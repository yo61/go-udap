# Mock Squeezebox Receiver design

**Date:** 2026-05-08
**Status:** Draft for review
**Depends on:** [2026-05-07 CLI redesign](2026-05-07-cli-redesign-design.md) (must be merged first)

## Goal

Build a software mock of a Squeezebox Receiver (SBR) that responds to UDAP
packets the way a real SBR does, so `go-udap` can be developed and tested
without real hardware. Multiple virtual devices can run side by side. The mock
is usable both as a standalone binary (real UDP loopback) and as an in-process
Go library (no network at all).

## Motivation

Today the `go-udap` test suite has no way to exercise the full
discover/get/set/save/reset workflow without plugging in a real SBR. End-to-end
verification is manual; CI can't catch regressions in the protocol layer or the
CLI's interaction with it. A software mock lets us:

- Run hermetic in-process tests of `udap.Client` against a faked transport.
- Run end-to-end tests of `go-udap` against the mock binary on UDP loopback.
- Develop `go-udap` features when no real SBR is available.

## Non-goals

- No real factory-reset semantics (a real SBR's factory reset is the front
  button; the mock just starts in factory state).
- No multi-machine mock cluster (everything runs on one host).
- No simulation of audio playback, button presses, or any non-UDAP behavior.
- No runtime control plane (failure injection is configured at process start
  only, not toggled while the mock is running).
- Phase 1 does not implement Phase 2/3 features; they are scoped here for
  consistent interface design but ship as separate implementation plans.

## Architecture overview

Three new pieces in the codebase:

1. **`udap.Transport` interface** — abstracts the network from `udap.Client`.
   Two implementations:
   - `UDPTransport` — wraps the real UDP socket, extracted from current
     `udap/client.go` socket-handling code. Production default.
   - `MockTransport` (lives in `mocksbr/`) — in-memory, hands packets directly
     to mocksbr device handlers in the same process. Used by hermetic tests.

2. **`mocksbr/` Go package** — N independent virtual SBR device state machines
   (working memory + NVRAM + identity + per-device knobs), packet-handler
   functions for each UDAP method, and a fan-out router that takes an inbound
   packet and dispatches to the matching device by destination MAC. Pure
   logic — no networking.

3. **`cmd/mocksbr/` binary** — wires `mocksbr.Network` to a `UDPTransport`
   listening on UDP/17784, plus CLI flag parsing for device count and
   per-device overrides.

### File layout

```
udap/
  transport.go             ← NEW: Transport interface + UDPTransport
  client.go                ← refactored to use Transport, no direct conn refs
  ...                      ← discovery.go, config.go also refactored
mocksbr/
  device.go                ← virtual SBR state machine
  network.go               ← Network of N devices, fan-out by destination MAC
  handlers.go              ← per-method packet handlers
  transport.go             ← MockTransport (couples Network to udap.Transport)
  identity.go              ← auto-generated MACs/UUIDs/names
  nvram.go                 ← shared INI loader (or imports from internal/ini)
  testdata/
    captures/              ← real-SBR packet captures (binary fixtures)
  testhelper/
    spawn.go               ← test-only helper: spawn cmd/mocksbr subprocess
cmd/
  mocksbr/
    main.go                ← CLI: flag parsing, wires Network + UDPTransport
internal/
  ini/                     ← extracted from cli/config.go; shared between
                              cli/ and mocksbr/ (only if needed; alternative
                              is for mocksbr to import cli/ — decided in
                              implementation)
```

### Test integration paths

- **In-process hermetic tests** (`udap` and `cli` package tests):
  ```go
  net := mocksbr.NewNetwork(3, udap.NewNoOpLogger())
  client, _ := udap.NewClientWithTransport(mocksbr.NewMockTransport(net), logger)
  ```
  No UDP, no port conflicts, deterministic timing.

- **End-to-end binary tests**: `mocksbr.SpawnMock(t, "--devices", "3")` spawns
  `cmd/mocksbr` as a subprocess; the test then runs `cli.Run([...])` against
  it. Real loopback UDP.

## `udap.Transport` interface

UDAP-aware (packet-shaped, not UDP-shaped):

```go
// Transport is the network abstraction underneath udap.Client. It handles
// broadcast send and asynchronous receive of raw UDAP packets; addressing
// is encoded in the packets themselves, not at the transport layer.
type Transport interface {
    // Send dispatches a UDAP packet from a client. The destination MAC is
    // encoded inside the packet. UDPTransport broadcasts to the LAN;
    // MockTransport feeds the packet directly to its connected mock devices.
    Send(packet []byte) error

    // Recv blocks until a packet arrives or ctx is cancelled. Returns the
    // packet bytes and an informational source identifier (e.g. an IP
    // string for UDPTransport; a MAC for MockTransport). The src is for
    // logging only; routing decisions use the packet's contents.
    Recv(ctx context.Context) (packet []byte, src string, err error)

    // Close releases transport resources.
    Close() error
}
```

