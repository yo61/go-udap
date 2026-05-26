# Shell Completions Implementation Plan (PR 2 of shell-completions feature)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship bash, zsh, and fish tab-completion for `go-udap`. Completions cover subcommands, flag names, the 26 NVRAM parameter names, dynamic `<mac>` arguments (live UDAP discovery with progressive timeout), and dynamic `--bind-interface` values (instant local enumeration). Distribution: completion scripts ship in the release tarball and are symlinked by the Homebrew Cask's `bash_completion` / `zsh_completion` / `fish_completion` stanzas.

**Architecture:** New `cli/completion.go` defines completion helpers (`completeMACs`, `completeInterfaces`, `completeParameterNames`) and a hidden `__dump-completions` build-time subcommand. Each subcommand file gets a `ValidArgsFunction` wired to the right helper; `RegisterFlagCompletionFunc("bind-interface", ...)` handles flag-value completion. A per-shell-session state file at `$TMPDIR/go-udap-complete-$PPID` carries a 1-byte "previous result count" signal so the second tab in a row (within 10 s) escalates the discovery timeout from 500 ms to `defaultTimeout`.

**Tech Stack:** Go 1.26, `github.com/spf13/cobra` (already a dep, used for `GenBashCompletionFileV2` / `GenZshCompletionFile` / `GenFishCompletionFile` and the `ValidArgsFunction` / `RegisterFlagCompletionFunc` hooks), GoReleaser v2.16+ (`homebrew_casks.completions:` map and `before:` hooks), Homebrew Cask v4+ stanzas.

**Spec:** [`docs/superpowers/specs/2026-05-26-shell-completions-design.md`](../specs/2026-05-26-shell-completions-design.md) — see "PR 2 — Completion implementation" section.

**Scope:** This plan covers PR 2 only. PR 1 (Cobra refactor) is already merged on `main` (commit `0472fc8`). This plan also drops the global `--timeout` default from 5 s to 2 s so the completion retry tier matches it.

**Branch:** `feat/shell-completions` (off `main`).

---

## Task 1: Create feature branch and verify post-PR-1 state

**Files (read-only check):**
- `cli/cli.go` — should contain `const defaultTimeout = 5 * time.Second` and Cobra `rootCmd`.
- `cli/params.go` — should contain `intWithPlaceholder` (added in PR 1 fix commit).

- [ ] **Step 1: Create the feature branch off main**

```bash
git checkout main
git pull --ff-only origin main
git checkout -b feat/shell-completions
git log --oneline -3
```

Expected: top commit is `0472fc8 refactor(cli): replace hand-rolled dispatch with spf13/cobra (#83)`. New branch is `feat/shell-completions`.

- [ ] **Step 2: Verify the post-PR-1 surface**

```bash
grep -n "defaultTimeout" cli/cli.go
grep -n "intWithPlaceholder" cli/params.go
go build ./... && task test
```

Expected:
- `cli/cli.go` defines `const defaultTimeout = 5 * time.Second`
- `cli/params.go` has `intWithPlaceholder`, `newIntWithPlaceholder`, etc.
- Build passes, tests pass.

If anything is missing the branch isn't off the right commit; sync and try again.

---

## Task 2: Drop `defaultTimeout` from 5 s to 2 s

**Files:**
- Modify: `cli/cli.go` — one-line constant change.
- Read: `cli/e2e_timeout_test.go` — audit for assertions that assume the default value.
- Possibly modify: `cli/e2e_timeout_test.go` — pin tests to `--timeout 5s` explicitly if they relied on default 5 s.
- Modify: `CLAUDE.md` — update the "Network timeouts default to 5 seconds" line.

- [ ] **Step 1: Read the timeout e2e test to assess impact**

```bash
cat cli/e2e_timeout_test.go
```

Look for assertions like `if elapsed > 5*time.Second` or `if elapsed < 5*time.Second`. Note them — they may need adjusting.

- [ ] **Step 2: Drop the constant**

In `cli/cli.go`, change:

```go
const defaultTimeout = 5 * time.Second
```

to:

```go
const defaultTimeout = 2 * time.Second
```

- [ ] **Step 3: Update CLAUDE.md**

In `CLAUDE.md`, find the line:

```
- Network timeouts default to 5 seconds (configurable via `--timeout`)
```

and change `5 seconds` to `2 seconds`. Also find:

```
Global flags: `--timeout DURATION` (default 5s), ...
```

and change `default 5s` to `default 2s`.

- [ ] **Step 4: Update flag help text**

In `cli/cli.go`, change the `--timeout` registration line:

```go
f.Var(flagTimeout, "timeout", "Operation timeout, e.g. 5s, 30s, 2m")
```

to:

