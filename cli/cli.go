package cli

import (
	"errors"
	"fmt"
	"io"
	"time"
)

// Version is the binary version string, surfaced by --version.
// Updated manually for now; release tooling can wire this to the git tag later.
const Version = "0.2.0"

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
func Run(args []string, stdout, stderr io.Writer) error {
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

	cmd := args[0]
	subArgs := args[1:]

	switch cmd {
	case "discover":
		return runDiscover(subArgs, stdout, stderr)
	case "info":
		return runInfo(subArgs, stdout, stderr)
	case "read":
		return runRead(subArgs, stdout, stderr)
	case "get":
		return runGet(subArgs, stdout, stderr)
	case "set":
		return runSet(subArgs, stdout, stderr)
	case "save":
		return runSave(subArgs, stdout, stderr)
	case "reset":
		return runReset(subArgs, stdout, stderr)
	case "commit":
		return runCommit(subArgs, stdout, stderr)
	default:
		return &ExitError{Code: 1, Err: fmt.Errorf("unknown command: %s", cmd)}
	}
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
  set <mac> [--config FILE] [--<param> VALUE ...]
                                 Set parameters from any combination of
                                 --config FILE (or --config - for stdin),
                                 piped stdin, and per-param --flags.
  save <mac>                     Save current config to NVRAM
  reset <mac>                    Reboot the device
  commit <mac>                   Save then reset

Global flags:
  --timeout DURATION  Operation timeout (default 5s)
  --verbose, -v       Debug logging to stderr
  --version           Print version and exit
  --help, -h          Print this help`)
}
