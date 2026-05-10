package udap

import (
	"sync"
	"testing"
	"time"
)

// TestSequenceCounterConcurrentAccess locks in the fix for the data
// race between any two goroutines that build packets simultaneously.
// Pre-fix, the read of c.sequence in createUdapPacket and the
// post-increment c.sequence++ both run unsynchronized; -race trips on
// the unprotected uint32. Post-fix (atomic.AddUint32) it must run
// cleanly.
//
// Run via `go test -race ./udap/ -run TestSequenceCounterConcurrentAccess`.
func TestSequenceCounterConcurrentAccess(t *testing.T) {
	c, err := newClientWithPort(0, NewNoOpLogger())
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	defer c.Close()

	dev := &Device{MAC: "00:04:20:00:00:01"}

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Three concurrent packet-builders, mirroring the public Create*
	// surface. Each one independently reads + writes c.sequence. The
	// packet/error returns are intentionally discarded — this test
	// only exercises the counter, not the packet contents.
	for _, fn := range []func(){
		func() { _ = c.CreateAdvancedDiscoveryPacket() },
		func() { _, _ = c.CreateGetDataPacket(dev, []string{"hostname"}) },
		func() { _, _ = c.CreateResetPacket(dev) },
	} {
		fn := fn
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					fn()
				}
			}
		}()
	}

	time.Sleep(100 * time.Millisecond)
	close(stop)
	wg.Wait()
}
