package mocksbr

import (
	"context"
	"sync"

	"go-udap/udap"
)

// MockTransport implements udap.Transport over an in-process Network.
// Used by hermetic tests to drive a udap.Client without opening a
// socket.
type MockTransport struct {
	net *Network

	mu      sync.Mutex
	pending [][]byte // reply packets waiting to be Recv'd
	wakeup  chan struct{}
	closed  bool
}

// NewMockTransport returns a MockTransport bound to net.
func NewMockTransport(net *Network) *MockTransport {
	return &MockTransport{
		net:    net,
		wakeup: make(chan struct{}, 1),
	}
}

// Send dispatches a packet to the network. Any replies the network
// produces are queued for retrieval by Recv.
func (t *MockTransport) Send(packet []byte) error {
	replies := t.net.Receive(packet)
	if len(replies) == 0 {
		return nil
	}
	t.mu.Lock()
	t.pending = append(t.pending, replies...)
	t.mu.Unlock()
	t.notifyWaiter()
	return nil
}

// Recv blocks until a reply packet is available or ctx is cancelled.
// The returned source identifier is the device's MAC, decoded from the
// reply packet's source address.
func (t *MockTransport) Recv(ctx context.Context) ([]byte, string, error) {
	for {
		t.mu.Lock()
		if t.closed {
			t.mu.Unlock()
			return nil, "", context.Canceled
		}
		if len(t.pending) > 0 {
			next := t.pending[0]
			t.pending = t.pending[1:]
			t.mu.Unlock()
			src := mockSourceMAC(next)
			return next, src, nil
		}
		t.mu.Unlock()

		select {
		case <-ctx.Done():
			return nil, "", ctx.Err()
		case <-t.wakeup:
		}
	}
}

// Close marks the transport closed; pending replies are discarded and
// future Recvs return immediately.
func (t *MockTransport) Close() error {
	t.mu.Lock()
	t.closed = true
	t.pending = nil
	t.mu.Unlock()
	t.notifyWaiter()
	return nil
}

func (t *MockTransport) notifyWaiter() {
	select {
	case t.wakeup <- struct{}{}:
	default:
	}
}

// mockSourceMAC extracts the source MAC from a reply packet. Returns
// "" if the packet is too short to contain a valid header.
func mockSourceMAC(reply []byte) string {
	if len(reply) < udap.UDAPHeaderSize {
		return ""
	}
	pkt, _, err := udap.ParsePacket(reply)
	if err != nil {
		return ""
	}
	return formatMAC(pkt.SrcAddress)
}
