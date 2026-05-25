package cli

import (
	"bytes"
	"testing"
)

func TestE2EBindInterfaceAndAllInterfacesMutuallyExclusive(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	err := Run([]string{"--bind-interface", "eth0", "--all-interfaces", "discover"}, &outBuf, &errBuf)
	if ExitCode(err) != 1 {
		t.Errorf("exit code %d, want 1 (flag conflict)", ExitCode(err))
	}
}

func TestE2EBindInterfaceUnknownNameIsExitOne(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	err := Run([]string{"--bind-interface", "definitely-not-a-real-interface", "discover", "--timeout", "100ms"}, &outBuf, &errBuf)
	if ExitCode(err) != 1 {
		t.Errorf("exit code %d, want 1 (unknown interface)", ExitCode(err))
	}
}
