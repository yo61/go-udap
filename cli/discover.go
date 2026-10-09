package cli

import (
	"context"
	"fmt"
	"io"
	"slices"
	"sort"

	"github.com/spf13/cobra"

	"go-udap/udap"
)

var discoverInfo bool

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover devices on the network",
	Long: `Broadcast a UDAP advanced-discover packet on UDP port 17784 and
print every Squeezebox device that responds within --timeout.

By default only MAC addresses are printed, one per line. Pass --info to
print full metadata per device (MAC, IP, Name, Model, Firmware, HW Rev,
UUID, State, plus IP / subnet / gateway via a follow-up get_ip query).
--format json or csv writes one record per device; with --info, every
device's queries complete before anything is written.

Sends always target the limited broadcast address 255.255.255.255 so
unconfigured devices (which have no DHCP lease and so no notion of a
subnet broadcast address) can hear them. On multi-homed hosts, use the
global --bind-interface or --all-interfaces flags to control which NIC
the broadcast leaves on.`,
	Args:        cobra.NoArgs,
	RunE:        runDiscover,
	Annotations: map[string]string{annotationResult: ""},
}

func init() {
	discoverCmd.Flags().BoolVar(&discoverInfo, "info", false, "Also print metadata per device")
	rootCmd.AddCommand(discoverCmd)
}

func runDiscover(cmd *cobra.Command, _ []string) error {
	stderr := cmd.ErrOrStderr()
	timeout := flagTimeout.Value()

	client, err := newClient(flagVerbose, stderr)
	if err != nil {
		return &ExitError{Code: exitFailure, Err: err}
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()
	stopProgress := startProgress(stderr, "Discovering", timeout)
	err = client.DiscoverDevicesWithContext(ctx)
	stopProgress()
	if err != nil {
		return &ExitError{Code: exitFailure, Err: fmt.Errorf("discovery failed: %w", err)}
	}

	devices := client.ListDevices()
	sort.Slice(devices, func(i, j int) bool {
		return devices[i].MAC.String() < devices[j].MAC.String()
	})

	if len(devices) == 0 {
		fmt.Fprintf(stderr, "no devices found within %s\n", timeout)
	}
	if !discoverInfo {
		return renderResult(cmd, discoverResult(devices))
	}
	return renderResult(cmd, queryDeviceDetails(ctx, client, devices, stderr))
}

// queryDeviceDetails runs the get_uuid fallback and get_ip for each
// device. Failures are soft: the UUID or network stays absent.
func queryDeviceDetails(
	ctx context.Context, client *udap.Client, devices []*udap.Device, stderr io.Writer,
) discoverInfoResult {
	result := make(discoverInfoResult, 0, len(devices))
	for _, d := range devices {
		maybeFillUUID(ctx, client, d, flagVerbose, stderr)
		discovered := discoveredDevice{device: d}
		nc, err := client.GetDeviceNetworkConfigWithContext(ctx, d)
		switch {
		case err == nil:
			discovered.network = &nc
		case flagVerbose:
			fmt.Fprintf(stderr, "warning: get_ip failed for %s: %v\n", d.MAC, err)
		}
		result = append(result, discovered)
	}
	return result
}

// discoverResult is the Result of `discover`.
type discoverResult []*udap.Device

type macRecord struct {
	MAC string `json:"mac"`
}

func (r discoverResult) WriteText(w io.Writer) error {
	for _, d := range r {
		if _, err := fmt.Fprintln(w, d.MAC); err != nil {
			return err
		}
	}
	return nil
}

func (r discoverResult) JSONValue() any {
	out := make([]macRecord, 0, len(r))
	for _, d := range r {
		out = append(out, macRecord{MAC: d.MAC.String()})
	}
	return out
}

func (r discoverResult) CSVHeader() []string { return []string{"mac"} }

func (r discoverResult) CSVRows() [][]*string {
	rows := make([][]*string, 0, len(r))
	for _, d := range r {
		rows = append(rows, []*string{new(d.MAC.String())})
	}
	return rows
}

// discoverInfoResult is the Result of `discover --info`.
type discoverInfoResult []discoveredDevice

// discoveredDevice is a device and its get_ip answer; network is nil
// when get_ip failed.
type discoveredDevice struct {
	device  *udap.Device
	network *udap.NetworkConfig
}

func (d discoveredDevice) networkRecord() *networkRecord {
	if d.network == nil {
		return nil
	}
	return new(newNetworkRecord(*d.network))
}

type discoveredDeviceRecord struct {
	deviceRecord
	Network *networkRecord `json:"network"`
}

func (r discoverInfoResult) WriteText(w io.Writer) error {
	for i, d := range r {
		if i > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		if err := formatDeviceInfo(w, d.device); err != nil {
			return err
		}
		var nc udap.NetworkConfig
		if d.network != nil {
			nc = *d.network
		}
		if err := formatNetworkConfig(w, nc); err != nil {
			return err
		}
	}
	return nil
}

func (r discoverInfoResult) JSONValue() any {
	out := make([]discoveredDeviceRecord, 0, len(r))
	for _, d := range r {
		out = append(out, discoveredDeviceRecord{
			deviceRecord: newDeviceRecord(d.device),
			Network:      d.networkRecord(),
		})
	}
	return out
}

func (r discoverInfoResult) CSVHeader() []string {
	header := slices.Clone(deviceHeader)
	for _, name := range networkHeader {
		header = append(header, "network_"+name)
	}
	return header
}

func (r discoverInfoResult) CSVRows() [][]*string {
	rows := make([][]*string, 0, len(r))
	for _, d := range r {
		network := make([]*string, len(networkHeader))
		if record := d.networkRecord(); record != nil {
			network = record.cells()
		}
		rows = append(rows, append(newDeviceRecord(d.device).cells(), network...))
	}
	return rows
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
