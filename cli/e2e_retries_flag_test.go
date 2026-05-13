package cli

import (
	"bytes"
	"context"
	"io"
	"sync/atomic"
	"testing"

	"go-udap/mocksbr"
	"go-udap/udap"
)

// TestE2ERetriesFlagSendsNPlus1Times: with --retries 2, the transport
// should receive 3 discovery requests (1 initial + 2 retries) for a single
// discover invocation.
func TestE2ERetriesFlagSendsNPlus1Times(t *testing.T) {
	var sendCount atomic.Int64
	network := mocksbr.NewNetwork(1, udap.NewNoOpLogger())
	t.Cleanup(func() { _ = network.Close() })

	// Wrap MockTransport to count Send calls.
	prev := newClient
	newClient = func(_ bool, _ io.Writer) (*udap.Client, error) {
		transport := mocksbr.NewMockTransport(network)
		counter := &countingMockTransport{inner: transport, count: &sendCount}
		c := udap.NewClientWithTransport(counter, udap.NewNoOpLogger())
		c.SetRetries(currentRetries)
		return c, nil
	}
	t.Cleanup(func() { newClient = prev })

	var outBuf, errBuf bytes.Buffer
	err := Run([]string{"--retries", "2", "discover", "--timeout", "300ms"}, &outBuf, &errBuf)
	if err != nil {
		t.Fatalf("Run returned %v; stderr=%s", err, errBuf.String())
	}
	if got := sendCount.Load(); got != 3 {
		t.Errorf("got %d Send calls into transport, want 3 (--retries 2 = 1 initial + 2 retries)", got)
	}
}

// countingMockTransport wraps a Transport and counts Send calls.
type countingMockTransport struct {
	inner udap.Transport
	count *atomic.Int64
}

func (t *countingMockTransport) Send(packet []byte) error {
	t.count.Add(1)
	return t.inner.Send(packet)
}

func (t *countingMockTransport) Recv(ctx context.Context) ([]byte, string, error) {
	return t.inner.Recv(ctx)
}

func (t *countingMockTransport) Close() error { return t.inner.Close() }
