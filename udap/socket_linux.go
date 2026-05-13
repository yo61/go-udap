//go:build linux

package udap

import (
	"fmt"
	"net"
	"syscall"
)

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
