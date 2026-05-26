# Shell completions for go-udap

**Date:** 2026-05-26
**Status:** Draft — awaiting implementation plan

## Summary

Add bash, zsh, and fish tab-completion for `go-udap`. Completion covers:

- Subcommand names (`discover`, `info`, `read`, `get`, `set`, `reboot`, `getip`, `interfaces`)
- Global flag names (`--timeout`, `--retries`, `--verbose`, `--bind-interface`, `--all-interfaces`, `--version`, `--help`)
- Per-param flag names on `set` (the 26 entries in `udap.Parameters`)
- Dynamic `<mac>` arguments via live UDAP discovery (500 ms first tab, `defaultTimeout` retry tier)
- Dynamic `--bind-interface NAME` from `udap.EnumerateInterfaces()`
- Parameter names on `get <mac> <param> [<param>...]`

Distribution: completion scripts ship inside the release tarball and are
symlinked into the standard shell-completion paths by the Homebrew Cask's
`bash_completion`, `zsh_completion`, and `fish_completion` stanzas. `brew
install` and `brew upgrade` wire them up; no manual steps required.

The work lands in two PRs:

1. **Cobra refactor** — replace the hand-rolled `cli/cli.go` switch dispatch
   with `github.com/spf13/cobra`. No end-user behaviour change on the happy
   path. Help and error text format shift to Cobra style.
2. **Completion** — add `ValidArgsFunction` / `RegisterFlagCompletionFunc`
   hooks, a hidden `__dump-completions` build-time generator subcommand,
   GoReleaser `before` hook and archive `files` entry, Cask completion
   stanzas. Drop the default `--timeout` from 5 s to 2 s so completion's
   retry tier matches the global default with no magic numbers.

## Motivation

- `go-udap` is installed via `brew install yo61/tap/go-udap` but ships no
  completions, so tab does nothing useful. Users have to memorise 8
  subcommands, ~30 flags, and 26 NVRAM parameter names.
- The MAC argument is the single hardest token to type — six hex bytes
  with colons. Even partial dynamic completion (matching what's on the
  network right now) is a large UX win.
- The interface name passed to `--bind-interface` is locally enumerable
  and cheap to complete — there's no reason to make the user run
  `go-udap interfaces` and copy-paste.

## References

- **Homebrew Cask Cookbook** (`docs.brew.sh/Cask-Cookbook`):
  - `bash_completion "..."` — symlinks into `$(brew --prefix)/etc/bash_completion.d`
  - `zsh_completion "..."` — symlinks into `$(brew --prefix)/share/zsh/site-functions`
  - `fish_completion "..."` — symlinks into `$(brew --prefix)/share/fish/vendor_completions.d`
- **Cobra completion docs** (`github.com/spf13/cobra/blob/main/site/content/completions/_index.md`):
  - `ValidArgsFunction(cmd, args, toComplete) ([]string, ShellCompDirective)` — positional-arg completion.
  - `RegisterFlagCompletionFunc(name, fn)` — flag-value completion.
  - `GenBashCompletionFileV2`, `GenZshCompletionFile`, `GenFishCompletionFile` — script generators.
  - `ShellCompDirective` bitmask: `NoFileComp` (no file-name fallback), `NoSpace` (suppress trailing space; we intentionally omit this so completions add a space).
- **Existing spec/plan:** `docs/superpowers/specs/2026-05-09-homebrew-tap-design.md` (Cask distribution rationale).
- **GoReleaser `homebrew_casks`:** generated cask uses the stanzas above when the relevant cask config keys are set.

## Two-PR split

### PR 1 — Cobra refactor

**Goal:** Replace the hand-rolled `cli/cli.go` dispatch with a Cobra command
tree. End-user behaviour on the happy path is identical: same subcommands,
same flag names, same stdout content, same exit codes, same global-flag
ordering rules. Help and error message format shift to Cobra's style — this
is the only visible change.

**Behaviour preserved (contract):**

