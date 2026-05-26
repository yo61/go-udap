package cli

import (
	"bytes"
	"strings"
	"testing"

	"go-udap/udap"
)

func TestE2EInterfacesSubcommandSmoke(t *testing.T) {
	// EnumerateInterfaces is real-OS-state-dependent; this test runs
	// against the actual host. It should always exit 0; the output is
	// either a table or "no usable interfaces found" on stderr.
	ifs, err := udap.EnumerateInterfaces()
	if err != nil {
		t.Skipf("EnumerateInterfaces error: %v", err)
	}

	t.Cleanup(resetFlagsForTesting)
	var outBuf, errBuf bytes.Buffer
	rerr := Execute([]string{"interfaces"}, &outBuf, &errBuf)
	if ExitCode(rerr) != 0 {
		t.Errorf("exit code %d, want 0", ExitCode(rerr))
	}
	if len(ifs) > 0 {
		if !strings.Contains(outBuf.String(), "NAME") {
			t.Errorf("expected table header in stdout; got:\n%s", outBuf.String())
		}
	} else {
		if !strings.Contains(errBuf.String(), "no usable interfaces") {
			t.Errorf("expected 'no usable interfaces' on stderr; got:\n%s", errBuf.String())
		}
	}
}