```go
f.Var(flagTimeout, "timeout", "Operation timeout, e.g. 2s, 30s, 2m")
```

(Cosmetic; the placeholder examples shouldn't promise a default value that no longer exists.)

- [ ] **Step 5: Audit and fix e2e_timeout_test.go**

If the test contains an assertion like "elapsed >= 5 s", either:

a) Add an explicit `--timeout 5s` to the invocation so the test's original intent is preserved, OR
b) Change the expected elapsed to `>= 2 s`.

Choose based on what the test was actually validating. If unsure, prefer (a) — it preserves the original semantic.

- [ ] **Step 6: Run tests**

```bash
task test
```

Expected: pass. If `e2e_timeout_test.go` fails, fix per Step 5.

- [ ] **Step 7: Commit**

```bash
git add cli/cli.go CLAUDE.md cli/e2e_timeout_test.go
git commit -m "$(cat <<'EOF'
feat(cli): drop default --timeout from 5s to 2s

UDAP devices respond within ~100 ms when present; 5 s was overly
generous. Tab-completion's retry tier (added in this PR) uses
defaultTimeout, so dropping to 2s also bounds the worst-case
completion delay to a still-tolerable value.

Update CLAUDE.md and the flag help text. The e2e timeout test pins
--timeout 5s explicitly where the original intent was to validate
the default value.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Add unit tests for completion helpers (TDD)

**Files:**
- Create: `cli/completion_test.go` — unit tests for `nextMACTimeout`, `recordMACAttempt`, `completionStatePath`, `completeInterfaces`, `completeParameterNames`.

Tests come BEFORE the implementation (TDD). The implementation lands in Task 4 and Task 5. Until Task 4 runs, these tests fail to compile (the symbols they reference don't exist yet).

- [ ] **Step 1: Create `cli/completion_test.go`**

```go
package cli

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestCompletionStatePath_ScopedByPPID(t *testing.T) {
	got := completionStatePath()
	ppid := strconv.Itoa(os.Getppid())
	if !strings.Contains(got, ppid) {
		t.Errorf("completionStatePath() = %q, want contains PPID %q", got, ppid)
	}
}

func TestNextMACTimeout_ColdWhenNoState(t *testing.T) {
	dir := t.TempDir()
	prev := completionStateDir
	completionStateDir = func() string { return dir }
	t.Cleanup(func() { completionStateDir = prev })

	got := nextMACTimeout()
	if got != 500*time.Millisecond {
		t.Errorf("nextMACTimeout() = %v, want 500ms (cold)", got)
	}
}

func TestNextMACTimeout_ColdWhenStateStale(t *testing.T) {
	dir := t.TempDir()
	prev := completionStateDir
	completionStateDir = func() string { return dir }
	t.Cleanup(func() { completionStateDir = prev })

	// Write state file with mtime 15s ago (older than the 10s freshness window).
	path := completionStatePath()
	if err := os.WriteFile(path, []byte("0"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-15 * time.Second)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}

	got := nextMACTimeout()
	if got != 500*time.Millisecond {
		t.Errorf("nextMACTimeout() = %v, want 500ms (stale)", got)
	}
}

func TestNextMACTimeout_RetryWhenRecentEmpty(t *testing.T) {
	dir := t.TempDir()
	prev := completionStateDir
	completionStateDir = func() string { return dir }
	t.Cleanup(func() { completionStateDir = prev })

	path := completionStatePath()
	if err := os.WriteFile(path, []byte("0"), 0o600); err != nil {
		t.Fatal(err)
	}

	got := nextMACTimeout()
	if got != defaultTimeout {
		t.Errorf("nextMACTimeout() = %v, want defaultTimeout %v (retry tier)", got, defaultTimeout)
	}
}

func TestNextMACTimeout_ColdWhenRecentNonempty(t *testing.T) {
	dir := t.TempDir()
	prev := completionStateDir
	completionStateDir = func() string { return dir }
	t.Cleanup(func() { completionStateDir = prev })

	path := completionStatePath()
	if err := os.WriteFile(path, []byte("3"), 0o600); err != nil {
		t.Fatal(err)
	}

	got := nextMACTimeout()
	if got != 500*time.Millisecond {
		t.Errorf("nextMACTimeout() = %v, want 500ms (recent-nonempty stays cold)", got)
	}
}

func TestRecordMACAttempt_WritesCount(t *testing.T) {
	dir := t.TempDir()
	prev := completionStateDir
	completionStateDir = func() string { return dir }
	t.Cleanup(func() { completionStateDir = prev })

	recordMACAttempt(5)

	data, err := os.ReadFile(completionStatePath())
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}
	if string(data) != "5" {
		t.Errorf("state file = %q, want %q", string(data), "5")
	}
}

