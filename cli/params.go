package cli

import "go-udap/udap"

// paramFlag describes one CLI --flag form of a UDAP parameter, derived
// from the udap.Parameters table.
type paramFlag struct {
	udapName string // canonical wire name, e.g. "wireless_SSID"
	flagName string // CLI form, e.g. "wireless-ssid"
	help     string
}

// paramFlags returns the CLI flag table, derived from udap.Parameters
// (the single source of truth). To add a new parameter, edit
// udap/parameters.go — this function picks it up automatically.
func paramFlags() []paramFlag {
	out := make([]paramFlag, len(udap.Parameters))
	for i, p := range udap.Parameters {
		out[i] = paramFlag{
			udapName: p.Name,
			flagName: p.FlagName(),
			help:     p.Help,
		}
	}
	return out
}
