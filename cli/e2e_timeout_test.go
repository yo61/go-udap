package cli

import (
	"testing"
	"time"

	"go-udap/mocksbr"
)

// TestE2ETotalWallClockBoundedByTimeoutFlag locks in the fix for review
// finding #4. Each device-targeted CLI command used to call
// discoverAndFind with the user's --timeout, then create a *fresh*
// context.WithTimeout with the same duration for the actual
// operation. Net result: total wall-clock could reach 2× --timeout
// (or 3× for `set --reboot`).
//
// With a 900 ms Slow device and --timeout 1s:
//   - Pre-fix: discover ~900 ms, GetData fresh 1 s ctx ~900 ms → ~1.8 s.
//   - Post-fix: shared 1 s ctx; ~900 ms in discover leaves ~100 ms for
//     GetData, which times out → total ~1 s.
//
// The bound below (1.4 s) admits the post-fix timing comfortably while
// rejecting the pre-fix double budget.
func TestE2ETotalWallClockBoundedByTimeoutFlag(t *testing.T) {
	env := startMockEnv(t, 0)
	if _, err := env.network.Add(mocksbr.DeviceConfig{
		MAC:  "aa:bb:cc:dd:ee:01",
		Slow: 900 * time.Millisecond,
	}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	const flagTimeout = 1 * time.Second
	const slack = 400 * time.Millisecond

	start := time.Now()
	_, _, _ = env.runCLI(t, "read", "aa:bb:cc:dd:ee:01", "--timeout", flagTimeout.String())
	elapsed := time.Since(start)

	if elapsed > flagTimeout+slack {
		t.Errorf("read took %v with --timeout %v; expected <= %v",
			elapsed, flagTimeout, flagTimeout+slack)
	}
}
