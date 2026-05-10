package cli

import (
	"strings"
	"testing"
)

func TestE2EReadDefaultSkipsFactoryDefaults(t *testing.T) {
	env := startMockEnv(t, 1)
	stdout, _, exitCode := env.runCLI(t, "read", "00:04:20:00:00:01", "--timeout", "500ms")
	if exitCode != 0 {
		t.Fatalf("exit code %d, want 0", exitCode)
	}
	// A factory-default device returns no non-default values, so read
	// (without --all) prints an empty document — but never an error.
	if strings.Contains(stdout, "offset_") {
		t.Errorf("default read should not include offset_NNN entries; got:\n%s", stdout)
	}
}

func TestE2EReadAllIncludesEverything(t *testing.T) {
	env := startMockEnv(t, 1)
	stdout, _, exitCode := env.runCLI(t, "read", "00:04:20:00:00:01", "--all", "--timeout", "500ms")
	if exitCode != 0 {
		t.Fatalf("exit code %d, want 0", exitCode)
	}
	// --all surfaces every recognised parameter; a few well-known ones
	// must appear regardless of value.
	for _, want := range []string{"hostname", "interface"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("--all output missing parameter %q; got:\n%s", want, stdout)
		}
	}
}

func TestE2EReadAfterSetSurfacesNewValue(t *testing.T) {
	env := startMockEnv(t, 1)
	if _, _, exit := env.runCLI(t, "set", "00:04:20:00:00:01",
		"--server-address", "192.168.1.42", "--timeout", "500ms"); exit != 0 {
		t.Fatalf("set exit %d", exit)
	}
	stdout, _, exitCode := env.runCLI(t, "read", "00:04:20:00:00:01", "--timeout", "500ms")
	if exitCode != 0 {
		t.Fatalf("read exit %d", exitCode)
	}
	if !strings.Contains(stdout, "192.168.1.42") {
		t.Errorf("read after set missing 192.168.1.42; got:\n%s", stdout)
	}
}
