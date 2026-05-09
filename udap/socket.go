package udap

import "net"

// EnableBroadcast is the public alias for the platform-specific
// enableBroadcast helpers in socket_unix.go and socket_windows.go.
// Used by cmd/mocksbr to set SO_BROADCAST on its listening socket so it
// can receive UDAP client broadcast discoveries.
func EnableBroadcast(conn *net.UDPConn, logger Logger) {
	enableBroadcast(conn, logger)
}
