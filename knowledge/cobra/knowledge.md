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
