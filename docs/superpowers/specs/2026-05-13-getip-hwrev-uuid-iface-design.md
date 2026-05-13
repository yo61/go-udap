# get_ip, hardware-rev/UUID surfacing, and interface selection

**Date:** 2026-05-13
**Status:** Draft — awaiting implementation plan

## Summary

Three additive feature deltas to go-udap, batched into a single design because
they share enough scope (CLI surface, output formatting, the `Device`
aggregate) to be worth specifying together:

1. **`get_ip` operation** (UCP method `0x0002`). Library method, new
   `getip <mac>` subcommand, and integration into `discover --info` so
   subnet/gateway appear alongside the existing discovery-TLV fields.
2. **Hardware revision and UUID surfacing.** Both arrive in the discovery
   reply today and are dropped on the floor; this surfaces them on `Device`
   and in CLI output.
3. **Interface selection.** New global flags `--interface NAME` and
   `--all-interfaces`, plus a new `interfaces` subcommand to enumerate usable
   interfaces. Default behaviour unchanged.

Every new behaviour is opt-in. `go-udap discover` with no flags behaves
identically to the current code.

## Motivation

- `get_ip` lets the user confirm the device's view of its own network state
  (IP, subnet mask, gateway), distinct from the UDP source IP that broadcast
  discovery happens to expose. Useful when validating a static-IP config the
  user just wrote with `set`.
- `HardwareRev` and `UUID` are useful metadata for distinguishing devices and
  reporting bugs against specific firmware/hardware combinations. The
  protocol already sends them; `udap/discovery.go` parses `HardwareRev` but
  has a "recorded for future use" comment and never surfaces it, and
  `0x0d UCP_CODE_UUID` isn't recognised at all.
- Interface selection matters on multi-homed hosts. The current
  `0.0.0.0`-bound socket sending to `255.255.255.255` is routed by the kernel
  via the default route's interface only, so devices on a secondary subnet
  are invisible to `discover`.

## References

- **Net-UDAP** (Perl reference implementation,
  `github.com/robinbowes/net-udap`):
  - `src/Net-UDAP/lib/Net/UDAP/Constant.pm` — `UCP_METHOD_GET_IP = 0x0002`,
    `UCP_CODE_IP_ADDR = 0x05`, `UCP_CODE_SUBNET_MASK = 0x06`,
    `UCP_CODE_GATEWAY_ADDR = 0x07`, `UCP_CODE_HARDWARE_REV = 0x0a`,
    `UCP_CODE_UUID = 0x0d`.
  - `src/Net-UDAP/lib/Net/UDAP/MessageOut.pm:111` — `get_ip` request has
    no payload ("nothing further to do for get_ip").
  - `src/Net-UDAP/lib/Net/UDAP/MessageIn.pm:199` — `get_ip` reply parses
    with the same TLV loop as advanced discovery.
- **Existing go-udap code:**
  - `udap/config.go:24-48` — `waitForDeviceReply` (reused for `get_ip`).
  - `udap/discovery.go:55-65` — discovery TLV codes table (`HardwareRev` and
    `UUID` added here).
  - `udap/transport.go:44-67` — existing `UDPTransport` (unchanged).

## Design lens

The whole batch is shaped to preserve one invariant: the `Transport` port
stays a single-concept abstraction. Every infrastructure change is a new
adapter (`UDPTransport`, `MultiTransport`, `MockTransport` — siblings, not
modes). The `Client` doesn't learn anything new about interfaces; it just
receives a differently-constructed `Transport`.

The same lens drives keeping `NetworkConfig` (the `get_ip` result) as a
separate value object rather than folding it into the `Device` aggregate:
`Device` carries everything observed at discovery time; `NetworkConfig` is
actively queried later. Different freshness, different invariants, different
types.

## Architecture

### Layering (unchanged)

| Layer | Members | Role |
|--|--|--|
| Domain | `Client`, `Device`, `Packet`, `Parameter`, `MAC`, `NetworkConfig` (new), `NetInterface` (new) | Model |
| Infrastructure | `UDPTransport`, `MultiTransport` (new), `MockTransport`, sockets, loggers | Adapters |
| Application | `cli` package | Use-case orchestration |

### New types

- **`NetworkConfig`** — value object. Fields: `IP net.IP`, `SubnetMask net.IP`,
  `Gateway net.IP`. Immutable, equality by attributes. Has `String()` and
  JSON tags. Returned by the new `get_ip` library call.
