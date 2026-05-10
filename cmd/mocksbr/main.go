// cmd/mocksbr is a standalone binary that runs a mock Squeezebox
// Receiver (or several) on a real UDP socket. The mock answers UDAP
// discovery, GetData, SetData and Reset requests; tests and developers
// can drive `go-udap` against it without real hardware.
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/pflag"

	"go-udap/mocksbr"
	"go-udap/udap"
)

const usage = `Usage: mocksbr [flags]

Run one or more mock Squeezebox Receivers on a UDP socket so go-udap (or
any UDAP client) can discover and configure them without real hardware.

Flags:`

func main() {
	if err := run(os.Args[1:], os.Stderr, os.Stdout); err != nil {
		switch {
		case errors.Is(err, errUsage):
			os.Exit(1)
		default:
			fmt.Fprintln(os.Stderr, "mocksbr:", err)
			os.Exit(2)
		}
	}
}

var errUsage = errors.New("usage error")

// Version is the binary version string, surfaced by --version.
// Set at build time via -ldflags "-X main.Version=...".
// Defaults to "dev" for un-stamped local builds.
var Version = "dev"

func run(args []string, stderr, stdout *os.File) error {
	fs := pflag.NewFlagSet("mocksbr", pflag.ContinueOnError)
	fs.SortFlags = false
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, usage)
		fs.PrintDefaults()
	}

	var (
		nDevices = fs.Int("devices", 1, "number of auto-generated virtual devices")
		listen   = fs.String("listen", fmt.Sprintf("0.0.0.0:%d", udap.Port), "UDP address to bind")
		verbose  = fs.BoolP("verbose", "v", false, "debug logging to stderr")
		showVer  = fs.Bool("version", false, "print version and exit")
		showHelp = fs.BoolP("help", "h", false, "print help and exit")
		device   = fs.StringArray("device", nil, "per-device override: idx=N,key=value,... (repeatable)")
	)

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			return nil
		}
		return fmt.Errorf("%w: %v", errUsage, err)
	}

	if *showHelp {
		fs.Usage()
		return nil
	}
	if *showVer {
		fmt.Fprintf(stdout, "mocksbr %s\n", Version)
		return nil
	}
	if *nDevices < 0 {
		return fmt.Errorf("%w: --devices must be non-negative, got %d", errUsage, *nDevices)
	}

	logger := udap.NewStructuredLoggerWith(stderr)
	if *verbose {
		logger.SetLevel(udap.LogLevelDebug)
	} else {
		logger.SetLevel(udap.LogLevelInfo)
	}

	overrides, err := parseDeviceFlags(*device, *nDevices)
	if err != nil {
		return fmt.Errorf("%w: %v", errUsage, err)
	}

	net := mocksbr.NewNetwork(*nDevices, logger)
	for _, ov := range overrides {
		ov.cfg.MAC = strings.ToLower(ov.cfg.MAC)
		// Override an auto-generated device: replace it.
		if err := replaceAutoDevice(net, ov.idx, ov.cfg); err != nil {
			return fmt.Errorf("%w: --device idx=%d: %v", errUsage, ov.idx, err)
		}
	}
	defer net.Close()

	addr, err := resolveUDPAddr(*listen)
	if err != nil {
		return fmt.Errorf("resolve --listen %q: %w", *listen, err)
	}
	conn, err := bindUDP(addr, logger)
	if err != nil {
		return fmt.Errorf("bind UDP %s: %w", addr, err)
	}
	defer conn.Close()

	logger.Info("mocksbr listening", "addr", conn.LocalAddr().String(), "devices", *nDevices+len(overrides))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	return serve(ctx, conn, net, logger)
}

// writeReply unicasts one scheduled reply back to src, deferring via
// time.AfterFunc when the responding device has DeviceConfig.Slow > 0
// so the wire-side delay matches what the in-process MockTransport
// already produces.
func writeReply(conn *net.UDPConn, src *net.UDPAddr, reply mocksbr.ScheduledReply, logger udap.Logger) {
	send := func() {
		if _, werr := conn.WriteToUDP(reply.Bytes, src); werr != nil {
			logger.Warn("mocksbr write reply", "to", src.String(), "error", werr)
		}
	}
	if reply.Delay <= 0 {
		send()
		return
	}
	time.AfterFunc(reply.Delay, send)
}

// serve is the read loop: pull packets off the socket, hand them to the
// network, unicast each reply back to the requesting source.
func serve(ctx context.Context, conn *net.UDPConn, network *mocksbr.Network, logger udap.Logger) error {
	buf := make([]byte, 2048)
	for {
		select {
		case <-ctx.Done():
			logger.Info("mocksbr shutting down")
			return nil
		default:
		}
		// Short read deadline so we re-check ctx promptly.
		_ = conn.SetReadDeadline(deadlineFromContext(ctx))
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				continue
			}
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			logger.Warn("mocksbr read error", "error", err)
			continue
		}
		packet := make([]byte, n)
		copy(packet, buf[:n])
		for _, reply := range network.ReceiveScheduled(packet) {
			writeReply(conn, src, reply, logger)
		}
	}
}
