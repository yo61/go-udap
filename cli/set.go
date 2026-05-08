package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/pflag"
)

func runSet(args []string, stdout, stderr io.Writer) error {
	fs := pflag.NewFlagSet("set", pflag.ContinueOnError)
	fs.SetOutput(stderr)
	timeout := fs.Duration("timeout", 5*time.Second, "Operation timeout")
	verbose := fs.BoolP("verbose", "v", false, "Debug logging to stderr")
	configPath := fs.String("config", "", "Read parameters from FILE (use - for stdin)")

	// Register a string flag for every known UDAP parameter.
	pf := paramFlags()
	for _, p := range pf {
		fs.String(p.flagName, "", p.help)
	}

	if err := fs.Parse(args); err != nil {
		return &ExitError{Code: 1, Err: err}
	}
	if fs.NArg() != 1 {
		return &ExitError{Code: 1, Err: fmt.Errorf("set: expected exactly one MAC argument")}
	}
	mac, err := normalizeMAC(fs.Arg(0))
	if err != nil {
		return &ExitError{Code: 1, Err: err}
	}

	// Collect per-param flag values that were actually set.
	flagValues := make(map[string]string)
	for _, p := range pf {
		if !fs.Changed(p.flagName) {
			continue
		}
		v, err := fs.GetString(p.flagName)
		if err != nil {
			return &ExitError{Code: 1, Err: err}
		}
		flagValues[p.udapName] = v
	}

	// Resolve --config (file path, "-" for stdin, or unset).
	var fileContent io.Reader
	var fileLabel string
	switch {
	case *configPath == "-":
		fileContent = os.Stdin
		fileLabel = "-"
	case *configPath != "":
		f, err := os.Open(*configPath)
		if err != nil {
			return &ExitError{Code: 1, Err: fmt.Errorf("open config: %w", err)}
		}
		defer f.Close()
		fileContent = f
		fileLabel = *configPath
	}

	// Detect piped stdin (only consulted when no --config was given).
	stdinPiped := isStdinPiped()
	var stdinContent io.Reader
	if stdinPiped {
		stdinContent = os.Stdin
	}

	merged, err := mergeSources(sourceInputs{
		fileContent:  fileContent,
		fileLabel:    fileLabel,
		stdinContent: stdinContent,
		stdinPiped:   stdinPiped,
		flags:        flagValues,
	}, stderr)
	if err != nil {
		return &ExitError{Code: 1, Err: err}
	}

	client, err := newClient(*verbose)
	if err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	defer client.Close()

	device, err := discoverAndFind(client, mac, *timeout)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := client.SetDeviceConfigWithContext(ctx, device, merged); err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("set failed: %w", err)}
	}

	// Echo what we sent for confirmation, sorted.
	if err := formatParamMap(stdout, merged); err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	return nil
}

// isStdinPiped returns true when stdin is not a terminal (i.e. data is piped
// or redirected from a file). False if stdin is interactive or unavailable.
func isStdinPiped() bool {
	st, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	if (st.Mode() & os.ModeCharDevice) != 0 {
		return false
	}
	return true
}
