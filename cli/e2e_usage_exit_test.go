package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// Invocations cobra rejects while parsing, before any subcommand runs,
// are usage errors and exit 2.
func TestE2ECobraUsageErrorsExitTwo(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"unknown flag", []string{"discover", "--flooble"}},
		{"unknown shorthand flag", []string{"discover", "-z"}},
		{"invalid duration value", []string{"discover", "--timeout", "soon"}},
		{"invalid integer value", []string{"discover", "--retries", "many"}},
		{"negative retries", []string{"discover", "--retries", "-1"}},
		{"missing MAC argument", []string{"info"}},
		{"too many arguments", []string{"reboot", "00:04:20:00:00:01", "extra"}},
		{"get without parameter names", []string{"get", "00:04:20:00:00:01"}},
		{"stray argument to argless command", []string{"interfaces", "extra"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := startMockEnv(t, 1)
			_, stderr, exitCode := env.runCLI(t, tc.args...)
			if exitCode != 2 {
				t.Errorf("exit code %d, want 2 (usage error); stderr:\n%s", exitCode, stderr)
			}
		})
	}
}

// Input the subcommand rejects before any UDAP traffic is a usage error
// and exits 2.
func TestE2EInvalidInputExitsTwo(t *testing.T) {
	dir := t.TempDir()
	missingConfig := filepath.Join(dir, "absent.ini")
	unknownKeyConfig := writeConfig(t, dir, "unknown-key.ini", "flooble = 1\n")
	noEqualsConfig := writeConfig(t, dir, "no-equals.ini", "hostname\n")
	cases := []struct {
		name string
		args []string
	}{
		{"info bad MAC", []string{"info", "not-a-mac"}},
		{"read bad MAC", []string{"read", "00:04:20"}},
		{"get bad MAC", []string{"get", "zz:zz:zz:zz:zz:zz", "hostname"}},
		{"set bad MAC", []string{"set", "not-a-mac", "--hostname", "x"}},
		{"reboot bad MAC", []string{"reboot", "not-a-mac"}},
		{"getip bad MAC", []string{"getip", "not-a-mac"}},
		{"get unknown parameter", []string{"get", "00:04:20:00:00:01", "flooble"}},
		{"set missing config file", []string{"set", "00:04:20:00:00:01", "--config", missingConfig}},
		{"set config unknown key", []string{"set", "00:04:20:00:00:01", "--config", unknownKeyConfig}},
		{"set config line without =", []string{"set", "00:04:20:00:00:01", "--config", noEqualsConfig}},
		{"set with nothing to set", []string{"set", "00:04:20:00:00:01"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := startMockEnv(t, 1)
			_, stderr, exitCode := env.runCLI(t, append(tc.args, "--timeout", "200ms")...)
			if exitCode != 2 {
				t.Errorf("exit code %d, want 2 (usage error); stderr:\n%s", exitCode, stderr)
			}
		})
	}
}

func writeConfig(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}
