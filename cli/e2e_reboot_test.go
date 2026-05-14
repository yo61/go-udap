package cli

import (
	"testing"
)

func TestE2ERebootSucceeds(t *testing.T) {
	env := startMockEnv(t, 1)
	_, _, exitCode := env.runCLI(t, "reboot", "00:04:20:00:00:01", "--timeout", "500ms")
	if exitCode != 0 {
		t.Fatalf("exit code %d, want 0", exitCode)
	}
}

// TestE2ERebootIsFireAndForget pins the post-skip-discover contract:
// reboot sends the UCP_METHOD_RESET packet and treats the absence of a
// reply as success, because the device legitimately can't ACK while
// it's rebooting. As a side-effect, a typo'd or non-existent MAC also
// returns exit 0 — the previous discover-first design caught that case
// with exit 2 but couldn't tell a successful reboot from a missing
// device once it got past discover. The trade-off was the price of
// making reboot work on configured Wi-Fi devices that don't respond to
// the discovery broadcast.
func TestE2ERebootIsFireAndForget(t *testing.T) {
	env := startMockEnv(t, 1)
	_, _, exitCode := env.runCLI(t, "reboot", "aa:bb:cc:dd:ee:ff", "--timeout", "200ms")
	if exitCode != 0 {
		t.Errorf("exit code %d, want 0 (fire-and-forget)", exitCode)
	}
}
