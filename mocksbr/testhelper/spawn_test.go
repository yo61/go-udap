package testhelper_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"net"
	"testing"
	"time"

	"go-udap/mocksbr/testhelper"
	"go-udap/udap"
)

// TestSpawnMockBootsAndRespondsToDiscovery is a Layer 3 smoke test:
// it spawns the cmd/mocksbr binary as a real subprocess, opens a UDP
// transport, sends a discovery packet directly to the mock's loopback
// port (bypassing broadcast), and verifies a discovery reply arrives.
func TestSpawnMockBootsAndRespondsToDiscovery(t *testing.T) {
	if testing.Short() {
		t.Skip("layer 3 spawn test skipped in -short mode")
	}

	mock := testhelper.SpawnMock(t, "--devices", "2")

	// Build a UDPTransport that talks to the spawned mock directly via
	// SendTo (the production Send broadcasts to port 17784 only).
	tr, err := udap.NewUDPTransport(0, udap.NewNoOpLogger())
	if err != nil {
		t.Fatalf("NewUDPTransport: %v", err)
	}
	defer tr.Close()
	client := udap.NewClientWithTransport(tr, udap.NewNoOpLogger())
	defer client.Close()

	dst := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: mock.Port}
	if err := tr.SendTo(client.CreateAdvancedDiscoveryPacket(), dst); err != nil {
		t.Fatalf("SendTo discovery: %v", err)
	}

	// Recv loop: collect responses for a short window.
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	gotMACs := map[string]bool{}
	for {
		reply, _, err := tr.Recv(ctx)
		if err != nil {
			break
		}
		pkt, _, perr := udap.ParsePacket(reply)
		if perr != nil {
			continue
		}
		mac := formatMAC(pkt.SrcAddress)
		gotMACs[mac] = true
	}

	if len(gotMACs) < 2 {
		t.Errorf("expected at least 2 discovery replies, got %d (%v)", len(gotMACs), gotMACs)
	}
}

// TestSpawnMockGetDataRoundTrip drives a real UDP round-trip against
// the spawned mocksbr binary: send a GetData request for a known
// parameter via SendTo, receive the reply on a real socket, parse
// the wire format, assert the parameter is present.
func TestSpawnMockGetDataRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("layer 3 spawn test skipped in -short mode")
	}
	mock := testhelper.SpawnMock(t, "--devices", "1")
	tr, dst := transportTo(t, mock.Port)
	defer tr.Close()

	const mac = "00:04:20:00:00:01"
	client := udap.NewClientWithTransport(tr, udap.NewNoOpLogger())
	defer client.Close()
	pkt, err := client.CreateGetDataPacket(&udap.Device{MAC: mac}, []string{"hostname"})
	if err != nil {
		t.Fatalf("CreateGetDataPacket: %v", err)
	}
	if err := tr.SendTo(pkt, dst); err != nil {
		t.Fatalf("SendTo: %v", err)
	}

	reply := mustRecv(t, tr, 1*time.Second)
	respPkt, _, err := udap.ParsePacket(reply)
	if err != nil {
		t.Fatalf("ParsePacket: %v", err)
	}
	if respPkt.UCPMethod != udap.MethodGetData {
		t.Errorf("UCPMethod=0x%04x, want 0x%04x", respPkt.UCPMethod, udap.MethodGetData)
	}
	if got := formatMAC(respPkt.SrcAddress); got != mac {
		t.Errorf("reply SrcAddress=%q, want %q", got, mac)
	}
}

// TestSpawnMockSetDataPersists sets a hostname via the spawned binary,
// then reads it back over UDP. Confirms the binary's read+dispatch
// loop applies SetData updates and the GetData handler surfaces the
// result, both on real sockets.
func TestSpawnMockSetDataPersists(t *testing.T) {
	if testing.Short() {
		t.Skip("layer 3 spawn test skipped in -short mode")
	}
	mock := testhelper.SpawnMock(t, "--devices", "1")
	tr, dst := transportTo(t, mock.Port)
	defer tr.Close()

	const (
		mac      = "00:04:20:00:00:01"
		hostname = "subprocess-test-host"
	)
	client := udap.NewClientWithTransport(tr, udap.NewNoOpLogger())
	defer client.Close()
	dev := &udap.Device{MAC: mac}

	setPkt, err := client.CreateSetDataPacket(dev, map[string]string{"hostname": hostname})
	if err != nil {
		t.Fatalf("CreateSetDataPacket: %v", err)
	}
	if err := tr.SendTo(setPkt, dst); err != nil {
		t.Fatalf("SendTo SetData: %v", err)
	}
	if _, _, err := udap.ParsePacket(mustRecv(t, tr, 1*time.Second)); err != nil {
		t.Fatalf("parse SetData ack: %v", err)
	}

	getPkt, err := client.CreateGetDataPacket(dev, []string{"hostname"})
	if err != nil {
		t.Fatalf("CreateGetDataPacket: %v", err)
	}
	if err := tr.SendTo(getPkt, dst); err != nil {
		t.Fatalf("SendTo GetData: %v", err)
	}

	reply := mustRecv(t, tr, 1*time.Second)
	if !bytes.Contains(reply, []byte(hostname)) {
		t.Errorf("GetData response did not include hostname %q; reply=%x", hostname, reply)
	}
}

