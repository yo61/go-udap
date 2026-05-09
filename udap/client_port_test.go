package udap

import (
	"net"
	"testing"
)

func TestNewClientWithPortZero(t *testing.T) {
	c, err := newClientWithPort(0, NewNoOpLogger())
	if err != nil {
		t.Fatalf("expected client on port 0, got error: %v", err)
	}
	defer c.Close()
	tr, ok := c.transport.(*UDPTransport)
	if !ok {
		t.Fatalf("expected *UDPTransport, got %T", c.transport)
	}
	addr, ok := tr.LocalAddr().(*net.UDPAddr)
	if !ok {
		t.Fatalf("expected *net.UDPAddr, got %T", tr.LocalAddr())
	}
	if addr.Port == 0 {
		t.Errorf("expected OS-assigned port (non-zero), got 0")
	}
	if addr.Port == Port {
		t.Errorf("port 0 request unexpectedly produced default Port %d", Port)
	}
}

func TestTwoClientsOnPortZeroDoNotConflict(t *testing.T) {
	a, err := newClientWithPort(0, NewNoOpLogger())
	if err != nil {
		t.Fatalf("client A: %v", err)
	}
	defer a.Close()

	b, err := newClientWithPort(0, NewNoOpLogger())
	if err != nil {
		t.Fatalf("client B: %v", err)
	}
	defer b.Close()
}