| Aspect | After PR 1 |
|---|---|
| Subcommand names | unchanged |
| Global flag names | unchanged; registered as `PersistentFlags()` on root |
| Per-param flags on `set` | unchanged; loop calls `setCmd.Flags()` |
| Global flags before or after subcommand | works natively via `PersistentFlags()` |
| Exit codes | 0 success, 1 usage error, 2 operation failure (via `cli.ExitCode` on the error returned by `Execute`) |
| `--bind-interface`/`--all-interfaces` mutual exclusion | `rootCmd.MarkFlagsMutuallyExclusive(...)` |
| stdout content (data payloads) | unchanged |
| stderr content (logs, progress, warnings) | unchanged |
| Progress bar TTY detection | unchanged (lives in `cli/progress.go`) |

**Behaviour that changes (visible):**

| Aspect | Before | After PR 1 |
|---|---|---|
| `--help` format | hand-rolled in `cli.go` | Cobra-generated usage block per subcommand |
| Unknown-subcommand error | `error: unknown command "xyz"` | `Error: unknown command "xyz" for "go-udap"\nRun 'go-udap --help'...` |
| Unknown-flag error | `error: unknown flag --foo` | `Error: unknown flag: --foo\nUsage:\n  ...` |
| Missing-arg error | hand-rolled | `Error: accepts 1 arg(s), received 0` |
| `--version` | hand-handled in `cli.go` | `rootCmd.Version = cli.Version`; Cobra prints `go-udap version X.Y.Z` |
| Top-level subcommand list | 8 commands | adds Cobra's `help` and `completion` (the latter is a no-op stub in PR 1, wired up in PR 2) |

**Files touched in PR 1:**

```
go.mod, go.sum                  add github.com/spf13/cobra
cli/cli.go                      rewrite: rootCmd + Execute(); delete
                                moveGlobalFlagsAfterSubcommand,
                                parseSubcommandFlags, errHelpRequested
cli/{discover,info,read,get,    each: top-level var <name>Cmd =
     set,reboot,getip,           &cobra.Command{Use, Short, Args, RunE};
     interfaces}.go              delete inlined flag parsing
cli/params.go                   loop registers flags on setCmd.Flags()
cli/source.go                   adapt signature to take *cobra.Command
                                (or its FlagSet)
main.go                         os.Exit(cli.ExitCode(cli.Execute(
                                  ctx, os.Args[1:], stdout, stderr)))
```

**`defaultTimeout` constant:** Consolidate the 8 inlined `5*time.Second`
defaults (one per subcommand file) into a single
`const defaultTimeout = 5*time.Second` on the root command. Value stays at
5 s in PR 1 — single source of truth, behaviour unchanged. PR 2 drops it
to 2 s.

**E2E test impact (`cli/e2e_*_test.go`, 19 files):**

Categorise every assertion. Update only the ones that depend on the
shifted text formats:

| Assertion category | Impact | Action |
|---|---|---|
| Stdout substrings (MACs, IPs, param values) | None | Verify unchanged |
| Exit codes (0/1/2) | None | Verify unchanged |
| `--help` substrings | High | Update to match Cobra format, or assert on stable fragments only |
| Unknown-subcommand / unknown-flag / missing-arg errors | High | Update to Cobra strings |
| Timing tests on default timeout (`e2e_timeout_test.go`) | Stable in PR 1 (5 s default unchanged) | Audit but expect no change |

Audit happens before refactor lands so the test surface is known up front.

**Risk register:**

| Risk | Likelihood | Mitigation |
|---|---|---|
| Hidden behaviour in `moveGlobalFlagsAfterSubcommand` not replicated by `PersistentFlags()` | medium | Keep flag-ordering e2e tests; add one if missing |
| Cobra's auto-added `completion` and `help` subcommands change `--help` UX | low | Acceptable; document |
| pflag flag types that don't round-trip through Cobra | low | All current usage is `StringVar`/`BoolVar`/`DurationVar`/`IntVar`, fully supported |
| New `cobra` dependency adds attack surface / maintenance burden | low | MIT-licensed, widely used, dependency surface is small (pflag, which we already use) |

### PR 2 — Completion implementation

**New file: `cli/completion.go`** — completion helpers and the hidden
build-time dumper. (The `defaultTimeout` constant referenced below lives
in `cli/cli.go` on the root command; PR 1 introduces it at 5 s, PR 2
drops it to 2 s as part of this change. See "Default-timeout drop" below.)

