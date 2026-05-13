package udap

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"
)

// Transport is the network abstraction underneath udap.Client. It handles
// broadcast send and asynchronous receive of raw UDAP packets; addressing
// is encoded in the packets themselves, not at the transport layer.
//
// Two implementations exist:
//   - UDPTransport (in this package): wraps a real *net.UDPConn.
//   - mocksbr.MockTransport: in-process, hands packets directly to mock
//     devices for hermetic tests.
type Transport interface {
	// Send dispatches a UDAP packet from a client. The destination MAC is
	// encoded inside the packet. UDPTransport broadcasts to the LAN;
	// MockTransport feeds the packet directly to its connected mock devices.
	Send(packet []byte) error

	// Recv blocks until a packet arrives or ctx is cancelled. Returns the
	// raw packet bytes and an informational source identifier (an IP
	// string for UDPTransport; a MAC for MockTransport). The src is for
	// logging only; routing decisions use the packet's contents.
	Recv(ctx context.Context) (packet []byte, src string, err error)

	// Close releases transport resources.
	Close() error
}

// UDPTransport implements Transport over a real *net.UDPConn.
type UDPTransport struct {
	conn   *net.UDPConn
	logger Logger
}

// NewUDPTransport binds a UDP socket on 0.0.0.0:port (port 0 lets the OS
// pick) and enables SO_BROADCAST so it can both broadcast and receive
// broadcasts. Use port=Port (17784) for production; port=0 in tests.
func NewUDPTransport(port int, logger Logger) (*UDPTransport, error) {
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return nil, fmt.Errorf("resolve UDP addr: %w", err)
	}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return nil, fmt.Errorf("listen UDP: %w", err)
	}
	enableBroadcast(conn, logger)
	logger.Debug("UDPTransport bound", "address", conn.LocalAddr().String())
	return &UDPTransport{conn: conn, logger: logger}, nil
}

// Send broadcasts the packet to 255.255.255.255:Port.
func (t *UDPTransport) Send(packet []byte) error {
	dst, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", Port))
	if err != nil {
		return fmt.Errorf("resolve broadcast addr: %w", err)
	}
	if _, err := t.conn.WriteToUDP(packet, dst); err != nil {
		return fmt.Errorf("UDP send: %w", err)
	}
	return nil
}

// SendTo sends a packet to a specific destination. Used by mocksbr's
// responder side, where the reply is unicast to the request's source
// (not broadcast).
func (t *UDPTransport) SendTo(packet []byte, dst *net.UDPAddr) error {
	if _, err := t.conn.WriteToUDP(packet, dst); err != nil {
		return fmt.Errorf("UDP send: %w", err)
	}
	return nil
}

// Recv blocks until a packet arrives or ctx is cancelled. Skips packets
// with the UDAP request flag set (the kernel-looped-back broadcast we
// just sent). Replies — including replies from a mocksbr running on
// localhost — fall through with the request bit clear and are returned.
func (t *UDPTransport) Recv(ctx context.Context) ([]byte, string, error) {
	buf := make([]byte, 2048)
	for {
		if err := ctx.Err(); err != nil {
			return nil, "", err
		}
		// Use a short read deadline so we can re-check ctx promptly.
		deadline := time.Now().Add(200 * time.Millisecond)
		if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
			deadline = d
		}
		_ = t.conn.SetReadDeadline(deadline)
		n, src, err := t.conn.ReadFromUDP(buf)
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				continue
			}
			return nil, "", fmt.Errorf("UDP recv: %w", err)
		}
		if isUDAPRequestPacket(buf, n) {
			continue
		}
		out := make([]byte, n)
		copy(out, buf[:n])
		return out, src.IP.String(), nil
	}
}

// RecvFrom is like Recv but also returns the source UDP address. The
// mocksbr binary uses this to unicast replies back to the requesting
// client.
func (t *UDPTransport) RecvFrom(ctx context.Context) ([]byte, *net.UDPAddr, error) {
	buf := make([]byte, 2048)
	for {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		deadline := time.Now().Add(200 * time.Millisecond)
		if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
			deadline = d
		}
		_ = t.conn.SetReadDeadline(deadline)
		n, src, err := t.conn.ReadFromUDP(buf)
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				continue
			}
			return nil, nil, fmt.Errorf("UDP recv: %w", err)
		}
		out := make([]byte, n)
		copy(out, buf[:n])
		return out, src, nil
	}
}

// LocalAddr returns the bound address (test helper).
func (t *UDPTransport) LocalAddr() net.Addr {
	return t.conn.LocalAddr()
}

// Close releases the underlying socket.
func (t *UDPTransport) Close() error {
	return t.conn.Close()
}

// NewUDPTransportOnInterface returns a UDPTransport bound to 0.0.0.0
// (so it can receive limited broadcasts), with the kernel constrained
// to send outbound packets through the given interface via a
// platform-specific socket option (IP_BOUND_IF on macOS,
// SO_BINDTODEVICE on Linux). The destination remains 255.255.255.255
// because unconfigured Squeezebox devices (source IP 0.0.0.0) only
// process limited broadcasts — directed-subnet broadcasts like
// 192.168.1.255 don't reach them.
//
// Earlier design (bind-to-interface-IP + send-to-directed-broadcast)
// was reverted after real-hardware testing showed pre-DHCP devices
// never replied. See `docs/superpowers/plans/2026-05-13-getip-hwrev-uuid-iface.md`
// "## Spike result" for the wire-trace evidence.
func NewUDPTransportOnInterface(iface NetInterface, port int, logger Logger) (*UDPTransport, error) {
	if iface.Addr == nil {
		return nil, fmt.Errorf("interface %s has no IPv4 address", iface.Name)
	}
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return nil, fmt.Errorf("resolve UDP addr: %w", err)
	}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return nil, fmt.Errorf("listen UDP: %w", err)
	}
	enableBroadcast(conn, logger)
	if err := bindToInterface(conn, iface, logger); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("bind to interface %s: %w", iface.Name, err)
	}
	logger.Debug("UDPTransport bound to interface",
		"interface", iface.Name, "address", conn.LocalAddr().String(),
		"egress_index", iface.Index)
	return &UDPTransport{
		conn:   conn,
		logger: logger,
	}, nil
}