func TestRecordMACAttempt_TolerantOfReadOnlyDir(t *testing.T) {
	// Point at a directory that doesn't exist — write must fail silently.
	prev := completionStateDir
	completionStateDir = func() string {
		return filepath.Join(t.TempDir(), "does-not-exist", "deeper")
	}
	t.Cleanup(func() { completionStateDir = prev })

	// Should not panic; should not propagate error.
	recordMACAttempt(7)
}

func TestCompleteInterfaces_ReturnsLocalInterfaces(t *testing.T) {
	cmd := &cobra.Command{Use: "fake"}
	out, directive := completeInterfaces(cmd, nil, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want NoFileComp", directive)
	}
	if len(out) == 0 {
		t.Skip("no usable interfaces on this host; skipping")
	}
	for _, entry := range out {
		if !strings.Contains(entry, "\t") {
			t.Errorf("entry %q missing tab separator (expected Name\\tAddr)", entry)
		}
	}
}

func TestCompleteParameterNames_FiltersAlreadyListed(t *testing.T) {
	cmd := &cobra.Command{Use: "get"}
	// args[0] is the MAC; args[1...] are already-listed param names.
	args := []string{"aa:bb:cc:dd:ee:ff", "server_address"}
	out, directive := completeParameterNames(cmd, args, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want NoFileComp", directive)
	}
	for _, entry := range out {
		name := strings.SplitN(entry, "\t", 2)[0]
		if name == "server_address" {
			t.Errorf("server_address should be filtered out, got %q in results", entry)
		}
	}
	if len(out) == 0 {
		t.Error("expected non-empty parameter list after filtering one entry")
	}
}

func TestCompleteParameterNames_EmptyArgsDelegatesToMACs(t *testing.T) {
	// When args is empty (the user is still typing the MAC), the helper
	// should delegate to completeMACs. We can't easily exercise the live
	// discovery path here; just verify the helper accepts empty args
	// without panicking and returns the MAC-completer's directive.
	cmd := &cobra.Command{Use: "get"}
	_, directive := completeParameterNames(cmd, nil, "")
	if directive != cobra.ShellCompDirectiveNoFileComp && directive != cobra.ShellCompDirectiveError {
		t.Errorf("directive = %v, want NoFileComp or Error", directive)
	}
}
```

- [ ] **Step 2: Verify the tests don't compile yet (expected)**

```bash
go test ./cli/... -run TestCompletion -count=1 2>&1 | head -10
```

Expected: errors like `undefined: completionStateDir`, `undefined: nextMACTimeout`, etc. These get defined in Task 4.

No commit at this step — the test file and the implementation it tests land in one commit at the end of Task 4 (TDD red-green-commit cycle for the helpers as a unit).

---

## Task 4: Implement completion helpers (cli/completion.go)

**Files:**
- Create: `cli/completion.go`

- [ ] **Step 1: Create `cli/completion.go`**

```go
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"go-udap/udap"
)

// completionStateDir is a seam for tests; production uses os.TempDir.
var completionStateDir = os.TempDir

// completionStatePath returns the per-shell-session state file path. The
// PPID scopes it to the invoking shell so two terminal sessions don't
// collide.
func completionStatePath() string {
	return filepath.Join(completionStateDir(),
		fmt.Sprintf("go-udap-complete-%d", os.Getppid()))
}

// nextMACTimeout picks the discovery timeout for completeMACs based on
// the previous attempt within the freshness window.
//
//   - File missing or mtime older than 10 s → cold tier (500 ms).
//   - File fresh AND last attempt returned 0 devices → retry tier
//     (defaultTimeout, currently 2 s).
//   - File fresh AND last attempt returned >0 devices → stay cold; the
//     network is responsive, no need to escalate.
func nextMACTimeout() time.Duration {
	const cold = 500 * time.Millisecond
	info, err := os.Stat(completionStatePath())
	if err != nil || time.Since(info.ModTime()) > 10*time.Second {
		return cold
	}
	data, err := os.ReadFile(completionStatePath())
	if err != nil {
		return cold
	}
	var n int
	if _, err := fmt.Sscanf(string(data), "%d", &n); err != nil {
		return cold
	}
	if n == 0 {
		return defaultTimeout
	}
	return cold
}

// recordMACAttempt writes the result count of the just-completed MAC
// completion attempt. Errors are swallowed — completion still works on
// the cold tier if state-writes fail.
func recordMACAttempt(count int) {
	_ = os.WriteFile(completionStatePath(),
		[]byte(strconv.Itoa(count)), 0o600)
}

