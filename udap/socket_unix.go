//go:build !windows

package udap

import (
	"net"
	"syscall"
)

// enableBroadcast enables SO_BROADCAST and SO_REUSEADDR on a UDP connection.
//
// IMPORTANT: this must use SyscallConn().Control(), NOT (*UDPConn).File().
// File() switches the socket to blocking mode, which on macOS prevents
// Close() from interrupting any pending recvfrom — symptom is that the
// process hangs after discovery. SyscallConn().Control() leaves the
// non-blocking-via-poller mode intact.
func enableBroadcast(conn *net.UDPConn, logger Logger) {
	rawConn, err := conn.SyscallConn()
	if err != nil {
		logger.Warn("Could not get raw connection", "error", err)
		return
	}

	rawConn.Control(func(fd uintptr) {
		if err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1); err != nil {
			logger.Warn("Failed to enable socket option", "option", "SO_BROADCAST", "error", err)
		} else {
			logger.Debug("Socket option enabled", "option", "SO_BROADCAST")
		}

		if err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
			logger.Warn("Failed to enable socket option", "option", "SO_REUSEADDR", "error", err)
		}
	})
}
