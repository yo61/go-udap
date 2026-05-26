package cli

import (
	"bytes"
	"testing"
)

func TestE2EBindInterfaceAndAllInterfacesMutuallyExclusive(t *testing.T) {
	t.Cleanup(resetFlagsForTesting)
	var outBuf, errBuf bytes.Buffer
	err := Execute([]string{"--bind-interface", "eth0", "--all-interfaces", "discover"}, &outBuf, &errBuf)
	if ExitCode(err) == 0 {
		t.Errorf("expected non-zero exit code for flag conflict, got 0")
	}
}

func TestE2EBindInterfaceUnknownNameIsExitOne(t *testing.T) {
	t.Cleanup(resetFlagsForTesting)
	var outBuf, errBuf bytes.Buffer
	err := Execute([]string{"--bind-interface", "definitely-not-a-real-interface", "discover", "--timeout", "100ms"}, &outBuf, &errBuf)
	if ExitCode(err) != 1 {
		t.Errorf("exit code %d, want 1 (unknown interface)", ExitCode(err))
	}
}