// newClientForCompletion wraps newClient but discards all logger output
// so completion subprocesses don't leak warnings to the user's prompt.
var newClientForCompletion = func(_ *cobra.Command) (*udap.Client, error) {
	// verbose=false would still go through the logger; route to io.Discard
	// to silence everything regardless of --verbose.
	return newClient(false, io.Discard)
}

// completeMACs is the ValidArgsFunction for info/read/get/set/reboot/getip.
// It runs a short-timeout UDAP discovery and returns "<MAC>\t<Name>" entries
// for Cobra to render. The first tab uses a 500 ms budget; if that returns
// empty, a second tab within 10 s uses defaultTimeout (the global --timeout
// default, currently 2 s).
func completeMACs(cmd *cobra.Command, args []string, _ string) (
	[]string, cobra.ShellCompDirective,
) {
	if len(args) >= 1 {
		// MAC already supplied; no more positional completion.
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), nextMACTimeout())
	defer cancel()
	client, err := newClientForCompletion(cmd)
	if err != nil {
		recordMACAttempt(0)
		return nil, cobra.ShellCompDirectiveError
	}
	defer client.Close()
	_ = client.DiscoverDevicesWithContext(ctx) // soft-fail; partial is fine
	devices := client.ListDevices()
	out := make([]string, 0, len(devices))
	for _, d := range devices {
		if d.Name != "" {
			out = append(out, fmt.Sprintf("%s\t%s", d.MAC, d.Name))
		} else {
			out = append(out, d.MAC.String())
		}
	}
	recordMACAttempt(len(out))
	return out, cobra.ShellCompDirectiveNoFileComp
}

