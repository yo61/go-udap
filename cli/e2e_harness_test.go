package cli

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"go-udap/mocksbr"
	"go-udap/udap"
)

// e2eEnv hosts an in-process mocksbr.Network that CLI invocations can
// drive via a MockTransport-backed udap.Client.
type e2eEnv struct {
	network *mocksbr.Network
}

// startMockEnv stands up a network of n auto-generated devices and
// substitutes the cli package's newClient seam with one that builds a
// MockTransport-backed Client. The previous seam is restored on test
// cleanup. e2e tests using this helper must not be t.Parallel — the
// seam is package-global.
func startMockEnv(t *testing.T, n int) *e2eEnv {
	t.Helper()
	network := mocksbr.NewNetwork(n, udap.NewNoOpLogger())

	prev := newClient
	newClient = func(_ bool, _ io.Writer) (*udap.Client, error) {
		return udap.NewClientWithTransport(
			mocksbr.NewMockTransport(network),
			udap.NewNoOpLogger(),
		), nil
	}
	t.Cleanup(func() {
		newClient = prev
		network.Close()
	})

	return &e2eEnv{network: network}
}

// runCLI invokes Execute with the given argv. Stdout, stderr, and the exit
// code are returned. Errors that Execute propagates are appended to stderr
// in the same "error: <msg>" form main.go prints, so tests see what
// the real binary would have shown the user.
func (e *e2eEnv) runCLI(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	t.Cleanup(resetFlagsForTesting)
	var outBuf, errBuf bytes.Buffer
	err := Execute(args, &outBuf, &errBuf)
	if err != nil {
		fmt.Fprintln(&errBuf, "error:", err)
	}
	return outBuf.String(), errBuf.String(), ExitCode(err)
}
