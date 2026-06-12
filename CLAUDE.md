# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Squeezebox UDAP (Universal Device Access Protocol) configuration tool written in Go. It provides a command-line interface for discovering and configuring Squeezebox devices on the network using the UDAP protocol over UDP port 17784.

The codebase has been modernized to use current Go best practices and idiomatic patterns.

## Architecture

The application is structured with a modular design:

- **main.go**: Thin entry point — parses os.Args and delegates to cli.Run.
- **cli/**: Single-shot CLI surface. cli.go dispatches subcommands;
  cli/{discover,info,read,get,set,reboot,getip,interfaces}.go implement
  them. cli/find.go has the discover-and-find-by-MAC helper used by
  every device-targeted command. cli/params.go is the CLI flag table
  derived from udap.Parameters; cli/source.go layers --config FILE /
  piped stdin / per-param flags for `set`. cli/progress.go and
  cli/stderr.go provide the progress bar (TTY-detected) and the mutex
  that serializes its output with the udap logger.
- **udap/client.go**: Core client (UDP socket, packet builders,
  capture). Also defines NewClientForInterface and
  NewClientForAllInterfaces constructors for per-interface and fan-out
  modes.
- **udap/transport.go**: UDPTransport (Transport interface impl over a
  real *net.UDPConn) and NewUDPTransportOnInterface (uses
  net.ListenConfig to set SO_REUSEPORT pre-bind and IP_BOUND_IF /
  SO_BINDTODEVICE for egress NIC selection).
- **udap/multi_transport.go**: MultiTransport composes N child
  Transports; Send fans out, Recv merges via per-child pump
  goroutines. Used by --all-interfaces.
- **udap/discovery.go**: Discovery broadcast + listener; populates
  Client.devices under a RWMutex. Parses HardwareRev (TLV 0x0a) and
  UUID (TLV 0x0d) into Device.
- **udap/config.go**: GetData / SetData / Reset operations
  (WithContext entry points only — no hardcoded-timeout legacy shims).
- **udap/getip.go**: CreateGetIPPacket + GetDeviceNetworkConfigWithContext
  for UCP_METHOD_GET_IP (0x0002). parseGetIPResponse decodes TLV 0x05
  (IP) / 0x06 (SubnetMask) / 0x07 (Gateway) into NetworkConfig.
- **udap/getuuid.go**: CreateGetUUIDPacket + GetDeviceUUIDWithContext
  for UCP_METHOD_GET_UUID (0x000b). parseGetUUIDResponse decodes TLV
  0x0d (16-byte UUID) into a 32-char lowercase hex string. Used by the
  CLI as a fallback when discovery's TLV 0x0d is missing (older
  firmware that omits UUID from adv_discover).
- **udap/netconfig.go**: NetworkConfig value object (result of get_ip).
- **udap/interfaces.go**: NetInterface value object + EnumerateInterfaces
  (filters: Up + Broadcast + !Loopback + has IPv4) + computeDirectedBroadcast.
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
- **udap/socket_{unix,darwin,linux,windows}.go**: Platform-specific
  socket-options helpers. socket_unix.go (!windows) defines
  enableBroadcast (SO_BROADCAST + SO_REUSEADDR post-bind), using
  SyscallConn().Control() (NOT File()) to keep the socket in
  non-blocking-via-poller mode on macOS. socket_darwin.go and
  socket_linux.go define bindToInterface (IP_BOUND_IF / SO_BINDTODEVICE
  for output-NIC selection) and setReusePortPreBind (SO_REUSEADDR +
  SO_REUSEPORT, called from net.ListenConfig.Control so the options
  land pre-bind, allowing multiple sockets on the same 0.0.0.0:port).
  socket_windows.go has stubs returning "not supported".

### Key Components

- **udap.Client**: UDP communication + device map (RWMutex-protected).
  Default constructor binds 0.0.0.0:17784 and broadcasts to
  255.255.255.255. NewClientForInterface and NewClientForAllInterfaces
  swap in alternate Transports.
- **udap.Device**: Discovered device metadata (MAC, IP, Name, Model,
  Firmware, HardwareRev, UUID, State, Parameters).
- **udap.NetworkConfig**: Value object — result of the get_ip query
  (IP / SubnetMask / Gateway as net.IP, all optional, "-" rendered
  for zero values).
- **udap.NetInterface**: Value object — anti-corruption layer over
  net.Interface. Carries Name, Index, Addr (IPv4), and Broadcast
  (informational only; UDAP sends always go to 255.255.255.255).
- **udap.Transport**: Interface (Send/Recv/Close) implemented by
  UDPTransport (real socket) and MultiTransport (fan-out over N
  children). mocksbr.MockTransport provides the in-process test path.
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
task security           # Run govulncheck + grype locally (matches CI)
task dev                # Run without building (go run)
task docs:dev           # Run the Fumadocs site locally with hot reload (no basePath)
task docs:serve         # Production-build the docs site and serve locally (no basePath)
task docs:build         # Production build of the docs site with /go-udap basePath (matches CI)
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

### Security scanning

CI runs `govulncheck` and `grype` against every PR, every push to `main`, and on a daily cron (`.github/workflows/security.yaml`). To reproduce locally:

```bash
task security
```

`govulncheck` (Go-native, reachability-aware) is run via `go run` so no install is needed. `grype` is optional locally — install with `brew install grype`. CI is authoritative; the local target is for quick iteration on dep upgrades.

SBOMs are produced two ways:
- **Per release:** `.goreleaser.yaml` emits SPDX-JSON and CycloneDX-JSON per archive (uploaded as release artifacts).
- **Per CI run:** `security.yaml` produces a CycloneDX SBOM artifact for Grype to scan.

## CLI Commands (when running the tool)

The tool is single-shot CLI; every operation is one invocation. There is no
interactive shell.

- `go-udap discover [--info]` — Discover devices; MACs only, or full metadata (including IP/subnet/gateway via per-device get_ip) with `--info`. Per-device get_ip failures are soft (dashes in output; warning gated on `--verbose`). When discovery omits TLV 0x0d (older firmware), `--info` falls back to `get_uuid` (UCP 0x000b) to populate UUID — also soft-fail with `--verbose`-gated warning.
- `go-udap info MAC` — Show metadata for one device (MAC, IP, Name, Model, Firmware, HW Rev, UUID, State). Same `get_uuid` fallback as `discover --info`.
- `go-udap read MAC [--all/-a]` — Read parameters from a device. By default skips factory-default values (so output round-trips cleanly through `set`); pass `--all`/`-a` to dump everything including factory defaults and unrecognized `offset_NNN` entries.
- `go-udap get MAC PARAM [PARAM...]` — Read specific parameters
- `go-udap set MAC [--reboot/-r] [--config FILE] [--<param> VALUE ...]` — Set parameters from file, piped stdin, and/or per-param flags (CLI flags win). The wire op writes NVRAM directly (every UCP_METHOD_SET_DATA writes — there is no separate save_data wire method per the Net::UDAP reference). Pass `--reboot/-r` to also reboot after writing.
- `go-udap reboot MAC` — Reboot the device
- `go-udap getip MAC` — Query the device's current IP / subnet / gateway via UCP_METHOD_GET_IP (0x0002). Distinct from discovery: discover passively observes; getip actively asks
- `go-udap interfaces` — List local network interfaces usable for UDAP discovery (Up + Broadcast + has IPv4 + not loopback). Useful for picking a value for `--bind-interface NAME`

Global flags: `--timeout DURATION` (default 2s), `--retries N` (default 0), `--verbose`/`-v`, `--version`, `--help`/`-h`, `--bind-interface NAME`, `--all-interfaces`.
Global flags are accepted before OR after the subcommand
(`go-udap -v read MAC` and `go-udap read -v MAC` are equivalent).

`--retries N` configures send-side retransmission: N is the number of **re-transmissions** beyond the initial send, so `--retries 2` results in 3 total sends (matches squeezeplay's hardcoded triple-send). Useful on lossy links; default 0 (one send, current behavior).

`--bind-interface NAME` binds discovery and all subsequent operations to a single named interface, validated pre-dispatch (unknown name → exit 1). `--all-interfaces` fans out across every usable interface via MultiTransport. The two flags are mutually exclusive (combining them → exit 1). On Windows both flags surface "not supported" since the platform-specific output-NIC binding isn't implemented there. The singular flag is `--bind-interface` (not `--interface`) so it doesn't collide with `set`'s per-param `--interface 0|1` flag (NVRAM byte at offset 52: 0=wireless, 1=wired).

Output is on stdout; logs and warnings on stderr. Exit codes: 0 success,
1 usage error, 2 operation failure.

## Development Notes

- Each invocation is independent; no persistent state between runs
- Network timeouts default to 2 seconds (configurable via `--timeout`)
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
client, err := udap.NewClient()                                // bind UDP 17784 (default, 0.0.0.0)
client, err := udap.NewClientForInterface("en0", logger)       // bind 0.0.0.0 + IP_BOUND_IF/SO_BINDTODEVICE → en0
client, err := udap.NewClientForAllInterfaces(logger)          // MultiTransport fan-out over all usable interfaces

err = client.DiscoverDevicesWithContext(ctx)                   // broadcast advanced discovery
device := client.GetDevice("00:04:20:16:05:8f")                // lookup by MAC (RWMutex-protected)
devices := client.ListDevices()                                // snapshot

err = client.GetAllDeviceConfigWithContext(ctx, device)        // read all 26 known params
m, err := client.GetDeviceConfigWithContext(ctx, device, names)// read selected
err = client.SetDeviceConfigWithContext(ctx, device, kvMap)    // write (RMW: read-modify-write all 26)
err = client.ResetDeviceWithContext(ctx, device)               // reboot
nc, err := client.GetDeviceNetworkConfigWithContext(ctx, device) // UCP_METHOD_GET_IP (0x0002) → NetworkConfig
uuid, err := client.GetDeviceUUIDWithContext(ctx, device)      // UCP_METHOD_GET_UUID (0x000b) → 32-char hex string

ifs, err := udap.EnumerateInterfaces()                         // []NetInterface: Up+Broadcast+!Loopback+IPv4
```

The `udap.Parameters` slice is the single source of truth for the 26
known NVRAM parameters (name, offset, length, CLI placeholder, help
text, factory default). The CLI's per-param flag table is derived
from it; adding a new parameter only requires editing that one slice.

### Discovery sends to limited broadcast (255.255.255.255), always

UDAP discovery is designed for unconfigured devices (source IP
`0.0.0.0`, no DHCP lease yet). Such devices don't know their subnet,
so they only process limited broadcast `255.255.255.255` — directed
subnet broadcasts like `192.168.1.255` don't reach them. Therefore
all UDAP sends go to `255.255.255.255`. To target a specific NIC on
a multi-homed host, `NewUDPTransportOnInterface` keeps the local bind
at `0.0.0.0` (so limited-broadcast replies come back) and uses
`IP_BOUND_IF` (macOS) / `SO_BINDTODEVICE` (Linux) to constrain output
to the chosen interface. `NetInterface.Broadcast` is informational
only — shown by `go-udap interfaces` but never used as a send
destination.
