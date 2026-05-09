package udap

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestDeviceMapConcurrentAccess locks in the fix for the data race
// between the discovery listener (writing to Client.devices on every
// received UDAP response) and CLI callers polling GetDevice /
// ListDevices / GetDevices (reading the same map). Pre-fix, this
// would trip Go's runtime "fatal error: concurrent map read and map
// write" with -race; post-fix it must run cleanly.
//
// Run via `go test -race ./udap/ -run TestDeviceMapConcurrentAccess`.
func TestDeviceMapConcurrentAccess(t *testing.T) {
	c, err := newClientWithPort(0, NewNoOpLogger())
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	defer c.Close()

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Writer: simulates the discovery listener storing devices.
	wg.Add(1)
	go func() {
		defer wg.Done()
		i := 0
		for {
			select {
			case <-stop:
				return
			default:
				c.recordDevice(&Device{
					MAC:      fmt.Sprintf("00:04:20:%02x:%02x:%02x", i&0xff, (i>>8)&0xff, (i>>16)&0xff),
					LastSeen: time.Now(),
				})
				i++
			}
		}
	}()

	// Three concurrent readers, mirroring the three exported lookups.
	for _, fn := range []func(){
		func() { _ = c.GetDevice("00:04:20:16:05:8f") },
		func() { _ = c.ListDevices() },
		func() { _ = c.GetDevices() },
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
