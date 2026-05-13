package cli

import (
	"fmt"
	"io"
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
func formatDeviceInfo(w io.Writer, d *udap.Device) {
	fmt.Fprintf(w, "MAC:      %s\n", d.MAC)
	fmt.Fprintf(w, "IP:       %s\n", d.IP)
	if d.Name != "" {
		fmt.Fprintf(w, "Name:     %s\n", d.Name)
	}
	if d.Model != "" {
		fmt.Fprintf(w, "Model:    %s\n", d.Model)
	}
	if d.Firmware != "" {
		fmt.Fprintf(w, "Firmware: %s\n", d.Firmware)
	}
	if d.HardwareRev != "" {
		fmt.Fprintf(w, "HW Rev:   %s\n", d.HardwareRev)
	}
	if d.UUID != "" {
		fmt.Fprintf(w, "UUID:     %s\n", d.UUID)
	}
	if d.State != "" {
		fmt.Fprintf(w, "State:    %s\n", d.State)
	}
}
