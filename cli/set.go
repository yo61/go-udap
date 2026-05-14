package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/pflag"

	"go-udap/udap"
)

func runSet(args []string, stdout, stderr io.Writer) error {
	fs := pflag.NewFlagSet("set", pflag.ContinueOnError)
	fs.SetOutput(stderr)
	timeout := newDurationWithPlaceholder("DURATION", 5*time.Second)
	fs.Var(timeout, "timeout", "Operation timeout, e.g. 5s, 30s, 2m")
	verbose := fs.BoolP("verbose", "v", false, "Debug logging to stderr")
	reboot := fs.BoolP("reboot", "r", false, "Reboot the device after applying changes")
	configPath := fs.String("config", "", "Read parameters from `FILE` (use - for stdin)")

	// Register a string flag for every known UDAP parameter, using
	// stringWithPlaceholder so each flag's --help line shows a semantic
	// placeholder (IP, NAME, 0|1, ...) instead of pflag's default "string".
	pf := paramFlags()
	paramValues := make(map[string]*stringWithPlaceholder, len(pf))
	for _, p := range pf {
		v := newStringWithPlaceholder(p.placeholder)
		fs.Var(v, p.flagName, p.help)
		paramValues[p.udapName] = v
	}

	if err := parseSubcommandFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return &ExitError{Code: 1, Err: fmt.Errorf("set: expected exactly one MAC argument")}
	}
	mac, err := normalizeMAC(fs.Arg(0))
	if err != nil {
		return &ExitError{Code: 1, Err: err}
	}

	// Collect per-param flag values that were actually set, and reject
	// invalid ones up front (review finding #2). The INI/stdin path
	// already runs through ParseINI which calls udap.ValidateParameter;
	// before this gate, --<param> VALUE flags slipped past it and were
	// silently zero-filled by CreateSetDataPacket on parse failure.
	flagValues := make(map[string]string)
	for _, p := range pf {
		if !fs.Changed(p.flagName) {
			continue
		}
		val := paramValues[p.udapName].String()
		if err := udap.ValidateParameter(p.udapName, val); err != nil {
			return &ExitError{Code: 1, Err: fmt.Errorf("--%s: %w", p.flagName, err)}
		}
		flagValues[p.udapName] = val
	}

	// Resolve --config (file path, "-" for stdin, or unset).
	var fileContent io.Reader
	var fileLabel string
	switch {
	case *configPath == "-":
		fileContent = stdinReader
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
	stdinPiped := stdinIsPiped()
	var stdinContent io.Reader
	if stdinPiped {
		stdinContent = stdinReader
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

	client, err := newClient(*verbose, stderr)
	if err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	defer client.Close()

	device, err := deviceFromMAC(mac)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout.Value())
	defer cancel()
	stop := startProgress(stderr, "set", timeout.Value())
	defer stop()

	if err := client.SetDeviceConfigWithContext(ctx, device, merged); err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("set failed: %w", err)}
	}
	if *reboot {
		if err := client.ResetDeviceWithContext(ctx, device); err != nil {
			return &ExitError{Code: 2, Err: fmt.Errorf("set --reboot failed during reset: %w", err)}
		}
	}
	stop()

	// Echo what we sent for confirmation, sorted.
	if err := formatParamMap(stdout, merged); err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	return nil
}

// stdinReader and stdinIsPiped are package-level seams used by e2e
// tests to substitute controlled stdin sources. Production code does
// not reassign them.
var (
	stdinReader  io.Reader = os.Stdin
	stdinIsPiped           = isStdinPiped
)

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
