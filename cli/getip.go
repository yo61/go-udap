package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var getipCmd = &cobra.Command{
	Use:   "getip <mac>",
	Short: "Query device IP / subnet / gateway via UCP get_ip",
	Args:  cobra.ExactArgs(1),
	RunE:  runGetIP,
}

func init() {
	rootCmd.AddCommand(getipCmd)
}

func runGetIP(cmd *cobra.Command, args []string) error {
	stdout := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()
	timeout := flagTimeout.Value()

	mac, err := normalizeMAC(args[0])
	if err != nil {
		return &ExitError{Code: 1, Err: err}
	}

	client, err := newClient(flagVerbose, stderr)
	if err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()
	stop := startProgress(stderr, "getip", timeout)
	device, err := discoverAndFind(ctx, client, mac)
	if err != nil {
		stop()
		return err
	}
	nc, err := client.GetDeviceNetworkConfigWithContext(ctx, device)
	stop()
	if err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("get_ip failed for %s: %w", mac, err)}
	}
	formatNetworkConfig(stdout, nc)
	return nil
}
