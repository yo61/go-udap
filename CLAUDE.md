# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Squeezebox UDAP (Universal Device Access Protocol) configuration tool written in Go. It provides a command-line interface for discovering and configuring Squeezebox devices on the network using the UDAP protocol over UDP port 17784.

The codebase has been modernized to use current Go best practices and idiomatic patterns.

## Architecture

The application is structured with a modular design:

- **main.go**: Single-shot CLI interface with command handling
- **udap/client.go**: Core client that handles UDP communication and device management
- **udap/discovery.go**: Device discovery implementation with context support
- **udap/config.go**: Device configuration management with context-aware operations
- **udap/protocol.go**: Low-level packet creation/parsing with TLV (Type-Length-Value) encoding
- **udap/socket_unix.go**: Unix-specific socket options (macOS, Linux)
- **udap/socket_windows.go**: Windows-specific socket options

### Key Components

- **UDAPClient**: Core client that handles UDP communication and device management
- **UDAPDevice**: Represents discovered Squeezebox devices with their properties
- **UDAP Protocol Implementation**: Low-level packet creation/parsing with TLV encoding
- **CLI**: Single-shot command-line interface for device discovery and configuration

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
go build -ldflags="-s -w" -trimpath -o go-udap main.go

# Cross-compile for Windows
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o go-udap.exe main.go

# Run tests
go test ./...

# Development
go run main.go
```

## CLI Commands (when running the tool)

The tool is single-shot CLI; every operation is one invocation. There is no
interactive shell.

- `go-udap discover [--info]` — Discover devices; MACs only, or full metadata with `--info`
- `go-udap info <mac>` — Show metadata for one device
- `go-udap read <mac>` — Read all parameters from a device
- `go-udap get <mac> <param> [<param>...]` — Read specific parameters
- `go-udap set <mac> [--reboot/-r] [--config FILE] [--<param> VALUE ...]` — Set parameters from file, piped stdin, and/or per-param flags (CLI flags win). The wire op writes NVRAM directly (every UCP_METHOD_SET_DATA writes — there is no separate save_data wire method per the Net::UDAP reference). Pass `--reboot/-r` to also reboot after writing.
- `go-udap reboot <mac>` — Reboot the device

Global flags: `--timeout DURATION` (default 5s), `--verbose`/`-v`, `--version`, `--help`/`-h`.

Output is on stdout; logs and warnings on stderr. Exit codes: 0 success,
1 usage error, 2 operation failure.

## Development Notes

- Each invocation is independent; no persistent state between runs
- Network timeouts default to 5 seconds (configurable via `--timeout`)
- Discovery uses broadcast UDP with configurable timeout
- All UDAP packets use big-endian byte order for network transmission

## Code Modernization Status

The codebase has been modernized with the following improvements:

### ✅ Completed Modernizations

1. **Switch Statement Conversions**: Converted if-else chains to tagged switch statements for better readability
2. **Modern Map Operations**: Replaced manual map copying loops with `maps.Copy()` from Go 1.21+
3. **Context Support**: Added context-aware versions of all timeout-sensitive operations:
   - `DiscoverDevicesWithContext()` and related discovery functions
   - `GetDeviceConfigWithContext()`, `SetDeviceConfigWithContext()`
   - `ResetDeviceWithContext()`, `SaveDeviceConfigWithContext()`
   - All functions now properly handle context cancellation and timeouts
4. **Error Handling**: Comprehensive error wrapping improvements:
   - All errors now use `%w` verb for proper error wrapping
   - Added device context (MAC addresses) to error messages
   - Enhanced protocol parsing errors with detailed information
   - Eliminated bare `return err` statements in favor of descriptive wrapped errors
5. **Pure Go Networking**: Removed `gopacket/pcap` dependency for cross-platform compatibility:
   - All networking now uses standard Go `net` package
   - Platform-specific socket options via build tags (`socket_unix.go`, `socket_windows.go`)
   - Enables cross-compilation to Windows, Linux, and other platforms without CGO
   - No external dependencies required (libpcap/WinPcap/Npcap)

### 🚧 In Progress

6. **Structured Logging**: Replace `fmt.Printf` calls with structured logging

### 📋 Pending Modernizations

7. **Goroutine Lifecycle Management**: Improve goroutine management patterns
8. **Struct Validation Methods**: Add validation methods to data structures
9. **Magic Number Constants**: Replace magic numbers with named constants
10. **Resource Cleanup Patterns**: Enhance resource cleanup and defer usage

## Cross-Platform Support

The tool builds and runs on multiple platforms without any external dependencies:

| Platform | Build Command | Binary Size (optimized) |
|----------|---------------|------------------------|
| macOS (amd64/arm64) | `go build` | ~2.8 MB |
| Windows (amd64) | `GOOS=windows GOARCH=amd64 go build` | ~2.9 MB |
| Linux (amd64) | `GOOS=linux GOARCH=amd64 go build` | ~2.8 MB |
| Linux (arm64) | `GOOS=linux GOARCH=arm64 go build` | ~2.7 MB |

**Note**: Windows binaries can be further compressed with UPX (`upx --best`) to ~1.2 MB.

## API Compatibility

All public APIs maintain backward compatibility. Context-aware functions are available alongside original timeout-based versions:

```go
// Legacy API (still supported)
err := client.DiscoverDevices(5 * time.Second)

// Modern context-aware API
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
err := client.DiscoverDevicesWithContext(ctx)
```
