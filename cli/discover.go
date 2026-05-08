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
	timeout := fs.Duration("timeout", 5*time.Second, "Discovery timeout")
	verbose := fs.BoolP("verbose", "v", false, "Debug logging to stderr")
	info := fs.Bool("info", false, "Also print metadata per device")
	if err := fs.Parse(args); err != nil {
		return &ExitError{Code: 1, Err: err}
	}

	client, err := newClient(*verbose)
	if err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := client.DiscoverDevicesWithContext(ctx); err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("discovery failed: %w", err)}
	}

	devices := client.ListDevices()
	sort.Slice(devices, func(i, j int) bool { return devices[i].MAC < devices[j].MAC })

	for i, d := range devices {
		if *info {
			if i > 0 {
				fmt.Fprintln(stdout)
			}
			formatDeviceInfo(stdout, d)
		} else {
			fmt.Fprintln(stdout, d.MAC)
		}
	}
	return nil
}

// newClient constructs a udap.Client; verbose controls log level.
func newClient(verbose bool) (*udap.Client, error) {
	logger := udap.NewStructuredLogger()
	if verbose {
		logger.SetLevel(udap.LogLevelDebug)
	} else {
		logger.SetLevel(udap.LogLevelWarn)
	}
	return udap.NewClientWithLogger(logger)
}
