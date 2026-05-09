package testhelper_test

import (
	"context"
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