### `UDPTransport`

Wraps a `*net.UDPConn` bound to UDP/17784 (or a configurable port — keeps
the `newClientWithPort` capability already added in the CLI redesign).
`Send` calls `WriteToUDP` with broadcast destination `255.255.255.255:17784`.
`Recv` calls `ReadFromUDP` with a deadline derived from `ctx`.

The current code's three "capture" code paths (`capturePacketWithContext`,
`capturePacketFromExistingConn`, the listener-goroutine model in
`discovery.go`) collapse into the single `Recv` loop. The
`SetReadDeadline`-juggling and racy goroutine cleanup go away as a
side-effect.

### `MockTransport`

Lives in `mocksbr/transport.go` to avoid the udap package depending on
mocksbr. `Send` calls `Network.Receive(packet)` synchronously and queues any
returned reply packets onto an internal channel. `Recv` reads from that
channel, blocking on `ctx`. No real network is involved.

### `udap.Client` changes

Public API gains one new constructor:

```go
// NewClientWithTransport constructs a Client using an arbitrary Transport.
// Used by tests that want to inject a MockTransport.
func NewClientWithTransport(t Transport, logger Logger) (*Client, error)
```

Existing `NewClient` and `NewClientWithLogger` are preserved; under the hood
they construct a `UDPTransport` and call `NewClientWithTransport`. The
private `newClientWithPort` (added in the CLI redesign) similarly delegates.

Inside `Client`, all `c.conn.WriteToUDP/ReadFromUDP/SetReadDeadline/Close`
sites are replaced by `c.transport.Send/Recv/Close` calls. The
broadcast-vs-unicast branching in `udap/config.go` and `udap/discovery.go`
goes away — `Client.Send` always pushes one packet through the transport;
per-device routing is the device's job (it filters by destination MAC in
the packet header).

Note that `udap.Client` is the *initiator* side: it always sends out
(broadcast in UDP mode); replies arrive via `Recv`. The mock
implementation handles the *responder* side and unicasts replies back to
the source — see the `cmd/mocksbr` Behavior section below.

The `PacketCaptureConfig`/`PacketCaptureResult` types and
`capturePacketWithContext`/`capturePacketFromExistingConn` helpers in
`udap/client.go` are removed; their callers in `udap/config.go` use
`c.transport.Recv(ctx)` directly with a packet-classification helper.

## `mocksbr` package public surface

```go
// Network is one or more virtual SBR devices sharing a single inbound
// packet queue, dispatched by destination MAC.
type Network struct { /* ... */ }

// NewNetwork constructs a Network of n auto-generated virtual devices.
// Auto-generated identities use deterministic MACs (00:04:20:00:00:01..N),
// UUIDs (mock-sbr-001..N), and names (Mock SBR 1..N), all with model
// "Mock" and firmware "0.0.0". All devices start in factory state with
// hardcoded factory-default NVRAM contents.
func NewNetwork(n int, logger udap.Logger) *Network

// Add appends one explicitly-configured device. Returns the assigned MAC.
// Used by tests and by cmd/mocksbr's per-device override flag.
func (n *Network) Add(cfg DeviceConfig) (mac string, err error)

// Receive feeds an inbound packet to the matching device (by destination
// MAC encoded in the packet) and returns zero or more reply packets.
// Discovery broadcasts produce N replies (one per device). Unicast
// requests produce one reply, or zero if the targeted device is
// Unreachable, currently Slow-delaying, or in its post-Reset reboot
// window.
func (n *Network) Receive(packet []byte) (replies [][]byte)

// Close releases per-device resources (Slow timers, reboot timers).
func (n *Network) Close() error

// DeviceConfig is the knobs for one virtual device.
type DeviceConfig struct {
    MAC      string  // required; must be a valid MAC address
    Name     string  // optional; defaults to "Mock SBR <n>"
    Model    string  // optional; defaults to "Mock"
    Firmware string  // optional; defaults to "0.0.0"
    UUID     string  // optional; defaults to "mock-sbr-<n>"

    // Phase 2: pre-configured state
    NVRAM map[string]string  // initial NVRAM contents (overrides factory
                             // defaults). Loaded from --device nvram=FILE
                             // by cmd/mocksbr.

    // Phase 3: failure injection
    FailOn      []Op            // return UDAP error for these ops
    Slow        time.Duration   // delay every reply by this duration
    Unreachable bool            // drop all packets, never reply
    RebootDelay time.Duration   // override default reboot window (100ms)
}

// Op identifies a UDAP operation for failure-injection knobs.
type Op string
const (
    OpDiscover Op = "discover"
    OpGet      Op = "get"
    OpSet      Op = "set"
    OpSave     Op = "save"
    OpReset    Op = "reset"
)

// MockTransport implements udap.Transport, backed by a Network.
type MockTransport struct { /* ... */ }
func NewMockTransport(net *Network) *MockTransport
```

