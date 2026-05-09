# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Squeezebox UDAP (Universal Device Access Protocol) configuration tool written in Go. It provides a command-line interface for discovering and configuring Squeezebox devices on the network using the UDAP protocol over UDP port 17784.

The codebase has been modernized to use current Go best practices and idiomatic patterns.

## Architecture

The application is structured with a modular design:

- **main.go**: Thin entry point — parses os.Args and delegates to cli.Run.
- **cli/**: Single-shot CLI surface. cli.go dispatches subcommands;
  cli/{discover,info,read,get,set,reboot}.go implement them.
  cli/find.go has the discover-and-find-by-MAC helper used by every
  device-targeted command. cli/params.go is the CLI flag table derived
  from udap.Parameters; cli/source.go layers --config FILE / piped
  stdin / per-param flags for `set`. cli/progress.go and cli/stderr.go
  provide the progress bar (TTY-detected) and the mutex that
  serializes its output with the udap logger.
- **udap/client.go**: Core client (UDP socket, packet builders, capture).
- **udap/discovery.go**: Discovery broadcast + listener; populates
  Client.devices under a RWMutex.
- **udap/config.go**: GetData / SetData / Reset operations
  (WithContext entry points only — no hardcoded-timeout legacy shims).
- **udap/protocol.go**: Packet struct, ParsePacket, TLV codecs, constants.
- **udap/parameters.go**: Single source of truth for the 26 known UDAP
  NVRAM parameters — name, offset, length, CLI placeholder, help text.
  Aliases (e.g. squeezecenter_address → server_address) live here too.
- **udap/getdata_response.go**: Decoder for the offset/length/value
  GetData response payload (verified against Net::UDAP wire captures).
- **udap/loopback.go**: isUDAPRequestPacket — UCPFlags-bit check that
  lets the capture path skip our own kernel-looped broadcast.
- **udap/logger.go**: Structured logger; takes an io.Writer so the CLI
  can route it through stderrSync.
- **udap/socket_{unix,windows}.go**: Platform-specific SO_BROADCAST
  setup. The Unix variant uses SyscallConn().Control() (NOT File())
  to keep the socket in non-blocking-via-poller mode on macOS.

### Key Components

- **udap.Client**: UDP communication + device map (RWMutex-protected).
- **udap.Device**: Discovered device metadata (MAC, IP, Name, Model,
  Firmware, State, Parameters).
- **udap.Parameters**: Canonical table of NVRAM parameters; CLI flag
  table is derived from it.
- **CLI**: Single-shot subcommand interface; global flags work before
  or after the subcommand.

### Key Protocol Details

- Uses UDP broadcast on port 17784 for device discovery (updated from original 3483)
- Implements UDAP packet format with magic number 0x75646170 ("udap")
- Supports message types: Discovery, SetData, GetData, DataResp, Error
- Device responses use TLV encoding for structured data
- Network byte order (big-endian) for all protocol fields

## Common Commands

This project uses [Task](https://taskfile.dev/) for build automation. Install with `brew install go-task`.

### Using Task (Recommended)
```bash
task build              # Build optimized binary for current platform
task build:all          # Build for all platforms (macOS, Windows, Linux)
task build:windows      # Cross-compile for Windows
task build:linux        # Cross-compile for Linux amd64
task build:linux-arm64  # Cross-compile for Linux arm64
task test               # Run all tests
task test:verbose       # Run tests with verbose output
task test:coverage      # Generate coverage report
task fmt                # Format all Go files
task lint               # Run go vet
task tidy               # Tidy go modules
task clean              # Remove build artifacts
task run                # Build and run
task dev                # Run without building (go run)
```

### Manual Commands
```bash
# Build optimized binary
go build -ldflags="-s -w" -trimpath -o go-udap .

# Cross-compile for Windows
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o go-udap.exe .

# Run tests (with race detector)
go test -race ./...

# Development
go run .
```

## CLI Commands (when running the tool)

The tool is single-shot CLI; every operation is one invocation. There is no
interactive shell.

- `go-udap discover [--info]` — Discover devices; MACs only, or full metadata with `--info`
- `go-udap info <mac>` — Show metadata for one device
- `go-udap read <mac> [--all/-a]` — Read parameters from a device. By default skips factory-default values (so output round-trips cleanly through `set`); pass `--all`/`-a` to dump everything including factory defaults and unrecognized `offset_NNN` entries.
- `go-udap get <mac> <param> [<param>...]` — Read specific parameters
- `go-udap set <mac> [--reboot/-r] [--config FILE] [--<param> VALUE ...]` — Set parameters from file, piped stdin, and/or per-param flags (CLI flags win). The wire op writes NVRAM directly (every UCP_METHOD_SET_DATA writes — there is no separate save_data wire method per the Net::UDAP reference). Pass `--reboot/-r` to also reboot after writing.
- `go-udap reboot <mac>` — Reboot the device

Global flags: `--timeout DURATION` (default 5s), `--verbose`/`-v`, `--version`, `--help`/`-h`.
Global flags are accepted before OR after the subcommand
(`go-udap -v read <mac>` and `go-udap read -v <mac>` are equivalent).

Output is on stdout; logs and warnings on stderr. Exit codes: 0 success,
1 usage error, 2 operation failure.

## Development Notes

- Each invocation is independent; no persistent state between runs
- Network timeouts default to 5 seconds (configurable via `--timeout`)
- Discovery uses broadcast UDP with configurable timeout
- All UDAP packets use big-endian byte order for network transmission

## Cross-Platform Support

The tool builds and runs on multiple platforms without any external dependencies:

| Platform | Build Command | Binary Size (optimized) |
|----------|---------------|------------------------|
| macOS (amd64/arm64) | `go build` | ~2.8 MB |
| Windows (amd64) | `GOOS=windows GOARCH=amd64 go build` | ~2.9 MB |
| Linux (amd64) | `GOOS=linux GOARCH=amd64 go build` | ~2.8 MB |
| Linux (arm64) | `GOOS=linux GOARCH=arm64 go build` | ~2.7 MB |

**Note**: Windows binaries can be further compressed with UPX (`upx --best`) to ~1.2 MB.

## udap package API

All operations take a `context.Context`; there are no timeout-based
shim entry points. The exported surface is:

```go
client, err := udap.NewClient()                                // bind UDP 17784
err = client.DiscoverDevicesWithContext(ctx)                   // broadcast advanced discovery
device := client.GetDevice("00:04:20:16:05:8f")                // lookup by MAC (RWMutex-protected)
devices := client.ListDevices()                                // snapshot

err = client.GetAllDeviceConfigWithContext(ctx, device)        // read all 26 known params
m, err := client.GetDeviceConfigWithContext(ctx, device, names)// read selected
err = client.SetDeviceConfigWithContext(ctx, device, kvMap)    // write (RMW: read-modify-write all 26)
err = client.ResetDeviceWithContext(ctx, device)               // reboot
```

The `udap.Parameters` slice is the single source of truth for the 26
known NVRAM parameters (name, offset, length, CLI placeholder, help
text, factory default). The CLI's per-param flag table is derived
from it; adding a new parameter only requires editing that one slice.
