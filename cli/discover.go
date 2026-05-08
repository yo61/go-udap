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

	client, err := newClient(*verbose, stderr)
	if err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	stopProgress := startProgress(stderr, "Discovering", *timeout)
	err = client.DiscoverDevicesWithContext(ctx)
	stopProgress()
	if err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("discovery failed: %w", err)}
	}

	devices := client.ListDevices()
	sort.Slice(devices, func(i, j int) bool { return devices[i].MAC < devices[j].MAC })

	if len(devices) == 0 {
		fmt.Fprintf(stderr, "no devices found within %s\n", *timeout)
		return nil
	}

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

// newClient constructs a udap.Client whose logger writes through the
// supplied stderr writer (typically a *stderrSync that serializes log
// output with the progress bar).
func newClient(verbose bool, stderr io.Writer) (*udap.Client, error) {
	logger := udap.NewStructuredLoggerWith(stderr)
	if verbose {
		logger.SetLevel(udap.LogLevelDebug)
	} else {
		logger.SetLevel(udap.LogLevelWarn)
	}
	return udap.NewClientWithLogger(logger)
}
