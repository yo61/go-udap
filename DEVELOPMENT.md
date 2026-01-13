# Development Guide

This document covers building, testing, and contributing to the Squeezebox UDAP Configuration Tool.

## Prerequisites

- Go 1.21 or later
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
go build -o squeezebox-udap main.go

# Optimized build (smaller binary)
go build -ldflags="-s -w" -trimpath -o squeezebox-udap main.go

# Run tests
go test ./...
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
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o squeezebox-udap.exe main.go

# Linux (amd64)
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o squeezebox-udap-linux-amd64 main.go

# Linux (arm64)
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -trimpath -o squeezebox-udap-linux-arm64 main.go
```

### Platform Support

| Platform | Architecture | Status |
|----------|--------------|--------|
| macOS | amd64, arm64 | Supported |
| Linux | amd64, arm64 | Supported |
| Windows | amd64 | Supported |

## Project Structure

```
squeezebox-udap/
├── main.go                 # CLI interface and command handling
├── udap/
│   ├── client.go          # UDP client and packet capture
│   ├── config.go          # Device configuration operations
│   ├── discovery.go       # Device discovery implementation
│   ├── protocol.go        # UDAP protocol constants and parsing
│   ├── validation.go      # Input validation functions
│   ├── logger.go          # Structured logging
│   ├── socket_unix.go     # Unix socket options (macOS/Linux)
│   └── socket_windows.go  # Windows socket options
├── Taskfile.yml           # Task automation configuration
├── go.mod                 # Go module definition
└── go.sum                 # Go module checksums
```

## Protocol Details

### UDAP Overview

- UDP broadcast on port 17784
- Packet format uses magic number and TLV (Type-Length-Value) encoding
- Network byte order (big-endian) for all protocol fields

### UCP Methods

| Method | Code | Description |
|--------|------|-------------|
| Discover | 0x0001 | Standard discovery |
| GetIP | 0x0002 | Get IP / Data response |
| Reset | 0x0004 | Device reset |
| GetData | 0x0005 | Get configuration data |
| SetData | 0x0006 | Set configuration data |
| Error | 0x0007 | Error response |
| SetDataAck | 0x0008 | SetData acknowledgment |
| AdvDisc | 0x0009 | Advanced discovery |

### Configuration Storage

Device configuration is stored in NVRAM at specific offsets. See `udap/protocol.go` for the complete mapping of parameter names to NVRAM offsets.

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

1. Put device in setup mode (factory reset or hold button during boot)
2. Run the tool and execute `discover`
3. Configure and test various parameters

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
- Inspired by the Perl [Net::UDAP](https://metacpan.org/pod/Net::UDAP) module
