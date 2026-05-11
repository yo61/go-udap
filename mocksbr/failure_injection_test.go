package mocksbr

import (
	"bytes"
	"context"
	"testing"
	"time"

	"go-udap/udap"
)

// Failure injection — DeviceConfig.Unreachable

func TestUnreachableDeviceDropsUnicastPackets(t *testing.T) {
	net := NewNetwork(0, udap.NewNoOpLogger())
	mac, err := net.Add(DeviceConfig{MAC: "aa:bb:cc:dd:ee:01", Unreachable: true})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	c := udap.NewClientWithTransport(NewMockTransport(net), udap.NewNoOpLogger())
	defer c.Close()
	dev := &udap.Device{MAC: mac}
	getPkt, err := c.CreateGetDataPacket(dev, []string{"hostname"})
	if err != nil {
		t.Fatalf("CreateGetDataPacket: %v", err)
	}

	if replies := net.Receive(getPkt); len(replies) != 0 {
		t.Errorf("expected 0 replies from unreachable device, got %d", len(replies))
	}
}

func TestUnreachableDeviceSkippedInDiscoveryFanout(t *testing.T) {
	net := NewNetwork(0, udap.NewNoOpLogger())
	if _, err := net.Add(DeviceConfig{MAC: "aa:bb:cc:dd:ee:01"}); err != nil {
		t.Fatalf("Add reachable: %v", err)
	}
	if _, err := net.Add(DeviceConfig{MAC: "aa:bb:cc:dd:ee:02", Unreachable: true}); err != nil {
		t.Fatalf("Add unreachable: %v", err)
	}
	if _, err := net.Add(DeviceConfig{MAC: "aa:bb:cc:dd:ee:03"}); err != nil {
		t.Fatalf("Add reachable: %v", err)
	}

	c := udap.NewClientWithTransport(NewMockTransport(net), udap.NewNoOpLogger())
	defer c.Close()
	disc := c.CreateAdvancedDiscoveryPacket()

	replies := net.Receive(disc)
	if len(replies) != 2 {
		t.Fatalf("expected 2 replies (one per reachable device), got %d", len(replies))
	}
	for i, reply := range replies {
		pkt, _, err := udap.ParsePacket(reply)
		if err != nil {
			t.Fatalf("reply %d: parse: %v", i, err)
		}
		gotMAC := formatMAC(pkt.SrcAddress)
		if gotMAC == "aa:bb:cc:dd:ee:02" {
			t.Errorf("unreachable device %s replied to discovery", gotMAC)
		}
	}
}

// Failure injection — DeviceConfig.Slow

