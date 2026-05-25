package cli

import (
	"context"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/spf13/pflag"

	"go-udap/udap"
)

func runDiscover(args []string, stdout, stderr io.Writer) error {
	fs := pflag.NewFlagSet("discover", pflag.ContinueOnError)
	fs.SetOutput(stderr)
	timeout := newDurationWithPlaceholder("DURATION", 5*time.Second)
	fs.Var(timeout, "timeout", "Discovery timeout, e.g. 5s, 30s, 2m")
	verbose := fs.BoolP("verbose", "v", false, "Debug logging to stderr")
	info := fs.Bool("info", false, "Also print metadata per device")
	if err := parseSubcommandFlags(fs, args); err != nil {
		return err
	}

	client, err := newClient(*verbose, stderr)
	if err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout.Value())
	defer cancel()
	stopProgress := startProgress(stderr, "Discovering", timeout.Value())
	err = client.DiscoverDevicesWithContext(ctx)
	stopProgress()
	if err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("discovery failed: %w", err)}
	}

	devices := client.ListDevices()
	sort.Slice(devices, func(i, j int) bool { return devices[i].MAC.String() < devices[j].MAC.String() })

	if len(devices) == 0 {
		fmt.Fprintf(stderr, "no devices found within %s\n", timeout.Value())
		return nil
	}

	for i, d := range devices {
		if *info {
			if i > 0 {
				fmt.Fprintln(stdout)
			}
			maybeFillUUID(ctx, client, d, *verbose, stderr)
			formatDeviceInfo(stdout, d)
			nc, err := client.GetDeviceNetworkConfigWithContext(ctx, d)
			if err != nil {
				// Soft-fail: emit dashes so the table is consistent.
				// The diagnostic message is gated behind --verbose because
				// the dash output already signals "network config not
				// available" — most users (especially on unconfigured
				// devices) don't need the wire-level error inline.
				if *verbose {
					fmt.Fprintf(stderr, "warning: get_ip failed for %s: %v\n", d.MAC, err)
				}
				nc = udap.NetworkConfig{}
			}
			formatNetworkConfig(stdout, nc)
		} else {
			fmt.Fprintln(stdout, d.MAC)
		}
	}
	return nil
}

// newClient constructs a udap.Client whose logger writes through the
// supplied stderr writer (typically a *stderrSync that serializes log
// output with the progress bar).
//
// Declared as a package variable so e2e tests can substitute a Client
// backed by mocksbr.MockTransport. Production code never reassigns it.
var newClient = func(verbose bool, stderr io.Writer) (*udap.Client, error) {
	logger := udap.NewStructuredLoggerWith(stderr)
	if verbose {
		logger.SetLevel(udap.LogLevelDebug)
	} else {
		logger.SetLevel(udap.LogLevelWarn)
	}
	sel := currentBindInterface
	var c *udap.Client
	var err error
	switch {
	case sel.name != "":
		c, err = udap.NewClientForInterface(sel.name, logger)
	case sel.all:
		c, err = udap.NewClientForAllInterfaces(logger)
	default:
		c, err = udap.NewClientWithLogger(logger)
	}
	if err != nil {
		return nil, err
	}
	c.SetRetries(currentRetries)
	return c, nil
}
