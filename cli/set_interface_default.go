package cli

import (
	"fmt"
	"io"

	"go-udap/udap"
)

// applyInterfaceDefault substitutes a concrete value for the NVRAM
// `interface` byte when the device reports the factory-default
// sentinel and the user didn't supply --interface.
//
// Squeezebox firmware refuses to bring the network up after reboot if
// the interface byte is left at its sentinel value, so SetDeviceConfig-
// WithContext's read-modify-write would faithfully write a value the
// device cannot act on. The CLI substitutes wired (1) by default —
// because the device is, by construction, already reachable over wire
// for this very write — or wireless (0) when --wireless-ssid signals
// that intent.
//
// The substitution is surfaced unconditionally on stderr so users see
// the magic happening; gating on --verbose would hide a behavior
// change from the very users it affects most (those configuring a
// freshly-reset device for the first time).
func applyInterfaceDefault(merged map[string]string, device *udap.Device, stderr io.Writer) {
	if _, userSet := merged["interface"]; userSet {
		return
	}
	p, ok := udap.ParameterByName("interface")
	if !ok {
		return
	}
	if device.Parameters["interface"] != p.FactoryDefault {
		return
	}
	if merged["wireless_SSID"] != "" {
		merged["interface"] = "0"
		fmt.Fprintln(stderr, "go-udap: device interface is unset; inferred wireless from --wireless-ssid")
		fmt.Fprintln(stderr, "  — pass --interface 1 to override")
		return
	}
	merged["interface"] = "1"
	fmt.Fprintln(stderr, "go-udap: device interface is unset; defaulting to wired")
	fmt.Fprintln(stderr, "  — pass --interface 0 for wireless or --interface 1 to silence this notice")
}
