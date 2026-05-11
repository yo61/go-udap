package mocksbr

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"go-udap/udap"
)

// TestDiscoveryFactoryMatchesFixture reproduces the captured factory
// discovery response byte-for-byte. The capture (real Squeezebox
// Receiver MAC 00:04:20:16:05:8f) was taken with a go-udap client
// using sequence=1 and an all-zeros source MAC.
func TestDiscoveryFactoryMatchesFixture(t *testing.T) {
	want := readFixture(t, "discovery-factory.bin")

	net := NewNetwork(0, udap.NewNoOpLogger())
	mac, err := net.Add(DeviceConfig{
		MAC: "00:04:20:16:05:8f",
		// Factory state: empty hostname.
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if mac != "00:04:20:16:05:8f" {
		t.Fatalf("MAC: got %s, want 00:04:20:16:05:8f", mac)
	}

	// Build a discovery request matching the client that produced the
	// capture: sequence=1, all-zeros src MAC, all-zeros dst MAC,
	// UCPMethod=AdvDisc.
	c := udap.NewClientWithTransport(NewMockTransport(net), udap.NewNoOpLogger())
	defer c.Close()
	req := c.CreateAdvancedDiscoveryPacket()

	replies := net.Receive(req)
	if len(replies) != 1 {
		t.Fatalf("expected 1 reply, got %d", len(replies))
	}
	got := replies[0]
	if !bytes.Equal(got, want) {
		t.Errorf("discovery response mismatch\n got %x\nwant %x", got, want)
	}
}

// TestDiscoveryConfiguredMatchesFixture reproduces the captured
// configured-device discovery response. The device has hostname
// "capture-test", which causes state="wait_slimserver".
func TestDiscoveryConfiguredMatchesFixture(t *testing.T) {
	want := readFixture(t, "discovery-configured.bin")

	net := NewNetwork(0, udap.NewNoOpLogger())
	if _, err := net.Add(DeviceConfig{
		MAC:  "00:04:20:16:05:8f",
		Name: "capture-test",
	}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	c := udap.NewClientWithTransport(NewMockTransport(net), udap.NewNoOpLogger())
	defer c.Close()
	req := c.CreateAdvancedDiscoveryPacket()

	replies := net.Receive(req)
	if len(replies) != 1 {
		t.Fatalf("expected 1 reply, got %d", len(replies))
	}
	if !bytes.Equal(replies[0], want) {
		t.Errorf("discovery response mismatch\n got %x\nwant %x", replies[0], want)
	}
}

// TestResetAckMatchesFixture reproduces the captured Reset ack:
// 27-byte header, UCPMethod=0x0004, Sequence=0x0002.
func TestResetAckMatchesFixture(t *testing.T) {
	want := readFixture(t, "reset-ack.bin")

	net := NewNetwork(0, udap.NewNoOpLogger())
	if _, err := net.Add(DeviceConfig{MAC: "00:04:20:16:05:8f"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	// The captured Sequence=0x0002 corresponds to the second client
	// request after discovery=0x0001. Our test client starts at 1, so
	// fire one no-op increment to align.
	c := udap.NewClientWithTransport(NewMockTransport(net), udap.NewNoOpLogger())
	defer c.Close()
	_ = c.CreateAdvancedDiscoveryPacket() // burns sequence=1
	dev := &udap.Device{MAC: udap.MustParseMAC("00:04:20:16:05:8f")}
	resetReq, err := c.CreateResetPacket(dev) // sequence=2
	if err != nil {
		t.Fatalf("CreateResetPacket: %v", err)
	}

	replies := net.Receive(resetReq)
	if len(replies) != 1 {
		t.Fatalf("expected 1 reply, got %d", len(replies))
	}
	if !bytes.Equal(replies[0], want) {
		t.Errorf("reset ack mismatch\n got %x\nwant %x", replies[0], want)
	}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "captures", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}
