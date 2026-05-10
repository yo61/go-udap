package udap

import (
	"encoding/binary"
	"runtime"
	"testing"
)

// TestParseGetDataResponseClampsSizeHint locks in the fix for review
// finding #9. parseGetDataResponse uses the network-supplied count as
// a make(map, N) size hint. Without clamping, count=65535 from a
// crafted UDP packet allocates a ~1.5 MB bucket array per response
// even when the body is too short to hold any items. The fix bounds
// the hint to the maximum item count that could fit in the payload.
//
// Pre-fix: ~1.5 MB allocated per call.
// Post-fix: well under 10 KB per call (~100× margin survives GC noise).
func TestParseGetDataResponseClampsSizeHint(t *testing.T) {
	payload := make([]byte, 6)
	binary.BigEndian.PutUint16(payload[:2], 0xFFFF)

	const iterations = 50

	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := 0; i < iterations; i++ {
		_, _ = parseGetDataResponse(payload)
	}
	runtime.ReadMemStats(&after)

	bytesPerCall := (after.TotalAlloc - before.TotalAlloc) / iterations
	const limit = 10 * 1024
	if bytesPerCall > limit {
		t.Errorf("parseGetDataResponse allocates %d bytes/call for count=0xFFFF; want < %d",
			bytesPerCall, limit)
	}
}

// TestParseGetDataResponseStillRejectsOversizedCount asserts the
// existing error path is preserved by the clamp — the function must
// still return an error when the declared count exceeds what the
// payload can hold.
func TestParseGetDataResponseStillRejectsOversizedCount(t *testing.T) {
	payload := make([]byte, 10)
	binary.BigEndian.PutUint16(payload[:2], 0xFFFF)

	if _, err := parseGetDataResponse(payload); err == nil {
		t.Fatalf("expected error for oversized count, got nil")
	}
}
