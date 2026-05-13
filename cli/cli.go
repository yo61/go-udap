package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

// Version is the binary version string, surfaced by --version.
// Set at build time via -ldflags "-X go-udap/cli.Version=...".
// Defaults to "dev" for un-stamped local builds (e.g. `go install`).
var Version = "dev"

// ExitError carries a process exit code alongside a message.
// Use it from subcommand handlers to control go-udap's exit status.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error { return e.Err }

// parseSubcommandFlags wraps fs.Parse and translates pflag.ErrHelp (the
// signal pflag returns when --help is requested, after it has already
// printed usage to stderr) into a nil-error, exit-0 ExitError sentinel.
// Other parse errors become exit-1 ExitErrors with the parse message.
//
// Subcommands call this and return its result directly; main.go's "error:"
// prefix is suppressed by the nil Err inside the sentinel ExitError.
func parseSubcommandFlags(fs *pflag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			return errHelpRequested
		}
		return &ExitError{Code: 1, Err: err}
	}
	return nil
}

// errHelpRequested is the sentinel returned by parseSubcommandFlags when
// --help was passed. Subcommands propagate it; cli.Run swaps it for nil
// so main.go reports exit 0 with no "error:" line.
var errHelpRequested = errors.New("help requested")

// ExitCode maps an error to a process exit code:
//   - nil           → 0
//   - *ExitError    → ee.Code
//   - any other err → 2 (operation failure)
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *ExitError
	if errors.As(err, &ee) {
		return ee.Code
	}
	return 2
}

// globalFlags holds values that apply to every subcommand.
type globalFlags struct {
	timeout time.Duration
	verbose bool
}

// Run parses the given arguments and dispatches to the appropriate subcommand.
// stdout receives all command output; stderr receives logs and warnings.
//
// Global flags (--verbose / -v, --timeout) may appear EITHER before the
// subcommand or after — `go-udap -v read <mac>` and `go-udap read -v
// <mac>` are equivalent. moveGlobalFlagsAfterSubcommand shuffles them
// past the subcommand name before dispatch so each subcommand's pflag
// FlagSet sees them in its expected position.
func Run(args []string, stdout, stderr io.Writer) error {
	args = moveGlobalFlagsAfterSubcommand(args)

	if len(args) == 0 {
		printUsage(stdout)
		return nil
	}

	switch args[0] {
	case "-h", "--help", "help":
		printUsage(stdout)
		return nil
	case "--version":
		fmt.Fprintf(stdout, "go-udap %s\n", Version)
		return nil
	}

	// Wrap stderr in a writer that serializes the progress bar with the
	// structured logger so they don't smash into each other on the same
	// terminal row. Subcommands and newClient both pull stderr from
	// here, so all writes go through the same mutex.
	syncErr := newStderrSync(stderr)

	cmd := args[0]
	subArgs := args[1:]

	err := dispatch(cmd, subArgs, stdout, syncErr)
	if errors.Is(err, errHelpRequested) {
		return nil
	}
	return err
}

func dispatch(cmd string, subArgs []string, stdout, syncErr io.Writer) error {
	switch cmd {
	case "discover":
		return runDiscover(subArgs, stdout, syncErr)
	case "info":
		return runInfo(subArgs, stdout, syncErr)
	case "read":
		return runRead(subArgs, stdout, syncErr)
	case "get":
		return runGet(subArgs, stdout, syncErr)
	case "set":
		return runSet(subArgs, stdout, syncErr)
	case "reboot":
		return runReboot(subArgs, stdout, syncErr)
	case "getip":
		return runGetIP(subArgs, stdout, syncErr)
	case "interfaces":
		return runInterfaces(subArgs, stdout, syncErr)
	default:
		return &ExitError{Code: 1, Err: fmt.Errorf("unknown command: %s", cmd)}
	}
}

// globalFlagsBoolean and globalFlagsValue list the flag forms recognized
// at the top level (i.e. before the subcommand). booleans take no value;
// value flags take the next arg, or accept the --foo=bar form.
var (
	globalFlagsBoolean = map[string]bool{
		"-v":        true,
		"--verbose": true,
	}
	globalFlagsValue = map[string]bool{
		"--timeout": true,
	}
)

// moveGlobalFlagsAfterSubcommand reorders args so leading global flags
// land after the subcommand. This lets `go-udap -v read <mac>` work in
// addition to `go-udap read -v <mac>`. Unknown flags or anything that
// doesn't look like a flag stop the scan — that token is treated as
// the subcommand.
//
// `--` is honored as the POSIX flag terminator: if it appears before
// any non-flag token, args are returned unchanged so the rest of the
// argv is treated as positional by the subcommand. (Without this guard
// the prior implementation would hoist the leading flag past `--` and
// then make `--` itself look like the subcommand name, producing
// "unknown command: --".)
func moveGlobalFlagsAfterSubcommand(args []string) []string {
	var leading []string
	i := 0
scan:
	for ; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			// POSIX terminator before subcommand — bail out and let
			// the subcommand parser see the original argv.
			return args
		}
		// --foo=bar form
		if strings.HasPrefix(a, "--") {
			if eq := strings.IndexByte(a, '='); eq > 0 {
				name := a[:eq]
				if globalFlagsBoolean[name] || globalFlagsValue[name] {
					leading = append(leading, a)
					continue
				}
				break scan
			}
		}
		if globalFlagsBoolean[a] {
			leading = append(leading, a)
			continue
		}
		if globalFlagsValue[a] {
			if i+1 >= len(args) {
				// Missing value — leave for subcommand parser to error on.
				break scan
			}
			leading = append(leading, a, args[i+1])
			i++
			continue
		}
		// Either a non-flag (subcommand) or an unknown flag — stop.
		break scan
	}
	if len(leading) == 0 || i >= len(args) {
		return args
	}
	rest := args[i:]
	out := make([]string, 0, len(args))
	out = append(out, rest[0])     // subcommand
	out = append(out, leading...)  // hoisted global flags
	out = append(out, rest[1:]...) // subcommand args
	return out
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, `go-udap — Squeezebox UDAP configuration tool

Usage:
  go-udap [global flags] <command> [args] [flags]

Commands:
  discover [--info]              Discover devices on the network
  info <mac>                     Show metadata for one device
  read <mac>                     Read all parameters from a device
  get <mac> <param> [<param>...] Read specific parameters
  set <mac> [--reboot] [--config FILE] [--<param> VALUE ...]
                                 Set parameters from any combination of
                                 --config FILE (or --config - for stdin),
                                 piped stdin, and per-param --flags.
                                 Pass --reboot/-r to also reboot the device
                                 after writing (the wire op writes NVRAM
                                 immediately, but some changes only take
                                 effect after reboot).
  reboot <mac>                   Reboot the device
  getip <mac>                    Query device IP / subnet / gateway via UCP get_ip
  interfaces                     List network interfaces usable for discovery

Global flags:
  --timeout DURATION  Operation timeout (default 5s)
  --verbose, -v       Debug logging to stderr
  --version           Print version and exit
  --help, -h          Print this help`)
}
