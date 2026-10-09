package cli

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"go-udap/udap"
)

var getipCmd = &cobra.Command{
	Use:   "getip MAC",
	Short: "Query device IP / subnet / gateway via UCP get_ip",
	Long: `Actively query the device's current network configuration via
UCP_METHOD_GET_IP (0x0002). Prints IP / subnet / gateway, one per line;
--format json or csv writes one record.

This is distinct from discover: discover passively observes the source
address of an adv-discover response, while getip explicitly asks the
device for its configured network parameters. Useful after a config
change to confirm the device picked up the new settings.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeMACs,
	RunE:              runGetIP,
	Annotations:       map[string]string{annotationResult: ""},
}

func init() {
	rootCmd.AddCommand(getipCmd)
}

func runGetIP(cmd *cobra.Command, args []string) error {
	stderr := cmd.ErrOrStderr()
	timeout := flagTimeout.Value()

	mac, err := normalizeMAC(args[0])
	if err != nil {
		return &ExitError{Code: exitUsage, Err: err}
	}

	client, err := newClient(flagVerbose, stderr)
	if err != nil {
		return &ExitError{Code: exitFailure, Err: err}
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
		return deviceOpError("getip", mac, timeout, err)
	}
	return renderResult(cmd, getipResult(nc))
}

// getipResult is the Result of `getip`.
type getipResult udap.NetworkConfig

func (r getipResult) WriteText(w io.Writer) error {
	return formatNetworkConfig(w, udap.NetworkConfig(r))
}

func (r getipResult) JSONValue() any { return newNetworkRecord(udap.NetworkConfig(r)) }

func (r getipResult) CSVHeader() []string { return networkHeader }

func (r getipResult) CSVRows() [][]*string {
	return [][]*string{newNetworkRecord(udap.NetworkConfig(r)).cells()}
}
