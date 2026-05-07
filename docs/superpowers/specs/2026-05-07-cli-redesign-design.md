# CLI redesign: replace interactive shell with single-shot subcommands

**Date:** 2026-05-07
**Status:** Draft for review

## Goal

Replace the readline-based interactive shell in `main.go` with a single-shot
CLI where every operation can be performed from one command line, suitable for
shell scripting and automation. The shell goes away entirely.

## Motivation

The interactive shell forces users to type `discover`, then `set`, then `commit`
inside a REPL. This is slow for one-off operations and awkward for automation
beyond piped here-docs. A CLI-first tool composes naturally with shell pipelines,
config management, and provisioning scripts.

## Non-goals

- No factory-reset command. Factory reset on a Squeezebox Receiver is a physical
  button operation (hold the front button for 6 seconds; see
  https://wiki.lyrion.org/index.php/SBRFrontButtonAndLED). The protocol does not
  expose it.
- No backwards compatibility with the old shell. The shell is removed in the
  same change, no migration shim, no `--shell` flag.
- No protocol changes. The `udap/` package's behavior and public API are
  unchanged in this redesign.

## Command surface

```
go-udap [global flags] <command> [args] [flags]
```

### Global flags

| Flag | Default | Purpose |
|---|---|---|
| `--timeout DURATION` | `5s` | Operation timeout (parsed by `time.ParseDuration`) |
| `--verbose, -v` | off | Debug-level structured logging to stderr |
| `--version` | — | Print version and exit |
| `--help, -h` | — | Print help (subcommand-aware) |

### Subcommands

| Command | Synopsis | Behavior |
|---|---|---|
| `discover` | `go-udap discover [--info]` | Broadcast a discovery packet, collect responses for `--timeout`, print one MAC address per line on stdout. With `--info`, also print metadata (Name, Model, Firmware, IP, UUID) per device in a multi-line block. |
| `info` | `go-udap info <mac>` | Targeted discovery; print metadata for the matching device. Exit code 2 if not found within `--timeout`. |
| `read` | `go-udap read <mac>` | Discover device, fetch all known parameters via GetData, print as `param=value` lines on stdout. |
| `get` | `go-udap get <mac> <param> [<param>...]` | Discover device, fetch the named parameters. Single param → bare value on stdout. Multiple params → `param=value` lines. |
| `set` | `go-udap set <mac> [--config FILE] [--<param> VALUE ...]` | Discover device, then send the merged parameter set. Sources are layered (see below). Errors if no parameters resolved. |
| `save` | `go-udap save <mac>` | Discover device, send SaveData to write current config to NVRAM. |
| `reset` | `go-udap reset <mac>` | Discover device, send Reset (reboots the device). |
| `commit` | `go-udap commit <mac>` | Save then reset (one operation, current shell semantics preserved). |

### Targeting

Every targeted command begins with a discovery preamble:

1. Broadcast a discovery packet on UDP/17784.
2. Collect responses up to `--timeout`.
3. If the requested MAC is among the responders, capture its IP and proceed.
4. If not, exit with code 2 and a clear error.

This validates the device exists before sending writes (the previous shell would
silently broadcast writes to non-existent MACs) and lets the udap layer unicast
subsequent traffic instead of broadcasting again.

### Exit codes

- `0` — success
- `1` — usage / argument error (missing MAC, unknown flag, malformed config file)
- `2` — operation failure (device not found, network error, device returned UDAP error)

### I/O streams

- All command output is on **stdout**.
- All log messages (info, warn, error, debug) are on **stderr** via slog.
- This guarantees stdout is always machine-parseable.

## `set` source layering

`set` accepts parameters from up to three sources, layered in this order:

1. **`--config FILE`** (or `--config -` for explicit stdin)
2. **Piped stdin** (auto-detected when stdin is not a tty)
3. **CLI per-param flags** (`--lan-ip-mode 1`)

Rules:

- Sources are merged; later sources override earlier ones (CLI flags always win).
- If `--config FILE` is given AND stdin is piped, the file wins and stdin is
  ignored with a warning on stderr. Use `--config -` to be explicit if stdin is
  the intended source.
- If no source supplies any parameters, exit with usage error.
- Unknown parameters in any source are an error. (No `--force` flag in the
  initial implementation; it can be added later if needed.)
- Each value is validated by `udap.validateParameter` before any packet is sent.

### Config file / stdin format

INI-style, `key=value` per line:

```
# comment
; also comment
lan_ip_mode=1
wireless_SSID=MyNet
wireless_wpa_psk=secret_value
```

Rules:

- One `key=value` per line.
- Lines beginning with `#` or `;` are comments.
- Blank lines ignored.
- Whitespace around key and value is trimmed.
- Values are not quoted; no escape sequences.
- Keys must match a known UDAP parameter name (canonical form, e.g.
  `wireless_SSID` not `wireless-ssid`).

The file format is identical to the output of `read`, so round-tripping works:

```bash
go-udap read <mac> > backup.conf
go-udap set <mac> --config backup.conf
```

Here-string scripting is preserved:

```bash
go-udap set <mac> <<< "lan_ip_mode=1
wireless_SSID=foo"
```

## Per-param flags

All ~25 known UDAP parameters get a corresponding CLI flag.

### Naming

CLI flags use lowercase-with-hyphens (GNU convention). The protocol and INI
file use the canonical UDAP names (mixed case with underscores, e.g.
`wireless_SSID`).

Translation happens at the CLI boundary. A flag table maps the two:

```go
type paramFlag struct {
    udapName string  // "wireless_SSID" — used in protocol & INI
    flagName string  // "wireless-ssid" — used on CLI
    help     string  // shown in --help
}
```

All flags are `pflag.String` (string-typed). Validation, including
type/range/length checks, is delegated to the existing
`udap.validateParameter`. The CLI uses `flagSet.Changed(name)` to distinguish
"flag was set to empty string" from "flag was not set" — only `Changed` flags
are added to the parameter map.

### Help text

Each entry in the flag table includes a one-line help string (e.g.
`"0=static, 1=DHCP"` for `lan_ip_mode`). Adding a new parameter is one new line
in the slice.

## File layout

```
main.go                      ← thin: parses subcommand, dispatches to cli package
cli/
  cli.go                     ← dispatcher, global flags, --help, --version
  discover.go                ← go-udap discover [--info]
  info.go                    ← go-udap info <mac>
  read.go                    ← go-udap read <mac>
  get.go                     ← go-udap get <mac> <params...>
  set.go                     ← go-udap set <mac> [--config FILE] [--<param>...]
  save.go                    ← go-udap save <mac>
  reset.go                   ← go-udap reset <mac>
  commit.go                  ← go-udap commit <mac>
  params.go                  ← flag table for all 25 known params
  config.go                  ← INI file parser, source layering
  output.go                  ← formatting helpers (param=value, --info blocks)
udap/                        ← protocol layer; behavior unchanged, API simplified
  client.go, discovery.go, config.go, protocol.go,
  socket_unix.go, socket_windows.go, validation.go, logger.go
```

## Removed

- `github.com/chzyer/readline` dependency.
- All shell code in `main.go`: `printUsage`, `parseParameters`,
  `createCompleter`, the readline loop, and history file handling.
- The `~/.squeezebox_udap_history` file (no migration; users can delete it
  manually if they wish).

## Added

- One new dependency: `github.com/spf13/pflag` (POSIX/GNU-compliant `flag`
  drop-in; supports flags interleaved with positional args, which stdlib does
  not). No transitive dependencies.

## udap/ package

No code changes in this redesign. Every CLI command does a discovery preamble
that populates `device.IP`, so the existing IP-aware code paths remain
exercised. The bootstrap-mode broadcast fallback (`device.IP == "0.0.0.0"`)
continues to handle unconfigured devices correctly, since discovery returns IP
`0.0.0.0` for those.

## Testing

### New tests (`cli/`)

- `cli/config_test.go` — INI parser: valid file, comments (`#` and `;`), blank
  lines, whitespace trimming, malformed line (no `=`), unknown parameter name
  rejected.
- `cli/set_test.go` — source layering: file alone, stdin alone, flags alone,
  file+flags (flags win), file+stdin (file wins, warning on stderr),
  `--config -` reads stdin explicitly, no sources = usage error.
- `cli/params_test.go` — every entry in the flag table maps to a real
  `ConfigSettings` key in `udap/protocol.go`. Catches drift between the flag
  table and the param list.
- `cli/output_test.go` — `read` formatting (`param=value` lines, sorted),
  `get` single-param (bare value), `get` multi-param (lines).

### Existing tests

- `udap/client_test.go`, `udap/protocol_test.go` — unchanged. Protocol behavior
  is not modified.

### Integration

No new integration tests against real hardware in this design; the existing
ad-hoc validation pattern (run against a real SBR) continues to apply.

## Supporting file changes

- **`README.md`** — rewrite the Usage section. Drop the interactive shell
  description and command table. New examples are single-shot CLI invocations.
  Move the (revised) here-string section into the main usage flow rather than a
  separate "Scripting" sub-section.
- **`Taskfile.yml`** — `task run` and `task dev` currently invoke the shell.
  Update to `go-udap --help` (or drop them; they have no obvious use in the
  CLI-first model).
- **`DEVELOPMENT.md`** — scan for shell-specific references; remove or update.
- **`go.mod`** — `go mod tidy` will drop `chzyer/readline` and add
  `spf13/pflag`.
- **`CLAUDE.md`** — update the "CLI Commands" section to reflect the new
  command surface.

## Migration impact

- The interactive shell is gone. Users running `./go-udap` with no arguments
  get help output instead of a prompt.
- The here-string scripting pattern still works, but the body changes from a
  sequence of REPL commands to an INI-formatted parameter list (see example
  above).
- The `~/.squeezebox_udap_history` file is no longer written or read; existing
  files are untouched.
- Any external scripts that piped REPL commands to the binary need to be
  rewritten to use subcommands. There is no automatic translation.

## Open items deferred to implementation

These are deliberate non-decisions; the implementation plan should resolve them
based on what fits cleanly:

- Exact wording of error messages.
- Exact layout of the `--info` metadata block (one line per field, or
  one device per block?).
- Whether `info` reuses `discover` internals or has its own targeted-broadcast
  helper. Both are reasonable; the implementation plan picks one.
