package cli

import (
	"bytes"
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

// runCLI invokes Run with the given argv. Stdout, stderr, and the exit
// code are returned. Errors are surfaced via exitCode (matching the
// real binary's behaviour).
func (e *e2eEnv) runCLI(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	err := Run(args, &outBuf, &errBuf)
	return outBuf.String(), errBuf.String(), ExitCode(err)
}