What is NOT exposed: packet-handler functions, internal device-state structs,
NVRAM internal types. Tests assert state by reading back via `udap.Client`
(the same way real-world callers would).

### Device state machine

Each virtual device holds two parameter maps:

- **Working memory** — what `GetData` returns; what `SetData` mutates.
- **NVRAM** — what `SaveData` copies working memory into; what `Reset`
  reloads working memory from.

Plus a small amount of per-device state: identity (MAC/Name/Model/Firmware/
UUID), reboot deadline (zero unless mid-reboot), per-device knobs from
`DeviceConfig`.

State transitions:

- `Discover` → emit discovery response. No state change.
- `GetData(params)` → return current working memory values for params.
- `SetData(params)` → update working memory. NVRAM untouched.
- `SaveData` → copy working memory → NVRAM atomically.
- `Reset` → emit ack; set reboot deadline = `now() + RebootDelay` (default
  100ms). During the reboot window, drop all incoming packets without
  replying. After the window expires, copy NVRAM → working memory.

### Reset/reboot semantics

- Reset packet receipt → mock returns ack immediately (UDAP
  `MethodGetData`/`0x0001` to match the documented net-udap behavior the
  existing `udap` code already accepts).
- Mock enters reboot window for `RebootDelay` (default 100ms).
- During the window, all incoming packets are silently dropped (no reply).
- At end of window, working memory is reloaded from NVRAM.

Default 100ms is short enough not to slow tests, long enough to be
observable for tests that want to verify "device disappears then comes
back". A real SBR takes ~10s; tests that specifically want to exercise
long-reboot timeout paths can override per-device with `reboot=10s`.

## `cmd/mocksbr` binary

This section describes the eventual full surface of the binary (the union
of Phases 1, 2, and 3). Phase 1 implements only the `idx`, `mac`, `name`,
`model`, `firmware`, `uuid` keys of `--device`; Phase 2 adds `nvram`;
Phase 3 adds `fail-on`, `slow`, `unreachable`, `reboot`. See "Phase
decomposition" at the end of this document for the per-phase scope.

```
mocksbr [flags]
```

### Flags

| Flag | Default | Purpose |
|---|---|---|
| `--devices N` | `1` | Number of auto-generated virtual devices |
| `--device SPEC` | — | Override a specific device's config; repeatable |
| `--listen ADDR` | `0.0.0.0:17784` | UDP address to bind |
| `--verbose, -v` | off | Debug logging to stderr |
| `--help, -h` | — | Print help |
| `--version` | — | Print version and exit |

### `--device SPEC` syntax

Comma-separated `key=value` pairs. The `idx` key is required and selects
which device this overrides (1-indexed, 1..`--devices`).

```
--device idx=1,nvram=fixtures/wireless-wpa2.conf
--device idx=2,fail-on=set,slow=2s
--device idx=3,mac=aa:bb:cc:dd:ee:ff,name=TestRig,unreachable=true
--device idx=1,fail-on=set+save,reboot=10s
```

Recognized keys: `idx`, `mac`, `name`, `model`, `firmware`, `uuid`,
`nvram`, `fail-on`, `slow`, `unreachable`, `reboot`. Unknown keys → usage
error. `idx` outside `1..--devices` → usage error. Multiple `--device`
flags for the same `idx` → usage error.

`fail-on` accepts multiple ops via `+`: `fail-on=set+save+reset`.

`slow` and `reboot` accept Go duration syntax (`100ms`, `2s`).

### Behavior

1. Parse flags. Build the list of `DeviceConfig`s: auto-generate `--devices`
   defaults, then apply each `--device` override (loading INI files for
   any `nvram=FILE` keys via the shared INI parser).
2. Construct `mocksbr.Network` from the configs.
3. Bind UDP socket on `--listen`.
4. Log startup info to stderr: each device's MAC, name, mode (factory or
   pre-configured), and any active failure knobs.
5. Read loop: receive packet via `ReadFromUDP` (capture source address) →
   `Network.Receive(pkt)` → for each reply packet, `WriteToUDP` to the
   source address that sent the original request. (Unicast back, not
   broadcast — matches real-SBR behavior, which lets the client receive
   the reply on its sending socket.)
6. On SIGINT/SIGTERM: log shutdown, close the socket, exit cleanly.

### Discovery response IP

The discovery response includes an IP field that `go-udap` parses into
`device.IP` and uses for subsequent unicast traffic. The mock has a
choice in what to put there:

