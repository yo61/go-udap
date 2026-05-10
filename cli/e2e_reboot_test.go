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

func TestE2ERebootMissingMACIsExitTwo(t *testing.T) {
	env := startMockEnv(t, 1)
	_, _, exitCode := env.runCLI(t, "reboot", "aa:bb:cc:dd:ee:ff", "--timeout", "200ms")
	if exitCode != 2 {
		t.Errorf("exit code %d, want 2", exitCode)
	}
}
