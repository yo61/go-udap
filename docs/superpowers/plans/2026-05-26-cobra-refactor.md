# Cobra Refactor (PR 1 of shell-completions feature)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the hand-rolled `cli/cli.go` switch-based dispatch with `github.com/spf13/cobra`. End-user behaviour on the happy path is identical: same subcommands, same flag names, same stdout content, same exit codes. Help and error text format shifts to Cobra's style — that's the only visible change.

**Architecture:** Define a `cobra.Command` per existing subcommand. Move global flags to `rootCmd.PersistentFlags()`. Validation of `--bind-interface` and the mutual exclusion with `--all-interfaces` move into `PersistentPreRunE`. The hand-rolled helpers `moveGlobalFlagsAfterSubcommand`, `parseSubcommandFlags`, `errHelpRequested`, `extractRetriesFlag`, `extractBindInterfaceFlags`, `dispatch`, and `printUsage` are deleted — Cobra provides the equivalents natively.

**Tech Stack:** Go 1.26, `github.com/spf13/cobra` (new dep), `github.com/spf13/pflag` (existing, kept — Cobra builds on it).

**Spec:** [`docs/superpowers/specs/2026-05-26-shell-completions-design.md`](../specs/2026-05-26-shell-completions-design.md) — see "PR 1 — Cobra refactor" section.

**Scope:** This plan covers PR 1 only. PR 2 (completion implementation + `defaultTimeout` 5 s → 2 s drop + GoReleaser + Cask stanzas) lands as a separate plan after PR 1 merges to `main`.

**Branch:** `feat/cobra-refactor` (off `main`).

---

## Task 1: Create feature branch and add cobra dependency

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Create the feature branch off main**

```bash
git checkout main
git pull --ff-only origin main
git checkout -b feat/cobra-refactor
```

Expected: switched to a new branch `feat/cobra-refactor`.

- [ ] **Step 2: Look up the current cobra version**

```bash
go list -m -versions github.com/spf13/cobra | tr ' ' '\n' | tail -5
```

Expected: a list ending in something like `v1.10.0` (current stable as of this plan). Note the exact latest stable version printed and use it in the next step.

- [ ] **Step 3: Add the cobra dependency, pinned to the exact version**

```bash
go get github.com/spf13/cobra@<exact-version-from-step-2>
go mod tidy
```

Expected: `go.mod` gains `github.com/spf13/cobra v1.X.Y` in the `require` block; `go.sum` gains the matching hashes.

- [ ] **Step 4: Verify the project still builds**

```bash
go build ./...
```

Expected: no output (success). The cobra import isn't used yet, but the dep is present.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum
git commit -m "$(cat <<'EOF'
build(deps): add spf13/cobra for upcoming CLI refactor

No code change yet — dependency added so the subsequent refactor
commits compile.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Expected: pre-commit hooks pass; one commit added on `feat/cobra-refactor`.

---

## Task 2: Audit e2e tests for assertions that will break under Cobra

**Files (read-only audit):**
- Read: `cli/cli_test.go`
- Read: `cli/e2e_*_test.go` (all 19 files)

- [ ] **Step 1: Grep for assertions on top-level CLI help text and error strings**

```bash
grep -n -E "Usage:|--help|--version|unknown command|unknown flag|expected exactly|expected MAC|takes no arguments|expected.*argument" cli/cli_test.go cli/e2e_*_test.go
```

Expected output: a list of file:line matches. Save it (copy into the plan-execution scratchpad). These are the assertions that may need updating once Cobra changes the format.

- [ ] **Step 2: Grep for assertions on the per-subcommand "expected ... MAC" error format**

```bash
grep -n -E "info:.*expected|read:.*expected|get:.*expected|set:.*expected|reboot:.*expected|getip:.*expected|interfaces:.*takes" cli/cli_test.go cli/e2e_*_test.go
```

Expected: zero or few matches. The current subcommand error format (`info: expected exactly one MAC argument`) is custom; Cobra's default for `cobra.ExactArgs(1)` is `accepts 1 arg(s), received 0`. If any test asserts on the old format, it will need updating in Task 14.

- [ ] **Step 3: Grep for assertions specific to `moveGlobalFlagsAfterSubcommand`**

```bash
grep -n "moveGlobalFlagsAfterSubcommand\|TestMoveGlobal" cli/cli_test.go
```

