//go:build darwin

package udap

import (
	"fmt"
	"net"
	"syscall"
)

// ipBoundIF is the macOS IPPROTO_IP socket option IP_BOUND_IF.
// Not exported by package syscall, so we use the raw constant value.
const ipBoundIF = 25

// SO_REUSEPORT allows multiple sockets to bind to the same address:port
// combination on macOS. Not exported by package syscall, so we use the
// raw constant value (0x0200 = 512 on BSD/macOS).
const SO_REUSEPORT = 0x0200

// bindToInterface constrains a UDP socket's outbound (and inbound)
// packets to the given interface, leaving the local IP binding
// (typically 0.0.0.0) unchanged so the socket can still receive
// limited broadcasts on that interface.
//
// macOS implementation: IP_BOUND_IF setsockopt with the interface
// index. This is the BSD-derived mechanism; Linux uses
// SO_BINDTODEVICE instead.
func bindToInterface(conn *net.UDPConn, iface NetInterface, logger Logger) error {
	rawConn, err := conn.SyscallConn()
	if err != nil {
		return fmt.Errorf("get raw conn for IP_BOUND_IF: %w", err)
	}
	var sockErr error
	cerr := rawConn.Control(func(fd uintptr) {
		sockErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, ipBoundIF, iface.Index)
	})
	if cerr != nil {
		return fmt.Errorf("Control for IP_BOUND_IF: %w", cerr)
	}
	if sockErr != nil {
		return fmt.Errorf("setsockopt IP_BOUND_IF=%d (%s): %w", iface.Index, iface.Name, sockErr)
	}
	logger.Debug("UDP socket bound to interface (IP_BOUND_IF)",
		"interface", iface.Name, "index", iface.Index)
	return nil
}

// setReusePortPreBind sets SO_REUSEADDR and SO_REUSEPORT on the socket
// described by fd. Must be called BEFORE bind() (i.e. from inside a
// net.ListenConfig.Control function) to enable multi-socket sharing
// of the same address:port — used by NewClientForAllInterfaces to
// stand up one socket per interface on the same 0.0.0.0:Port.
func setReusePortPreBind(fd uintptr) error {
	if err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		return fmt.Errorf("SO_REUSEADDR: %w", err)
	}
	if err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, SO_REUSEPORT, 1); err != nil {
		return fmt.Errorf("SO_REUSEPORT: %w", err)
	}
	return nil
}
