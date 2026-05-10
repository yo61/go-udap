package cli

import (
	"strings"
	"testing"
)

func TestE2EInfoPrintsDeviceMetadata(t *testing.T) {
	env := startMockEnv(t, 2)
	stdout, _, exitCode := env.runCLI(t, "info", "00:04:20:00:00:02", "--timeout", "500ms")
	if exitCode != 0 {
		t.Fatalf("exit code %d, want 0", exitCode)
	}
	for _, want := range []string{
		"00:04:20:00:00:02",
		"Squeezebox Receiver",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing %q; got:\n%s", want, stdout)
		}
	}
}

func TestE2EInfoMissingMACIsExitCodeTwo(t *testing.T) {
	env := startMockEnv(t, 1)
	_, _, exitCode := env.runCLI(t, "info", "aa:bb:cc:dd:ee:ff", "--timeout", "200ms")
	if exitCode != 2 {
		t.Errorf("exit code %d, want 2 (device not found)", exitCode)
	}
}
