package cli

import (
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"

	"go-udap/udap"
)

var discoverInfo bool

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover devices on the network",
	Args:  cobra.NoArgs,
	RunE:  runDiscover,
}

func init() {
	discoverCmd.Flags().BoolVar(&discoverInfo, "info", false, "Also print metadata per device")
	rootCmd.AddCommand(discoverCmd)
}

func runDiscover(cmd *cobra.Command, _ []string) error {
	stdout := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()
	timeout := flagTimeout.Value()

	client, err := newClient(flagVerbose, stderr)
	if err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()
	stopProgress := startProgress(stderr, "Discovering", timeout)
	err = client.DiscoverDevicesWithContext(ctx)
	stopProgress()
	if err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("discovery failed: %w", err)}
	}

	devices := client.ListDevices()
	sort.Slice(devices, func(i, j int) bool { return devices[i].MAC.String() < devices[j].MAC.String() })

	if len(devices) == 0 {
		fmt.Fprintf(stderr, "no devices found within %s\n", timeout)
		return nil
	}

	for i, d := range devices {
		if discoverInfo {
			if i > 0 {
				fmt.Fprintln(stdout)
			}
			maybeFillUUID(ctx, client, d, flagVerbose, stderr)
			formatDeviceInfo(stdout, d)
			nc, err := client.GetDeviceNetworkConfigWithContext(ctx, d)
			if err != nil {
				if flagVerbose {
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
// supplied stderr writer. Declared as a package variable so e2e tests
// can substitute a Client backed by mocksbr.MockTransport.
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