- **UDP mode (`cmd/mocksbr` running):** the discovery response IP is the
  IP address the source of the inbound discovery saw the mock at — i.e.,
  the destination IP of the inbound discovery packet (often `127.0.0.1`
  on loopback, or the dev box's LAN IP). This guarantees that
  `go-udap`'s subsequent unicasts (which go to `device.IP:17784`)
  actually reach the mock.

- **MockTransport mode (in-process tests):** the discovery response IP
  is the device's configured `lan_network_address` from NVRAM (or
  `0.0.0.0` for factory state). No real routing happens, so the field
  is purely informational; tests reading `device.IP` see realistic data.

Implication: a mock device pre-configured with `nvram=FILE` containing
`lan_network_address=192.168.1.50` will, in UDP mode, advertise
`127.0.0.1` (or wherever the mock is listening) — *not* `192.168.1.50`.
The configured value is still returned by `GetData(["lan_network_address"])`,
just not used for routing. Documenting this so it's not surprising; full
"pretend the mock is at an arbitrary IP" support would require IP aliasing
and is explicitly out of scope.

### Stdout/stderr split

stdout is empty in normal operation (the binary is a daemon). All
structured logs go to stderr via `udap.Logger`. Same convention as
`go-udap`.

### Exit codes

- `0` — clean shutdown (SIGINT/SIGTERM).
- `1` — usage / flag error.
- `2` — socket bind failure or runtime error.

### Examples

```bash
# Three default factory-state devices on the standard port
mocksbr --devices 3

# One device pre-configured to a wireless WPA2 setup
mocksbr --device idx=1,nvram=fixtures/wireless-wpa2.conf

# Two devices, second one fails set+save with 2s delay
mocksbr --devices 2 --device idx=2,fail-on=set+save,slow=2s

# Custom port (e.g. avoid conflict with another mocksbr instance on the
# same box) — go-udap can target it with a future --port flag if added
mocksbr --listen 127.0.0.1:27784
```

## INI loader sharing

`cli/config.go`'s `ParseINI` is needed by `mocksbr` for `--device nvram=FILE`.
Two ways to share:

- Move it to `internal/ini/` and have both `cli/` and `mocksbr/` import it.
- Have `mocksbr/` import `cli/` directly (Go allows this since they're in
  the same module; the dependency graph stays acyclic).

Recommendation: extract to `internal/ini/` for clarity. `cli/` shouldn't
own a function that `mocksbr/` consumes — they're peer concerns. The
extraction is a one-file move and re-export.

## Capture session

Pre-implementation step: capture real-SBR packets to lock in the wire
format and resolve TLV ambiguities the current parser doesn't fully
understand.

### Setup

One real SBR powered on, on the same LAN as the dev box. tcpdump or tshark
capturing UDP/17784:

```bash
sudo tcpdump -i en0 -w sbr-capture.pcap 'udp port 17784'
```

### Sequences to record

1. **Discovery — factory-reset device.** Front-button factory reset (hold
   6 seconds), then `go-udap discover`. Capture device's response. Goal:
   nail down the TLV layout — what bytes are at offsets `0x1a`, `0xad`,
   `0xb7` (the `unknown_0x*` fields the current parser only half-
   understands), what fields appear, canonical IP/Name/Model/Firmware/
   UUID encoding for an unconfigured device.

2. **Discovery — configured device.** Configure the device via `go-udap
   set + commit` to a known network state. `go-udap discover` again,
   capture, diff against (1).

3. **GetData responses.** `go-udap read <mac>`. Capture both directions.
   Goal: response TLV format for every known param; identify any params
   the current code mis-parses.

4. **SetData ack.** `go-udap set <mac> --hostname mock-test`. Capture
   response. Goal: confirm exact ack format.

5. **SaveData ack.** `go-udap save <mac>`. Capture response.

6. **Reset behavior.** `go-udap reset <mac>`. Capture the response and
   time how long the device is unresponsive (loop discover until it
   answers again). Calibrates the default `RebootDelay`.

7. **Error response.** Force an invalid set
   (`go-udap set <mac> --wireless-keylen 99` etc). Capture whether the
   device returns a `MethodError` packet, what's in it, or whether it
   silently ignores invalid input. Determines whether mock failure
   injection produces error responses or silently drops.

### Fixtures stored in repo

`mocksbr/testdata/captures/` as raw binary files, one per packet:
`discovery-factory.bin`, `discovery-configured.bin`, `getdata-response.bin`,
`setdata-ack.bin`, `savedata-ack.bin`, `reset-ack.bin`, `error-response.bin`.

Tests in `mocksbr/` compare the mock's generated bytes to these fixtures
byte-for-byte.

### Documented in this spec (post-capture)

