//go:build windows

package udap

import (
	"fmt"
	"net"
	"syscall"
)

// enableBroadcast enables SO_BROADCAST and SO_REUSEADDR on a UDP connection
func enableBroadcast(conn *net.UDPConn, logger Logger) {
	rawConn, err := conn.SyscallConn()
	if err != nil {
		logger.Warn("Could not get raw connection", "error", err)
		return
	}

	rawConn.Control(func(fd uintptr) {
		// Enable SO_BROADCAST for broadcast packets
		if err := syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1); err != nil {
			logger.Warn("Failed to enable socket option", "option", "SO_BROADCAST", "error", err)
		} else {
			logger.Debug("Socket option enabled", "option", "SO_BROADCAST")
		}

		// Enable SO_REUSEADDR to allow multiple listeners
		if err := syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
			logger.Warn("Failed to enable socket option", "option", "SO_REUSEADDR", "error", err)
		}
	})
}

// bindToInterface is not yet implemented on Windows. Returns a clear
// error so --interface NAME surfaces "not supported" rather than
// silently misbehaving. The Windows equivalent is IP_UNICAST_IF
// (IPPROTO_IP option), but implementation is out of scope for now.
func bindToInterface(_ *net.UDPConn, _ NetInterface, _ Logger) error {
	return fmt.Errorf("--interface NAME is not yet supported on Windows; omit the flag to use the default discovery mode")
}

// setReusePortPreBind is a no-op on Windows because SO_REUSEPORT
// doesn't exist there. NewClientForAllInterfaces is unreachable on
// Windows anyway (bindToInterface returns "not supported"), so this
// is defensive — the function exists so transport.go compiles on
// Windows.
func setReusePortPreBind(fd uintptr) error {
	return nil
}
