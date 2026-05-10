package mocksbr

import (
	"context"
	"testing"
	"time"

	"go-udap/udap"
)

// TestGetAllDeviceConfigClearsStaleOffsetKeys locks in the fix for the
// last item from the review's "outstanding LOW findings": long-running
// consumers calling GetAllDeviceConfigWithContext repeatedly used to
// accumulate `offset_NNN` synthetic keys in device.Parameters
// indefinitely. parseGetDataResponse emits those keys for NVRAM
// offsets unknown to the udap.Parameters table; maps.Copy merged them
// in but never deleted stale entries from prior calls.
//
// Post-fix, GetAll clears any existing `offset_NNN` keys before
// merging the new response, so the map size is bounded by what the
// device actually returns on the most recent call.
func TestGetAllDeviceConfigClearsStaleOffsetKeys(t *testing.T) {
	net := NewNetwork(1, udap.NewNoOpLogger())
	transport := NewMockTransport(net)
	client := udap.NewClientWithTransport(transport, udap.NewNoOpLogger())
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := client.DiscoverDevicesWithContext(ctx); err != nil {
		t.Fatalf("Discover: %v", err)
	}
	dev := client.GetDevice("00:04:20:00:00:01")
	if dev == nil {
		t.Fatalf("device not discovered")
	}

	// Simulate a leftover offset_NNN entry from a prior call against a
	// real device whose firmware exposes an NVRAM offset the
	// udap.Parameters table doesn't know about.
	if dev.Parameters == nil {
		dev.Parameters = map[string]string{}
	}
	dev.Parameters["offset_999"] = "stale-from-previous-call"

	if err := client.GetAllDeviceConfigWithContext(ctx, dev); err != nil {
		t.Fatalf("GetAll: %v", err)
	}

	if got, present := dev.Parameters["offset_999"]; present {
		t.Errorf("stale offset_999=%q survived a fresh GetAll", got)
	}
}
