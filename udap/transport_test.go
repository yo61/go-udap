package udap

import (
	"bytes"
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

func TestTransportInterfaceShape(t *testing.T) {
	// Compile-time check that *UDPTransport satisfies Transport.
	var _ Transport = (*UDPTransport)(nil)
}

func TestUDPTransportRecvCancelledContextReturnsContextErr(t *testing.T) {
	tr, err := NewUDPTransport(0, NewNoOpLogger())
	if err != nil {
		t.Fatalf("NewUDPTransport: %v", err)
	}
	defer tr.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err = tr.Recv(ctx)
	if err == nil {
		t.Fatalf("expected error from cancelled context")
	}
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.Canceled or DeadlineExceeded, got %v", err)
	}
}

func TestUDPTransportRoundTrip(t *testing.T) {
	a, err := NewUDPTransport(0, NewNoOpLogger())
	if err != nil {
		t.Fatalf("NewUDPTransport a: %v", err)
	}
	defer a.Close()

	b, err := NewUDPTransport(0, NewNoOpLogger())
	if err != nil {
		t.Fatalf("NewUDPTransport b: %v", err)
	}
	defer b.Close()

	// b sends to a's bound port on loopback; a receives. Direct unicast
	// on loopback avoids relying on broadcast semantics, which are
	// unreliable across platforms. LocalAddr returns 0.0.0.0:port, so
	// rewrite to 127.0.0.1:port for delivery.
	bound := a.LocalAddr().(*net.UDPAddr)
	addr := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: bound.Port}
	// Pad the packet so it's longer than ucpFlagsOffset; otherwise
	// isUDAPRequestPacket short-circuits to false anyway, but the
	// "request bit set" filter would also drop the packet. Use a
	// non-request UDAP-shaped buffer (UCPFlags=0).
	payload := make([]byte, 32)
	payload[ucpFlagsOffset] = 0x00 // response (not request)

	if err := b.SendTo(payload, addr); err != nil {
		t.Fatalf("SendTo: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	got, _, err := a.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("got %v, want %v", got, payload)
	}
}

func TestNewUDPTransportOnInterfaceBindsToAddr(t *testing.T) {
	ifs, err := EnumerateInterfaces()
	if err != nil || len(ifs) == 0 {
		t.Skip("no usable interfaces")
	}
	iface := ifs[0]
	tr, err := NewUDPTransportOnInterface(iface, 0, NewNoOpLogger())
	if err != nil {
		t.Fatalf("NewUDPTransportOnInterface: %v", err)
	}
	defer tr.Close()
	addr := tr.LocalAddr().String()
	if !strings.Contains(addr, iface.Addr.String()) {
		t.Errorf("LocalAddr = %s, want it to contain %s", addr, iface.Addr)
	}
}
