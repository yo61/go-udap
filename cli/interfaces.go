package cli

import (
	"fmt"
	"io"
	"net"
	"strconv"

	"github.com/spf13/cobra"

	"go-udap/udap"
)

// enumerateInterfaces is a seam for tests.
var enumerateInterfaces = udap.EnumerateInterfaces

var interfacesCmd = &cobra.Command{
	Use:   "interfaces",
	Short: "List network interfaces usable for discovery",
	Long: `List the local network interfaces that satisfy the filter go-udap
applies to discovery: up, broadcast-capable, has an IPv4 address, and
not a loopback. The default text output is a table; --format json or
csv writes one record per interface.

Useful for picking a value for the global --bind-interface flag on
multi-homed hosts. The Broadcast column is informational only — UDAP
discovery always targets the limited broadcast 255.255.255.255 so
unconfigured devices can hear it.`,
	Args:        cobra.NoArgs,
	RunE:        runInterfaces,
	Annotations: map[string]string{annotationResult: ""},
}

func init() {
	rootCmd.AddCommand(interfacesCmd)
}

func runInterfaces(cmd *cobra.Command, _ []string) error {
	stderr := cmd.ErrOrStderr()

	ifs, err := enumerateInterfaces()
	if err != nil {
		return &ExitError{Code: exitFailure, Err: fmt.Errorf("enumerate interfaces: %w", err)}
	}
	if len(ifs) == 0 {
		fmt.Fprintln(stderr, "no usable interfaces found")
	}
	return renderResult(cmd, interfacesResult(ifs))
}

// interfacesResult is the Result of `interfaces`.
type interfacesResult []udap.NetInterface

type interfaceRecord struct {
	Name      string  `json:"name"`
	Index     int     `json:"index"`
	Address   *string `json:"address"`
	Broadcast *string `json:"broadcast"`
}

func (r interfacesResult) records() []interfaceRecord {
	out := make([]interfaceRecord, 0, len(r))
	for _, ni := range r {
		out = append(out, interfaceRecord{
			Name:      ni.Name,
			Index:     ni.Index,
			Address:   ipOrNull(ni.Addr),
			Broadcast: ipOrNull(ni.Broadcast),
		})
	}
	return out
}

func (r interfacesResult) WriteText(w io.Writer) error {
	formatInterfacesTable(w, r)
	return nil
}

func (r interfacesResult) JSONValue() any { return r.records() }

func (r interfacesResult) CSVHeader() []string {
	return []string{"name", "index", "address", "broadcast"}
}

func (r interfacesResult) CSVRows() [][]*string {
	rows := make([][]*string, 0, len(r))
	for _, rec := range r.records() {
		index := strconv.Itoa(rec.Index)
		rows = append(rows, []*string{&rec.Name, &index, rec.Address, rec.Broadcast})
	}
	return rows
}

// ipOrNull returns ip as a string, or nil when it is absent.
func ipOrNull(ip net.IP) *string {
	if len(ip) == 0 {
		return nil
	}
	return new(ip.String())
}
