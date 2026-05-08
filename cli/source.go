package cli

import (
	"fmt"
	"io"
)

// sourceInputs collects the three possible parameter sources for `set`.
//   - fileContent: bytes from --config FILE (or --config - for stdin); fileLabel
//     identifies it for error messages ("-" if it was explicit stdin).
//   - stdinContent: piped stdin when no explicit --config was given.
//   - stdinPiped: true if stdin was piped (caller's responsibility to detect).
//   - flags: per-param values from CLI flags.
type sourceInputs struct {
	fileContent  io.Reader
	fileLabel    string
	stdinContent io.Reader
	stdinPiped   bool
	flags        map[string]string
}

// mergeSources combines parameter sources in layered order
// (file/stdin first, then CLI flags overlay) and returns the merged map.
// Warnings (e.g. piped stdin ignored when a --config FILE was given) are
// written to warn. Returns an error if no source supplies any parameters.
func mergeSources(in sourceInputs, warn io.Writer) (map[string]string, error) {
	merged := make(map[string]string)

	switch {
	case in.fileContent != nil:
		params, err := ParseINI(in.fileContent)
		if err != nil {
			label := in.fileLabel
			if label == "" {
				label = "config"
			}
			return nil, fmt.Errorf("%s: %w", label, err)
		}
		for k, v := range params {
			merged[k] = v
		}
		if in.stdinPiped && in.fileLabel != "-" {
			fmt.Fprintln(warn, "warning: --config supplied; ignoring piped stdin")
		}
	case in.stdinPiped && in.stdinContent != nil:
		params, err := ParseINI(in.stdinContent)
		if err != nil {
			return nil, fmt.Errorf("stdin: %w", err)
		}
		for k, v := range params {
			merged[k] = v
		}
	}

	for k, v := range in.flags {
		merged[k] = v
	}

	if len(merged) == 0 {
		return nil, fmt.Errorf("no parameters specified")
	}
	return merged, nil
}
