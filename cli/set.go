package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"go-udap/udap"
)

var (
	setReboot     bool
	setConfigPath string
	// setParamValues holds one stringWithPlaceholder per known UDAP
	// parameter, keyed by canonical wire name. Populated in init() and
	// read in runSet to decide which --<param> flags were given.
	setParamValues = make(map[string]*stringWithPlaceholder)
)

var setCmd = &cobra.Command{
	Use:               "set <mac>",
	Short:             "Set parameters from any combination of --config FILE, piped stdin, and per-param --flags",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeMACs,
	RunE:              runSet,
}

func init() {
	setCmd.Flags().BoolVarP(&setReboot, "reboot", "r", false, "Reboot the device after applying changes")
	setCmd.Flags().StringVar(&setConfigPath, "config", "", "Read parameters from `FILE` (use - for stdin)")

	// Register one --<flag> for every known UDAP parameter, using
	// stringWithPlaceholder so each flag's --help line shows a semantic
	// placeholder (IP, NAME, 0|1, ...) instead of pflag's default "string".
	for _, p := range paramFlags() {
		v := newStringWithPlaceholder(p.placeholder)
		setCmd.Flags().Var(v, p.flagName, p.help)
		setParamValues[p.udapName] = v
	}

	rootCmd.AddCommand(setCmd)
}

func runSet(cmd *cobra.Command, args []string) error {
	stdout := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()
	timeout := flagTimeout.Value()

	mac, err := normalizeMAC(args[0])
	if err != nil {
		return &ExitError{Code: 1, Err: err}
	}

	// Collect per-param flag values that were actually set, and reject
	// invalid ones up front. The INI/stdin path validates via
	// ParseINI -> udap.ValidateParameter; this matches that gate for
	// flag input.
	flagValues := make(map[string]string)
	for _, p := range paramFlags() {
		if !cmd.Flags().Changed(p.flagName) {
			continue
		}
		val := setParamValues[p.udapName].String()
		if err := udap.ValidateParameter(p.udapName, val); err != nil {
			return &ExitError{Code: 1, Err: fmt.Errorf("--%s: %w", p.flagName, err)}
		}
		flagValues[p.udapName] = val
	}

	// Resolve --config (file path, "-" for stdin, or unset).
	var fileContent io.Reader
	var fileLabel string
	switch {
	case setConfigPath == "-":
		fileContent = stdinReader
		fileLabel = "-"
	case setConfigPath != "":
		f, err := os.Open(setConfigPath)
		if err != nil {
			return &ExitError{Code: 1, Err: fmt.Errorf("open config: %w", err)}
		}
		defer f.Close()
		fileContent = f
		fileLabel = setConfigPath
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

	client, err := newClient(flagVerbose, stderr)
	if err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()
	stop := startProgress(stderr, "set", timeout)
	defer stop()
	device, err := discoverAndFind(ctx, client, mac)
	if err != nil {
		return err
	}

	// Pre-populate device.Parameters via an explicit read so
	// applyInterfaceDefault can inspect the device's current interface
	// byte. SetDeviceConfigWithContext's own RMW skips its prelude read
	// when device.Parameters is already populated, so this is one read,
	// not two.
	if err := client.GetAllDeviceConfigWithContext(ctx, device); err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("read current parameters: %w", err)}
	}
	applyInterfaceDefault(merged, device, stderr)

	if err := client.SetDeviceConfigWithContext(ctx, device, merged); err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("set failed: %w", err)}
	}
	if setReboot {
		if err := client.ResetDeviceWithContext(ctx, device); err != nil {
			return &ExitError{Code: 2, Err: fmt.Errorf("set --reboot failed during reset: %w", err)}
		}
	}
	stop()

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

// isStdinPiped returns true when stdin is not a terminal.
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
