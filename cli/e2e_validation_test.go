package cli

import (
	"strings"
	"testing"
)

// TestE2ESetRejectsInvalidIPViaFlag locks in the fix for review
// finding #2. Per-flag values for `set` previously bypassed
// udap.ValidateParameter (only the INI/stdin path validated). Combined
// with finding #3 — CreateSetDataPacket silently zero-filling on parse
// failure — a typo like --server-address 192.168.1.x would write
// 0.0.0.0 to NVRAM with only a stderr warning.
//
// Post-fix, the CLI rejects the bad value before any packet is built.
func TestE2ESetRejectsInvalidIPViaFlag(t *testing.T) {
	env := startMockEnv(t, 1)
	_, stderr, exitCode := env.runCLI(t, "set", "00:04:20:00:00:01",
		"--server-address", "192.168.1.x",
		"--timeout", "500ms")
	// Exit 1 is usage error (CLI rejects up front, before UDP I/O);
	// exit 2 would mean the rejection only happens at the udap layer
	// after a wasted discovery + read round-trip.
	if exitCode != 1 {
		t.Fatalf("expected exit 1 (usage error), got %d", exitCode)
	}
	if !strings.Contains(stderr, "server-address") && !strings.Contains(stderr, "server_address") {
		t.Errorf("stderr should name the offending flag; got:\n%s", stderr)
	}
}

func TestE2ESetRejectsInvalidUint8ViaFlag(t *testing.T) {
	env := startMockEnv(t, 1)
	_, _, exitCode := env.runCLI(t, "set", "00:04:20:00:00:01",
		"--wireless-channel", "999",
		"--timeout", "500ms")
	if exitCode != 1 {
		t.Fatalf("expected exit 1 (usage error), got %d", exitCode)
	}
}