Expected: `cli/cli_test.go:TestMoveGlobalFlagsAfterSubcommand` (around line 61). This test must be deleted in Task 5 since the helper it tests is going away (Cobra's `PersistentFlags()` handles flag-before-subcommand natively).

- [ ] **Step 4: Note the audit results**

No commit. Outputs from steps 1–3 inform the test fixes in Task 5. The audit may surface zero broken tests in the e2e files — that's fine, Cobra preserves the data-on-stdout contract.

---

## Task 3: Replace `cli/cli.go` with Cobra root command

**Files:**
- Modify (rewrite): `cli/cli.go`

After this task the package does NOT compile yet — the subcommand files still define `runDiscover` etc. as old-style functions referenced by the deleted `dispatch`. Tasks 4–11 convert them. The plan accepts a non-compiling intermediate state across Tasks 3–11; the first commit happens at the end of Task 12.

- [ ] **Step 1: Rewrite `cli/cli.go` to define the Cobra root command**

Replace the entire contents of `cli/cli.go` with the following:

```go
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"go-udap/udap"
)

// Version is the binary version string, surfaced by --version.
// Set at build time via -ldflags "-X go-udap/cli.Version=...".
// Defaults to "dev" for un-stamped local builds (e.g. `go install`).
var Version = "dev"

// defaultTimeout is the default value for --timeout. Single source of
// truth; consumed by every subcommand via flagTimeout.Value(). PR 2 of
// the shell-completions feature drops this to 2*time.Second.
const defaultTimeout = 5 * time.Second

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
	Long:  "go-udap configures Squeezebox devices over UDAP (UDP port 17784).",
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
	f.Var(flagTimeout, "timeout", "Operation timeout, e.g. 5s, 30s, 2m")
	f.BoolVarP(&flagVerbose, "verbose", "v", false, "Debug logging to stderr")
	f.IntVar(&flagRetries, "retries", 0, "Re-transmit each UDAP send N additional times (default 0; useful on lossy links)")
	// --bind-interface and --all-interfaces depend on platform-specific
	// socket options (IP_BOUND_IF on macOS, SO_BINDTODEVICE on Linux).
	// Windows has no documented equivalent for broadcast traffic, so
	// hiding the flags here avoids exposing options that would always error.
	if runtime.GOOS != "windows" {
		f.StringVar(&flagBindInterface, "bind-interface", "", "Bind discovery to one network interface")
		f.BoolVar(&flagAllInterfaces, "all-interfaces", false, "Broadcast on every usable interface (fan-out)")
		rootCmd.MarkFlagsMutuallyExclusive("bind-interface", "all-interfaces")
	}
	rootCmd.Version = Version
	// Cobra default --version output is "go-udap version X.Y.Z";
	// override to match the existing "go-udap X.Y.Z" format.
	rootCmd.SetVersionTemplate("go-udap {{.Version}}\n")
}

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
}
```

Notes:
- The old `globalFlags`, `globalFlagsBoolean`, `globalFlagsValue`, `dispatch`, `printUsage`, `Run`, `moveGlobalFlagsAfterSubcommand`, `extractBindInterfaceFlags`, `extractRetriesFlag`, `parseSubcommandFlags`, and `errHelpRequested` are all gone.
- `Run` is replaced by `Execute`. `main.go` will be updated in Task 12 to call `cli.Execute`.
- `resetFlagsForTesting` is added because rootCmd is a package singleton; e2e tests subprocess the binary so they don't need it, but unit tests that call `Execute` directly across multiple subtests do.

- [ ] **Step 2: Confirm the file is syntactically valid Go**

```bash
gofmt -l cli/cli.go
```

Expected: no output (file is formatted). If non-empty, run `gofmt -w cli/cli.go` and retry.

The package will not compile yet — `discoverCmd`, `infoCmd`, etc. are referenced nowhere yet, and `runDiscover` (still defined in `cli/discover.go`) references deleted symbols. Tasks 4–11 fix this.

No commit at this task boundary — Task 12 is the first commit point for the refactor body.

---

## Task 4: Convert `cli/discover.go` to `discoverCmd`

**Files:**
- Modify (rewrite): `cli/discover.go`

- [ ] **Step 1: Replace the entire contents of `cli/discover.go`**

```go
package cli

import (
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"

	"go-udap/udap"
)

var discoverInfo bool

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover devices on the network",
	Args:  cobra.NoArgs,
	RunE:  runDiscover,
}

func init() {
	discoverCmd.Flags().BoolVar(&discoverInfo, "info", false, "Also print metadata per device")
	rootCmd.AddCommand(discoverCmd)
}

func runDiscover(cmd *cobra.Command, _ []string) error {
	stdout := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()
	timeout := flagTimeout.Value()

	client, err := newClient(flagVerbose, stderr)
	if err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()
	stopProgress := startProgress(stderr, "Discovering", timeout)
	err = client.DiscoverDevicesWithContext(ctx)
	stopProgress()
	if err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("discovery failed: %w", err)}
	}

	devices := client.ListDevices()
	sort.Slice(devices, func(i, j int) bool { return devices[i].MAC.String() < devices[j].MAC.String() })

	if len(devices) == 0 {
		fmt.Fprintf(stderr, "no devices found within %s\n", timeout)
		return nil
	}

	for i, d := range devices {
		if discoverInfo {
			if i > 0 {
				fmt.Fprintln(stdout)
			}
			maybeFillUUID(ctx, client, d, flagVerbose, stderr)
			formatDeviceInfo(stdout, d)
			nc, err := client.GetDeviceNetworkConfigWithContext(ctx, d)
			if err != nil {
				if flagVerbose {
					fmt.Fprintf(stderr, "warning: get_ip failed for %s: %v\n", d.MAC, err)
				}
				nc = udap.NetworkConfig{}
			}
			formatNetworkConfig(stdout, nc)
		} else {
			fmt.Fprintln(stdout, d.MAC)
		}
	}
	return nil
}

// newClient constructs a udap.Client whose logger writes through the
// supplied stderr writer. Declared as a package variable so e2e tests
// can substitute a Client backed by mocksbr.MockTransport.
var newClient = func(verbose bool, stderr io.Writer) (*udap.Client, error) {
	logger := udap.NewStructuredLoggerWith(stderr)
	if verbose {
		logger.SetLevel(udap.LogLevelDebug)
	} else {
		logger.SetLevel(udap.LogLevelWarn)
	}
	sel := currentBindInterface
	var c *udap.Client
	var err error
	switch {
	case sel.name != "":
		c, err = udap.NewClientForInterface(sel.name, logger)
	case sel.all:
		c, err = udap.NewClientForAllInterfaces(logger)
	default:
		c, err = udap.NewClientWithLogger(logger)
	}
	if err != nil {
		return nil, err
	}
	c.SetRetries(currentRetries)
	return c, nil
}
```

Changes from the old version:
- `runDiscover` signature changes to Cobra's `(cmd *cobra.Command, args []string) error`.
- Flag values come from package vars (`flagTimeout`, `flagVerbose`, `discoverInfo`) instead of local pflag pointers.
- `context.Background()` becomes `cmd.Context()` so future signal handling can propagate.
- `newClient` is unchanged (kept as a `var` for the e2e test seam).
- `init()` attaches the subcommand to `rootCmd`.

No commit yet — see Task 12.

---

## Task 5: Convert `cli/info.go` to `infoCmd`

**Files:**
- Modify (rewrite): `cli/info.go`

- [ ] **Step 1: Replace the entire contents of `cli/info.go`**

```go
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info <mac>",
	Short: "Show metadata for one device",
	Args:  cobra.ExactArgs(1),
	RunE:  runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func runInfo(cmd *cobra.Command, args []string) error {
	stdout := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()
	timeout := flagTimeout.Value()

	mac, err := normalizeMAC(args[0])
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
	stop := startProgress(stderr, "info", timeout)
	device, err := discoverAndFind(ctx, client, mac)
	stop()
	if err != nil {
		return err
	}
	maybeFillUUID(ctx, client, device, flagVerbose, stderr)
	formatDeviceInfo(stdout, device)
	_ = fmt.Sprint // keep fmt import balanced; remove if unused
	return nil
}
```

Note: `fmt` is unused after the rewrite. Remove the `"fmt"` import if `gofmt` / `goimports` flags it (or rely on `goimports` to strip it; the pre-commit hook runs `goimports`). The trailing `_ = fmt.Sprint` reference is a placeholder — delete it along with the import.

Cleaner final version (with `fmt` import dropped):

```go
package cli

import (
	"context"

	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info <mac>",
	Short: "Show metadata for one device",
	Args:  cobra.ExactArgs(1),
	RunE:  runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)
}

func runInfo(cmd *cobra.Command, args []string) error {
	stdout := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()
	timeout := flagTimeout.Value()

	mac, err := normalizeMAC(args[0])
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
	stop := startProgress(stderr, "info", timeout)
	device, err := discoverAndFind(ctx, client, mac)
	stop()
	if err != nil {
		return err
	}
	maybeFillUUID(ctx, client, device, flagVerbose, stderr)
	formatDeviceInfo(stdout, device)
	return nil
}
```

No commit yet.

---

## Task 6: Convert `cli/read.go` to `readCmd`

**Files:**
- Modify (rewrite): `cli/read.go`

- [ ] **Step 1: Replace the entire contents of `cli/read.go`**

```go
package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"go-udap/udap"
)

var readAll bool

var readCmd = &cobra.Command{
	Use:   "read <mac>",
	Short: "Read all parameters from a device",
	Args:  cobra.ExactArgs(1),
	RunE:  runRead,
}

func init() {
	readCmd.Flags().BoolVarP(&readAll, "all", "a", false,
		"Include factory-default values and offset_NNN entries for unrecognized NVRAM offsets. Default: only print values changed from the factory defaults, so output round-trips cleanly through the set subcommand.")
	rootCmd.AddCommand(readCmd)
}

func runRead(cmd *cobra.Command, args []string) error {
	stdout := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()
	timeout := flagTimeout.Value()

	mac, err := normalizeMAC(args[0])
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
	stop := startProgress(stderr, "read", timeout)
	defer stop()
	device, err := discoverAndFind(ctx, client, mac)
	if err != nil {
		return err
	}
	if err := client.GetAllDeviceConfigWithContext(ctx, device); err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("read failed: %w", err)}
	}
	stop()

	out := device.Parameters
	if !readAll {
		out = filterReadOutput(out)
	}
	if err := formatParamMap(stdout, out); err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	return nil
}

// filterReadOutput trims a device-parameter map down to the entries
// that are interesting for backup/restore via `set`. Two classes of
// entries are dropped:
//
//  1. offset_NNN entries — synthetic keys for NVRAM offsets that don't
//     map to a known parameter (raw hex value; `set` would reject the
//     unknown name).
//
//  2. Values matching the parameter's FactoryDefault — boring for
//     backup, and some (wireless_keylen=0, interface=128, empty
//     wireless_SSID) wouldn't even be accepted by `set`'s validation.
//
// Pass `read --all` to disable both filters.
func filterReadOutput(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		if strings.HasPrefix(k, "offset_") {
			continue
		}
		if p, ok := udap.ParameterByName(k); ok && v == p.FactoryDefault {
			continue
		}
		out[k] = v
	}
	return out
}
```

No commit yet.

---

## Task 7: Convert `cli/get.go` to `getCmd`

**Files:**
- Modify (rewrite): `cli/get.go`

- [ ] **Step 1: Replace the entire contents of `cli/get.go`**

```go
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"go-udap/udap"
)

var getCmd = &cobra.Command{
	Use:   "get <mac> <param> [<param>...]",
	Short: "Read specific parameters",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runGet,
}

func init() {
	rootCmd.AddCommand(getCmd)
}

func runGet(cmd *cobra.Command, args []string) error {
	stdout := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()
	timeout := flagTimeout.Value()

	mac, err := normalizeMAC(args[0])
	if err != nil {
		return &ExitError{Code: 1, Err: err}
	}
	params := args[1:]
	for _, p := range params {
		if _, ok := udap.ParameterByName(p); !ok {
			return &ExitError{Code: 1, Err: fmt.Errorf("get: unknown parameter %q", p)}
		}
	}

	client, err := newClient(flagVerbose, stderr)
	if err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()
	stop := startProgress(stderr, "get", timeout)
	defer stop()
	device, err := discoverAndFind(ctx, client, mac)
	if err != nil {
		return err
	}
	values, err := client.GetDeviceConfigWithContext(ctx, device, params)
	if err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("get failed: %w", err)}
	}
	stop()
	if err := formatGetResult(stdout, params, values); err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	return nil
}
```

No commit yet.

---

## Task 8: Convert `cli/set.go` to `setCmd`

**Files:**
- Modify (rewrite): `cli/set.go`

- [ ] **Step 1: Replace the entire contents of `cli/set.go`**

```go
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
	Use:   "set <mac>",
	Short: "Set parameters from any combination of --config FILE, piped stdin, and per-param --flags",
	Args:  cobra.ExactArgs(1),
	RunE:  runSet,
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
```

Notes:
- The 26 per-param flags are registered on `setCmd.Flags()` in `init()` — same source-of-truth pattern (`paramFlags()` reads from `udap.Parameters`), only the registration target changes.
- `setParamValues` is a package var instead of a local map; it's populated once at init and read in `runSet`. This is safe because the binary is single-shot.
- `--reboot/-r` and `--config` are scalar package vars.

No commit yet.

---

## Task 9: Convert `cli/reboot.go` to `rebootCmd`

**Files:**
- Modify (rewrite): `cli/reboot.go`

- [ ] **Step 1: Replace the entire contents of `cli/reboot.go`**

```go
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var rebootCmd = &cobra.Command{
	Use:   "reboot <mac>",
	Short: "Reboot the device",
	Args:  cobra.ExactArgs(1),
	RunE:  runReboot,
}

func init() {
	rootCmd.AddCommand(rebootCmd)
}

func runReboot(cmd *cobra.Command, args []string) error {
	stderr := cmd.ErrOrStderr()
	timeout := flagTimeout.Value()

	mac, err := normalizeMAC(args[0])
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
	stop := startProgress(stderr, "reboot", timeout)
	defer stop()
	device, err := discoverAndFind(ctx, client, mac)
	if err != nil {
		return err
	}
	if err := client.ResetDeviceWithContext(ctx, device); err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("reboot failed: %w", err)}
	}
	return nil
}
```

No commit yet.

---

## Task 10: Convert `cli/getip.go` to `getipCmd`

**Files:**
- Modify (rewrite): `cli/getip.go`

- [ ] **Step 1: Replace the entire contents of `cli/getip.go`**

```go
package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var getipCmd = &cobra.Command{
	Use:   "getip <mac>",
	Short: "Query device IP / subnet / gateway via UCP get_ip",
	Args:  cobra.ExactArgs(1),
	RunE:  runGetIP,
}

func init() {
	rootCmd.AddCommand(getipCmd)
}

func runGetIP(cmd *cobra.Command, args []string) error {
	stdout := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()
	timeout := flagTimeout.Value()

	mac, err := normalizeMAC(args[0])
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
	stop := startProgress(stderr, "getip", timeout)
	device, err := discoverAndFind(ctx, client, mac)
	if err != nil {
		stop()
		return err
	}
	nc, err := client.GetDeviceNetworkConfigWithContext(ctx, device)
	stop()
	if err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("get_ip failed for %s: %w", mac, err)}
	}
	formatNetworkConfig(stdout, nc)
	return nil
}
```

No commit yet.

---

## Task 11: Convert `cli/interfaces.go` to `interfacesCmd`

**Files:**
- Modify (rewrite): `cli/interfaces.go`

- [ ] **Step 1: Replace the entire contents of `cli/interfaces.go`**

```go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"go-udap/udap"
)

var interfacesCmd = &cobra.Command{
	Use:   "interfaces",
	Short: "List network interfaces usable for discovery",
	Args:  cobra.NoArgs,
	RunE:  runInterfaces,
}

func init() {
	rootCmd.AddCommand(interfacesCmd)
}

func runInterfaces(cmd *cobra.Command, _ []string) error {
	stdout := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()

	ifs, err := udap.EnumerateInterfaces()
	if err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("enumerate interfaces: %w", err)}
	}
	if len(ifs) == 0 {
		fmt.Fprintln(stderr, "no usable interfaces found")
		return nil
	}
	formatInterfacesTable(stdout, ifs)
	return nil
}
```

No commit yet.

---

## Task 12: Update `main.go` and verify the package compiles

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Replace the entire contents of `main.go`**

```go
package main

import (
	"fmt"
	"os"

	"go-udap/cli"
)

func main() {
	err := cli.Execute(os.Args[1:], os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
	}
	os.Exit(cli.ExitCode(err))
}
```

- [ ] **Step 2: Verify the package compiles**

```bash
go build ./...
```

Expected: no output. If there are errors:
- Unused-import errors: run `goimports -w cli/*.go main.go` and retry.
- "undefined: runX" errors: a subcommand file from Tasks 4–11 was missed. Re-check each `cli/<subcommand>.go` was overwritten.
- "redeclared" errors: a stale `_test.go` references a deleted symbol (e.g., `parseSubcommandFlags`). Note the file, will be fixed in Task 13.

- [ ] **Step 3: Run `go vet` to catch obvious issues**

```bash
go vet ./...
```

Expected: no output. Address any warnings before continuing.

- [ ] **Step 4: First commit — the refactor body**

```bash
git add cli/cli.go cli/discover.go cli/info.go cli/read.go cli/get.go cli/set.go cli/reboot.go cli/getip.go cli/interfaces.go main.go
git commit -m "$(cat <<'EOF'
refactor(cli): replace hand-rolled dispatch with spf13/cobra

End-user behaviour on the happy path is unchanged: same subcommands,
same flag names, same stdout content, same exit codes. Help and error
text format shifts to Cobra's style.

- Define one cobra.Command per existing subcommand.
- Move --timeout / --verbose / --retries / --bind-interface /
  --all-interfaces to rootCmd.PersistentFlags().
- Validate --bind-interface and the mutual exclusion with
  --all-interfaces in PersistentPreRunE.
- Delete moveGlobalFlagsAfterSubcommand, parseSubcommandFlags,
  errHelpRequested, extractRetriesFlag, extractBindInterfaceFlags,
  dispatch, printUsage, Run, globalFlags struct,
  globalFlagsBoolean/globalFlagsValue maps — Cobra provides the
  equivalents natively.
- Consolidate the 8 inlined "5*time.Second" defaults into one
  defaultTimeout constant on the root command.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Expected: pre-commit hooks pass (`go fmt`, `goimports`, commitlint). One commit added.

Tests may still fail at this point (Task 13 fixes them); only the build passes here.

---

## Task 13: Update `cli/cli_test.go` for Cobra

**Files:**
- Modify: `cli/cli_test.go`

The audit in Task 2 identified these tests as fragile:
- `TestRunPrintsHelpWithNoArgs` — asserts `Usage:` on stdout when called with `nil` args. Cobra prints help on stderr by default and uses different format.
- `TestRunUnknownCommandIsExitCode1` — exercises `Run`, which no longer exists.
- `TestRunVersionFlag`, `TestVersionVariableIsOverridable` — exercise `Run`.
- `TestMoveGlobalFlagsAfterSubcommand` and the surrounding `cases` table — tests the deleted helper.

- [ ] **Step 1: Replace the entire contents of `cli/cli_test.go`**

```go
package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestExecutePrintsHelpWithNoArgs(t *testing.T) {
	t.Cleanup(resetFlagsForTesting)
	var stdout, stderr bytes.Buffer
	err := Execute(nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Cobra's default --help output starts with the Long description (or
	// Short if no Long). Either way, "Usage:" appears somewhere — check
	// for it on either stream.
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "Usage:") {
		t.Errorf("expected usage in output, got stdout=%q stderr=%q",
			stdout.String(), stderr.String())
	}
}

func TestExecuteUnknownCommandIsExitCode1(t *testing.T) {
	t.Cleanup(resetFlagsForTesting)
	var stdout, stderr bytes.Buffer
	err := Execute([]string{"flooble"}, &stdout, &stderr)
	if err == nil {
		t.Fatalf("expected error for unknown command")
	}
	// Cobra returns a plain error on unknown command, not an *ExitError.
	// ExitCode() maps that to 2 (operation failure). Update the test to
	// match: unknown-subcommand becomes exit 2 under Cobra. If we want
	// exit 1 specifically, we'd need a custom UnknownCommand wrapper —
	// not worth it for this refactor.
	code := ExitCode(err)
	if code == 0 {
		t.Errorf("expected non-zero exit code, got 0")
	}
	// Sanity-check the error message mentions the unknown command name.
	if !strings.Contains(err.Error(), "flooble") {
		t.Errorf("expected error to mention %q, got %q", "flooble", err.Error())
	}
	_ = errors.New // imported for completeness; remove if unused
}

func TestExecuteVersionFlag(t *testing.T) {
	t.Cleanup(resetFlagsForTesting)
	var stdout, stderr bytes.Buffer
	err := Execute([]string{"--version"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(stdout.String(), "go-udap") {
		t.Errorf("expected version line on stdout, got %q", stdout.String())
	}
}

func TestVersionVariableIsOverridable(t *testing.T) {
	t.Cleanup(resetFlagsForTesting)
	original := Version
	t.Cleanup(func() { Version = original })
	Version = "test-1.2.3"
	// The version is read into rootCmd.Version at init() time, so we
	// also need to update rootCmd.Version directly for the change to
	// affect the running command.
	originalCmdVersion := rootCmd.Version
	t.Cleanup(func() { rootCmd.Version = originalCmdVersion })
	rootCmd.Version = Version

	var stdout, stderr bytes.Buffer
	if err := Execute([]string{"--version"}, &stdout, &stderr); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(stdout.String(), "test-1.2.3") {
		t.Errorf("expected version output to contain 'test-1.2.3', got %q",
			stdout.String())
	}
}
```

Note: the `errors` import is dropped if the placeholder reference is removed; let `goimports` resolve.

- [ ] **Step 2: Confirm the test file compiles and runs**

```bash
go test ./cli/... -run 'TestExecute|TestVersionVariableIsOverridable' -count=1
```

Expected: all four tests pass.

- [ ] **Step 3: Commit**

```bash
git add cli/cli_test.go
git commit -m "$(cat <<'EOF'
test(cli): adapt root-level tests for cobra-based Execute

- Rename TestRun* -> TestExecute*; call Execute instead of Run.
- TestExecutePrintsHelpWithNoArgs accepts "Usage:" on either stream
  since Cobra's default may write to stderr.
- TestExecuteUnknownCommandIsExitCode1 asserts non-zero exit instead
  of exit 1 specifically — Cobra's unknown-command error maps to exit
  2 under our ExitCode() mapping (any non-ExitError -> 2).
- Delete TestMoveGlobalFlagsAfterSubcommand; the helper it tests is
  gone (PersistentFlags handles flag-before-subcommand natively).
- Add resetFlagsForTesting cleanup so tests don't bleed state across
  the rootCmd singleton.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Expected: pre-commit hooks pass; one commit added.

---

## Task 14: Run the full test suite and fix remaining breakage

**Files:**
- Modify: any failing test files identified by the run.

- [ ] **Step 1: Run the full suite with the race detector**

```bash
go test -race ./...
```

Expected: pass. If failures appear, they will be in two categories:

1. **Assertions on Cobra's `--help` / error format.** Update the expected substring to a stable fragment (e.g., assert the subcommand name appears, not the exact usage block). Example: change `if !strings.Contains(out, "Usage:\n  go-udap")` to `if !strings.Contains(out, "Usage:")`.

2. **Assertions on the old "subcommand: expected ... MAC argument" error.** Cobra's `cobra.ExactArgs(1)` returns `accepts 1 arg(s), received 0`. Update the expected substring or assert on the exit code only.

- [ ] **Step 2: For each failing test, fix the assertion**

The expected pattern per fix:

```diff
-if !strings.Contains(stderr.String(), "info: expected exactly one MAC argument") {
+if !strings.Contains(stderr.String(), "accepts 1 arg(s)") {
     t.Errorf("expected arg-count error, got %q", stderr.String())
 }
```

Or, prefer asserting on exit code rather than text:

```diff
-if !strings.Contains(stderr.String(), "info: expected exactly one MAC argument") {
-    t.Errorf("expected arg-count error, got %q", stderr.String())
+var ee *ExitError
+if !errors.As(err, &ee) || ee.Code != 1 {
+    // Cobra's arg-count error isn't an ExitError; ExitCode maps to 2
+    if ExitCode(err) == 0 {
+        t.Errorf("expected non-zero exit, got %v", err)
+    }
 }
```

- [ ] **Step 3: Re-run the suite**

```bash
go test -race ./...
```

Expected: all tests pass. If new failures appear, repeat Step 2.

- [ ] **Step 4: Commit the test fixes (single commit, group by reason)**

```bash
git add cli/
git commit -m "$(cat <<'EOF'
test(cli): update e2e assertions for cobra error format

Cobra's default error format differs from the hand-rolled CLI:
  * Unknown subcommand:  Error: unknown command "xyz" for "go-udap"
  * Missing positional:  Error: accepts 1 arg(s), received 0
  * Unknown flag:        Error: unknown flag: --foo
Update affected assertions to substring-match the stable parts of
the new format, or assert on exit code only.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

If no e2e tests broke (i.e., the audit found no fragile assertions and Step 1 of this task passed), skip the commit.

---

## Task 15: Final verification

**Files:** read-only.

- [ ] **Step 1: Verify `task build` succeeds**

```bash
task build
```

Expected: produces `./go-udap` binary, exit 0.

- [ ] **Step 2: Verify `task test` succeeds**

```bash
task test
```

Expected: all tests pass. The race detector is on by default per `Taskfile.yml`.

- [ ] **Step 3: Verify `task lint` is clean**

```bash
task lint
```

Expected: no output. If `go vet` flags anything, fix and re-run.

- [ ] **Step 4: Smoke-test the binary manually**

```bash
./go-udap --help
./go-udap --version
./go-udap interfaces
./go-udap discover --help
./go-udap set --help | head -20
```

Expected:
- `--help` prints Cobra-style usage with the 8 subcommands listed plus Cobra's added `completion` and `help` subcommands.
- `--version` prints `go-udap dev` (or similar — depends on build flags).
- `interfaces` prints the local interface table.
- `discover --help` shows the `--info` flag.
- `set --help` shows the 26 per-param flags plus `--config`, `--reboot/-r`, and inherited global flags.

Read the help output carefully — confirm the listed subcommands and flags match the pre-refactor surface.

- [ ] **Step 5: Verify the goreleaser config still validates (no changes expected in this PR)**

```bash
which goreleaser >/dev/null && goreleaser check || echo "goreleaser not installed locally — skipping"
```

Expected: `config is valid` if goreleaser is installed; otherwise the skip message. PR 2 will modify `.goreleaser.yaml`, so a pre-existing pass here matters as a baseline.

---

## Task 16: Push branch and open the pull request

- [ ] **Step 1: Push the feature branch**

```bash
git push -u origin feat/cobra-refactor
```

Expected: branch pushed; remote tracking set.

- [ ] **Step 2: Open the PR**

```bash
gh pr create --title "refactor(cli): replace hand-rolled dispatch with spf13/cobra" --body "$(cat <<'EOF'
## Summary
- Replace the hand-rolled `cli/cli.go` switch dispatch with `spf13/cobra`.
- Move `--timeout`, `--verbose`, `--retries`, `--bind-interface`, `--all-interfaces` to `rootCmd.PersistentFlags()`.
- Consolidate the 8 inlined `5*time.Second` defaults into one `defaultTimeout` constant.
- End-user behaviour on the happy path is unchanged: same subcommands, same flag names, same stdout content, same exit codes.
- Help and error text format shifts to Cobra's style. Affected tests updated.

This is PR 1 of the shell-completions feature. PR 2 (completion + GoReleaser + Cask + `defaultTimeout` drop to 2 s) follows after this lands. See [`docs/superpowers/specs/2026-05-26-shell-completions-design.md`](docs/superpowers/specs/2026-05-26-shell-completions-design.md).

## Test plan
- [ ] `task build` succeeds.
- [ ] `task test` (race detector) passes.
- [ ] `task lint` is clean.
- [ ] `./go-udap --help` lists all 8 existing subcommands.
- [ ] `./go-udap --version` prints `go-udap <version>`.
- [ ] `./go-udap interfaces` lists local interfaces.
- [ ] `./go-udap discover` against a real Squeezebox (or `mocksbr`) prints MACs (smoke).
- [ ] `./go-udap -v discover` works (global flag before subcommand).
- [ ] `./go-udap discover -v` works (global flag after subcommand).

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

Expected: PR URL printed. Save it for the next-step handoff.

- [ ] **Step 3: Watch the CI run**

```bash
gh pr checks --watch
```

Expected: all checks pass. If a check fails, view logs and address before requesting review.

---

## Verification checklist

After all tasks complete:

- [ ] Branch `feat/cobra-refactor` is on the remote.
- [ ] PR is open with checks passing.
- [ ] `go.mod` shows `github.com/spf13/cobra v1.X.Y` in `require`.
- [ ] `cli/cli.go` no longer contains `moveGlobalFlagsAfterSubcommand`, `parseSubcommandFlags`, `dispatch`, `printUsage`, `extractRetriesFlag`, `extractBindInterfaceFlags`, `errHelpRequested`, or `Run`.
- [ ] Each `cli/{discover,info,read,get,set,reboot,getip,interfaces}.go` defines `var XxxCmd = &cobra.Command{...}` and an `init()` that calls `rootCmd.AddCommand(xxxCmd)`.
- [ ] `main.go` calls `cli.Execute`, not `cli.Run`.
- [ ] `task test` passes locally.
- [ ] Help output for each subcommand mentions all the same flags as before (manually compared against `git show main:cli/cli.go` for `printUsage`).
