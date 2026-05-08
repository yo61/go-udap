package cli

import "go-udap/udap"

// paramFlag describes one CLI --flag form of a UDAP parameter, derived
// from the udap.Parameters table.
type paramFlag struct {
	udapName    string // canonical wire name, e.g. "wireless_SSID"
	flagName    string // CLI form, e.g. "wireless-ssid"
	placeholder string // value form shown in --help (IP, NAME, 0|1, ...)
	help        string
}

// paramFlags returns the CLI flag table, derived from udap.Parameters
// (the single source of truth). To add a new parameter, edit
// udap/parameters.go — this function picks it up automatically.
func paramFlags() []paramFlag {
	out := make([]paramFlag, len(udap.Parameters))
	for i, p := range udap.Parameters {
		out[i] = paramFlag{
			udapName:    p.Name,
			flagName:    p.FlagName(),
			placeholder: p.Placeholder,
			help:        p.Help,
		}
	}
	return out
}

// stringWithPlaceholder is a pflag.Value that holds a string but
// reports a custom placeholder via Type(). pflag's --help renders
// "--<flag> <Type()>" in the value-form column; the stock String value
// always returns "string", which is uninformative. This lets us show
// "IP", "NAME", "0|1", etc. without polluting the description text.
type stringWithPlaceholder struct {
	val         string
	placeholder string
}

func newStringWithPlaceholder(placeholder string) *stringWithPlaceholder {
	return &stringWithPlaceholder{placeholder: placeholder}
}

func (s *stringWithPlaceholder) String() string { return s.val }
func (s *stringWithPlaceholder) Set(v string) error {
	s.val = v
	return nil
}
func (s *stringWithPlaceholder) Type() string {
	if s.placeholder == "" {
		return "string"
	}
	return s.placeholder
}
