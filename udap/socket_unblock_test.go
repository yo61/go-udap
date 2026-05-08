package udap

import (
	"context"
	"testing"
	"time"
)

// TestClientCloseUnblocksAfterDiscovery is a regression test for the hang
// where (*net.UDPConn).File() in enableBroadcast switched the socket to
// blocking mode, after which Close() could not interrupt the listener
// goroutine's pending recvfrom on macOS. Symptom: `go-udap discover`
// printed "Listener goroutine did not exit in time" and never exited.
func TestClientCloseUnblocksAfterDiscovery(t *testing.T) {
	client, err := newClientWithPort(0, NewNoOpLogger())
	if err != nil {
		t.Fatalf("newClientWithPort: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_ = client.DiscoverDevicesWithContext(ctx)

	done := make(chan error, 1)
	go func() { done <- client.Close() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Close returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("client.Close() did not return within 2s; listener goroutine hung in recvfrom")
	}
}
