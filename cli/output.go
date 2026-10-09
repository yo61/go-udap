package cli

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"sort"

	"go-udap/udap"
)

// formatParamMap writes "key=value\n" lines to w, sorted by key.
// Used by `read` and multi-param `get`.
func formatParamMap(w io.Writer, m map[string]string) error {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if _, err := fmt.Fprintf(w, "%s=%s\n", k, m[k]); err != nil {
			return err
		}
	}
	return nil
}

// formatGetResult writes the result of a `get` command. Single-param requests
// produce a bare value (one line, no key=); multi-param requests produce
// key=value lines (one per requested param, in request order).
func formatGetResult(w io.Writer, requested []string, values map[string]string) error {
	if len(requested) == 1 {
		_, err := fmt.Fprintf(w, "%s\n", values[requested[0]])
		return err
	}
	for _, k := range requested {
		if _, err := fmt.Fprintf(w, "%s=%s\n", k, values[k]); err != nil {
			return err
		}
	}
	return nil
}

// formatDeviceInfo writes a multi-line metadata block for one device.
// Used by `info` and by `discover --info`. Empty fields are skipped so
// we don't show e.g. "State:" with nothing after it.
func formatDeviceInfo(w io.Writer, d *udap.Device) error {
	var b bytes.Buffer
	fmt.Fprintf(&b, "MAC:      %s\n", d.MAC)
	fmt.Fprintf(&b, "IP:       %s\n", d.IP)
	for _, field := range []struct{ label, value string }{
		{"Name:     ", d.Name},
		{"Model:    ", d.Model},
		{"Firmware: ", d.Firmware},
		{"HW Rev:   ", d.HardwareRev},
		{"UUID:     ", d.UUID},
		{"State:    ", d.State},
	} {
		if field.value != "" {
			fmt.Fprintf(&b, "%s%s\n", field.label, field.value)
		}
	}
	_, err := w.Write(b.Bytes())
	return err
}

// formatNetworkConfig writes IP / Subnet / Gateway lines, using "-"
// for any absent field.
func formatNetworkConfig(w io.Writer, nc udap.NetworkConfig) error {
	_, err := fmt.Fprintf(w, "IP:      %s\nSubnet:  %s\nGateway: %s\n",
		ipOrDashCLI(nc.IP), ipOrDashCLI(nc.SubnetMask), ipOrDashCLI(nc.Gateway))
	return err
}

func ipOrDashCLI(ip net.IP) string {
	if s := configuredIPOrNull(ip); s != nil {
		return *s
	}
	return "-"
}

// deviceRecord is a Device's metadata as the CLI names it. An empty
// field is absent, as it is omitted in text.
type deviceRecord struct {
	MAC         string  `json:"mac"`
	IP          *string `json:"ip"`
	Name        *string `json:"name"`
	Model       *string `json:"model"`
	Firmware    *string `json:"firmware"`
	HardwareRev *string `json:"hardware_rev"`
	UUID        *string `json:"uuid"`
	State       *string `json:"state"`
}

var deviceHeader = []string{
	"mac", "ip", "name", "model", "firmware", "hardware_rev", "uuid", "state",
}

func newDeviceRecord(d *udap.Device) deviceRecord {
	return deviceRecord{
		MAC:         d.MAC.String(),
		IP:          stringOrNull(d.IP),
		Name:        stringOrNull(d.Name),
		Model:       stringOrNull(d.Model),
		Firmware:    stringOrNull(d.Firmware),
		HardwareRev: stringOrNull(d.HardwareRev),
		UUID:        stringOrNull(d.UUID),
		State:       stringOrNull(d.State),
	}
}

func (r deviceRecord) cells() []*string {
	return []*string{&r.MAC, r.IP, r.Name, r.Model, r.Firmware, r.HardwareRev, r.UUID, r.State}
}

func stringOrNull(s string) *string {
	if s == "" {
		return nil
	}
	return new(s)
}

// networkRecord is a NetworkConfig as the CLI names it. A zero IP is
// absent, as it is "-" in text.
type networkRecord struct {
	IP         *string `json:"ip"`
	SubnetMask *string `json:"subnet_mask"`
	Gateway    *string `json:"gateway"`
}

var networkHeader = []string{"ip", "subnet_mask", "gateway"}

func newNetworkRecord(nc udap.NetworkConfig) networkRecord {
	return networkRecord{
		IP:         configuredIPOrNull(nc.IP),
		SubnetMask: configuredIPOrNull(nc.SubnetMask),
		Gateway:    configuredIPOrNull(nc.Gateway),
	}
}

func (r networkRecord) cells() []*string { return []*string{r.IP, r.SubnetMask, r.Gateway} }

// configuredIPOrNull is ipOrNull that also treats 0.0.0.0 as absent.
func configuredIPOrNull(ip net.IP) *string {
	if ip.IsUnspecified() {
		return nil
	}
	return ipOrNull(ip)
}

// formatInterfacesTable writes a fixed-column table for NetInterfaces.
// If the slice is empty, writes nothing.
func formatInterfacesTable(w io.Writer, ifs []udap.NetInterface) {
	if len(ifs) == 0 {
		return
	}
	fmt.Fprintln(w, "NAME            INDEX  ADDRESS            BROADCAST")
	for _, ni := range ifs {
		fmt.Fprintf(w, "%-15s %-5d  %-18s %s\n", ni.Name, ni.Index, ni.Addr, ni.Broadcast)
	}
}