- **`NetInterface`** — value object. Fields: `Name string`, `Index int`,
  `Addr net.IP`, `Broadcast net.IP`. Context-local representation of a
  usable interface, translating from `net.Interface` (a tiny anti-corruption
  layer keeping Go-stdlib terminology out of the udap domain).
- **`MultiTransport`** — new `Transport` implementation. Composes
  `[]Transport`. `Send` fans out across all children; `Recv` merges replies
  via per-child goroutines and a shared channel.

### New constructors

- `NewClient()` — unchanged. Binds `0.0.0.0:17784`, single socket, OS default
  routing.
- `NewClientForInterface(name string, logger Logger) (*Client, error)` —
  builds a `UDPTransport` bound to the named interface's IP.
- `NewClientForAllInterfaces(logger Logger) (*Client, error)` — enumerates
  usable interfaces, builds N `UDPTransport`s (one per interface), wraps in
  `MultiTransport`.

The CLI picks the constructor based on the global flags before any other
operation runs.

### Wire format additions

- **`MethodGetIP` request:** 27-byte header only. `UCPMethod = 0x0002`,
  `DstBroadcast = 0`, destination MAC = target device's MAC. No payload.
- **`MethodGetIP` reply:** 27-byte header + TLV stream. TLV format identical
  to discovery: 1-byte code, 1-byte length, value bytes. Relevant codes:
  `0x05 IP_ADDR` (4 bytes), `0x06 SUBNET_MASK` (4 bytes),
  `0x07 GATEWAY_ADDR` (4 bytes). Devices may omit fields.
