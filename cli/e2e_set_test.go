package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2ESetWithFlag(t *testing.T) {
	env := startMockEnv(t, 1)
	stdout, _, exitCode := env.runCLI(t, "set", "00:04:20:00:00:01",
		"--server-address", "10.20.30.40", "--timeout", "500ms")
	if exitCode != 0 {
		t.Fatalf("exit code %d, want 0", exitCode)
	}
	if !strings.Contains(stdout, "server_address=10.20.30.40") {
		t.Errorf("set echo missing applied value; got:\n%s", stdout)
	}
}

func TestE2ESetWithConfigFile(t *testing.T) {
	env := startMockEnv(t, 1)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.ini")
	cfgBody := "server_address = 192.168.10.1\nhostname = test-host\n"
	if err := os.WriteFile(cfgPath, []byte(cfgBody), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	stdout, _, exitCode := env.runCLI(t, "set", "00:04:20:00:00:01",
		"--config", cfgPath, "--timeout", "500ms")
	if exitCode != 0 {
		t.Fatalf("exit code %d, want 0", exitCode)
	}
	for _, want := range []string{"server_address=192.168.10.1", "hostname=test-host"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("set echo missing %q; got:\n%s", want, stdout)
		}
	}
}

func TestE2ESetWithPipedStdin(t *testing.T) {
	env := startMockEnv(t, 1)

	prevReader := stdinReader
	prevPiped := stdinIsPiped
	stdinReader = strings.NewReader("hostname = piped-host\n")
	stdinIsPiped = func() bool { return true }
	t.Cleanup(func() {
		stdinReader = prevReader
		stdinIsPiped = prevPiped
	})

	stdout, _, exitCode := env.runCLI(t, "set", "00:04:20:00:00:01", "--timeout", "500ms")
	if exitCode != 0 {
		t.Fatalf("exit code %d, want 0", exitCode)
	}
	if !strings.Contains(stdout, "hostname=piped-host") {
		t.Errorf("set echo missing piped value; got:\n%s", stdout)
	}
}

func TestE2ESetFlagOverridesConfig(t *testing.T) {
	env := startMockEnv(t, 1)

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.ini")
	if err := os.WriteFile(cfgPath, []byte("server_address = 10.0.0.1\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	stdout, _, exitCode := env.runCLI(t, "set", "00:04:20:00:00:01",
		"--config", cfgPath,
		"--server-address", "10.0.0.99",
		"--timeout", "500ms")
	if exitCode != 0 {
		t.Fatalf("exit code %d, want 0", exitCode)
	}
	if !strings.Contains(stdout, "server_address=10.0.0.99") {
		t.Errorf("flag should override config; got:\n%s", stdout)
	}
	if strings.Contains(stdout, "10.0.0.1") {
		t.Errorf("config value leaked through despite flag override; got:\n%s", stdout)
	}
}

func TestE2ESetWithRebootTriggersReset(t *testing.T) {
	env := startMockEnv(t, 1)
	_, _, exitCode := env.runCLI(t, "set", "00:04:20:00:00:01",
		"--server-address", "10.0.0.5",
		"--reboot",
		"--timeout", "500ms")
	if exitCode != 0 {
		t.Fatalf("exit code %d, want 0", exitCode)
	}
	// We can't directly inspect mocksbr device state from here, but a
	// successful exit confirms both SetData and Reset acks were
	// processed by the client without error.
}
