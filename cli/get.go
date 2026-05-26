package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"go-udap/udap"
)

var getCmd = &cobra.Command{
	Use:               "get <mac> <param> [<param>...]",
	Short:             "Read specific parameters",
	Args:              cobra.MinimumNArgs(2),
	ValidArgsFunction: completeParameterNames,
	RunE:              runGet,
}

func init() {
	rootCmd.AddCommand(getCmd)
}

func runGet(cmd *cobra.Command, args []string) error {
	stdout := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()
	timeout := flagTimeout.Value()

	mac, err := normalizeMAC(args[0])
	if err != nil {
		return &ExitError{Code: 1, Err: err}
	}
	params := args[1:]
	for _, p := range params {
		if _, ok := udap.ParameterByName(p); !ok {
			return &ExitError{Code: 1, Err: fmt.Errorf("get: unknown parameter %q", p)}
		}
	}

	client, err := newClient(flagVerbose, stderr)
	if err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()
	stop := startProgress(stderr, "get", timeout)
	defer stop()
	device, err := discoverAndFind(ctx, client, mac)
	if err != nil {
		return err
	}
	values, err := client.GetDeviceConfigWithContext(ctx, device, params)
	if err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("get failed: %w", err)}
	}
	stop()
	if err := formatGetResult(stdout, params, values); err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	return nil
}