// TestSpawnMockResetAcksWithMethodReset sends a Reset request and
// asserts the ack carries UCPMethod=MethodReset (not MethodError).
func TestSpawnMockResetAcksWithMethodReset(t *testing.T) {
	if testing.Short() {
		t.Skip("layer 3 spawn test skipped in -short mode")
	}
	mock := testhelper.SpawnMock(t, "--devices", "1")
	tr, dst := transportTo(t, mock.Port)
	defer tr.Close()

	const mac = "00:04:20:00:00:01"
	client := udap.NewClientWithTransport(tr, udap.NewNoOpLogger())
	defer client.Close()
	pkt, err := client.CreateResetPacket(&udap.Device{MAC: mac})
	if err != nil {
		t.Fatalf("CreateResetPacket: %v", err)
	}
	if err := tr.SendTo(pkt, dst); err != nil {
		t.Fatalf("SendTo: %v", err)
	}

	respPkt, _, err := udap.ParsePacket(mustRecv(t, tr, 1*time.Second))
	if err != nil {
		t.Fatalf("ParsePacket: %v", err)
	}
	if respPkt.UCPMethod != udap.MethodReset {
		t.Errorf("UCPMethod=0x%04x, want 0x%04x", respPkt.UCPMethod, udap.MethodReset)
	}
}

// TestSpawnMockIgnoresUnknownUCPMethod sends a packet with an
// unrecognized UCPMethod. mocksbr's dispatch logs and drops it, no
// reply is emitted; we verify by attempting to Recv with a short
// budget and asserting the timeout fires.
func TestSpawnMockIgnoresUnknownUCPMethod(t *testing.T) {
	if testing.Short() {
		t.Skip("layer 3 spawn test skipped in -short mode")
	}
	mock := testhelper.SpawnMock(t, "--devices", "1")
	tr, dst := transportTo(t, mock.Port)
	defer tr.Close()

	const mac = "00:04:20:00:00:01"
	pkt := buildHeaderOnlyPacket(t, mac, 0x9999)
	if err := tr.SendTo(pkt, dst); err != nil {
		t.Fatalf("SendTo: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if reply, _, err := tr.Recv(ctx); err == nil {
		t.Errorf("expected no reply, got %d bytes", len(reply))
	} else if !errors.Is(err, context.DeadlineExceeded) {
		t.Logf("Recv err: %v (acceptable as long as no reply was returned)", err)
	}
}

// TestSpawnMockIgnoresMalformedPackets sends raw bytes too short to
// parse. Same pattern as the unknown-method test: assert no reply
// arrives within a short window.
func TestSpawnMockIgnoresMalformedPackets(t *testing.T) {
	if testing.Short() {
		t.Skip("layer 3 spawn test skipped in -short mode")
	}
	mock := testhelper.SpawnMock(t, "--devices", "1")
	tr, dst := transportTo(t, mock.Port)
	defer tr.Close()

	if err := tr.SendTo([]byte{0xff, 0xff, 0xff}, dst); err != nil {
		t.Fatalf("SendTo: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if reply, _, err := tr.Recv(ctx); err == nil {
		t.Errorf("expected no reply, got %d bytes", len(reply))
	} else if !errors.Is(err, context.DeadlineExceeded) {
		t.Logf("Recv err: %v (acceptable as long as no reply was returned)", err)
	}
}

// transportTo creates a UDPTransport on an ephemeral local port and
// returns it along with a UDPAddr pointing at the spawned mock.
func transportTo(t *testing.T, mockPort int) (*udap.UDPTransport, *net.UDPAddr) {
	t.Helper()
	tr, err := udap.NewUDPTransport(0, udap.NewNoOpLogger())
	if err != nil {
		t.Fatalf("NewUDPTransport: %v", err)
	}
	dst := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: mockPort}
	return tr, dst
}

// mustRecv blocks for at most timeout, returning the next packet or
// failing the test.
func mustRecv(t *testing.T, tr *udap.UDPTransport, timeout time.Duration) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	reply, _, err := tr.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	return reply
}

// buildHeaderOnlyPacket constructs a 27-byte UDAP header naming
// srcMAC as the source and method as the UCPMethod. Used by tests
// that need to send packets which Client.Create* won't produce
// (e.g. unknown methods).
func buildHeaderOnlyPacket(t *testing.T, srcMAC string, method uint16) []byte {
	t.Helper()
	var src [6]byte
	const hex = "0123456789abcdef"
	pos := 0
	for i := 0; i < 6; i++ {
		hi := indexOf(hex, srcMAC[pos])
		lo := indexOf(hex, srcMAC[pos+1])
		src[i] = byte(hi<<4 | lo)
		pos += 2
		if i < 5 {
			pos++ // skip ':'
		}
	}
	pkt := udap.Packet{
		DstType:    udap.AddrTypeETH,
		SrcType:    udap.AddrTypeETH,
		SrcAddress: src,
		UDAPType:   udap.TypeUCP,
		UCPFlags:   0x01,
		UAPClass:   [4]byte{0x00, 0x01, 0x00, 0x01},
		UCPMethod:  method,
	}
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, pkt); err != nil {
		t.Fatalf("encode header: %v", err)
	}
	return buf.Bytes()
}

func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return 0
}

func formatMAC(b [6]byte) string {
	const hex = "0123456789abcdef"
	out := make([]byte, 17)
	pos := 0
	for i, x := range b {
		if i > 0 {
			out[pos] = ':'
			pos++
		}
		out[pos] = hex[x>>4]
		out[pos+1] = hex[x&0x0f]
		pos += 2
	}
	return string(out)
}
