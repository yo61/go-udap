package udap

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeTransport is a controllable Transport for MultiTransport tests.
type fakeTransport struct {
	sent    [][]byte
	sendErr error
	recvCh  chan recvOut
	closed  bool
}

type recvOut struct {
	pkt []byte
	src string
	err error
}

func newFakeTransport() *fakeTransport {
	return &fakeTransport{recvCh: make(chan recvOut, 8)}
}

func (f *fakeTransport) Send(p []byte) error {
	if f.sendErr != nil {
		return f.sendErr
	}
	f.sent = append(f.sent, p)
	return nil
}

func (f *fakeTransport) Recv(ctx context.Context) ([]byte, string, error) {
	select {
	case <-ctx.Done():
		return nil, "", ctx.Err()
	case ro, ok := <-f.recvCh:
		if !ok {
			return nil, "", errors.New("transport closed")
		}
		return ro.pkt, ro.src, ro.err
	}
}

func (f *fakeTransport) Close() error {
	if !f.closed {
		f.closed = true
		close(f.recvCh)
	}
	return nil
}

func TestMultiTransportSendFansOut(t *testing.T) {
	a, b := newFakeTransport(), newFakeTransport()
	mt := NewMultiTransport([]Transport{a, b}, NewNoOpLogger())
	defer mt.Close()

	if err := mt.Send([]byte{1, 2, 3}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(a.sent) != 1 || len(b.sent) != 1 {
		t.Errorf("expected each child to receive one send, got a=%d b=%d", len(a.sent), len(b.sent))
	}
}

func TestMultiTransportSendSucceedsIfAnyChildSucceeds(t *testing.T) {
	a, b := newFakeTransport(), newFakeTransport()
	a.sendErr = errors.New("child A broken")
	mt := NewMultiTransport([]Transport{a, b}, NewNoOpLogger())
	defer mt.Close()
	if err := mt.Send([]byte{1}); err != nil {
		t.Errorf("Send should succeed when at least one child succeeds, got %v", err)
	}
	if len(b.sent) != 1 {
		t.Errorf("expected B to receive the packet, got %d sends", len(b.sent))
	}
}

func TestMultiTransportSendFailsWhenAllChildrenFail(t *testing.T) {
	a, b := newFakeTransport(), newFakeTransport()
	a.sendErr = errors.New("A broken")
	b.sendErr = errors.New("B broken")
	mt := NewMultiTransport([]Transport{a, b}, NewNoOpLogger())
	defer mt.Close()
	if err := mt.Send([]byte{1}); err == nil {
		t.Error("Send should fail when all children fail")
	}
}

func TestMultiTransportRecvMergesReplies(t *testing.T) {
	a, b := newFakeTransport(), newFakeTransport()
	mt := NewMultiTransport([]Transport{a, b}, NewNoOpLogger())
	defer mt.Close()

	a.recvCh <- recvOut{pkt: []byte{0x0a}, src: "ifA"}
	b.recvCh <- recvOut{pkt: []byte{0x0b}, src: "ifB"}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	got := make(map[byte]bool)
	for range 2 {
		pkt, _, err := mt.Recv(ctx)
		if err != nil {
			t.Fatalf("Recv: %v", err)
		}
		got[pkt[0]] = true
	}
	if !got[0x0a] || !got[0x0b] {
		t.Errorf("did not see both replies; got %v", got)
	}
}

func TestMultiTransportRecvCancelledByCtx(t *testing.T) {
	a := newFakeTransport()
	mt := NewMultiTransport([]Transport{a}, NewNoOpLogger())
	defer mt.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, _, err := mt.Recv(ctx)
	if err == nil {
		t.Error("Recv should return ctx error after deadline")
	}
}

func TestMultiTransportCloseClosesChildren(t *testing.T) {
	a, b := newFakeTransport(), newFakeTransport()
	mt := NewMultiTransport([]Transport{a, b}, NewNoOpLogger())
	if err := mt.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	if !a.closed || !b.closed {
		t.Errorf("expected both children closed; a=%v b=%v", a.closed, b.closed)
	}
}
