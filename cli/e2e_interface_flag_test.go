package cli

import (
	"bytes"
	"strings"
	"testing"
)

// The conflict is reported before the interface name is validated, so the
// result doesn't depend on which interfaces the host has.
func TestE2EBindInterfaceAndAllInterfacesMutuallyExclusive(t *testing.T) {
	t.Cleanup(resetFlagsForTesting)
	var outBuf, errBuf bytes.Buffer
	args := []string{"--bind-interface", "no-such-if", "--all-interfaces", "discover"}
	err := Execute(args, &outBuf, &errBuf)
	if err == nil || !strings.Contains(err.Error(), "cannot be used together") {
		t.Fatalf("expected the flag-conflict error, got %v", err)
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code %d, want 2 (usage error)", ExitCode(err))
	}
}

func TestE2EBindInterfaceUnknownNameIsUsageError(t *testing.T) {
	t.Cleanup(resetFlagsForTesting)
	var outBuf, errBuf bytes.Buffer
	err := Execute([]string{"--bind-interface", "definitely-not-a-real-interface", "discover", "--timeout", "100ms"}, &outBuf, &errBuf)
	if ExitCode(err) != 2 {
		t.Errorf("exit code %d, want 2 (unknown interface)", ExitCode(err))
	}
}