```go
// completionStateDir is a seam for tests.
var completionStateDir = os.TempDir

func completionStatePath() string {
    return filepath.Join(completionStateDir(),
        fmt.Sprintf("go-udap-complete-%d", os.Getppid()))
}

// nextMACTimeout returns 500ms on a cold start, defaultTimeout when the
// previous tab within ~10s returned zero devices.
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

func recordMACAttempt(count int) {
    _ = os.WriteFile(completionStatePath(),
        []byte(strconv.Itoa(count)), 0o600) // best-effort
}

// completeMACs is the ValidArgsFunction for info/read/get/set/reboot/getip.
func completeMACs(cmd *cobra.Command, args []string, toComplete string) (
    []string, cobra.ShellCompDirective,
) {
    if len(args) >= 1 {
        return nil, cobra.ShellCompDirectiveNoFileComp
    }
    ctx, cancel := context.WithTimeout(cmd.Context(), nextMACTimeout())
    defer cancel()
    client, err := newClientForCompletion(cmd) // logger -> io.Discard
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

// completeInterfaces is RegisterFlagCompletionFunc target for --bind-interface.
func completeInterfaces(cmd *cobra.Command, args []string, toComplete string) (
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
func completeParameterNames(cmd *cobra.Command, args []string, toComplete string) (
    []string, cobra.ShellCompDirective,
) {
    if len(args) == 0 { // still waiting on MAC
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

// __dump-completions writes the three scripts to <out-dir>. Hidden; used
// by GoReleaser's before-hook to bundle scripts into the release archive.
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
```

**Wiring** (per subcommand file):

```go
var infoCmd = &cobra.Command{
    Use:               "info <mac>",
    Args:              cobra.ExactArgs(1),
    ValidArgsFunction: completeMACs,
    RunE:              runInfo,
}
// ... read, get (uses completeParameterNames), set, reboot, getip likewise
```

```go
// in cli/cli.go init()
rootCmd.RegisterFlagCompletionFunc("bind-interface", completeInterfaces)
```

**Cobra `ShellCompDirective` convention:** All completers return
`cobra.ShellCompDirectiveNoFileComp` alone — never combined with
`ShellCompDirectiveNoSpace`. This yields "don't fall back to file-name
completion if our list is empty" and (by virtue of `NoSpace` absent) "add a
trailing space after the completed token". The cursor lands ready for the
next argument.

**Tab-separated descriptions:** Completions returned as `"<value>\t<hint>"`
have Cobra insert only `<value>` into the command line; `<hint>` is shown
to the user as a description. The trailing-space rule still applies to
the inserted value. Example: `00:04:20:16:05:8f\tbasement-radio` →
inserts `00:04:20:16:05:8f ` (with trailing space), displays
`basement-radio` as the hint.

**Default-timeout drop (5 s → 2 s):**

This PR drops `defaultTimeout` from 5 s to 2 s. Motivation: the
completion retry tier uses `defaultTimeout` so there's a single source of
truth; 5 s on a tab press feels too long. UDAP devices typically respond
within ~100 ms when present.

Knock-on changes in PR 2:

- `CLAUDE.md` — update the "Network timeouts default to 5 seconds" line to 2 seconds, and the `--timeout DURATION` doc.
- README — search for any 5-second references in user-facing docs.
- Flag help text — currently `"Discovery timeout, e.g. 5s, 30s, 2m"`. Change to `"e.g. 2s, 30s, 2m"`.
- `cli/e2e_timeout_test.go` — pin tests to `--timeout 5s` explicitly where the original intent was "5 s default", or adjust expected elapsed to 2 s if the test validates "default" behaviour.

**Distribution (`.goreleaser.yaml` changes):**

```yaml
before:
  hooks:
    - go mod tidy
    - go run . __dump-completions ./completions   # NEW

archives:
  - id: go-udap
    files:
      - LICENSE
      - README.md
      - src: completions/*                          # NEW
        dst: completions/
        info:
          mode: 0644

homebrew_casks:
  - name: go-udap
    completions:                                    # NEW
      bash: "completions/go-udap.bash"
      zsh:  "completions/_go-udap"
      fish: "completions/go-udap.fish"
```

