# cobra: verified behaviour

Checked against spf13/cobra v1.10.2 on 2026-10-09 with a throwaway probe
(see #244). Re-check after a cobra upgrade.

- `--help` returns before `PersistentPreRunE` runs, so hooks never see a
  help invocation.
- The built-in `--version` (`cmd.Version` set) prints its template before
  any hook or `RunE`, so it can't react to other flags.
- The hidden `__complete` command does run root's `PersistentPreRunE`.
  Hooks that reject flags must exempt it, or shell completion breaks.
- `MarkFlagsMutuallyExclusive` works across persistent flags, including
  ones given by shorthand. Its check runs after `PersistentPreRunE`, and
  its error message is cobra's generic wording.
- A root command without `RunE` prints help for a bare invocation
  without running any hook. Adding a root `RunE` doesn't change the
  "unknown command" error for stray arguments.

## Usage-error hooks (checked for #252)

`Command.execute` runs in this order, and the first error returns:
`Find` (unknown subcommand) → `ParseFlags` (via `FlagErrorFunc`) →
`ValidateArgs` → `PersistentPreRunE` → `PreRunE` →
`ValidateRequiredFlags` → `ValidateFlagGroups` → `RunE`.

- Every error cobra raises itself is an untyped `fmt.Errorf`. Telling
  them apart means matching on message text.
- Only flag parse errors have a hook (`SetFlagErrorFunc`, inherited by
  subcommands). Argument-count errors could be wrapped per validator.
- "Unknown command" comes from `legacyArgs` inside `Find`, which only
  runs when the root command has no `Args` set. Setting root `Args`
  removes the check, and a non-runnable root then prints help and
  returns nil, so the exit is 0. There is no hook for it.
- Flag-group errors (`MarkFlagsMutuallyExclusive`) are raised *after*
  `PersistentPreRunE`, so a flag set in that hook can't separate parse
  errors from command errors.
- What does work: wrap every `RunE` to record that it started. Any
  untyped error returned before that point came from cobra's parsing
  and validation (`cli.Execute`). cobra's own `help` and `__complete`
  commands use `Run`, which can't return an error, and they're added
  during `ExecuteC`, so a tree walk done beforehand never sees them.
