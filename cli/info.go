package cli

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"go-udap/udap"
)

var infoCmd = &cobra.Command{
	Use:   "info MAC",
	Short: "Show metadata for one device",
	Long: `Run a discovery cycle and print the metadata for one device by MAC
address: MAC, IP, Name, Model, Firmware, HW Rev, UUID, and State.
--format json or csv writes one record.

If the discovery response omits UUID (older firmware does), info falls
back to a get_uuid query (UCP 0x000b) to fill that field. Failures of the
fallback are soft; pass --verbose to see them on stderr.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeMACs,
	RunE:              runInfo,
	Annotations:       map[string]string{annotationResult: ""},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func runInfo(cmd *cobra.Command, args []string) error {
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
	stop := startProgress(stderr, "info", timeout)
	device, err := discoverAndFind(ctx, client, mac)
	stop()
	if err != nil {
		return err
	}
	maybeFillUUID(ctx, client, device, flagVerbose, stderr)
	return renderResult(cmd, infoResult{device})
}

// infoResult is the Result of `info`.
type infoResult struct{ device *udap.Device }

func (r infoResult) WriteText(w io.Writer) error {
	return formatDeviceInfo(w, r.device)
}

func (r infoResult) JSONValue() any { return newDeviceRecord(r.device) }

func (r infoResult) CSVHeader() []string { return deviceHeader }

func (r infoResult) CSVRows() [][]*string {
	return [][]*string{newDeviceRecord(r.device).cells()}
}
