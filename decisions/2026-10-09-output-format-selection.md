## Decision: Typed per-command Results rendered by a format registry; stdout only; ignored format flags are errors

Every Data command builds a typed Result before writing anything. A new
package `go-udap/cli/output` owns `Format` (`text`, `json`, `csv`), the
`Result` interface and `Render(w, Format, Result) error`. `Result`
requires all three renderings, so the compiler, not a runtime check,
guarantees every command supports every format.

- Flags: global `--format`/`-o`, with `--json` as shorthand; the two
  are mutually exclusive (exit 2, usage error). No environment variable.
- Only stdout changes. Errors, warnings, progress and logs stay as plain
  text on stderr; exit codes stay the failure signal.
- `text` output stays byte-identical to today.
- JSON is compact, with a bare top level (array or object), every key
  always present and `null` for absent values. Parameter values are
  strings.
- CSV is RFC 4180 with a header row always present. Nested objects are
  flattened with a prefix, and parameter maps use long form
  (`name,value`).
- A format flag on a command that writes no Result (`reboot`, bare
  root, `completion`) exits 2. The exceptions are `--help`, which wins,
  and `__complete`.
- `--version` takes the format flags. `--build-info` adds VCS and
  runtime metadata from `debug.ReadBuildInfo`, with no new ldflags.
  `--verbose` combined with either exits 2.
- JSON and CSV shapes are a public contract: removing or renaming a
  key is a breaking (`feat!:`) change, and adding a key is not.

## Context: issue #244 (selectable output formats for every textual command), settled in a grilling session on 2026-10-09

Before this, `cli/output.go` held five hand-written text formatters
called directly by each command.

## Alternatives considered:

- **Hold data as JSON or `map[string]any` and format from that:**
  rejected. It loses types and field order, and text rendering (such as
  `-` for an absent IP) needs the types.
- **Generic document model (`Table`/`Record`) rendered generically:**
  rejected. Today's hand-shaped text output (a bare value for a
  single-param `get`, the aligned `info` block) would either be lost or
  bend the model.
- **Optional per-format interfaces with a runtime "unsupported format"
  error:** deferred until a format exists that genuinely suits only
  some commands.
- **Register format flags only on the commands that support them:**
  rejected. It breaks the rule that global flags work before or after
  the subcommand.
- **Warn on an ignored format flag instead of failing:** rejected. A
  script that asked for JSON would silently receive nothing.
- **Stream `discover --info` per device:** rejected in favour of one
  render path. NDJSON can come later if it's needed.
- **Pretty-printed JSON:** rejected. `jq` pretty-prints on demand.
- **`--version --verbose` for build metadata:** rejected. It would give
  `--verbose` a second meaning; `--build-info` is used instead.

## Reasoning:

Typed Results keep the data intact until the last moment and give the
CLI ownership of its JSON names, independent of `udap.Device`'s tags.
One render path means text, JSON and CSV cannot drift apart in content.
Rejecting ignored flags makes misuse visible instead of silent.

## Trade-offs accepted:

- `discover --info` text no longer appears device by device. The bytes
  are unchanged; only when they appear changes.
- `reboot --help` still lists `--format` under "Global Flags", even
  though using it there is an error.
- Pre-existing silently ignored global flags (`--timeout`, `--retries`
  and the interface flags on `interfaces` and `--version`) are left for
  a follow-up issue.

## Supersedes: none
