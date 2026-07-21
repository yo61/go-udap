package cli

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"go-udap/udap"
)

var readAll bool

var readCmd = &cobra.Command{
	Use:   "read MAC",
	Short: "Read all parameters from a device",
	Long: `Read every known NVRAM parameter from a device and print them in
config-file format (one "name = value" per line).

By default the output is filtered down to values changed from the
factory defaults, so piping it back through "go-udap set --config -"
restores only what the user actually set. Pass --all (-a) to include
factory defaults and offset_NNN entries for unrecognized NVRAM offsets.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeMACs,
	RunE:              runRead,
}

func init() {
	readCmd.Flags().BoolVarP(&readAll, "all", "a", false,
		"Include factory-default values and offset_NNN entries for unrecognized NVRAM offsets. Default: only print values changed from the factory defaults, so output round-trips cleanly through the set subcommand.")
	rootCmd.AddCommand(readCmd)
}

func runRead(cmd *cobra.Command, args []string) error {
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
	stop := startProgress(stderr, "read", timeout)
	defer stop()
	device, err := discoverAndFind(ctx, client, mac)
	if err != nil {
		return err
	}
	if err := client.GetAllDeviceConfigWithContext(ctx, device); err != nil {
		return deviceOpError("read", mac, timeout, err)
	}
	stop()

	out := device.Parameters
	if !readAll {
		out = filterReadOutput(out)
	}
	if err := formatParamMap(stdout, out); err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	return nil
}

// filterReadOutput trims a device-parameter map down to the entries
// that are interesting for backup/restore via `set`. Two classes of
// entries are dropped:
//
//  1. offset_NNN entries — synthetic keys for NVRAM offsets that don't
//     map to a known parameter (raw hex value; `set` would reject the
//     unknown name).
//
//  2. Values matching the parameter's FactoryDefault — boring for
//     backup, and some (wireless_keylen=0, interface=128, empty
//     wireless_SSID) wouldn't even be accepted by `set`'s validation.
//
// Pass `read --all` to disable both filters.
func filterReadOutput(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		if strings.HasPrefix(k, "offset_") {
			continue
		}
		if p, ok := udap.ParameterByName(k); ok && v == p.FactoryDefault {
			continue
		}
		out[k] = v
	}
	return out
}
