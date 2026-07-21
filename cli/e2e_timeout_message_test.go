package cli

import (
	"strings"
	"testing"

	"go-udap/mocksbr"
)

// #110: a device-op timeout must surface as a plain-English message
// naming the command, the device, and the timeout — never Go's
// "context deadline exceeded" wrap chain. Each device (found via
// discovery) drops the operation request so the op times out while
// discovery still succeeds.

func TestE2EGetIPTimeoutMessageIsPlainEnglish(t *testing.T) {
	env := startMockEnv(t, 0)
	if _, err := env.network.Add(mocksbr.DeviceConfig{
		MAC:       "00:04:20:00:00:01",
		DropGetIP: true,
	}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	_, stderr, exit := env.runCLI(t, "getip", "00:04:20:00:00:01", "--timeout", "200ms")

	assertTimeoutMessage(t, "getip", "00:04:20:00:00:01", "200ms", stderr, exit)
}

func TestE2EReadTimeoutMessageIsPlainEnglish(t *testing.T) {
	env := startMockEnv(t, 0)
	if _, err := env.network.Add(mocksbr.DeviceConfig{
		MAC:         "00:04:20:00:00:01",
		DropGetData: true,
	}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	_, stderr, exit := env.runCLI(t, "read", "00:04:20:00:00:01", "--timeout", "200ms")

	assertTimeoutMessage(t, "read", "00:04:20:00:00:01", "200ms", stderr, exit)
}

func TestE2EGetTimeoutMessageIsPlainEnglish(t *testing.T) {
	env := startMockEnv(t, 0)
	if _, err := env.network.Add(mocksbr.DeviceConfig{
		MAC:         "00:04:20:00:00:01",
		DropGetData: true,
	}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	_, stderr, exit := env.runCLI(t, "get", "00:04:20:00:00:01", "server_address", "--timeout", "200ms")

	assertTimeoutMessage(t, "get", "00:04:20:00:00:01", "200ms", stderr, exit)
}

func TestE2ESetTimeoutMessageIsPlainEnglish(t *testing.T) {
	env := startMockEnv(t, 0)
	if _, err := env.network.Add(mocksbr.DeviceConfig{
		MAC:         "00:04:20:00:00:01",
		DropGetData: true,
	}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	_, stderr, exit := env.runCLI(t, "set", "00:04:20:00:00:01", "--server-address", "192.168.1.5", "--timeout", "200ms")

	assertTimeoutMessage(t, "set", "00:04:20:00:00:01", "200ms", stderr, exit)
}

// assertTimeoutMessage checks the plain-English form is present, the
// Go internals are absent, and the exit code is 2 (operation failure).
func assertTimeoutMessage(t *testing.T, op, mac, timeout, stderr string, exit int) {
	t.Helper()
	want := op + ": no reply from " + mac + " within " + timeout
	if !strings.Contains(stderr, want) {
		t.Errorf("stderr missing %q; got:\n%s", want, stderr)
	}
	if strings.Contains(stderr, "context deadline exceeded") {
		t.Errorf("stderr leaks Go internals; got:\n%s", stderr)
	}
	if exit != 2 {
		t.Errorf("exit code %d, want 2", exit)
	}
}
