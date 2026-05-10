package cli

import (
	"strings"
	"testing"

	"go-udap/mocksbr"
)

// TestE2ESetErrorsWhenPriorReadFails locks in the fix for review
// finding #1. SetDeviceConfigWithContext does a read-modify-write to
// preserve NVRAM regions the caller didn't explicitly set; before the
// fix, if that prelude read failed, the function logged a warning and
// proceeded with only the user's overrides — exactly the corrupt
// state the read was supposed to prevent. Post-fix, the read failure
// propagates, set exits non-zero, and no SetData is sent.
func TestE2ESetErrorsWhenPriorReadFails(t *testing.T) {
	env := startMockEnv(t, 0)
	if _, err := env.network.Add(mocksbr.DeviceConfig{
		MAC:    "aa:bb:cc:dd:ee:01",
		FailOn: []mocksbr.Op{mocksbr.OpGet},
	}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	_, stderr, exitCode := env.runCLI(t, "set", "aa:bb:cc:dd:ee:01",
		"--hostname", "new-name",
		"--timeout", "500ms")
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit when prelude read fails, got 0; stderr:\n%s", stderr)
	}
	if !strings.Contains(strings.ToLower(stderr), "read") &&
		!strings.Contains(strings.ToLower(stderr), "get") {
		t.Errorf("stderr should mention the failed read; got:\n%s", stderr)
	}
}
