package cli

import (
	"fmt"

	"go-udap/udap"
)

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
//
// The Placeholder is wrapped in backticks at the start of the help text
// so pflag uses it as the value-placeholder shown in --help (e.g. the
// `IP` in "--lan-gateway IP"). pflag strips the backticks but keeps the
// word in the description, prefixed with an em-dash for readability.
func paramFlags() []paramFlag {
	out := make([]paramFlag, len(udap.Parameters))
	for i, p := range udap.Parameters {
		help := p.Help
		if p.Placeholder != "" {
			help = fmt.Sprintf("`%s` — %s", p.Placeholder, p.Help)
		}
		out[i] = paramFlag{
			udapName: p.Name,
			flagName: p.FlagName(),
			help:     help,
		}
	}
	return out
}