func TestSlowDeviceReplyDelayedByConfiguredDuration(t *testing.T) {
	const delay = 80 * time.Millisecond
	net := NewNetwork(0, udap.NewNoOpLogger())
	mac, err := net.Add(DeviceConfig{MAC: "aa:bb:cc:dd:ee:01", Slow: delay})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	transport := NewMockTransport(net)
	client := udap.NewClientWithTransport(transport, udap.NewNoOpLogger())
	defer client.Close()

	dev := &udap.Device{MAC: mac}
	getPkt, err := client.CreateGetDataPacket(dev, []string{"hostname"})
	if err != nil {
		t.Fatalf("CreateGetDataPacket: %v", err)
	}

	start := time.Now()
	if err := transport.Send(getPkt); err != nil {
		t.Fatalf("Send: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if _, _, err := transport.Recv(ctx); err != nil {
		t.Fatalf("Recv: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed < delay {
		t.Errorf("reply arrived in %v, expected at least %v (Slow delay)", elapsed, delay)
	}
	if elapsed > delay+200*time.Millisecond {
		t.Errorf("reply arrived in %v, expected ~%v (Slow delay + small skew)", elapsed, delay)
	}
}

func TestSlowDeviceTimesOutWhenDeadlineShorter(t *testing.T) {
	const slow = 200 * time.Millisecond
	const ctxBudget = 40 * time.Millisecond
	net := NewNetwork(0, udap.NewNoOpLogger())
	mac, err := net.Add(DeviceConfig{MAC: "aa:bb:cc:dd:ee:01", Slow: slow})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	client := udap.NewClientWithTransport(NewMockTransport(net), udap.NewNoOpLogger())
	defer client.Close()

	dev := &udap.Device{MAC: mac}
	ctx, cancel := context.WithTimeout(context.Background(), ctxBudget)
	defer cancel()

	_, err = client.GetDeviceConfigWithContext(ctx, dev, []string{"hostname"})
	if err == nil {
		t.Fatalf("expected deadline-exceeded error, got nil")
	}
}

// Failure injection — DeviceConfig.FailOn

func TestFailOnGetReturnsErrorResponse(t *testing.T) {
	net := NewNetwork(0, udap.NewNoOpLogger())
	mac, err := net.Add(DeviceConfig{MAC: "aa:bb:cc:dd:ee:01", FailOn: []Op{OpGet}})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	client := udap.NewClientWithTransport(NewMockTransport(net), udap.NewNoOpLogger())
	defer client.Close()

	dev := &udap.Device{MAC: mac}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err = client.GetDeviceConfigWithContext(ctx, dev, []string{"hostname"})
	if err == nil {
		t.Fatalf("expected error response, got nil")
	}
}

func TestFailOnSetReturnsErrorResponseWithMessage(t *testing.T) {
	net := NewNetwork(0, udap.NewNoOpLogger())
	mac, err := net.Add(DeviceConfig{MAC: "aa:bb:cc:dd:ee:01", FailOn: []Op{OpSet}})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	client := udap.NewClientWithTransport(NewMockTransport(net), udap.NewNoOpLogger())
	defer client.Close()

	// Pre-load device.Parameters to skip the SetData read-modify-write
	// prelude that would otherwise fail first.
	dev := &udap.Device{MAC: mac, Parameters: map[string]string{"hostname": "x"}}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = client.SetDeviceConfigWithContext(ctx, dev, map[string]string{"hostname": "y"})
	if err == nil {
		t.Fatalf("expected SetDeviceConfig to return error, got nil")
	}
}

// TestFailOnSetDoesNotMutateCachedParameters verifies the aggregate
// invariant on Device: when SetDeviceConfigWithContext fails (here, the
// device replies with MethodError), the in-memory device.Parameters
// cache must not reflect the attempted write. Before the fix,
// device.Parameters was updated in the same loop that built the wire
// payload, so a failing round-trip left the cache showing values that
// were never written to NVRAM. A subsequent `read` against the cached
// device would then return phantom values.
func TestFailOnSetDoesNotMutateCachedParameters(t *testing.T) {
	net := NewNetwork(0, udap.NewNoOpLogger())
	mac, err := net.Add(DeviceConfig{MAC: "aa:bb:cc:dd:ee:01", FailOn: []Op{OpSet}})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	client := udap.NewClientWithTransport(NewMockTransport(net), udap.NewNoOpLogger())
	defer client.Close()

	const original = "before"
	const attempted = "after"
	dev := &udap.Device{MAC: mac, Parameters: map[string]string{"hostname": original}}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := client.SetDeviceConfigWithContext(ctx, dev, map[string]string{"hostname": attempted}); err == nil {
		t.Fatalf("expected SetDeviceConfig to return error, got nil")
	}
	if got := dev.Parameters["hostname"]; got != original {
		t.Errorf("device.Parameters[hostname] = %q after failed write, want %q (phantom write regression)", got, original)
	}
}

// TestFailOnResetReturnsErrorThroughClient verifies that a MethodError
// reply to a Reset request propagates as an error from
// ResetDeviceWithContext. Before review finding #10's fix, the client
// silently accepted any reply method as success.
func TestFailOnResetReturnsErrorThroughClient(t *testing.T) {
	net := NewNetwork(0, udap.NewNoOpLogger())
	mac, err := net.Add(DeviceConfig{MAC: "aa:bb:cc:dd:ee:01", FailOn: []Op{OpReset}})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	client := udap.NewClientWithTransport(NewMockTransport(net), udap.NewNoOpLogger())
	defer client.Close()

	dev := &udap.Device{MAC: mac}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = client.ResetDeviceWithContext(ctx, dev)
	if err == nil {
		t.Fatalf("expected ResetDeviceWithContext to return error, got nil")
	}
}

func TestFailOnResetEmitsErrorResponseOnWire(t *testing.T) {
	// Verifies the wire-level response only. The udap.Client's
	// ResetDeviceWithContext currently treats any reply as success
	// (bug #10 in the review); once that's fixed, this can be promoted
	// to an end-to-end test through the client.
	net := NewNetwork(0, udap.NewNoOpLogger())
	mac, err := net.Add(DeviceConfig{MAC: "aa:bb:cc:dd:ee:01", FailOn: []Op{OpReset}})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	c := udap.NewClientWithTransport(NewMockTransport(net), udap.NewNoOpLogger())
	defer c.Close()
	dev := &udap.Device{MAC: mac}
	resetPkt, err := c.CreateResetPacket(dev)
	if err != nil {
		t.Fatalf("CreateResetPacket: %v", err)
	}

	replies := net.Receive(resetPkt)
	if len(replies) != 1 {
		t.Fatalf("expected 1 reply, got %d", len(replies))
	}
	pkt, _, err := udap.ParsePacket(replies[0])
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if pkt.UCPMethod != udap.MethodError {
		t.Errorf("UCPMethod=0x%04x, want 0x%04x (MethodError)", pkt.UCPMethod, udap.MethodError)
	}
}

// Failure injection — DeviceConfig.Malformed

func TestMalformedOversizedCountIsRejectedByClient(t *testing.T) {
	net := NewNetwork(0, udap.NewNoOpLogger())
	mac, err := net.Add(DeviceConfig{
		MAC:       "aa:bb:cc:dd:ee:01",
		Malformed: MalformedOversizedCount,
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	client := udap.NewClientWithTransport(NewMockTransport(net), udap.NewNoOpLogger())
	defer client.Close()

	dev := &udap.Device{MAC: mac}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err = client.GetDeviceConfigWithContext(ctx, dev, []string{"hostname"})
	if err == nil {
		t.Fatalf("expected error from malformed GetData response, got nil")
	}
}

func TestMalformedLengthExceedsPayloadIsRejectedByClient(t *testing.T) {
	net := NewNetwork(0, udap.NewNoOpLogger())
	mac, err := net.Add(DeviceConfig{
		MAC:       "aa:bb:cc:dd:ee:01",
		Malformed: MalformedLengthExceedsPayload,
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	client := udap.NewClientWithTransport(NewMockTransport(net), udap.NewNoOpLogger())
	defer client.Close()

	dev := &udap.Device{MAC: mac}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err = client.GetDeviceConfigWithContext(ctx, dev, []string{"hostname"})
	if err == nil {
		t.Fatalf("expected error from malformed GetData response, got nil")
	}
}

func TestMalformedUnknownMethodIsRejectedByClient(t *testing.T) {
	net := NewNetwork(0, udap.NewNoOpLogger())
	mac, err := net.Add(DeviceConfig{
		MAC:       "aa:bb:cc:dd:ee:01",
		Malformed: MalformedUnknownMethod,
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	client := udap.NewClientWithTransport(NewMockTransport(net), udap.NewNoOpLogger())
	defer client.Close()

	dev := &udap.Device{MAC: mac}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err = client.GetDeviceConfigWithContext(ctx, dev, []string{"hostname"})
	if err == nil {
		t.Fatalf("expected error for unknown UCP method, got nil")
	}
}

// MockTransport.InjectReply — spoofing helper for source-validation
// tests (lays the groundwork for the future fix to review finding #6).

func TestMockTransportInjectReplyUsesProvidedSrc(t *testing.T) {
	net := NewNetwork(0, udap.NewNoOpLogger())
	transport := NewMockTransport(net)
	defer transport.Close()

	want := []byte{0xde, 0xad, 0xbe, 0xef}
	transport.InjectReply(want, "192.168.1.99")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	got, src, err := transport.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("got bytes %x, want %x", got, want)
	}
	if src != "192.168.1.99" {
		t.Errorf("got src %q, want %q", src, "192.168.1.99")
	}
}

func TestMockTransportNetworkRepliesUseExtractedMAC(t *testing.T) {
	// Ensures the existing path (network.Receive → enqueue) keeps
	// surfacing the reply's source MAC, so InjectReply doesn't break
	// non-spoofed reply delivery.
	net := NewNetwork(1, udap.NewNoOpLogger())
	transport := NewMockTransport(net)
	defer transport.Close()

	c := udap.NewClientWithTransport(transport, udap.NewNoOpLogger())
	if err := transport.Send(c.CreateAdvancedDiscoveryPacket()); err != nil {
		t.Fatalf("Send: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, src, err := transport.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if src != "00:04:20:00:00:01" {
		t.Errorf("got src %q, want %q", src, "00:04:20:00:00:01")
	}
}

func TestFailOnDiscoverDevicesAreSkipped(t *testing.T) {
	net := NewNetwork(0, udap.NewNoOpLogger())
	if _, err := net.Add(DeviceConfig{MAC: "aa:bb:cc:dd:ee:01"}); err != nil {
		t.Fatalf("Add reachable: %v", err)
	}
	if _, err := net.Add(DeviceConfig{MAC: "aa:bb:cc:dd:ee:02", FailOn: []Op{OpDiscover}}); err != nil {
		t.Fatalf("Add fail-discover: %v", err)
	}

	c := udap.NewClientWithTransport(NewMockTransport(net), udap.NewNoOpLogger())
	defer c.Close()
	disc := c.CreateAdvancedDiscoveryPacket()

	replies := net.Receive(disc)
	if len(replies) != 1 {
		t.Fatalf("expected 1 reply (FailOn=discover device skipped), got %d", len(replies))
	}
	pkt, _, err := udap.ParsePacket(replies[0])
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := formatMAC(pkt.SrcAddress); got != "aa:bb:cc:dd:ee:01" {
		t.Errorf("reply src MAC=%q, want %q", got, "aa:bb:cc:dd:ee:01")
	}
}
