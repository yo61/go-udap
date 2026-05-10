package cli

import (
	"strings"
	"testing"
)

func TestE2EGetSingleParamPrintsBareValue(t *testing.T) {
	env := startMockEnv(t, 1)
	// Set a known value, then get it back.
	if _, _, exit := env.runCLI(t, "set", "00:04:20:00:00:01",
		"--server-address", "10.0.0.7", "--timeout", "500ms"); exit != 0 {
		t.Fatalf("set exit %d", exit)
	}
	stdout, _, exitCode := env.runCLI(t, "get", "00:04:20:00:00:01", "server_address",
		"--timeout", "500ms")
	if exitCode != 0 {
		t.Fatalf("exit code %d, want 0", exitCode)
	}
	got := strings.TrimSpace(stdout)
	if got != "10.0.0.7" {
		t.Errorf("got %q, want %q", got, "10.0.0.7")
	}
}

func TestE2EGetMultipleParamsPrintsKeyEqValue(t *testing.T) {
	env := startMockEnv(t, 1)
	if _, _, exit := env.runCLI(t, "set", "00:04:20:00:00:01",
		"--server-address", "10.0.0.7",
		"--hostname", "my-sbr",
		"--timeout", "500ms"); exit != 0 {
		t.Fatalf("set exit %d", exit)
	}
	stdout, _, exitCode := env.runCLI(t, "get", "00:04:20:00:00:01",
		"server_address", "hostname", "--timeout", "500ms")
	if exitCode != 0 {
		t.Fatalf("exit code %d, want 0", exitCode)
	}
	for _, want := range []string{"server_address=10.0.0.7", "hostname=my-sbr"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout missing %q; got:\n%s", want, stdout)
		}
	}
}
