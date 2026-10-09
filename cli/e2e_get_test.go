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

func TestE2EGetAliasReturnsTheCanonicalParametersValue(t *testing.T) {
	env := startMockEnv(t, 1)
	if _, _, exit := env.runCLI(t, "set", "00:04:20:00:00:01",
		"--server-address", "10.0.0.7", "--timeout", "500ms"); exit != 0 {
		t.Fatalf("set exit %d", exit)
	}
	cases := map[string]struct {
		args []string
		want string
	}{
		"single": {[]string{"squeezecenter_address"}, "10.0.0.7\n"},
		"multi": {
			[]string{"squeezecenter_address", "lan_ip_mode"},
			"squeezecenter_address=10.0.0.7\nlan_ip_mode=1\n",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			args := append([]string{"get", "00:04:20:00:00:01"}, tc.args...)
			stdout, stderr, exitCode := env.runCLI(t, append(args, "--timeout", "500ms")...)
			if exitCode != 0 {
				t.Fatalf("exit code %d, want 0; stderr:\n%s", exitCode, stderr)
			}
			if stdout != tc.want {
				t.Errorf("stdout: got %q, want %q", stdout, tc.want)
			}
		})
	}
}