- **Discovery TLVs added to recognition:** `0x0a HARDWARE_REV`
  (already received, now surfaced as an opaque string — same shape as
  `Firmware`; real captures and `mocksbr` use 4-ASCII-byte forms like
  `"0005"`, and Net-UDAP doesn't fix a numeric wire format) and
  `0x0d UUID` (newly recognised, surfaced as lowercased hex string of
  the raw bytes).

## Components

### `udap/` (library)

**New files**

- `udap/netconfig.go` — `NetworkConfig` value object, `String()`, JSON tags.
- `udap/interfaces.go` — `NetInterface` value object;
  `EnumerateInterfaces() ([]NetInterface, error)`. Filter:
  `FlagUp && FlagBroadcast && !FlagLoopback` and ≥1 IPv4 address. Computes
  directed-broadcast per interface as `addr | ^mask`.
- `udap/multi_transport.go` — `MultiTransport` implementing `Transport`.
- `udap/getip.go` — `Client.CreateGetIPPacket(*Device) ([]byte, error)`
  and `Client.GetDeviceNetworkConfigWithContext(ctx, *Device) (NetworkConfig, error)`.
  Decoder for TLV codes `0x05`/`0x06`/`0x07`.

**Modified files**

- `udap/transport.go` — add
  `NewUDPTransportOnInterface(iface NetInterface, port int, logger Logger) (*UDPTransport, error)`
  that binds to the interface's IP and sends to its directed broadcast
  (e.g. `192.168.1.255`) instead of `255.255.255.255`. Existing
  `NewUDPTransport` unchanged.
- `udap/client.go` — add `NewClientForInterface` and
  `NewClientForAllInterfaces` constructors. `NewClient` unchanged.
- `udap/protocol.go` — add `Device.HardwareRev string` and
  `Device.UUID string` (JSON tags, both `,omitempty`). Add
  `tlvUUID = 0x0d` constant.
- `udap/discovery.go` — wire `tlvHardwareRev` (already received) and
  `tlvUUID` (new) into `Device`. HW rev kept as the raw TLV bytes
  interpreted as UTF-8 (same shape as `Firmware`); UUID encoded as
  lowercase hex string of the raw bytes.

### `cli/` (CLI)

**New files**

- `cli/getip.go` — `getip <mac>` subcommand. Discovers, looks up by MAC,
  calls library, renders `NetworkConfig`.
- `cli/interfaces.go` — `interfaces` subcommand. Calls
  `EnumerateInterfaces`, renders as a table (name, index, address,
  directed broadcast).

**Modified files**

- `cli/cli.go` — register `getip` and `interfaces` subcommands; add
  `--interface NAME` and `--all-interfaces` to the global flag set,
  validate mutual exclusion at parse, route to the right client
  constructor.
- `cli/discover.go` — when `--info` is set, after broadcast discovery
  completes, fire `GetDeviceNetworkConfigWithContext` per device. On
  per-device failure, log and continue with blank network fields.
- `cli/info.go` — include `HardwareRev` and `UUID` in output.
- `cli/output.go` — render `NetworkConfig` rows; render `NetInterface`
  table.

### `cmd/mocksbr/` (test mock)

- Add a handler for `MethodGetIP` (`0x0002`): respond with TLV-encoded
  IP/subnet/gateway. Defaults configurable per mock device.
- Add config knobs: drop-`get_ip`-requests mode (for timeout tests) and
  reply-with-`MethodError` mode (for error-method tests).

### What stays untouched

- `udap/parameters.go`, `udap/getdata_response.go`, `udap/loopback.go`,
  `udap/mac.go`, `udap/socket*.go`, `udap/logger.go`.
- Existing tests covering `UDPTransport` single-conn path and `Client` core
  operations.

## Data flow

### Flow A — `go-udap getip <mac>`

1. CLI parses args and global flags; selects client constructor.
2. `client.DiscoverDevicesWithContext(ctx)` populates the device map.
3. `client.GetDevice(mac)` looks up the requested device.
4. `client.GetDeviceNetworkConfigWithContext(ctx, device)`:
   1. `CreateGetIPPacket(device)` builds the 27-byte header-only request.
   2. `transport.Send(packet)`.
   3. `waitForDeviceReply(ctx, device)` reuses the existing helper
      (anti-spoof source-IP check + MAC matching).
   4. On `MethodGetIP` reply, parse TLVs `0x05`/`0x06`/`0x07`, return
      `NetworkConfig`.
5. CLI renders to stdout.

### Flow B — `go-udap discover --info`

1. `client.DiscoverDevicesWithContext(ctx)` collects N devices.
2. For each device: `client.GetDeviceNetworkConfigWithContext(ctx, d)`.
3. On per-device error: log `Warn`, continue, render network fields as `-`.
4. CLI prints aggregated table with subnet/gateway columns.

Per-device `get_ip` is sequential by default — bounded by the shared
parent `ctx`, so the whole operation is gated by the global `--timeout`.
No per-device timeout multiplication.

### Flow C — `--all-interfaces` discovery fan-out

1. CLI sees `--all-interfaces`; calls `NewClientForAllInterfaces`.
2. `EnumerateInterfaces()` returns usable interfaces.
3. For each: `NewUDPTransportOnInterface(iface, Port, logger)`. On
   per-child bind failure, log `Warn` and skip.
4. If zero children: constructor returns error.
5. `MultiTransport` wraps the children.
6. `client.DiscoverDevicesWithContext`:
   - `transport.Send(packet)` → each child sends to its directed-broadcast
     address (e.g. `192.168.1.255`).
   - `transport.Recv(ctx)` merges replies via per-child goroutines into a
     shared channel.
   - `Client.recordDevice` keys on MAC; same device replying via two
     interfaces is deduped (last writer wins on `IP`).

## Error handling

### `GetDeviceNetworkConfigWithContext`

| Condition | Behaviour |
|--|--|
| `device.MAC` is zero | Error from `CreateGetIPPacket`; never sent |
| `transport.Send` fails | Wrapped error |
| `ctx` cancelled / deadline exceeded | Returns `ctx.Err()`; CLI maps to exit-code 2 |
| Reply method `MethodGetIP` (`0x0002`) | Parse TLVs, return `NetworkConfig` |
| Reply method `MethodError` (`0x0007`) | Parse TLV `0x03`, return as error |
| Reply method `MethodCredentialsError` (`0x0008`) | Return "rejected credentials" error |
| Reply method anything else | Return "unexpected response method 0x%04x" error |
| Required TLVs missing | Soft-fail: return what was present (zero values for missing fields), log `Warn` |
| TLV length doesn't match expected | Skip that TLV with `Warn`; same rationale |

### `discover --info` per-device failure

Already covered in Flow B: log `Warn`, continue, render network fields as
`-`. Exit code 0 (discovery succeeded).

### `--interface NAME`

| Condition | Behaviour |
|--|--|
| `--interface` and `--all-interfaces` both set | Flag-parse error, exit 1 |
| Named interface doesn't exist | Constructor error, exit 1 |
| Named interface has no IPv4 address | Constructor error, exit 1 |
| Named interface is down or non-broadcast | Constructor error, exit 1 |
| Bind fails (perms, in-use) | Constructor error, exit 2 |

### `--all-interfaces`

| Condition | Behaviour |
|--|--|
| Zero usable interfaces | Constructor error, exit 1 |
| Some children fail to bind | Warning per child, continue |
| All children fail to bind | Constructor error, exit 2 |

### `MultiTransport` runtime

- `Send`: collect per-child errors. Success if **any** child succeeded;
  aggregated error only if all failed. Per-child failures logged at `Warn`.
- `Recv`: per-child goroutine exits cleanly on `ctx` cancellation, shared
  done-channel close from `Close`, or non-timeout error from the child
  (logged, goroutine exits, others continue).
- `Close`: closes all children, signals done-channel, waits for goroutines
  to drain with a short bounded timeout so a stuck child can't hang.

### `interfaces` subcommand

| Condition | Behaviour |
|--|--|
| Zero usable interfaces | Exit 0 with message ("no usable interfaces found") |
| `net.Interfaces()` syscall fails | Exit 2, error wrapped |

### Soft-fail rationale for partial `get_ip` TLVs

Real firmware variants are known to drop TLVs (e.g. no `0x07` when no
gateway is configured). A device that answers `0x0002` at all is healthy;
treating partial TLVs as success-with-blank-field matches the same
forgiveness `parseDiscoveryResponse` already applies. Logging at `Warn`
keeps observability without breaking scripts.

## Testing

### Unit tests (pure, no network)

- `udap/getip_test.go`:
  - `TestCreateGetIPPacket` — table-driven wire-format check.
  - `TestCreateGetIPPacketZeroMAC` — error path.
  - `TestParseGetIPResponse` — TLV variants, partial cases, malformed
    lengths, empty payload. Never panics.
- `udap/discovery_test.go` (extend): TLV stream containing
  `0x0a UCP_CODE_HARDWARE_REV` and `0x0d UCP_CODE_UUID`. Verify
  `Device.HardwareRev` (string passthrough) and `Device.UUID`
  (lowercase hex of raw bytes). Missing-TLV cases produce empty
  strings.
- `udap/netconfig_test.go`: `String()` rendering, JSON roundtrip,
  equality.
- `udap/interfaces_test.go`:
  - `TestNetInterfaceBroadcast` — table-driven `addr | ^mask` over /24,
    /16, /22, /30.
  - `TestEnumerateInterfaces` — asserts every returned `NetInterface`
    satisfies the filter; skips on hosts with zero usable interfaces.
- `udap/multi_transport_test.go`:
  - Send fans out to all children.
  - Recv merges from all children (non-deterministic order).
  - Send partial failure: one child errors, one succeeds → success.
  - Send all-fail: aggregated error.
  - Close propagates and unblocks Recv.
  - Context cancellation mid-recv unblocks immediately.
- `cli/getip_test.go`, `cli/interfaces_test.go`: output formatting /
  golden-file style, matching existing `cli/output_test.go` patterns.

### End-to-end tests via mocksbr

New `cli/e2e_*_test.go` cases:

- `e2e_getip_test.go` — `getip <mac>` against mocksbr that answers
  `0x0002`. Verify three-line output, exit 0.
- `e2e_getip_timeout_test.go` — mocksbr drops `0x0002`. Exit 2,
  device named on stderr.
- `e2e_getip_error_method_test.go` — mocksbr replies `MethodError`
  with TLV `0x03` message. Verify message surfaces verbatim.
- `e2e_discover_info_with_getip_test.go` — both discovery and
  `get_ip` answered. Verify subnet/gateway columns appear.
- `e2e_discover_info_partial_test.go` — two mocksbrs; one answers,
  one drops. Both devices appear; dropped one shows dashes. Exit 0.
- `e2e_interfaces_test.go` — `interfaces` subcommand sanity check
  on the host (skip if zero).

### mocksbr additions

- Handler for `MethodGetIP` (`0x0002`); per-device IP/subnet/gateway
  defaults.
- Drop-`get_ip`-requests mode.
- Reply-with-`MethodError` mode.

### `--interface` / `--all-interfaces` testing

- Library-level: `NewClientForInterface("nonexistent")` returns a clean
  error before any I/O. Success case uses a real interface discovered
  via `EnumerateInterfaces` at test-time; skips on hosts with none.
- CLI-level: flag-parse rules (mutual exclusion, unknown-interface
  error) without touching the network.
- No automated "fan-out actually emits packets on multiple interfaces"
  test — covered by manual spike below.

### Manual broadcast verification spike (prerequisite)

Before implementation, verify the current claim that
`0.0.0.0` + `255.255.255.255` goes out the default-route interface only.

1. On a multi-homed host, run `sudo tcpdump -i any 'udp port 17784'`.
2. Run current `go-udap discover` in another terminal.
3. Observe which interface(s) the outbound broadcast leaves on.

Expected: only the default-route interface. If observed, design stands.
If broadcasts already fan out, `--all-interfaces` becomes redundant with
the default and scope should be pruned.

Repeat post-implementation with `--all-interfaces` to confirm
per-interface fan-out. Result documented in the implementation plan;
no checked-in artifact.

### Out of scope for testing

- Real-device wire-format verification (mocksbr is the contract).
- Cross-platform interface enumeration differences (Linux/macOS/Windows).
  Filter is on standard `net.Flags`; directed-broadcast is pure
  arithmetic. Windows behaviour validated manually if/when a Windows
  build is needed.

## Backward compatibility

- All new operations are purely additive.
- `NewClient()` keeps its behaviour (binds `0.0.0.0`, single socket, OS
  default routing). No existing tests break.
- New global flags default to off; `discover` without `--info` is
  unchanged.
- New `Device` fields (`HardwareRev`, `UUID`) default to zero/empty when
  the TLVs are absent — same forgiveness pattern existing fields use.
- JSON output of `Device` gains two fields. Consumers that strict-decode
  the JSON will need to accept them; tolerant decoders (the default in
  Go's `encoding/json`) are unaffected.

## Open questions / risks

- **Real-device TLV coverage.** We don't have wire captures of `get_ip`
  replies from every Squeezebox model. The decoder is permissive
  (missing fields → blanks) to absorb variation. If a model surfaces
  unusual codes (e.g. an `0x04 USE_DHCP` flag), they'll be silently
  ignored; we can extend later.
- **Windows interface enumeration.** Not exercised in CI. The standard
  `net.Interfaces()` covers it, but flag semantics may differ. Test on
  the first Windows build attempt.
- **`--all-interfaces` on hosts with VPN/Docker/bridge interfaces.**
  Fan-out will hit all broadcast-capable interfaces. This is the
  intended behaviour — user-opt-in flag, user owns the host topology —
  but worth noting in the `--help` text.
- **Discovery over Tailscale / WireGuard not supported by this design.**
  WireGuard is a routed L3 VPN: the tunnel interface is point-to-point
  (`FlagPointToPoint`, no `FlagBroadcast`) and WireGuard silently drops
  packets destined for `255.255.255.255` because there's no peer
  associated with the broadcast address. So the filter chosen here
  (`FlagUp && FlagBroadcast` + IPv4) excludes Tailscale interfaces
  automatically, and that's the correct behaviour — including them
  would only contribute timeout latency for a fan-out branch that
  never reaches anything. If discovery across a Tailscale Subnet
  Router is wanted later, the right mechanism is a directed broadcast
  to the remote LAN's broadcast address (e.g. `192.168.1.255`) routed
  unicast through the tunnel, which is a separate feature with a
  different CLI surface (something like `--broadcast-address IP`) and
  no `MultiTransport` involvement.

## Implementation order

The implementation plan should sequence as follows (subject to the
writing-plans skill refining it):

1. **Spike first.** Run the manual `tcpdump` verification described in
   the testing section. Confirm or refute the current-behaviour
   assumption. If refuted, return to spec.
2. **HW rev / UUID surfacing.** Smallest, lowest-risk change. Touches
   `protocol.go`, `discovery.go`, output rendering, one test extension.
3. **`get_ip` library + subcommand.** Self-contained. New file, new
   subcommand, mocksbr handler, e2e tests.
4. **`discover --info` integration.** Depends on (3).
5. **`NetInterface` value object + `EnumerateInterfaces` +
   `interfaces` subcommand.** Self-contained.
6. **`UDPTransport` per-interface binding +
   `NewClientForInterface`.** Depends on (5).
7. **`MultiTransport` + `NewClientForAllInterfaces` + flag wiring.**
   Depends on (6).

Each step is independently reviewable and testable. Steps (2)–(7) can
land as separate commits or PRs.
