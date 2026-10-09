## Decision: Exit 1 means operation failure and exit 2 means usage error; cobra's errors are recognised by when they happen, not by their text

- `0` success, `1` operation failure, `2` usage error. No other codes.
- Untyped errors default to `1`.
- `cli.Execute` turns an untyped error into a usage error (`2`) when no
  subcommand's `RunE` had started. Every `RunE` is wrapped once to
  record that it started.
- Our own usage errors (bad MAC, unknown parameter, invalid parameter
  value, unreadable or invalid `--config`, unusable `--bind-interface`)
  return `ExitError{Code: exitUsage}`.
- `PersistentPreRunE` checks the `--bind-interface`/`--all-interfaces`
  conflict before validating the interface name, so a conflict is
  reported as such whatever interfaces the host has. cobra's
  `MarkFlagsMutuallyExclusive` stays, for shell completion.

## Context: issue #252

The documented contract was `1` usage and `2` operation failure, the
reverse of the usual CLI convention. Errors from cobra (unknown
subcommand, unknown flag, invalid flag value, wrong number of
arguments, conflicting flags) were untyped and fell through to `2`.

## Alternatives considered:

- **A hook for each kind of error** (`SetFlagErrorFunc`, wrapped `Args`
  validators, a mutual-exclusion check repeated in
  `PersistentPreRunE`): rejected. cobra has no hook for "unknown
  command" (see `knowledge/cobra/knowledge.md`), so this leaves a gap
  and needs one hook per category.
- **Matching on cobra's error text**: rejected. It breaks without
  warning when cobra changes its wording.
- **Treating every untyped error as a usage error**: rejected. Issue
  #252 requires untyped errors to default to operation failure.

## Reasoning:

Every kind of cobra parse or validation error is raised before `RunE`,
and everything our hooks return is already typed. So "untyped, and no
`RunE` has started" picks out exactly cobra's errors. That includes
kinds cobra may add later, such as required-flag errors, and needs no
text matching.

## Trade-offs accepted:

- One more piece of package-level state (`reachedRunE`), reset at the
  start of each `Execute`.
- An untyped error from a hook that runs before `RunE`
  (`PersistentPreRunE`, `PreRunE`) would be classified as a usage
  error. Those hooks must keep returning `*ExitError`.

## Supersedes: the exit-code meanings in CLAUDE.md and `quality/criteria.md` before #252