// completeInterfaces is the flag-value completer for --bind-interface.
// Uses udap.EnumerateInterfaces() — instant, local, no network.
func completeInterfaces(_ *cobra.Command, _ []string, _ string) (
	[]string, cobra.ShellCompDirective,
) {
	ifs, err := udap.EnumerateInterfaces()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(ifs))
	for _, i := range ifs {
		out = append(out, fmt.Sprintf("%s\t%s", i.Name, i.Addr))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

// completeParameterNames is the ValidArgsFunction for `get <mac> <param>...`.
// First positional arg (MAC) delegates to completeMACs; subsequent args
// complete from udap.Parameters, filtering out names already on the line.
func completeParameterNames(cmd *cobra.Command, args []string, toComplete string) (
	[]string, cobra.ShellCompDirective,
) {
	if len(args) == 0 {
		return completeMACs(cmd, args, toComplete)
	}
	out := make([]string, 0, len(udap.Parameters))
	for _, p := range udap.Parameters {
		if slices.Contains(args[1:], p.Name) {
			continue
		}
		out = append(out, fmt.Sprintf("%s\t%s", p.Name, p.Help))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

// dumpCompletionsCmd is hidden; GoReleaser's before-hook invokes it to
// produce the three completion scripts that ship in the release archive.
var dumpCompletionsCmd = &cobra.Command{
	Use:    "__dump-completions <out-dir>",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := args[0]
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		if err := rootCmd.GenBashCompletionFileV2(
			filepath.Join(dir, "go-udap.bash"), true); err != nil {
			return err
		}
		if err := rootCmd.GenZshCompletionFile(
			filepath.Join(dir, "_go-udap")); err != nil {
			return err
		}
		return rootCmd.GenFishCompletionFile(
			filepath.Join(dir, "go-udap.fish"), true)
	},
}

func init() {
	rootCmd.AddCommand(dumpCompletionsCmd)
}
```

- [ ] **Step 2: Run the unit tests from Task 3**

```bash
go test ./cli/... -run TestCompletion -count=1 -v
```

Expected: all 10 tests pass.

- [ ] **Step 3: Run the full test suite**

```bash
go test -race ./... -count=1
```

Expected: pass.

- [ ] **Step 4: Smoke-test `__dump-completions`**

```bash
mkdir -p /tmp/go-udap-completions-test
go run . __dump-completions /tmp/go-udap-completions-test
ls /tmp/go-udap-completions-test
```

Expected: `go-udap.bash`, `_go-udap`, `go-udap.fish` all present and non-empty.

Quick syntax check (these are likely already installed on macOS — skip what isn't):

```bash
bash -n /tmp/go-udap-completions-test/go-udap.bash && echo bash OK
zsh -n /tmp/go-udap-completions-test/_go-udap && echo zsh OK
fish -c "source /tmp/go-udap-completions-test/go-udap.fish" && echo fish OK
```

Expected: each command prints `OK`.

Clean up:

```bash
rm -rf /tmp/go-udap-completions-test
```

- [ ] **Step 5: Commit**

```bash
git add cli/completion.go cli/completion_test.go
git commit -m "$(cat <<'EOF'
feat(cli): add completion helpers and __dump-completions subcommand

cli/completion.go defines four ValidArgsFunction / flag-completer
helpers — completeMACs (live UDAP discovery with progressive
timeout), completeInterfaces (instant local enumeration),
completeParameterNames (udap.Parameters minus already-listed) — plus
a hidden __dump-completions subcommand that emits bash/zsh/fish
scripts into a directory. GoReleaser's before-hook will call
__dump-completions at release time so the scripts ride along in the
tarball.

Per-shell-session state file at $TMPDIR/go-udap-complete-$PPID with
10s freshness window: first tab uses 500ms discovery, second tab
within window uses defaultTimeout (2s). State file errors are
swallowed; completion still works on the cold tier.

ValidArgsFunction wiring on subcommands comes in the next commit.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Wire `ValidArgsFunction` and flag completion

**Files:**
- Modify: `cli/info.go`, `cli/read.go`, `cli/get.go`, `cli/set.go`, `cli/reboot.go`, `cli/getip.go` — add `ValidArgsFunction` to each subcommand.
- Modify: `cli/cli.go` — `init()` adds `rootCmd.RegisterFlagCompletionFunc("bind-interface", completeInterfaces)`.

- [ ] **Step 1: Update each subcommand file to add `ValidArgsFunction`**

For `cli/info.go`, find:

```go
var infoCmd = &cobra.Command{
	Use:   "info <mac>",
	Short: "Show metadata for one device",
	Args:  cobra.ExactArgs(1),
	RunE:  runInfo,
}
```

Add a `ValidArgsFunction: completeMACs,` line:

```go
var infoCmd = &cobra.Command{
	Use:               "info <mac>",
	Short:             "Show metadata for one device",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeMACs,
	RunE:              runInfo,
}
```

Apply the same change to:
- `cli/read.go` — add `ValidArgsFunction: completeMACs,` to `readCmd`.
- `cli/set.go` — add `ValidArgsFunction: completeMACs,` to `setCmd`.
- `cli/reboot.go` — add `ValidArgsFunction: completeMACs,` to `rebootCmd`.
- `cli/getip.go` — add `ValidArgsFunction: completeMACs,` to `getipCmd`.

For `cli/get.go`, the wired helper is different — use `completeParameterNames`:

```go
var getCmd = &cobra.Command{
	Use:               "get <mac> <param> [<param>...]",
	Short:             "Read specific parameters",
	Args:              cobra.MinimumNArgs(2),
	ValidArgsFunction: completeParameterNames,
	RunE:              runGet,
}
```

(`completeParameterNames` internally delegates to `completeMACs` when `len(args) == 0`.)

`cli/discover.go` takes no positional args (`cobra.NoArgs`), so no `ValidArgsFunction` is needed.
`cli/interfaces.go` takes no positional args either, same.

- [ ] **Step 2: Register the `--bind-interface` flag completer in `cli/cli.go`**

In `cli/cli.go` `init()`, after the `MarkFlagsMutuallyExclusive` call and before `rootCmd.Version = Version`, add:

```go
	if err := rootCmd.RegisterFlagCompletionFunc("bind-interface", completeInterfaces); err != nil {
		// Cobra panics on duplicate registration; this should never
		// happen unless init() runs twice (it doesn't). Log via panic
		// so an init-order bug is loud.
		panic(fmt.Sprintf("register bind-interface completion: %v", err))
	}
```

You may need to add `"fmt"` to the imports if not already there. (It's already imported in cli/cli.go.)

- [ ] **Step 3: Verify build and tests pass**

```bash
go build ./... && task test
```

Expected: pass. The new wiring adds no new tests yet — the next task does that.

- [ ] **Step 4: Manual smoke-test the `__complete` behaviour**

```bash
go build -o go-udap .
./go-udap __complete info ""
./go-udap __complete --bind-interface ""
./go-udap __complete get "" ""
```

Expected:
- First call: stdout has either `:4` (NoFileComp directive on its own line at the end) with no entries if no devices on the network, OR `<MAC>\t<name>\n:4` if devices respond. The first invocation will use 500 ms; immediately re-running it will use `defaultTimeout`.
- Second call: lists local interfaces (e.g., `en0\t192.168.1.241`) followed by `:4`.
- Third call: lists local interfaces (because `get <TAB>` while empty delegates to MACs via completeParameterNames). The next position would complete parameter names.

Note: Cobra outputs the directive as `:N` where N is the bitmask (`:4` = NoFileComp). The exact format is internal to Cobra; we don't depend on it directly.

- [ ] **Step 5: Commit**

```bash
git add cli/info.go cli/read.go cli/get.go cli/set.go cli/reboot.go cli/getip.go cli/cli.go
git commit -m "$(cat <<'EOF'
feat(cli): wire ValidArgsFunction and --bind-interface completion

Each subcommand that takes <mac> as its first positional arg
(info, read, get, set, reboot, getip) gets ValidArgsFunction =
completeMACs. The `get` subcommand uses completeParameterNames
which delegates to completeMACs for the first arg and then
completes from udap.Parameters (minus already-listed) for
subsequent args.

RegisterFlagCompletionFunc("bind-interface", completeInterfaces)
makes --bind-interface tab-complete from local interfaces.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Add `.gitignore` entry for completions/ and `task completions` target

**Files:**
- Modify: `.gitignore`
- Modify: `Taskfile.yml`

- [ ] **Step 1: Append to `.gitignore`**

Add at the end of `.gitignore`:

```
# Generated by `task completions` and goreleaser before-hook.
completions/
```

- [ ] **Step 2: Add a `task completions` target**

Read `Taskfile.yml`. Add a new task in the tasks: section (adjacent to `task build`):

```yaml
  completions:
    desc: Generate shell completion scripts into ./completions
    cmds:
      - mkdir -p ./completions
      - go run . __dump-completions ./completions
      - echo "Wrote ./completions/{go-udap.bash,_go-udap,go-udap.fish}"
```

- [ ] **Step 3: Verify the new task works**

```bash
task completions
ls completions/
```

Expected: `go-udap.bash`, `_go-udap`, `go-udap.fish` in `./completions/`.

```bash
bash -n completions/go-udap.bash && echo bash OK
zsh -n completions/_go-udap && echo zsh OK
fish -c "source completions/go-udap.fish" && echo fish OK
```

Clean up so the directory doesn't end up tracked:

```bash
rm -rf completions/
git status
```

Expected: `.gitignore` and `Taskfile.yml` are the only modified files. The `completions/` dir is now gitignored even if recreated.

- [ ] **Step 4: Commit**

```bash
git add .gitignore Taskfile.yml
git commit -m "$(cat <<'EOF'
build: add task completions target and gitignore completions/

`task completions` writes bash/zsh/fish scripts via the hidden
__dump-completions subcommand. Useful for local smoke-testing.
The completions/ directory is a build artifact (also produced by
the goreleaser before-hook); gitignored so stray runs don't
accidentally pollute git status.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Update `.goreleaser.yaml` for completion scripts

**Files:**
- Modify: `.goreleaser.yaml`

- [ ] **Step 1: Read the current `.goreleaser.yaml`**

Locate three sections that need changes:
- `before.hooks:` — currently has `go mod tidy`. Add a second hook.
- `archives:` — currently has `files: [LICENSE, README.md]`. Add a `completions/*` entry.
- `homebrew_casks:` — currently has the `binary` line and `hooks.post.install`. Add a `completions:` map.

- [ ] **Step 2: Add the completions before-hook**

Change:

```yaml
before:
  hooks:
    - go mod tidy
```

to:

```yaml
before:
  hooks:
    - go mod tidy
    - mkdir -p ./completions
    - go run . __dump-completions ./completions
```

- [ ] **Step 3: Add completions to the archive `files:`**

Change:

```yaml
archives:
  - id: go-udap
    name_template: >-
      ...
    formats:
      - tar.gz
    files:
      - LICENSE
      - README.md
    format_overrides:
      ...
```

to:

```yaml
archives:
  - id: go-udap
    name_template: >-
      ...
    formats:
      - tar.gz
    files:
      - LICENSE
      - README.md
      - src: completions/go-udap.bash
        dst: completions/go-udap.bash
      - src: completions/_go-udap
        dst: completions/_go-udap
      - src: completions/go-udap.fish
        dst: completions/go-udap.fish
    format_overrides:
      ...
```

(Listing files explicitly rather than using a glob, so the archive layout is deterministic and reviewable in goreleaser's audit output.)

- [ ] **Step 4: Add the cask completions map**

Find the `homebrew_casks:` block. After the `hooks:` block and before any trailing `# ...` comment, add a `completions:` map:

```yaml
homebrew_casks:
  - name: go-udap
    repository:
      ...
    homepage: ...
    description: ...
    license: MIT
    url:
      verified: github.com/yo61/go-udap
    completions:
      bash: completions/go-udap.bash
      zsh: completions/_go-udap
      fish: completions/go-udap.fish
    hooks:
      post:
        install: |
          if OS.mac?
            system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{staged_path}/go-udap"]
          end
```

- [ ] **Step 5: Validate the goreleaser config**

```bash
goreleaser check
```

Expected: `config is valid`. If goreleaser isn't installed locally, skip this step — CI catches it.

- [ ] **Step 6: Smoke-test the release pipeline locally (snapshot mode)**

```bash
goreleaser release --snapshot --clean --skip=publish 2>&1 | tail -40
```

Expected: the snapshot succeeds. In `dist/`, look for:
- `go-udap_<version>_macos_x86_64.tar.gz` (etc.) — extract one and verify `completions/` is inside.
- `dist/homebrew/Casks/go-udap.rb` — verify it contains `bash_completion "completions/go-udap.bash"`, `zsh_completion "completions/_go-udap"`, `fish_completion "completions/go-udap.fish"`.

Quick check:

```bash
grep -E "bash_completion|zsh_completion|fish_completion" dist/homebrew/Casks/go-udap.rb
```

Expected: three matches.

```bash
tar -tzf dist/go-udap_*_macos_arm64.tar.gz | grep completions
```

Expected: three completion files listed inside the tarball.

If goreleaser isn't installed, skip this step.

- [ ] **Step 7: Commit**

```bash
git add .goreleaser.yaml
git commit -m "$(cat <<'EOF'
build: ship bash/zsh/fish completions via release tarball + cask

Three changes to .goreleaser.yaml:

1. before.hooks: invoke `go run . __dump-completions ./completions`
   so the three scripts are generated before archiving.
2. archives.files: bundle completions/go-udap.bash, /_go-udap, and
   /go-udap.fish into each release tarball.
3. homebrew_casks.completions: GoReleaser writes the three
   bash_completion / zsh_completion / fish_completion stanzas into
   the generated cask. brew install/upgrade then symlinks them into
   $HOMEBREW_PREFIX/etc/bash_completion.d, share/zsh/site-functions,
   share/fish/vendor_completions.d.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Add CI smoke job for completion scripts

**Files:**
- Modify: `.github/workflows/ci.yaml` (or whichever workflow runs tests).

- [ ] **Step 1: Locate the CI workflow file**

```bash
ls .github/workflows/
```

Find the workflow that runs `go test` on PR pushes. Likely `ci.yaml` or `test.yaml`.

- [ ] **Step 2: Add a `completion-smoke` job**

Append (alongside existing jobs):

```yaml
  completion-smoke:
    name: completion syntax smoke
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - name: Install zsh and fish
        run: |
          sudo apt-get update
          sudo apt-get install -y --no-install-recommends zsh fish
      - name: Generate completion scripts
        run: |
          mkdir -p ./completions
          go run . __dump-completions ./completions
          ls -la ./completions
      - name: Bash syntax check
        run: bash -n ./completions/go-udap.bash
      - name: Zsh syntax check
        run: zsh -n ./completions/_go-udap
      - name: Fish syntax check
        run: fish -c "source ./completions/go-udap.fish"
```

Adjust the existing action SHAs to match the project's existing pins — find any other workflow that uses `actions/checkout` or `actions/setup-go` and copy the SHA + version comment. The project pins actions to SHA hashes (see `CLAUDE.md`).

- [ ] **Step 3: Confirm the workflow is syntactically valid**

```bash
which actionlint && actionlint .github/workflows/ || echo "actionlint not installed locally; CI will catch it"
```

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/
git commit -m "$(cat <<'EOF'
ci: add completion-smoke job (bash/zsh/fish syntax check)

Generates the three completion scripts via __dump-completions, then
runs `bash -n`, `zsh -n`, and `fish -c source` to syntax-check each.
This catches malformed scripts before they ride out in a release.
Syntax-only — does not simulate tab presses.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Update README with completion section

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Read the current README structure**

```bash
cat README.md
```

Find the "Installation" or "Usage" section. The new completion section goes there.

- [ ] **Step 2: Add a "Shell completions" section**

Insert after the install section:

```markdown
## Shell completions

`brew install yo61/tap/go-udap` installs bash, zsh, and fish completions automatically.

For installations outside Homebrew, generate the script for your shell and source it:

```bash
# Bash (Linux: ~/.local/share/bash-completion/completions/go-udap)
go-udap completion bash > ~/.local/share/bash-completion/completions/go-udap

# Zsh (anywhere on your $fpath, e.g. ~/.zsh/completions/_go-udap)
go-udap completion zsh > ~/.zsh/completions/_go-udap

# Fish
go-udap completion fish > ~/.config/fish/completions/go-udap.fish
```

Completions cover subcommand names, flag names, the 26 NVRAM parameter
names on `get` / `set`, MAC addresses (via short-timeout UDAP discovery),
and `--bind-interface` values.
```

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs(readme): add shell completions section

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 10: Final verification

**Files (read-only):**

- [ ] **Step 1: Run all the things**

```bash
task build && echo "build OK"
task test && echo "test OK"
task lint && echo "lint OK"
task completions && ls completions/ && bash -n completions/go-udap.bash && zsh -n completions/_go-udap && fish -c "source completions/go-udap.fish" && echo "completions OK"
rm -rf completions/
```

Expected: every line ends with `... OK`.

- [ ] **Step 2: Manual UI smoke test**

In a fresh terminal, run:

```bash
./go-udap completion zsh > /tmp/_go-udap
echo "fpath=(/tmp \$fpath); autoload -U compinit; compinit; ./go-udap <TAB>" | zsh -i 2>&1 | head -20
```

This exercises Cobra's own user-facing `completion` subcommand and tests that the generated script is loadable. The exact output is shell-dependent; just confirm no syntax errors.

- [ ] **Step 3: Verify default timeout actually changed**

```bash
./go-udap --help 2>&1 | grep timeout
```

Expected: `--timeout DURATION        Operation timeout, e.g. 2s, 30s, 2m (default 2s)`.

The `(default 2s)` is the key signal.

- [ ] **Step 4: Verify __dump-completions is hidden**

```bash
./go-udap --help 2>&1 | grep -i dump
```

Expected: no match. `__dump-completions` is `Hidden: true` and should not appear in help.

It must still be runnable explicitly:

```bash
mkdir -p /tmp/dump-test && ./go-udap __dump-completions /tmp/dump-test && ls /tmp/dump-test && rm -rf /tmp/dump-test
```

Expected: three files produced.

---

## Task 11: Push branch and open PR

- [ ] **Step 1: Push**

```bash
git push -u origin feat/shell-completions
```

- [ ] **Step 2: Open PR**

```bash
gh pr create --title "feat: shell completions (bash/zsh/fish) via Homebrew Cask" --body "$(cat <<'EOF'
## Summary
- Add tab-completion for subcommands, flags, the 26 NVRAM parameter names, `<mac>` arguments (live UDAP discovery, progressive 500ms→default timeout), and `--bind-interface` values.
- Ship completion scripts via the release tarball; GoReleaser writes `bash_completion`/`zsh_completion`/`fish_completion` stanzas into the cask; `brew install`/`brew upgrade` wires them up.
- Drop the default `--timeout` from 5s to 2s. UDAP devices typically respond in ~100ms; the completion retry tier uses `defaultTimeout` so dropping it also bounds the worst-case tab delay.

This is PR 2 of the shell-completions feature; PR 1 (Cobra refactor, #83) is already merged. See [`docs/superpowers/specs/2026-05-26-shell-completions-design.md`](docs/superpowers/specs/2026-05-26-shell-completions-design.md) and [`docs/superpowers/plans/2026-05-26-shell-completions.md`](docs/superpowers/plans/2026-05-26-shell-completions.md).

## Test plan
- [x] `task build` succeeds.
- [x] `task test` (race detector) passes.
- [x] `task lint` is clean.
- [x] `task completions` produces three syntactically valid scripts.
- [x] `./go-udap --help` shows `--timeout DURATION ... (default 2s)`.
- [x] `__dump-completions` is hidden from `--help` but runs explicitly.
- [x] CI `completion-smoke` job passes (bash/zsh/fish syntax check).
- [ ] After merge: `brew upgrade yo61/tap/go-udap` and confirm tab-completion works in a fresh shell.

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

- [ ] **Step 3: Watch CI**

```bash
gh pr checks --watch
```

Expected: all checks pass including the new `completion-smoke` job.

---

## Verification checklist (after all tasks)

- [ ] Branch `feat/shell-completions` is on the remote.
- [ ] PR is open with all checks passing (incl. `completion-smoke`).
- [ ] `cli/completion.go` exists; `cli/completion_test.go` exists; all 10 unit tests pass.
- [ ] Each subcommand that takes `<mac>` has `ValidArgsFunction` wired.
- [ ] `rootCmd.RegisterFlagCompletionFunc("bind-interface", completeInterfaces)` lives in `cli/cli.go init()`.
- [ ] `defaultTimeout` is `2 * time.Second`.
- [ ] `.goreleaser.yaml` has the before-hook, archive `files` entries, and `homebrew_casks.completions:` block.
- [ ] `Taskfile.yml` has `task completions`.
- [ ] `.gitignore` excludes `completions/`.
- [ ] `CLAUDE.md` updated to say "Network timeouts default to 2 seconds".
- [ ] `README.md` has a shell-completions section.

## Post-merge action items (out of scope of this PR)

These verify the feature reaches users after release:

1. Tag a new release once PR 2 is on `main`. release-please will produce the version-bump PR; merge that, then verify the tag pushes the bump to the tap.
2. On macOS: `brew upgrade yo61/tap/go-udap`; confirm `go-udap <TAB>` completes subcommands in zsh.
3. Verify the tap audit workflow (`yo61/homebrew-tap` → `audit.yaml`) passes on the new cask.
