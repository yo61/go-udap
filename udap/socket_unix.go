//go:build !windows

package udap

import (
	"net"
	"syscall"
)

// enableBroadcast enables SO_BROADCAST and SO_REUSEADDR on a UDP connection
func enableBroadcast(conn *net.UDPConn, logger Logger) {
	file, err := conn.File()
	if err != nil {
		logger.Warn("Could not get socket file descriptor", "error", err)
		return
	}
	defer file.Close()

	fd := int(file.Fd())

	// Enable SO_BROADCAST for receiving broadcast packets
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1); err != nil {
		logger.Warn("Failed to enable socket option", "option", "SO_BROADCAST", "error", err)
	} else {
		logger.Debug("Socket option enabled", "option", "SO_BROADCAST")
	}

	// Enable SO_REUSEADDR to allow multiple listeners
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		logger.Warn("Failed to enable socket option", "option", "SO_REUSEADDR", "error", err)
	}
}

// enableBroadcastSimple enables SO_BROADCAST on a UDP connection (minimal logging)
func enableBroadcastSimple(conn *net.UDPConn, logger Logger) {
	file, err := conn.File()
	if err != nil {
		return
	}
	defer file.Close()

	fd := int(file.Fd())
	syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
	syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
}
