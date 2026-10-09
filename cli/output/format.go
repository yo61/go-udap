// Package output renders a command's Result to stdout in the chosen
// Output format: text, json or csv.
package output

import (
	"errors"
	"slices"
	"strings"
)

// Format is an Output format. The zero value is Text. It implements
// pflag.Value so it can back the --format flag directly.
type Format int

// The Output formats, in the order they are listed to the user.
const (
	Text Format = iota
	JSON
	CSV
)

var formatNames = []string{"text", "json", "csv"}

var errInvalidFormat = errors.New("want " + strings.Join(formatNames, ", "))

// Names returns the format names, as accepted by Set.
func Names() []string { return slices.Clone(formatNames) }

func (f Format) String() string { return formatNames[f] }

// Set parses an exact, lowercase format name.
func (f *Format) Set(value string) error {
	for i, name := range formatNames {
		if value == name {
			*f = Format(i)
			return nil
		}
	}
	return errInvalidFormat
}

// Type is the placeholder pflag shows in help: "-o, --format FORMAT".
func (f *Format) Type() string { return "FORMAT" }
