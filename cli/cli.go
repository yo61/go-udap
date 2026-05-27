package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"go-udap/udap"
)

// Version is the binary version string, surfaced by --version.
// Set at build time via -ldflags "-X go-udap/cli.Version=...".
// Defaults to "dev" for un-stamped local builds (e.g. `go install`).
var Version = "dev"

// defaultTimeout is the default value for --timeout. Single source of
// truth; consumed by every subcommand via flagTimeout.Value(). PR 2 of
// the shell-completions feature drops this to 2*time.Second.
const defaultTimeout = 2 * time.Second

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
	ee, ok := errors.AsType[*ExitError](err)
	if ok {
		return ee.Code
	}
	return 2
}

// bindInterfaceSelection captures the global --bind-interface /
// --all-interfaces flags. Populated by rootCmd.PersistentPreRunE before
// any subcommand RunE executes.
type bindInterfaceSelection struct {
	name string // empty unless --bind-interface was set
	all  bool   // true if --all-interfaces was set
}

var currentBindInterface bindInterfaceSelection

// currentRetries holds the --retries N flag value. Populated by
// PersistentPreRunE.
var currentRetries int

// Package-level flag-value holders. Cobra reads into these; subcommand
// RunE bodies read out of them. Module-level state is intentional —
// the CLI process is single-shot and these vars are reset to defaults
// when the binary re-runs.
var (
	flagTimeout       = newDurationWithPlaceholder("DURATION", defaultTimeout)
	flagVerbose       bool
	flagRetries       int
	flagBindInterface string
	flagAllInterfaces bool
)

// rootCmd is the entry point for the CLI. Subcommands are attached in
// init() functions in their respective files.
var rootCmd = &cobra.Command{
	Use:   "go-udap",
	Short: "Squeezebox UDAP configuration tool",
	Long: `go-udap discovers and configures Squeezebox devices over UDAP
(Universal Device Access Protocol) on UDP port 17784.

It is single-shot: every invocation runs one subcommand to completion
and exits. There is no daemon or interactive shell. Subcommands map
one-to-one onto UDAP wire methods (discover, get_data, set_data, reset,
get_ip, get_uuid).

UDAP only talks to devices in setup mode (the front light flashes red).
Brand-new devices arrive in setup mode; existing devices can be put
back into it by holding the front button for 3-6 seconds.`,
	// SilenceUsage prevents Cobra printing the full usage block on every
	// RunE error — the CLI prints only "error: <msg>" via main.go.
	SilenceUsage: true,
	// SilenceErrors prevents Cobra printing the error itself; main.go
	// handles that with the "error:" prefix.
	SilenceErrors: true,
	// PersistentPreRunE runs before every subcommand RunE. It populates
	// currentBindInterface and currentRetries from the parsed flags and
	// validates the --bind-interface / --all-interfaces combination.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		currentRetries = flagRetries
		sel := bindInterfaceSelection{name: flagBindInterface, all: flagAllInterfaces}
		if sel.name != "" {
			ifs, err := udap.EnumerateInterfaces()
			if err != nil {
				return &ExitError{Code: 2, Err: fmt.Errorf("enumerate interfaces: %w", err)}
			}
			found := false
			for _, iface := range ifs {
				if iface.Name == sel.name {
					found = true
					break
				}
			}
			if !found {
				return &ExitError{Code: 1, Err: fmt.Errorf("--bind-interface: %q is not usable (must be up, broadcast-capable, with an IPv4 address)", sel.name)}
			}
		}
		currentBindInterface = sel
		return nil
	},
}

func init() {
	f := rootCmd.PersistentFlags()
	f.Var(flagTimeout, "timeout", "Operation timeout, e.g. 2s, 30s, 2m")
	f.BoolVarP(&flagVerbose, "verbose", "v", false, "Debug logging to stderr")
	f.Var(newIntWithPlaceholder("N", 0, &flagRetries), "retries", "Re-transmit each UDAP send N additional times (default 0; useful on lossy links)")
	// --bind-interface and --all-interfaces are accepted on all platforms.
	// Validation (against EnumerateInterfaces) runs in PersistentPreRunE on
	// every platform — unknown names exit 1 everywhere. Platform-specific
	// behaviour is in newClient: Windows socket-binding returns "not
	// supported" when the flag is actually used.
	f.StringVar(&flagBindInterface, "bind-interface", "", "Bind discovery to one network interface")
	f.BoolVar(&flagAllInterfaces, "all-interfaces", false, "Broadcast on every usable interface (fan-out)")
	rootCmd.MarkFlagsMutuallyExclusive("bind-interface", "all-interfaces")
	if err := rootCmd.RegisterFlagCompletionFunc("bind-interface", completeInterfaces); err != nil {
		panic(fmt.Sprintf("register bind-interface completion: %v", err))
	}
	rootCmd.Version = Version
	// Cobra default --version output is "go-udap version X.Y.Z";
	// override to match the existing "go-udap X.Y.Z" format.
	rootCmd.SetVersionTemplate("go-udap {{.Version}}\n")
}

// Root returns the assembled cobra command tree. Intended for tooling
// that walks the tree without running it — e.g. the man-page generator
// in cmd/docs. The tree is a package-level singleton populated by init()
// side-effects in each subcommand file; callers must not mutate it.
func Root() *cobra.Command { return rootCmd }

// Execute parses args and runs the appropriate subcommand. stdout and
// stderr are the writers the command tree should produce output through;
// stderr is wrapped in a stderrSync writer so the progress bar and the
// structured logger don't smash together on the same terminal row.
func Execute(args []string, stdout, stderr io.Writer) error {
	syncErr := newStderrSync(stderr)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(syncErr)
	rootCmd.SetArgs(args)
	return rootCmd.ExecuteContext(context.Background())
}

// resetFlagsForTesting restores all package-level flag holders to their
// defaults. Tests that call Execute multiple times must invoke this in
// t.Cleanup to avoid bleed-through between runs.
func resetFlagsForTesting() {
	flagTimeout = newDurationWithPlaceholder("DURATION", defaultTimeout)
	flagVerbose = false
	flagRetries = 0
	flagBindInterface = ""
	flagAllInterfaces = false
	currentBindInterface = bindInterfaceSelection{}
	currentRetries = 0
	// Re-register the PersistentFlag for --timeout since the holder is
	// a fresh instance. (Other flag vars are scalars — pflag already
	// holds a pointer, so they don't need re-registration.)
	if f := rootCmd.PersistentFlags().Lookup("timeout"); f != nil {
		f.Value = flagTimeout
	}
	// Reset pflag's Changed tracking across the entire command tree so
	// mutual-exclusion checks and "was this flag set?" logic don't bleed
	// between calls on the rootCmd singleton.
	resetChangedInTree(rootCmd)
	// Also reset per-subcommand vars that Cobra doesn't manage through
	// pflag's Changed field. These are the underlying Go vars that flag
	// callbacks write to; resetting them prevents bleed across Execute calls.
	discoverInfo = false
	readAll = false
	setReboot = false
	setConfigPath = ""
}

// resetChangedInTree recursively clears the Changed flag on every pflag
// flag in a cobra command and all its subcommands.
func resetChangedInTree(cmd *cobra.Command) {
	clearChanged := func(f *pflag.Flag) { f.Changed = false }
	cmd.PersistentFlags().VisitAll(clearChanged)
	cmd.Flags().VisitAll(clearChanged)
	for _, sub := range cmd.Commands() {
		resetChangedInTree(sub)
	}
}
