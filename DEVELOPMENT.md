# Development Guide

This document covers building, testing, and contributing to the Squeezebox UDAP Configuration Tool.

## Prerequisites

- Go 1.26 or later (see `go.mod`)
- [Task](https://taskfile.dev/) (optional, for build automation)

## Building from Source

### Using Task (Recommended)

```bash
# Build optimized binary for current platform
task build

# Run tests
task test

# Run tests with verbose output
task test:verbose

# Format code
task fmt

# Run linter
task lint

# Clean build artifacts
task clean
```

### Using Go Directly

```bash
# Development build
go build -o go-udap .

# Optimized build (smaller binary)
go build -ldflags="-s -w" -trimpath -o go-udap .

# Run tests (with race detector)
go test -race ./...
```

## Cross-Compilation

The tool uses pure Go networking (no cgo dependencies), enabling easy cross-compilation.

### Using Task

```bash
# Windows (amd64)
task build:windows

# Linux (amd64)
task build:linux

# Linux (arm64)
task build:linux-arm64

# All platforms
task build:all
```

### Using Go Directly

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o go-udap.exe .

# Linux (amd64)
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o go-udap-linux-amd64 .

# Linux (arm64)
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -trimpath -o go-udap-linux-arm64 .
```

### Platform Support

| Platform | Architecture | Status |
|----------|--------------|--------|
| macOS | amd64, arm64 | Supported |
| Linux | amd64, arm64 | Supported |
| Windows | amd64 | Supported |

## Project Structure

```
go-udap/
├── main.go                       # 16-line entry point; calls cli.Run
├── cli/                          # CLI surface (one file per subcommand)
│   ├── cli.go                    # dispatcher, global flag hoisting
│   ├── discover.go info.go read.go get.go set.go reboot.go
│   ├── find.go                   # discover-and-find-by-MAC helper
│   ├── params.go                 # CLI flag table derived from udap.Parameters
│   ├── source.go                 # layered set sources (file/stdin/flags)
│   ├── config.go                 # INI parser
│   ├── output.go                 # formatParamMap, formatDeviceInfo
│   ├── progress.go stderr.go     # TTY progress bar + log/bar mutex
│   └── *_test.go                 # unit tests
├── udap/                         # protocol + transport
│   ├── client.go                 # UDP socket, packet builders, capture
│   ├── discovery.go              # broadcast + listener, RWMutex-protected device map
│   ├── config.go                 # GetData / SetData / Reset (WithContext)
│   ├── protocol.go               # Packet struct, ParsePacket, constants
│   ├── parameters.go             # ★ single source of truth for 26 NVRAM params
│   ├── getdata_response.go       # offset/length/value response decoder
│   ├── loopback.go               # isUDAPRequestPacket — kernel-loopback filter
│   ├── validation.go             # parameter / packet validation
│   ├── logger.go                 # structured logger (takes io.Writer)
│   ├── socket_unix.go            # SO_BROADCAST via SyscallConn().Control
│   ├── socket_windows.go
│   ├── testdata/captures/*.bin   # captured Net::UDAP wire payloads
│   └── *_test.go
├── docs/superpowers/             # planning specs/plans (history)
├── Taskfile.yml                  # Task automation
├── go.mod / go.sum               # module definition (Go 1.26.3, only pflag dep)
└── README.md / CLAUDE.md / DEVELOPMENT.md
```

## Protocol Details

### UDAP Overview

- UDP broadcast on port 17784
- Packet format uses magic number and TLV (Type-Length-Value) encoding
- Network byte order (big-endian) for all protocol fields

### UCP Methods

Per the [Net::UDAP](https://github.com/robinbowes/net-udap) Constant.pm
reference (`UCP_METHOD_*`):

| Method | Code | Description |
|--------|------|-------------|
| Discover | 0x0001 | Basic discovery (broadcast) |
| GetIP | 0x0002 | Returns network-config TLVs (lan_ip_mode, lan_*_address). Not a generic "data response". |
| Reset | 0x0004 | Reboot the device (header-only request; device echoes 0x0004) |
| GetData | 0x0005 | Read NVRAM. Request: `[16 zero user][16 zero pass][uint16 count][N×(offset, length)]`. Response: same method, with `[uint16 count][N×(offset, length, value)]` |
| SetData | 0x0006 | Write NVRAM. Same wire shape as GetData but with values appended; device echoes 0x0006 with a uint16 count of accepted params. There is no separate save_data wire method — every 0x0006 writes NVRAM. |
| Error | 0x0007 | Generic error response |
| CredentialsError | 0x0008 | Device rejected the request's user/pass fields |
| AdvDisc | 0x0009 | Advanced discovery (broadcast) — what the CLI uses |

### Configuration Storage

Device configuration is stored in NVRAM at specific byte offsets. The
authoritative mapping lives in `udap/parameters.go` (the `Parameters`
slice). Each entry carries the wire offset, length, the CLI flag's
placeholder hint (`IP`, `0|1`, `NAME`, ...), the help text, and the
factory-default value (used by `read` to filter uninteresting output).

## Testing

### Running Tests

```bash
# All tests
task test

# Verbose output
task test:verbose

# With coverage report
task test:coverage
```

### Manual Testing

With a physical Squeezebox device:

1. Put the device in setup mode by holding the front button for ~3
   seconds until the LED blinks slow red (or factory-reset by holding
   ~6 seconds until it blinks fast red — see
   <https://wiki.lyrion.org/index.php/SBRFrontButtonAndLED>).
2. Run `go-udap discover --info` to confirm the device appears.
3. Configure with `set` (use `--reboot/-r` to apply changes that take
   effect on reboot).
4. Read back with `read` to verify (output is filtered to non-default
   values — pass `--all` to see everything).

## Code Style

- Follow standard Go formatting (`go fmt`)
- Use `go vet` for static analysis
- Structured logging with key-value pairs
- Context-aware functions for cancellation support

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `task test`
5. Format code: `task fmt`
6. Submit a pull request

## Acknowledgments

- UDAP protocol based on [LMS-Community/squeezeplay](https://github.com/LMS-Community/squeezeplay)
- Wire format and constant tables verified against the Perl
  [Net::UDAP](https://github.com/robinbowes/net-udap) reference
  implementation (the `Constant.pm`, `Client.pm`, and `Shell.pm`
  sources in particular).
