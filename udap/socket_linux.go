//go:build linux

package udap

import (
	"fmt"
	"net"
	"syscall"
)

// SO_REUSEPORT allows multiple sockets to bind to the same address:port
// combination, as long as each socket is used to receive or send data
// to a distinct interface. This constant is available in Linux kernel 3.9+.
const SO_REUSEPORT = 15

// bindToInterface constrains a UDP socket's outbound packets to the
// given interface, leaving the local IP binding (typically 0.0.0.0)
// unchanged so the socket can still receive limited broadcasts.
//
// Linux implementation: SO_BINDTODEVICE setsockopt with the interface
// name. Requires CAP_NET_RAW (or root) on most distributions;
// EPERM on failure surfaces in the error message.
func bindToInterface(conn *net.UDPConn, iface NetInterface, logger Logger) error {
	rawConn, err := conn.SyscallConn()
	if err != nil {
		return fmt.Errorf("get raw conn for SO_BINDTODEVICE: %w", err)
	}
	var sockErr error
	cerr := rawConn.Control(func(fd uintptr) {
		sockErr = syscall.SetsockoptString(int(fd), syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, iface.Name)
	})
	if cerr != nil {
		return fmt.Errorf("Control for SO_BINDTODEVICE: %w", cerr)
	}
	if sockErr != nil {
		return fmt.Errorf("setsockopt SO_BINDTODEVICE=%s: %w (may require CAP_NET_RAW)", iface.Name, sockErr)
	}
	logger.Debug("UDP socket bound to interface (SO_BINDTODEVICE)",
		"interface", iface.Name, "index", iface.Index)
	return nil
}

// setReusePortPreBind sets SO_REUSEADDR and SO_REUSEPORT on the socket
// described by fd. Must be called BEFORE bind() (i.e. from inside a
// net.ListenConfig.Control function) to enable multi-socket sharing
// of the same address:port — used by NewClientForAllInterfaces to
// stand up one socket per interface on the same 0.0.0.0:Port.
//
// SO_REUSEPORT requires Linux kernel 3.9+.
func setReusePortPreBind(fd uintptr) error {
	if err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		return fmt.Errorf("SO_REUSEADDR: %w", err)
	}
	if err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, SO_REUSEPORT, 1); err != nil {
		return fmt.Errorf("SO_REUSEPORT: %w", err)
	}
	return nil
}