A "UDAP packet reference" appendix listing each packet type's exact byte
layout based on the captures, including what each previously-unknown TLV
byte means. Becomes the canonical reference for both the mock and future
maintenance of the `udap` package.

The capture session happens AFTER the spec is approved but BEFORE the
implementation plans are written, so the plans can reference specific
fixture files and packet layouts. The appendix is appended to this spec
as a follow-up commit.

## Factory defaults

The mock's factory-default NVRAM (used unless `nvram=FILE` overrides) is a
single hardcoded `map[string]string` in `mocksbr/device.go`. Specific values
to be locked in during the capture session, based on what a real
factory-reset SBR reports via `go-udap read`. Placeholder until then:

```go
var factoryDefaults = map[string]string{
    "lan_ip_mode":         "1",  // DHCP
    "interface":           "0",  // Wireless
    "wireless_mode":       "0",  // Infrastructure
    // ... TBD from capture
}
```

The capture session step 1 (factory-reset device read) provides the
authoritative values. The placeholder is good enough for the implementation
plan to reference; the actual values are filled in after capture.

## Testing strategy

Three layers, each catching different failure modes.

### Layer 1 — `mocksbr/` package unit tests (pure logic, no UDP)

- Device state machine transitions: factory → SetData → GetData reflects
  change → SaveData → Reset → working memory matches NVRAM.
- Network fan-out: send a discovery broadcast packet, assert N replies
  (one per device).
- NVRAM loading from INI: `nvram=FILE` populates initial state correctly.
- Failure injection:
  - `FailOn=[OpSet]` → SetData returns UDAP error packet.
  - `Unreachable=true` → no replies for any op.
  - `Slow=100ms` → replies delayed by ~100ms.
- Reboot window: post-Reset, device drops packets for `RebootDelay`, then
  resumes.
- Packet generation: byte-for-byte compare against captured fixtures.

### Layer 2 — `udap/` integration tests using `MockTransport` (in-process, no UDP)

- `udap.NewClientWithTransport(mocksbr.NewMockTransport(net), logger)`
  end-to-end: discover → read → set → save → reset → read (verify changes
  persisted across reboot).
- Discovery: client correctly parses N device responses from a Network
  with N devices.