The exact GoReleaser key for cask completions (`completions:` map vs
`extra_install:` raw Ruby vs another) will be confirmed in the
implementation plan against the current GoReleaser docs. Whichever key
is correct, the generated cask must include the three stanzas
`bash_completion "completions/go-udap.bash"`,
`zsh_completion "completions/_go-udap"`,
`fish_completion "completions/go-udap.fish"`.

**`.gitignore`:**

```
completions/      # build artifact; generated by goreleaser before-hook
```

**Local dev convenience (`Taskfile.yml`):**

```yaml
tasks:
  completions:
    desc: Generate shell completion scripts into ./completions
    cmds:
      - go run . __dump-completions ./completions
```

**Files touched in PR 2:**

```
cli/completion.go              new file: helpers + __dump-completions
cli/completion_test.go         new file: unit tests
cli/completion_mac_test.go     new file: mocksbr-backed MAC test
cli/e2e_completion_test.go     new file: __complete wire-format e2e
cli/{info,read,get,set,        add ValidArgsFunction wiring (no logic change)
     reboot,getip}.go
cli/cli.go                     drop defaultTimeout 5s -> 2s; register
                               completeInterfaces for --bind-interface
cli/params.go                  no change (flag completion is automatic)
.goreleaser.yaml               before-hook + archive files + cask completions
.gitignore                     add completions/
Taskfile.yml                   add `task completions`
CLAUDE.md                      update timeout default doc; mention completions
README.md                      mention completions if 5s default is mentioned
.github/workflows/ci.yaml      add completion-smoke job (bash -n / zsh -n / fish -c)
```

## Error handling and UX edges

| Scenario | Behaviour |
|---|---|
| Tab on `<mac>`, no devices respond in cold tier (500 ms) | Empty result; state file records `0`. Within 10 s, next tab uses `defaultTimeout` (2 s). |
| Tab on `<mac>`, retry tier also returns empty | Empty result; state file again `0`. No further escalation — bounded. After 10 s idle, the next attempt is "cold" again. |
| `newClient` fails (port 17784 in use, etc.) | `ShellCompDirectiveError`; shell shows no completion. Stderr suppressed (Cobra routes `__complete` stderr to `/dev/null` by default; helper uses `io.Discard` logger as belt-and-braces). |
| State file write fails (read-only `/tmp`, full disk) | `recordMACAttempt` ignores error. Completion still works on the cold tier; the retry tier is degraded silently. |
| User has `--bind-interface garbage` mid-line | `newClient` errors; completer returns `ShellCompDirectiveError`. No completion, no visible error. |
| User runs `go-udap completion zsh` (no brew) | Cobra prints the script to stdout; user redirects to their `fpath`. Documented in README. |
| `brew upgrade go-udap` | Cask uninstall removes old symlinks; install creates new ones with updated content. No user action needed. |
| User on Linux without `bash-completion` package | bash completion file is symlinked but not auto-loaded. Documented in README; user must `source $(brew --prefix)/etc/bash_completion.d/go-udap.bash` or install `bash-completion`. |

**Stderr suppression in completion subprocess:**

