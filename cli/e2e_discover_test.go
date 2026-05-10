package cli

import (
	"strings"
	"testing"
)

func TestE2EDiscoverListsAllMACs(t *testing.T) {
	env := startMockEnv(t, 3)
	_ = env

	stdout, _, exitCode := env.runCLI(t, "discover", "--timeout", "500ms")
	if exitCode != 0 {
		t.Fatalf("exit code %d, want 0", exitCode)
	}
	for _, mac := range []string{
		"00:04:20:00:00:01",
		"00:04:20:00:00:02",
		"00:04:20:00:00:03",
	} {
		if !strings.Contains(stdout, mac) {
			t.Errorf("stdout missing MAC %s; got:\n%s", mac, stdout)
		}
	}
}

func TestE2EDiscoverWithInfoIncludesMetadata(t *testing.T) {
	env := startMockEnv(t, 1)
	stdout, _, exitCode := env.runCLI(t, "discover", "--info", "--timeout", "500ms")
	if exitCode != 0 {
		t.Fatalf("exit code %d, want 0", exitCode)
	}
	if !strings.Contains(stdout, "00:04:20:00:00:01") {
		t.Errorf("stdout missing MAC; got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Squeezebox") {
		t.Errorf("stdout missing model name; got:\n%s", stdout)
	}
}