- Read-modify-write: SetData on a single param doesn't clobber other
  params (the udap client's internal RMW logic).
- Error handling: when mock device has `FailOn=[OpSet]`,
  `client.SetDeviceConfig(...)` returns the device's error wrapped
  properly.

### Layer 3 — `cli/` end-to-end tests (real loopback UDP)

- `mocksbr.SpawnMock(t, "--devices", "3")` returns a started subprocess
  + a Cleanup that kills + waits.
- Each `go-udap` subcommand has at least one happy-path E2E test.
- Failure injection cases: `mocksbr --device idx=1,fail-on=set` then
  `cli.Run([..."set", mac, "--hostname", "x"])` → expect stderr
  containing the device error and exit code 2.

### Test infrastructure

`mocksbr/testhelper/spawn.go` provides `SpawnMock(t, args...) *MockHandle`.
`MockHandle` exposes the bound UDP port and a `Close()` for cleanup. All
Layer 3 tests use `SpawnMock`; no test ever opens UDP/17784 on its own.
Layer 1 and 2 tests use no real network.

### Hardware verification (manual, not automated)

After Phase 1 ships, run `go-udap` against a real SBR alongside `mocksbr`
and verify the same input produces equivalent observable output (allowing
for IP/MAC differences). Manual gate before declaring the mock
production-ready, repeated whenever the captured fixtures change.

## Phase decomposition

The full design above ships as **three independent implementation plans**.
Each plan produces working, mergeable software on its own.

### Phase 1 — Core mock + Transport refactor

- `udap.Transport` interface + `UDPTransport` implementation.
- `udap.Client` refactor to use Transport.
- New constructor `NewClientWithTransport`.
- `mocksbr/` package: Network, Device, handlers, MockTransport.
- Auto-generated identities only; factory-state-only devices.
- All 8 `go-udap` subcommands work end-to-end against `mocksbr` (binary
  + via MockTransport).
- `cmd/mocksbr` binary with `--devices`, `--listen`, `--verbose`,
  `--help`, `--version`, plus `--device idx=N,mac=...` (identity overrides
  only; `nvram=`, `fail-on=`, `slow=`, `unreachable=`, `reboot=` deferred
  to Phase 2/3).
- Layer 1 + 2 + 3 tests for the above.
- Spec appendix updated with capture-session results.

### Phase 2 — Pre-configured state

- `--device idx=N,nvram=FILE` flag.
- INI parser extraction to `internal/ini/` (or kept in `cli/` if simpler).
- `DeviceConfig.NVRAM` honored in device construction.
- Fixture INI files in `mocksbr/testdata/fixtures/` for common
  configurations (DHCP wireless, static wired, etc.).
- Tests: load fixture, verify GetData returns those values; round-trip
  test (`go-udap read > x.conf; mocksbr --device nvram=x.conf` produces
  identical state).

### Phase 3 — Failure injection

- `DeviceConfig.FailOn`, `Slow`, `Unreachable`, `RebootDelay` honored.
- `--device fail-on=`, `slow=`, `unreachable=`, `reboot=` flag parsing.
- Tests: each knob individually + combinations.

## Migration impact

- Existing `udap.NewClient` and `udap.NewClientWithLogger` keep their
  signatures. Internal implementation changes (now wraps a `UDPTransport`).
- Existing `go-udap` users see no change.
- Tests of `udap` and `cli` packages can be rewritten to use `MockTransport`
  for hermetic testing; the existing `newClientWithPort(0, ...)` test
  helper from the CLI redesign remains valid for tests that still want
  real loopback.
- The `udap.PacketCaptureConfig` and `PacketCaptureResult` types are
  removed (they were internal helpers that become unnecessary). No
  external consumers.

## Open items deferred to implementation

- Exact factory-default param values (filled in after capture session).
- Whether to extract `cli.ParseINI` to `internal/ini/` or have `mocksbr`
  import `cli` directly (decided in Phase 2 implementation plan).
- Per-device-overlay struct vs flat slice for the auto-gen + override
  combination (decided in Phase 1 plan based on what's cleanest in code).
- Exact UDAP error packet format the mock returns for `FailOn` (depends on
  capture session step 7).

---

## Appendix A: UDAP packet reference (from real-SBR captures)

Captured 2026-05-08 against two Squeezebox Receivers on the same LAN.
Both report `Model="squeezebox"` via discovery; firmware version is
not advertised as an obvious string but the `0x09=07` and `0x0b=37`
TLVs likely encode the firmware revision (see TLV table below).
Source pcap: `sbr-capture.pcap` in repo working tree (untracked,
not committed). Per-packet binary fixtures at
`mocksbr/testdata/captures/`.

### Packet header (27 bytes)

The earlier `UDAPHeaderSize = 25` was wrong; the struct serialises to 27
bytes. Fixed in commit `239b11b` on `robin/cli-redesign`.

| Offset | Bytes | Field | Notes |
|---|---|---|---|
| 0 | 1 | DstBroadcast | 0x01 in client→device requests; 0x00 in responses |
| 1 | 1 | DstType | 0x01 = ETH (always seen) |
| 2 | 6 | DstAddress | target MAC; all-zeros in client broadcasts; client's MAC in responses |
| 8 | 1 | SrcBroadcast | always 0x00 |
| 9 | 1 | SrcType | 0x01 = ETH (always seen) |
| 10 | 6 | SrcAddress | sender's MAC; all-zeros in client requests; device MAC in responses |
| 16 | 2 | Sequence | client picks; device echoes |
| 18 | 2 | UDAPType | 0xC001 = UCP (always seen) |
| 20 | 1 | UCPFlags | 0x01 = request; 0x00 = response |
| 21 | 4 | UAPClass | always 0x00, 0x01, 0x00, 0x01 |
| 25 | 2 | UCPMethod | see method table below |

### UCP methods observed

| Method | Direction | Meaning |
|---|---|---|
| 0x0001 | request (broadcast) | Discover (basic) |
| 0x0002 | response | DataResp / GetIP — used as ack for **set** with status |
| 0x0003 | request | (seen as small-payload response from device after large request — may be hardware-revision/info; needs further investigation) |
| 0x0004 | request → echo response | Reset (request to device; device echoes method on ack) |
| 0x0005 | request | GetData |
| 0x0006 | request → response | SetData / SaveData (same wire method); response carries 2-byte status payload |
| 0x0007 | response | Error (not observed in this capture; device silently ignores invalid input) |
| 0x0008 | response | "ack-only". Originally believed to be GetData's normal response; later proven wrong (see GetData response section below) — real SBRs return method 0x0005 with offset/length/value triples. The 0x0008 ack was the device's silent rejection of our broken TLV-of-names request. |
| 0x0009 | request (broadcast) | AdvDisc (advanced discovery) |

### Discovery response TLVs (factory device)

61-byte packet = 27-byte header + 34-byte TLV payload. TLVs are
plain `type:1 length:1 value:length` bytes (no padding).

Fixture: `mocksbr/testdata/captures/discovery-factory.bin`

| TLV type | Length | Value | Meaning |
|---|---|---|---|
| 0x0c | 4 | `init` | **Device state** — `init` = factory state, ready for setup |
| 0x0b | 2 | `07` | Firmware revision part — major or hardware? |
| 0x0a | 4 | `0005` | Hardware/device class ID (constant across both SBRs) |
| 0x09 | 2 | `77` | Firmware revision part — minor or build? |
| 0x03 | 10 | `squeezebox` | **Model** |
| 0x02 | 0 | (empty) | **Name** — empty when device has no configured hostname |

### Discovery response TLVs (configured device, IP 192.168.1.116)

84-byte packet = 27-byte header + 57-byte TLV payload. Differences
from factory state:

Fixture: `mocksbr/testdata/captures/discovery-configured.bin`

| TLV type | Length | Value | Notes vs. factory |
|---|---|---|---|
| 0x0c | 15 | `wait_slimserver` | **State changed** — device is configured, waiting to connect to LMS |
| 0x0b | 2 | `07` | unchanged |
| 0x0a | 4 | `0005` | unchanged |
| 0x09 | 2 | `77` | unchanged |
| 0x03 | 10 | `squeezebox` | unchanged |
| 0x02 | 12 | `capture-test` | **Name now populated** with the configured hostname |

**Important behavioural insight:** the source IP of a configured
device's discovery response is its real LAN IP (e.g. `192.168.1.116`),
not `0.0.0.0`. Factory devices respond from `0.0.0.0`. The udap
client's `device.IP` is set from the UDP source IP of the discovery
response, so this works automatically — no IP TLV in the discovery
payload.

### SetData request payload format (verified)

Documented in `udap/client.go:CreateSetDataPacket` and confirmed by
captures:

```
| 16 bytes username (zeros) | 16 bytes password (zeros) |
| 2 bytes count (uint16 BE) |
| repeated count times: { 2 bytes offset BE | 2 bytes length BE | length bytes value } |
```

Offset+length identifies the param via `udap.ConfigSettings`. Value
encoding per length: 4 → IPv4 octets, 1/2 → big-endian unsigned
integer, other → string with trailing-zero padding to length.

### SetData/SaveData response (`0x0006` ack)

29-byte packet = 27-byte header + 2-byte payload.

The 2-byte payload is **the count of params accepted**, big-endian uint16.

| Status | Capture context |
|---|---|
| 0x0003 | initial set with 3 params (hostname, interface, lan_ip_mode) |
| 0x0001 | set with 1 param (`wireless_keylen=99`) |
| 0x0000 | save with empty payload (RMW fell back to no params) |

Fixtures: `setdata-status-ack.bin` (status=0x0003), `savedata-status-ack.bin` (status=0x0000).

### GetData request and response (verified against Perl Net::UDAP)

A second capture session run with the Perl Net::UDAP reference shell
(`perl_code.pcap`, `perl_shell_session.txt`) proved that real SBRs DO
return GetData responses — the earlier "0x0008 ack means no data"
finding was an artefact of `go-udap`'s wrong request format. The
correct format mirrors SetData minus the value bytes.

#### GetData request (0x0005)

Frame 6 of `perl_code.pcap`, fixture
`udap/testdata/captures/getdata-request-26params.bin` (165 bytes):

```
| 27-byte UDAP header (UCPMethod=0x0005)               |
| 16 bytes username (zeros)                            |
| 16 bytes password (zeros)                            |
| 2 bytes count (uint16 BE) — number of items requested|
| count × { 2 bytes NVRAM offset BE | 2 bytes length BE } |
```

Each (offset, length) pair identifies a parameter to read. No value
bytes — that's the only difference from a SetData request.

#### GetData response (0x0005)

Frame 7 of `perl_code.pcap`, fixture
`udap/testdata/captures/getdata-response-26params.bin` (387 bytes):

```
| 27-byte UDAP header (UCPMethod=0x0005, UCPFlags=0x00)|
| 2 bytes count (uint16 BE) — number of items returned |
| count × { 2 bytes offset | 2 bytes length | length bytes value } |
```

Same offset/length triple as a SetData REQUEST, but in the response
direction. Value encoding matches SetData: 1- and 2-byte numerics as
big-endian unsigned integers, 4-byte values as raw IPv4 octets, longer
values as NUL-padded strings.

#### What the 0x0008 ack actually meant

When `go-udap` historically sent a TLV-of-parameter-names GetData
request, the device couldn't parse it, dropped the request silently,
and the only thing left in the capture was an unrelated header-only
0x0008 ack from a different operation. The `setdata-empty-ack.bin`
fixture is now a misnomer — it's not a GetData response at all.

#### Implication for the mock

The mock implements GetData faithfully — parse the offset/length pairs
from the request, look up each (offset, length) → param-name via
`udap.ConfigSettings`, and return a 0x0005 response with offset/length/
value triples. **No deliberate divergence from real-SBR behavior is
needed.** The mock should also include `squeezecenter_name` (NVRAM
offset 83, length 33) which the Perl session reads — `udap.ConfigSettings`
gained that entry in commit `9469dac` on `robin/cli-redesign`.

### Reset request and response

Request: 27-byte header-only packet, method 0x0004.

Response: 27-byte header-only packet, method 0x0004 (echo). Fixture:
`reset-ack.bin`.

**Reboot duration: ~11 seconds** (measured from user's session).
Mock default `RebootDelay = 100ms` remains correct for fast tests;
tests wanting realistic timing can set 10s.

### Error responses — empirical finding

Step 7 of the playbook attempted to elicit an error response by
setting `wireless-keylen=99` (invalid per the udap validation table).
**The device accepted the value silently** — responding with a
normal SetData ack (`status=0x0001` = 1 param accepted). No
`MethodError (0x0007)` packet was observed.

Implication for Phase 3 failure injection: **real SBRs do NOT emit
UDAP error packets for invalid input.** Mock failure injection
(`FailOn=[OpSet]` etc.) should therefore default to **dropping
responses** (causing client timeouts) rather than emitting error
packets — that matches real-device "failure" behavior. An optional
`FailMode` knob could later allow synthetic error-packet emission
for tests that specifically want to exercise the client's
error-response code path.

### Device state machine — observed

The `0x0c` TLV value reveals state. Observed values:

- `init` — factory-reset, ready to accept configuration via UDAP.
- `wait_slimserver` — configured, looking for LMS.
- (other states presumably exist for "playing", "buffering", etc.)

An earlier draft of this section claimed that **devices in
`wait_slimserver` state stop responding meaningfully to UDAP commands
other than discovery and reset.** That claim has been retracted: the
Perl Net::UDAP session showed a configured `wait_slimserver` device
responding correctly to `get_data`, `set_data`, and `reset`. The
apparent state-dependent failures `go-udap` saw were caused by:
1. The wrong GetData wire format (TLV-of-names instead of
   offset/length).
2. The packet-capture `SourceIP="0.0.0.0"` filter rejecting responses
   from configured devices' real LAN IPs.

Both fixed on `robin/cli-redesign`. State `wait_slimserver` does not
imply a UDAP service restriction.

**Implication for the mock:** the mock's devices remain UDAP-responsive
in every state. Test scenarios that need a "no-response" path can use
Phase 3's `Unreachable=true` knob.

### Summary of plan tweaks driven by these captures

These will be folded into the Phase 1 implementation plan:

1. **`buildDiscoveryResponse` TLV byte mapping** (Phase 1 plan
   Task 11) — replace placeholder bytes with the real layout from
   this appendix:
   - 0x0c (state, default `init`)
   - 0x0b (`07`)
   - 0x0a (`0005`)
   - 0x09 (`77` — keep as a placeholder for now since we don't have
     a confirmed firmware decode)
   - 0x03 (`squeezebox`)
   - 0x02 (name, empty in factory)
2. **Discovery response: factory-state advertises `0.0.0.0`** as
   source IP via the UDP layer; configured devices advertise their
   real IP. Mock just responds from its actual binding socket — no
   IP TLV in payload.
3. **`buildDataResp` (Phase 1 plan Task 12)** — must produce the
   real-SBR offset/length/value response (UCPMethod=0x0005), not a
   TLV-of-names DataResp. Parse the incoming request's
   `(offset, length)` pairs, look each up in `udap.ConfigSettings`
   to get the param name, fetch the current value from working
   memory, and emit it back as `(offset, length, value)`. No
   divergence from real-SBR behavior.
4. **The "if `len(params) > 5` also save" heuristic** in Task 13's
   SetData handler can be DELETED. SetData and SaveData are the
   same wire method 0x0006; the mock can't tell them apart from
   the wire format. The mock should treat every 0x0006 as
   "set into working memory" and rely on the explicit Save path
   for NVRAM updates. (For tests that assert NVRAM behavior, the
   mock can't see "Save vs Set intent" — but tests can verify by
   sending Reset and reading back.)
5. **SetData/SaveData ack: 2-byte payload = uint16 BE param count.**
   The mock's `buildAck` for these methods should include the
   accepted-param-count payload, not a header-only ack.
6. **Reset ack: header-only packet, method 0x0004 echo.**
   Mock's `buildAck` for Reset should NOT use 0x0008
   (SetDataAck — what the existing client code accepts) but the
   echo of 0x0004. The existing `udap` client also accepts 0x0001
   (per net-udap notes in the codebase); the mock can pick either.
7. **Failure injection (Phase 3) should default to "no response"**
   rather than synthetic error packets, matching observed real-SBR
   behavior on invalid input.

### Pcap-not-committed reminder

The source `sbr-capture.pcap` is left untracked at the repo root.
Add to `.gitignore` separately if convenient. Per-packet `.bin`
fixtures in `mocksbr/testdata/captures/` ARE committed so tests
can reference them.
