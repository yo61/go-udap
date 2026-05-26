package cli

import (
	"context"

	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:               "info <mac>",
	Short:             "Show metadata for one device",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeMACs,
	RunE:              runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func runInfo(cmd *cobra.Command, args []string) error {
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
	stop := startProgress(stderr, "info", timeout)
	device, err := discoverAndFind(ctx, client, mac)
	stop()
	if err != nil {
		return err
	}
	maybeFillUUID(ctx, client, device, flagVerbose, stderr)
	formatDeviceInfo(stdout, device)
	return nil
}
