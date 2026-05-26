package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var rebootCmd = &cobra.Command{
	Use:               "reboot <mac>",
	Short:             "Reboot the device",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeMACs,
	RunE:              runReboot,
}

func init() {
	rootCmd.AddCommand(rebootCmd)
}

func runReboot(cmd *cobra.Command, args []string) error {
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
	stop := startProgress(stderr, "reboot", timeout)
	defer stop()
	device, err := discoverAndFind(ctx, client, mac)
	if err != nil {
		return err
	}
	if err := client.ResetDeviceWithContext(ctx, device); err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("reboot failed: %w", err)}
	}
	return nil
}