`completeMACs` constructs a client with `io.Discard` as the logger
sink (`newClientForCompletion(cmd)` wraps `newClient` and swaps the
logger). The progress bar in `cli/progress.go` already checks
`isatty(stderr)` before drawing; when invoked under `__complete`,
stderr is `/dev/null` (Cobra's generated shell scripts redirect it),
so the bar is suppressed for free.

**Mutually exclusive flag handling during completion:**

`rootCmd.MarkFlagsMutuallyExclusive("bind-interface", "all-interfaces")`
errors during normal `Execute`. During `__complete`, Cobra does not
treat the conflict as fatal — completion proceeds. This is the right
behaviour: the user is mid-typing, the conflict is informational.

**Parameter value completion (out of scope):**

`go-udap set --interface <TAB>` would benefit from completing `0` and
`1` (wireless / wired). 26 parameters with hand-crafted value
completers is a lot of work for marginal gain. Skip in this design.
If users specifically request value completion for, e.g.,
`--interface` or `--lan_ip_mode`, add case-by-case later.

## Testing strategy

### Unit tests (new `cli/completion_test.go`)

| Test | Coverage |
|---|---|
| `TestCompleteInterfaces_ReturnsLocalInterfaces` | `completeInterfaces` returns at least one entry in `Name\tAddr` form |
| `TestCompleteParameterNames_FiltersAlreadyListed` | `args=["mac","server_address"]`: `server_address` excluded, other 25 included |
| `TestCompleteParameterNames_EmptyArgsDelegatesToMACs` | `args=[]` triggers MAC path |
| `TestNextMACTimeout_ColdWhenNoState` | No state file → 500 ms |
| `TestNextMACTimeout_ColdWhenStateStale` | mtime >10 s → 500 ms |
| `TestNextMACTimeout_RetryWhenRecentEmpty` | Recent state file with `0` → `defaultTimeout` (2 s) |
| `TestNextMACTimeout_ColdWhenRecentNonempty` | Recent state file with `3` → 500 ms |
| `TestRecordMACAttempt_WritesCount` | After `recordMACAttempt(5)`, file contains `5` |
| `TestRecordMACAttempt_TolerantOfReadOnlyTmp` | Write failure → no panic, no error returned |
| `TestCompletionStatePath_ScopedByPPID` | Path contains `os.Getppid()` |

Tests swap the `completionStateDir` package var to `t.TempDir`.

### Mocksbr-backed MAC test (new `cli/completion_mac_test.go`)

Uses the existing `mocksbr.MockTransport` to inject fake devices into the
`udap.Transport` interface, exercising `completeMACs` end-to-end without
UDP sockets:

```go
func TestCompleteMACs_ReturnsDiscoveredDevices(t *testing.T) {
    devices := []mocksbr.FakeDevice{{
        MAC: "00:04:20:16:05:8f", Name: "basement-radio",
    }}
    // ...inject via transport seam, invoke completeMACs, assert result
}
```

### `__complete` wire-format e2e (new `cli/e2e_completion_test.go`)

Subprocess-invokes `go-udap __complete info ""` (Cobra's hidden entry
point that shell scripts call) and asserts:

- Exit code 0
- Stdout contains expected MAC lines (mocked via existing e2e harness)
- Stderr empty (no log leakage)
- Output ends with Cobra's `:NoFileComp` directive line

This pins the contract the shell scripts depend on; if Cobra ever
changes its `__complete` output format we catch it here.

### E2E audit pass (PR 1)

Inventory every assertion in the 19 existing `cli/e2e_*_test.go` files:

| Category | Impact | Action |
|---|---|---|
| Stdout data substrings | None | Verify |
| Exit codes | None | Verify |
| `--help` substrings | High | Update to Cobra format, prefer stable fragments |
| Unknown-subcommand / unknown-flag / missing-arg errors | High | Update strings |
| Timing assertions on default timeout | PR 2 only | In PR 2: pin to `--timeout 5s` or adjust to 2 s based on test intent |

### Completion script smoke (new CI job)

```yaml
completion-smoke:
  runs-on: ubuntu-latest
  steps:
    - run: go run . __dump-completions ./completions
    - run: bash -n completions/go-udap.bash
    - run: zsh -n completions/_go-udap
    - run: fish -c "source completions/go-udap.fish"
```

Catches malformed generated scripts before release. Syntax-only — does
not simulate tab presses.

### Explicitly not tested

- Interactive tab behaviour in real shells (brittle; mostly tests Cobra).
- `brew install` end-to-end (covered by `yo61/homebrew-tap` audit workflow).
- Cask `bash_completion`/`zsh_completion`/`fish_completion` symlink wiring
  (Homebrew's responsibility; smoke-test manually after release).

## Non-goals

- PowerShell completion. Add later if Windows users request it.
- Persistent MAC cache. Live discovery with progressive timeout is the
  chosen UX.
- Per-parameter value completion (e.g., `--interface 0|1`). Add
  case-by-case if requested.
- Cobra dynamic completion for `--config FILE`. Cobra's default
  file-name fallback already handles this.
- Switching back from Cask to Formula. Cask remains correct per the
  rationale in `b0d48d5d feat(release): switch Homebrew distribution
  from formula to cask`.

## Open questions

- GoReleaser cask-completions key: confirm during plan whether it's
  `completions:`, `extra_install:` raw Ruby, or another key. Either
  way, the generated cask must contain the three completion stanzas.
- E2E test failure modes around Cobra error text — likely tractable;
  audit happens before refactor lands.
