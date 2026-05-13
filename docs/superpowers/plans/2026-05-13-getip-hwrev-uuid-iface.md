# get_ip, hwrev/uuid surfacing, interface selection — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add three additive features to go-udap — active `get_ip` (UCP 0x0002) query, HardwareRev/UUID surfacing from discovery TLVs, and per-interface broadcast control (`--interface NAME`, `--all-interfaces`, `interfaces` subcommand).

**Architecture:** Each feature is purely additive — `NewClient()` and `discover` without flags retain current behaviour. New value objects `NetworkConfig` and `NetInterface` live in the `udap` domain. Fan-out is implemented as a new `MultiTransport` adapter satisfying the existing `Transport` port; the `Client` is agnostic to single vs. multi-socket transports. Per-device `get_ip` failures during `discover --info` are soft-failed (`-` in output).

**Tech Stack:** Go 1.x, `spf13/pflag`, in-process `mocksbr` for e2e, prek pre-commit hooks. All work on branch `feat/getip-hwrev-uuid-interface-selection`.

**Spec:** `docs/superpowers/specs/2026-05-13-getip-hwrev-uuid-iface-design.md`.

---

## File structure

### New files in `udap/`

| Path | Responsibility |
|--|--|
| `udap/netconfig.go` | `NetworkConfig` value object (IP/SubnetMask/Gateway) — result of `get_ip` |
| `udap/netconfig_test.go` | Tests for `NetworkConfig.String()`, JSON roundtrip, equality |
| `udap/getip.go` | `CreateGetIPPacket`, `GetDeviceNetworkConfigWithContext`, `parseGetIPResponse` |
| `udap/getip_test.go` | Wire-format + TLV parse tests for `get_ip` |
| `udap/interfaces.go` | `NetInterface` value object + `EnumerateInterfaces()` |
| `udap/interfaces_test.go` | Directed-broadcast math + enumeration smoke test |
| `udap/multi_transport.go` | `MultiTransport` composing N `Transport`s |
| `udap/multi_transport_test.go` | Fan-out send, merged recv, partial/all failure, close, cancel |

### Modified files in `udap/`

| Path | Change |
|--|--|
| `udap/protocol.go` | Add `Device.HardwareRev string`, `Device.UUID string`, `tlvUUID = 0x0d` |
| `udap/discovery.go` | Parse TLV `0x0a` (HW rev as string passthrough) and `0x0d` (UUID as hex) |
| `udap/discovery_test.go` | New cases for HW rev / UUID in discovery TLVs |
| `udap/transport.go` | Add `NewUDPTransportOnInterface` constructor |
| `udap/client.go` | Add `NewClientForInterface`, `NewClientForAllInterfaces` constructors |

### New files in `cli/`

| Path | Responsibility |
|--|--|
| `cli/getip.go` | `getip <mac>` subcommand |
| `cli/interfaces.go` | `interfaces` subcommand |
| `cli/e2e_getip_test.go` | E2E happy/timeout/error-method for `getip` |
| `cli/e2e_discover_info_getip_test.go` | E2E for `discover --info` integration |
| `cli/e2e_interfaces_test.go` | E2E smoke for `interfaces` subcommand |
| `cli/e2e_interface_flag_test.go` | E2E for `--interface` / `--all-interfaces` flag handling |

### Modified files in `cli/`

| Path | Change |
|--|--|
| `cli/cli.go` | Register `getip` and `interfaces` subcommands; add `--interface` / `--all-interfaces` global flags |
| `cli/discover.go` | `--info` fires per-device `get_ip`, renders subnet/gateway |
| `cli/output.go` | Render `NetworkConfig`, render `NetInterface` table, render HW rev / UUID in `formatDeviceInfo` |

### Modified files in `mocksbr/`

| Path | Change |
|--|--|
| `mocksbr/responses.go` | Emit TLV `0x0d UUID` in discovery; add `buildGetIPResponse` |
| `mocksbr/responses.go` | New `getDataItem` parsing not needed for `get_ip` (no payload to parse) |
| `mocksbr/handlers.go` | Dispatch `MethodGetIP` (0x0002) |
| `mocksbr/device.go` | Add `NetworkConfig` fields to `DeviceConfig` (IP, SubnetMask, Gateway with defaults) |

---

## Phase 1 — Manual broadcast spike (no code)

### Task 1.1: Verify current broadcast behaviour

**Files:** none (manual)

- [ ] **Step 1: Identify a multi-homed host** — either your dev Mac with WiFi + Ethernet both up, or a Linux box with two NICs. If single-homed, add a temporary loopback alias: `sudo ifconfig lo0 alias 127.0.0.2/8` (macOS) or `sudo ip addr add 127.0.0.2/8 dev lo` (Linux). Note the names of all up interfaces (`ifconfig -a` or `ip addr`).

- [ ] **Step 2: Build the current binary on this branch**

```bash
task build
```

Expected: produces `./go-udap` in repo root.

- [ ] **Step 3: Capture in one terminal**

```bash
sudo tcpdump -i any -n -nn 'udp port 17784'
```

Expected: tcpdump runs, waiting for packets.

- [ ] **Step 4: Trigger discovery in another terminal**

```bash
./go-udap discover
```

- [ ] **Step 5: Observe tcpdump output**

Note the interface names that show outbound `255.255.255.255.17784` traffic. Record the observation:

- Multi-interface host: expected to see broadcast leaving on **only one** interface (the default-route's). If observed → spec stands; proceed to Phase 2.
- If broadcast leaves on **all** interfaces → spec premise refuted. Stop. Report back; we'd prune Phase 7 from scope.

Write the observation into a one-paragraph note appended at the bottom of `docs/superpowers/plans/2026-05-13-getip-hwrev-uuid-iface.md` (this file) under a `## Spike result` heading.

- [ ] **Step 6: Commit the spike note**

```bash
git add docs/superpowers/plans/2026-05-13-getip-hwrev-uuid-iface.md
git commit -m "docs(plan): record broadcast-behaviour spike result"
```

---

## Phase 2 — HardwareRev + UUID surfacing

Smallest, lowest-risk feature. Adds two string fields to `Device`, parses two TLVs (one already received but discarded), and surfaces in CLI output.

### Task 2.1: Add Device fields and tlvUUID constant

**Files:**
- Modify: `udap/protocol.go`

- [ ] **Step 1: Add fields to Device struct**

In `udap/protocol.go`, find the `type Device struct` block (around line 103). Add two new fields after `State`:

```go
type Device struct {
	MAC         MAC               `json:"mac"`
	IP          string            `json:"ip"`
	Name        string            `json:"name"`
	Model       string            `json:"model"`
	Firmware    string            `json:"firmware"`
	HardwareRev string            `json:"hardware_rev,omitempty"`
	UUID        string            `json:"uuid,omitempty"`
	State       string            `json:"state,omitempty"`
	LastSeen    time.Time         `json:"last_seen"`
	Parameters  map[string]string `json:"parameters"`
}
```

- [ ] **Step 2: Build to verify no break**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add udap/protocol.go
git commit -m "feat(udap): add Device.HardwareRev and Device.UUID fields"
```

### Task 2.2: Wire HardwareRev TLV parsing

**Files:**
- Modify: `udap/discovery.go`
- Test: `udap/discovery_test.go`

- [ ] **Step 1: Write the failing test**

Append to `udap/discovery_test.go` (create the file if it doesn't exist; check for existing tests first with `grep -l "func Test" udap/discovery_test.go`):

```go
func TestParseDiscoveryResponsePopulatesHardwareRev(t *testing.T) {
	c := mustNewClient(t)
	defer c.Close()

	// TLV stream containing only hardware_rev = "0005"
	data := []byte{
		0x0a, 0x04, '0', '0', '0', '5', // hw rev
	}
	pkt := &Packet{
		SrcType:    AddrTypeETH,
		SrcAddress: [6]byte{0x00, 0x04, 0x20, 0x00, 0x00, 0x01},
	}
	device := c.parseDiscoveryResponse(data, "192.168.1.50", pkt)
	if device == nil {
		t.Fatal("parseDiscoveryResponse returned nil")
	}
	if device.HardwareRev != "0005" {
		t.Errorf("HardwareRev = %q, want %q", device.HardwareRev, "0005")
	}
}
```

If `mustNewClient` doesn't exist in the test file, also add this helper at the top:

```go
func mustNewClient(t *testing.T) *Client {
	t.Helper()
	c, err := NewClientWithLogger(NewNoOpLogger())
	if err != nil {
		t.Fatalf("NewClientWithLogger: %v", err)
	}
	return c
}
```

(Confirm first whether `mustNewClient` is already defined in `udap/`: `grep -rn "func mustNewClient" udap/` — if it exists, don't re-declare.)

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run TestParseDiscoveryResponsePopulatesHardwareRev ./udap/ -v
```

Expected: FAIL — `HardwareRev = "", want "0005"`.

- [ ] **Step 3: Wire HW rev in discovery.go**

In `udap/discovery.go`, find the `parseDiscoveryResponse` function's TLV switch (around line 122). Add a `case tlvHardwareRev:` arm. Replace the existing `case tlvHardwareRev:` comment-only handling:

Before (around line 133-134):
```go
		case tlvHardwareRev:
			// Not surfaced today; recorded for future use.
```

After:
```go
		case tlvHardwareRev:
			device.HardwareRev = string(value)
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run TestParseDiscoveryResponsePopulatesHardwareRev ./udap/ -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add udap/discovery.go udap/discovery_test.go
git commit -m "feat(udap): surface HardwareRev from discovery TLV 0x0a"
```

### Task 2.3: Add tlvUUID constant and parse UUID

**Files:**
- Modify: `udap/discovery.go`
- Test: `udap/discovery_test.go`

- [ ] **Step 1: Write the failing test**

Append to `udap/discovery_test.go`:

```go
func TestParseDiscoveryResponsePopulatesUUID(t *testing.T) {
	c := mustNewClient(t)
	defer c.Close()

	// TLV stream containing only uuid = 16 raw bytes 0x00..0x0f
	data := []byte{
		0x0d, 0x10, // tlvUUID, length 16
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
	}
	pkt := &Packet{
		SrcType:    AddrTypeETH,
		SrcAddress: [6]byte{0x00, 0x04, 0x20, 0x00, 0x00, 0x01},
	}
	device := c.parseDiscoveryResponse(data, "192.168.1.50", pkt)
	if device == nil {
		t.Fatal("parseDiscoveryResponse returned nil")
	}
	want := "000102030405060708090a0b0c0d0e0f"
	if device.UUID != want {
		t.Errorf("UUID = %q, want %q", device.UUID, want)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run TestParseDiscoveryResponsePopulatesUUID ./udap/ -v
```

Expected: FAIL — `UUID = "", want "000102..."`.

- [ ] **Step 3: Add tlvUUID constant**

In `udap/discovery.go`, find the const block at lines 58-65:

```go
const (
	tlvDeviceName   = 0x02
	tlvDeviceType   = 0x03
	tlvFirmwareRev  = 0x09
	tlvHardwareRev  = 0x0a
	tlvDeviceID     = 0x0b
	tlvDeviceStatus = 0x0c
)
```

Add `tlvUUID = 0x0d`:

```go
const (
	tlvDeviceName   = 0x02
	tlvDeviceType   = 0x03
	tlvFirmwareRev  = 0x09
	tlvHardwareRev  = 0x0a
	tlvDeviceID     = 0x0b
	tlvDeviceStatus = 0x0c
	tlvUUID         = 0x0d
)
```

- [ ] **Step 4: Add UUID case to switch and import encoding/hex**

In `udap/discovery.go` imports, add `"encoding/hex"`:

```go
import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)
```

(Use `go imports` after editing — or let the prek hook fix it.)

In the TLV switch, add an arm after `case tlvHardwareRev`:

```go
		case tlvUUID:
			device.UUID = hex.EncodeToString(value)
```

- [ ] **Step 5: Run test to verify it passes**

```bash
go test -run TestParseDiscoveryResponsePopulatesUUID ./udap/ -v
```

Expected: PASS.

- [ ] **Step 6: Run full udap tests to confirm nothing else broke**

```bash
go test ./udap/...
```

Expected: PASS (all tests).

- [ ] **Step 7: Commit**

```bash
git add udap/discovery.go udap/discovery_test.go
git commit -m "feat(udap): recognise and surface UUID from discovery TLV 0x0d"
```

### Task 2.4: Surface HW rev + UUID in CLI output

**Files:**
- Modify: `cli/output.go`
- Test: `cli/output_test.go`

- [ ] **Step 1: Write the failing test**

Check what exists in `cli/output_test.go`:

```bash
grep -n "^func Test" cli/output_test.go
```

Append a new test:

```go
func TestFormatDeviceInfoIncludesHardwareRevAndUUID(t *testing.T) {
	mac, err := udap.ParseMAC("00:04:20:00:00:01")
	if err != nil {
		t.Fatalf("ParseMAC: %v", err)
	}
	d := &udap.Device{
		MAC:         mac,
		IP:          "192.168.1.50",
		Name:        "test",
		HardwareRev: "0005",
		UUID:        "000102030405060708090a0b0c0d0e0f",
	}
	var buf bytes.Buffer
	formatDeviceInfo(&buf, d)
	out := buf.String()
	for _, want := range []string{"0005", "000102030405060708090a0b0c0d0e0f", "HW Rev", "UUID"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q; got:\n%s", want, out)
		}
	}
}
```

If `bytes` or `strings` aren't already imported in the test file, add them.

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run TestFormatDeviceInfoIncludesHardwareRevAndUUID ./cli/ -v
```

Expected: FAIL — output missing fields.

- [ ] **Step 3: Update formatDeviceInfo**

In `cli/output.go`, modify `formatDeviceInfo` to emit the new fields. Insert after the `Firmware` block (around line 56):

```go
func formatDeviceInfo(w io.Writer, d *udap.Device) {
	fmt.Fprintf(w, "MAC:      %s\n", d.MAC)
	fmt.Fprintf(w, "IP:       %s\n", d.IP)
	if d.Name != "" {
		fmt.Fprintf(w, "Name:     %s\n", d.Name)
	}
	if d.Model != "" {
		fmt.Fprintf(w, "Model:    %s\n", d.Model)
	}
	if d.Firmware != "" {
		fmt.Fprintf(w, "Firmware: %s\n", d.Firmware)
	}
	if d.HardwareRev != "" {
		fmt.Fprintf(w, "HW Rev:   %s\n", d.HardwareRev)
	}
	if d.UUID != "" {
		fmt.Fprintf(w, "UUID:     %s\n", d.UUID)
	}
	if d.State != "" {
		fmt.Fprintf(w, "State:    %s\n", d.State)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run TestFormatDeviceInfoIncludesHardwareRevAndUUID ./cli/ -v
```

Expected: PASS.

- [ ] **Step 5: Run full cli tests**

```bash
go test ./cli/...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add cli/output.go cli/output_test.go
git commit -m "feat(cli): surface HardwareRev and UUID in device info output"
```

### Task 2.5: Make mocksbr emit UUID in discovery and accept UUID in DeviceConfig

**Files:**
- Modify: `mocksbr/device.go`
- Modify: `mocksbr/responses.go`
- Test: extend an existing mocksbr test

- [ ] **Step 1: Inspect current mocksbr discovery emission**

Already known: `mocksbr/responses.go:68-85` writes TLVs 0x0c, 0x0b, 0x0a, 0x09, 0x03, 0x02. UUID (0x0d) is NOT emitted.

`DeviceConfig` already has a `UUID string` field (mocksbr/device.go:33) but it's unused on the wire.

- [ ] **Step 2: Write the failing test**

Append to `mocksbr/integration_test.go`:

```go
func TestDiscoveryResponseIncludesUUID(t *testing.T) {
	n := NewNetwork(0, udap.NewNoOpLogger())
	defer n.Close()
	n.Add(DeviceConfig{
		MAC:  "00:04:20:00:00:01",
		UUID: "deadbeefcafebabe1122334455667788",
	})

	// Build a discovery request packet to feed into Receive.
	client, _ := udap.NewClientWithLogger(udap.NewNoOpLogger())
	req := client.CreateAdvancedDiscoveryPacket()
	client.Close()

	replies := n.Receive(req)
	if len(replies) != 1 {
		t.Fatalf("got %d replies, want 1", len(replies))
	}
	if !bytes.Contains(replies[0], []byte("deadbeefcafebabe1122334455667788")) {
		t.Errorf("reply does not contain UUID; reply hex=%x", replies[0])
	}
}
```

Add `"bytes"` to imports if not already there.

- [ ] **Step 3: Run test to verify it fails**

```bash
go test -run TestDiscoveryResponseIncludesUUID ./mocksbr/ -v
```

Expected: FAIL — UUID bytes not in reply.

- [ ] **Step 4: Emit UUID TLV in discovery response**

In `mocksbr/responses.go`, modify `buildDiscoveryResponse` (around line 68). Decode the UUID config (hex string) back to bytes for the wire; if it's not valid hex, fall back to writing it as raw bytes (matches existing forgiveness pattern):

```go
func (d *device) buildDiscoveryResponse(req *udap.Packet) []byte {
	hdr := buildHeader(req, d.cfg.MAC, req.UCPMethod)

	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, hdr)

	state := d.state()
	hostname := d.snapshotWorking()["hostname"]

	writeTLV(buf, 0x0c, []byte(state))
	writeTLV(buf, 0x0b, []byte(d.cfg.DeviceID))
	writeTLV(buf, 0x0a, []byte(d.cfg.Hardware))
	writeTLV(buf, 0x09, []byte(d.cfg.Firmware))
	writeTLV(buf, 0x03, []byte(d.cfg.Model))
	writeTLV(buf, 0x02, []byte(hostname))
	if d.cfg.UUID != "" {
		writeTLV(buf, 0x0d, uuidWireBytes(d.cfg.UUID))
	}

	return buf.Bytes()
}

// uuidWireBytes converts a hex-string UUID config into wire bytes. If
// the config isn't valid hex (e.g. "mock-sbr-001"), the bytes of the
// string are used directly; the client's hex.EncodeToString will then
// surface those bytes hex-encoded, which is harmless for tests.
func uuidWireBytes(uuid string) []byte {
	b, err := hex.DecodeString(uuid)
	if err != nil {
		return []byte(uuid)
	}
	return b
}
```

Add `"encoding/hex"` to imports.

- [ ] **Step 5: Run test to verify it passes**

```bash
go test -run TestDiscoveryResponseIncludesUUID ./mocksbr/ -v
```

Expected: PASS.

- [ ] **Step 6: Run full mocksbr suite**

```bash
go test ./mocksbr/...
```

Expected: PASS. The pre-existing `autoConfig` uses non-hex UUIDs like `"mock-sbr-001"`; verify the fallback path doesn't break anything.

- [ ] **Step 7: Commit**

```bash
git add mocksbr/responses.go mocksbr/integration_test.go
git commit -m "feat(mocksbr): emit UUID TLV 0x0d in discovery responses"
```

### Task 2.6: E2E test for HW rev + UUID surfacing through the CLI

**Files:**
- Test: `cli/e2e_info_test.go` (extend)

- [ ] **Step 1: Write the failing test**

Append to `cli/e2e_info_test.go`:

```go
func TestE2EInfoPrintsHardwareRevAndUUID(t *testing.T) {
	network := mocksbr.NewNetwork(0, udap.NewNoOpLogger())
	t.Cleanup(network.Close)
	network.Add(mocksbr.DeviceConfig{
		MAC:      "00:04:20:00:00:01",
		Hardware: "0005",
		UUID:     "deadbeefcafebabe1122334455667788",
	})
	prev := newClient
	newClient = func(_ bool, _ io.Writer) (*udap.Client, error) {
		return udap.NewClientWithTransport(
			mocksbr.NewMockTransport(network),
			udap.NewNoOpLogger(),
		), nil
	}
	t.Cleanup(func() { newClient = prev })

	var outBuf, errBuf bytes.Buffer
	err := Run([]string{"info", "00:04:20:00:00:01", "--timeout", "500ms"}, &outBuf, &errBuf)
	if err != nil {
		t.Fatalf("Run returned %v; stderr=%s", err, errBuf.String())
	}
	for _, want := range []string{"HW Rev:", "0005", "UUID:", "deadbeefcafebabe1122334455667788"} {
		if !strings.Contains(outBuf.String(), want) {
			t.Errorf("stdout missing %q; got:\n%s", want, outBuf.String())
		}
	}
}
```

Imports in this file already include `mocksbr`, `udap`, `bytes`, `strings`, `io`, `testing`. Add any that are missing.

- [ ] **Step 2: Run test to verify it passes**

```bash
go test -run TestE2EInfoPrintsHardwareRevAndUUID ./cli/ -v
```

Expected: PASS (because all the plumbing is in place from earlier tasks).

If it fails, debug — there's a wire-format issue.

- [ ] **Step 3: Commit**

```bash
git add cli/e2e_info_test.go
git commit -m "test(cli): e2e verify HardwareRev and UUID surface in info"
```

---

## Phase 3 — `get_ip` library operation + CLI subcommand

### Task 3.1: NetworkConfig value object

**Files:**
- Create: `udap/netconfig.go`
- Create: `udap/netconfig_test.go`

- [ ] **Step 1: Write the failing test**

Create `udap/netconfig_test.go`:

```go
package udap

import (
	"encoding/json"
	"net"
	"strings"
	"testing"
)

func TestNetworkConfigString(t *testing.T) {
	nc := NetworkConfig{
		IP:         net.IPv4(192, 168, 1, 50),
		SubnetMask: net.IPv4(255, 255, 255, 0),
		Gateway:    net.IPv4(192, 168, 1, 1),
	}
	got := nc.String()
	for _, want := range []string{"192.168.1.50", "255.255.255.0", "192.168.1.1"} {
		if !strings.Contains(got, want) {
			t.Errorf("String() missing %q; got %q", want, got)
		}
	}
}

func TestNetworkConfigJSONRoundtrip(t *testing.T) {
	in := NetworkConfig{
		IP:         net.IPv4(10, 0, 0, 1),
		SubnetMask: net.IPv4(255, 0, 0, 0),
		Gateway:    net.IPv4(10, 0, 0, 254),
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var out NetworkConfig
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !in.IP.Equal(out.IP) || !in.SubnetMask.Equal(out.SubnetMask) || !in.Gateway.Equal(out.Gateway) {
		t.Errorf("roundtrip mismatch: in=%+v out=%+v", in, out)
	}
}

func TestNetworkConfigZeroValueStringIsSafe(t *testing.T) {
	var nc NetworkConfig
	// Must not panic; output content is implementation-defined.
	_ = nc.String()
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run TestNetworkConfig ./udap/ -v
```

Expected: FAIL — `NetworkConfig` not declared.

- [ ] **Step 3: Create udap/netconfig.go**

```go
package udap

import (
	"fmt"
	"net"
)

// NetworkConfig is the result of a UCP_METHOD_GET_IP (0x0002) query.
// Returned by Client.GetDeviceNetworkConfigWithContext. Distinct from
// the Device aggregate: Device reflects what discovery passively
// observed; NetworkConfig is what the device reports actively when
// asked.
//
// All fields are optional — devices may omit TLVs (notably Gateway on
// static-IP-without-gateway configurations). Missing fields are zero
// IPs.
type NetworkConfig struct {
	IP         net.IP `json:"ip,omitempty"`
	SubnetMask net.IP `json:"subnet_mask,omitempty"`
	Gateway    net.IP `json:"gateway,omitempty"`
}

// String returns a multi-line representation suitable for CLI output.
func (n NetworkConfig) String() string {
	return fmt.Sprintf("IP:      %s\nSubnet:  %s\nGateway: %s",
		ipOrDash(n.IP), ipOrDash(n.SubnetMask), ipOrDash(n.Gateway))
}

func ipOrDash(ip net.IP) string {
	if len(ip) == 0 || ip.IsUnspecified() {
		return "-"
	}
	return ip.String()
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run TestNetworkConfig ./udap/ -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add udap/netconfig.go udap/netconfig_test.go
git commit -m "feat(udap): add NetworkConfig value object"
```

### Task 3.2: CreateGetIPPacket — wire-format builder

**Files:**
- Create: `udap/getip.go`
- Create: `udap/getip_test.go`

- [ ] **Step 1: Write the failing test**

Create `udap/getip_test.go`:

```go
package udap

import (
	"net"
	"testing"
)

func TestCreateGetIPPacketHeaderOnly(t *testing.T) {
	c := mustNewClient(t)
	defer c.Close()
	device := &Device{MAC: MAC([6]byte{0x00, 0x04, 0x20, 0x16, 0x05, 0x8f})}
	pkt, err := c.CreateGetIPPacket(device)
	if err != nil {
		t.Fatalf("CreateGetIPPacket: %v", err)
	}
	if len(pkt) != UDAPHeaderSize {
		t.Errorf("packet size %d, want %d (header only)", len(pkt), UDAPHeaderSize)
	}
	hdr, payload, err := ParsePacket(pkt)
	if err != nil {
		t.Fatalf("ParsePacket: %v", err)
	}
	if len(payload) != 0 {
		t.Errorf("payload length %d, want 0", len(payload))
	}
	if hdr.UCPMethod != MethodGetIP {
		t.Errorf("UCPMethod = 0x%04x, want 0x%04x", hdr.UCPMethod, MethodGetIP)
	}
	if hdr.DstBroadcast != 0 {
		t.Errorf("DstBroadcast = %d, want 0", hdr.DstBroadcast)
	}
	wantMAC := [6]byte{0x00, 0x04, 0x20, 0x16, 0x05, 0x8f}
	if hdr.DstAddress != wantMAC {
		t.Errorf("DstAddress = %x, want %x", hdr.DstAddress, wantMAC)
	}
}

func TestCreateGetIPPacketRejectsZeroMAC(t *testing.T) {
	c := mustNewClient(t)
	defer c.Close()
	device := &Device{MAC: MAC([6]byte{})}
	_, err := c.CreateGetIPPacket(device)
	if err == nil {
		t.Error("CreateGetIPPacket with zero MAC returned nil error")
	}
}

func TestParseGetIPResponseHappyPath(t *testing.T) {
	data := []byte{
		0x05, 0x04, 192, 168, 1, 50, // IP
		0x06, 0x04, 255, 255, 255, 0, // SubnetMask
		0x07, 0x04, 192, 168, 1, 1, // Gateway
	}
	nc, err := parseGetIPResponse(data)
	if err != nil {
		t.Fatalf("parseGetIPResponse: %v", err)
	}
	if !nc.IP.Equal(net.IPv4(192, 168, 1, 50)) {
		t.Errorf("IP = %v, want 192.168.1.50", nc.IP)
	}
	if !nc.SubnetMask.Equal(net.IPv4(255, 255, 255, 0)) {
		t.Errorf("SubnetMask = %v, want 255.255.255.0", nc.SubnetMask)
	}
	if !nc.Gateway.Equal(net.IPv4(192, 168, 1, 1)) {
		t.Errorf("Gateway = %v, want 192.168.1.1", nc.Gateway)
	}
}

func TestParseGetIPResponsePartialTLVs(t *testing.T) {
	// IP only — no subnet, no gateway. Should not error.
	data := []byte{
		0x05, 0x04, 10, 0, 0, 1,
	}
	nc, err := parseGetIPResponse(data)
	if err != nil {
		t.Fatalf("parseGetIPResponse: %v", err)
	}
	if !nc.IP.Equal(net.IPv4(10, 0, 0, 1)) {
		t.Errorf("IP = %v, want 10.0.0.1", nc.IP)
	}
	if nc.SubnetMask != nil {
		t.Errorf("SubnetMask = %v, want nil", nc.SubnetMask)
	}
	if nc.Gateway != nil {
		t.Errorf("Gateway = %v, want nil", nc.Gateway)
	}
}

func TestParseGetIPResponseEmptyPayloadIsZeroValue(t *testing.T) {
	nc, err := parseGetIPResponse(nil)
	if err != nil {
		t.Fatalf("parseGetIPResponse(nil): %v", err)
	}
	var zero NetworkConfig
	if nc != zero {
		t.Errorf("got %+v, want zero value", nc)
	}
}

func TestParseGetIPResponseSkipsWrongLengthTLV(t *testing.T) {
	// IP is correct (4 bytes); subnet has wrong length (3 bytes). The
	// 3-byte subnet should be skipped; IP should still parse.
	data := []byte{
		0x05, 0x04, 10, 0, 0, 1,
		0x06, 0x03, 255, 255, 0, // wrong length for IPv4
		0x07, 0x04, 10, 0, 0, 254,
	}
	nc, err := parseGetIPResponse(data)
	if err != nil {
		t.Fatalf("parseGetIPResponse: %v", err)
	}
	if !nc.IP.Equal(net.IPv4(10, 0, 0, 1)) {
		t.Errorf("IP = %v, want 10.0.0.1", nc.IP)
	}
	if nc.SubnetMask != nil {
		t.Errorf("SubnetMask = %v, want nil (skipped)", nc.SubnetMask)
	}
	if !nc.Gateway.Equal(net.IPv4(10, 0, 0, 254)) {
		t.Errorf("Gateway = %v, want 10.0.0.254", nc.Gateway)
	}
}

func TestParseGetIPResponseMalformedLengthRunoff(t *testing.T) {
	// One TLV declares length 4 but provides 2 bytes; should stop, not panic.
	data := []byte{
		0x05, 0x04, 1, 2,
	}
	_, err := parseGetIPResponse(data)
	if err != nil {
		t.Fatalf("parseGetIPResponse should soft-fail: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run "TestCreateGetIP|TestParseGetIP" ./udap/ -v
```

Expected: FAIL — `CreateGetIPPacket` and `parseGetIPResponse` not declared.

- [ ] **Step 3: Create udap/getip.go**

```go
package udap

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
)

// CreateGetIPPacket builds a UCP_METHOD_GET_IP (0x0002) request — a
// 27-byte UDAP header addressed to device.MAC with no payload. The
// device replies with method=0x0002 and a TLV stream of network-config
// codes (0x05 IP, 0x06 SubnetMask, 0x07 Gateway).
//
// Reference: Net::UDAP MessageOut.pm — "nothing further to do for
// get_ip" once the header is built.
func (c *Client) CreateGetIPPacket(device *Device) ([]byte, error) {
	if device.MAC.IsZero() {
		return nil, fmt.Errorf("cannot build GetIP packet: device has zero MAC address")
	}
	packet := c.createUdapPacket(
		device.MAC.Bytes(),
		MethodGetIP, // 0x0002
		0x01,        // request flag
		false,       // unicast
	)
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, packet); err != nil {
		return nil, fmt.Errorf("encode GetIP header: %w", err)
	}
	return buf.Bytes(), nil
}

// GetDeviceNetworkConfigWithContext sends a get_ip request to device and
// parses the response. Soft-fails on missing or malformed TLVs (returns
// a NetworkConfig with zero-value fields for whichever pieces are
// missing); hard-fails on transport errors, context cancellation,
// MethodError, MethodCredentialsError, or unexpected reply methods.
func (c *Client) GetDeviceNetworkConfigWithContext(ctx context.Context, device *Device) (NetworkConfig, error) {
	packet, err := c.CreateGetIPPacket(device)
	if err != nil {
		return NetworkConfig{}, fmt.Errorf("build GetIP packet: %w", err)
	}
	if err := c.transport.Send(packet); err != nil {
		return NetworkConfig{}, fmt.Errorf("send GetIP: %w", err)
	}
	c.logger.Info("Sent GetIP request", "device_mac", device.MAC)

	respPacket, data, err := c.waitForDeviceReply(ctx, device)
	if err != nil {
		return NetworkConfig{}, err
	}

	switch respPacket.UCPMethod {
	case MethodGetIP:
		return parseGetIPResponse(data)
	case MethodError:
		if len(data) > 0 {
			for _, tlv := range DecodeTLV(data) {
				if tlv.Type == TLVTypeErrorMessage {
					return NetworkConfig{}, fmt.Errorf("device %s error: %s", device.MAC, string(tlv.Value))
				}
			}
		}
		return NetworkConfig{}, fmt.Errorf("device %s returned error response", device.MAC)
	case MethodCredentialsError:
		return NetworkConfig{}, fmt.Errorf("device %s rejected credentials", device.MAC)
	default:
		return NetworkConfig{}, fmt.Errorf("device %s: unexpected response method 0x%04x", device.MAC, respPacket.UCPMethod)
	}
}

// parseGetIPResponse decodes a get_ip reply payload. TLV format
// matches discovery: 1-byte code, 1-byte length, value bytes.
// Recognised codes:
//
//	0x05 UCP_CODE_IP_ADDR       (4 bytes IPv4)
//	0x06 UCP_CODE_SUBNET_MASK   (4 bytes IPv4 mask)
//	0x07 UCP_CODE_GATEWAY_ADDR  (4 bytes IPv4)
//
// Unknown codes are skipped. Wrong-length codes are skipped. The
// function never panics on malformed input; truncated TLVs cause the
// scan to stop where it is and return whatever was parsed up to that
// point.
func parseGetIPResponse(data []byte) (NetworkConfig, error) {
	var nc NetworkConfig
	for offset := 0; offset+2 <= len(data); {
		tagType := data[offset]
		length := int(data[offset+1])
		offset += 2
		if offset+length > len(data) {
			break
		}
		value := data[offset : offset+length]
		offset += length

		switch tagType {
		case tlvIPAddr:
			if length == 4 {
				nc.IP = net.IPv4(value[0], value[1], value[2], value[3])
			}
		case tlvSubnetMask:
			if length == 4 {
				nc.SubnetMask = net.IPv4(value[0], value[1], value[2], value[3])
			}
		case tlvGatewayAddr:
			if length == 4 {
				nc.Gateway = net.IPv4(value[0], value[1], value[2], value[3])
			}
		}
	}
	return nc, nil
}

// get_ip / discovery TLV codes per Net::UDAP Constant.pm:
//
//	UCP_CODE_IP_ADDR       = 0x05
//	UCP_CODE_SUBNET_MASK   = 0x06
//	UCP_CODE_GATEWAY_ADDR  = 0x07
const (
	tlvIPAddr      = 0x05
	tlvSubnetMask  = 0x06
	tlvGatewayAddr = 0x07
)
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test -run "TestCreateGetIP|TestParseGetIP" ./udap/ -v
```

Expected: PASS for all six subtests.

- [ ] **Step 5: Run full udap suite to confirm no regressions**

```bash
go test ./udap/...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add udap/getip.go udap/getip_test.go
git commit -m "feat(udap): implement get_ip (UCP method 0x0002) operation"
```

### Task 3.3: mocksbr handler for MethodGetIP

**Files:**
- Modify: `mocksbr/device.go`
- Modify: `mocksbr/responses.go`
- Modify: `mocksbr/handlers.go`
- Test: extend `mocksbr/integration_test.go`

- [ ] **Step 1: Add NetworkConfig fields to DeviceConfig**

In `mocksbr/device.go`, extend `DeviceConfig`:

```go
type DeviceConfig struct {
	MAC      string
	Name     string
	Model    string
	DeviceID string
	Firmware string
	Hardware string
	UUID     string

	// Network configuration reported by the get_ip operation
	// (UCP method 0x0002). All optional — empty values are emitted
	// as zero IPs in the wire response.
	IP         string // e.g. "192.168.1.50"
	SubnetMask string // e.g. "255.255.255.0"
	Gateway    string // e.g. "192.168.1.1"

	NVRAM       map[string]string
	FailOn      []Op
	Slow        time.Duration
	Unreachable bool
	RebootDelay time.Duration

	// DropGetIP makes the device silently ignore get_ip requests
	// (for timeout testing).
	DropGetIP bool

	Malformed MalformedMode
}
```

- [ ] **Step 2: Write the failing test**

Append to `mocksbr/integration_test.go`:

```go
func TestGetIPHandlerReturnsConfiguredIPs(t *testing.T) {
	n := NewNetwork(0, udap.NewNoOpLogger())
	defer n.Close()
	n.Add(DeviceConfig{
		MAC:        "00:04:20:00:00:01",
		IP:         "192.168.1.50",
		SubnetMask: "255.255.255.0",
		Gateway:    "192.168.1.1",
	})

	client, _ := udap.NewClientWithLogger(udap.NewNoOpLogger())
	mac, _ := udap.ParseMAC("00:04:20:00:00:01")
	dev := &udap.Device{MAC: mac}
	req, err := client.CreateGetIPPacket(dev)
	if err != nil {
		t.Fatalf("CreateGetIPPacket: %v", err)
	}
	client.Close()

	replies := n.Receive(req)
	if len(replies) != 1 {
		t.Fatalf("got %d replies, want 1", len(replies))
	}

	_, payload, err := udap.ParsePacket(replies[0])
	if err != nil {
		t.Fatalf("ParsePacket: %v", err)
	}
	// Verify IP TLV (0x05) is present with the configured value bytes
	if !bytes.Contains(payload, []byte{0x05, 0x04, 192, 168, 1, 50}) {
		t.Errorf("payload missing IP TLV; got %x", payload)
	}
}

func TestGetIPHandlerHonoursDropGetIP(t *testing.T) {
	n := NewNetwork(0, udap.NewNoOpLogger())
	defer n.Close()
	n.Add(DeviceConfig{
		MAC:       "00:04:20:00:00:01",
		DropGetIP: true,
	})

	client, _ := udap.NewClientWithLogger(udap.NewNoOpLogger())
	mac, _ := udap.ParseMAC("00:04:20:00:00:01")
	dev := &udap.Device{MAC: mac}
	req, _ := client.CreateGetIPPacket(dev)
	client.Close()

	replies := n.Receive(req)
	if len(replies) != 0 {
		t.Errorf("got %d replies, want 0 (dropped)", len(replies))
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test -run TestGetIPHandler ./mocksbr/ -v
```

Expected: FAIL — `MethodGetIP` not dispatched, no replies.

- [ ] **Step 4: Add buildGetIPResponse in responses.go**

Append to `mocksbr/responses.go`:

```go
// buildGetIPResponse constructs a get_ip reply (UCPMethod=0x0002) with
// TLV-encoded IP / SubnetMask / Gateway from DeviceConfig. Missing
// fields are encoded as zero IPv4 (0.0.0.0). The wire TLV codes match
// Net::UDAP: 0x05 IP_ADDR, 0x06 SUBNET_MASK, 0x07 GATEWAY_ADDR.
func (d *device) buildGetIPResponse(req *udap.Packet) []byte {
	hdr := buildHeader(req, d.cfg.MAC, udap.MethodGetIP)
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, hdr)
	writeIPTLV(buf, 0x05, d.cfg.IP)
	writeIPTLV(buf, 0x06, d.cfg.SubnetMask)
	writeIPTLV(buf, 0x07, d.cfg.Gateway)
	return buf.Bytes()
}

// writeIPTLV emits a 4-byte IPv4 TLV. Empty or unparseable inputs
// produce a 0.0.0.0 value.
func writeIPTLV(buf *bytes.Buffer, t byte, ipStr string) {
	out := []byte{0, 0, 0, 0}
	if ipStr != "" {
		if ip := net.ParseIP(ipStr); ip != nil {
			if ip4 := ip.To4(); ip4 != nil {
				out = ip4
			}
		}
	}
	buf.WriteByte(t)
	buf.WriteByte(0x04)
	buf.Write(out)
}
```

`net` should already be imported; verify after editing.

- [ ] **Step 5: Dispatch MethodGetIP in handlers.go**

In `mocksbr/handlers.go`, add a case to `ReceiveScheduled`:

```go
	switch pkt.UCPMethod {
	case udap.MethodDiscover, udap.MethodAdvDisc:
		return n.handleDiscovery(pkt)
	case udap.MethodGetData:
		return n.dispatchUnicast(pkt, OpGet, func(d *device) []byte {
			return d.buildGetDataResponse(pkt, payload)
		})
	case udap.MethodSetData:
		return n.dispatchUnicast(pkt, OpSet, func(d *device) []byte {
			return d.handleSetData(pkt, payload)
		})
	case udap.MethodReset:
		return n.dispatchUnicast(pkt, OpReset, func(d *device) []byte {
			ack := d.buildResetAck(pkt)
			d.startReboot()
			d.applyReset()
			return ack
		})
	case udap.MethodGetIP:
		return n.dispatchUnicast(pkt, OpGetIP, func(d *device) []byte {
			if d.cfg.DropGetIP {
				return nil
			}
			return d.buildGetIPResponse(pkt)
		})
	default:
		n.logger.Debug("mocksbr: unhandled UCPMethod",
			"method", fmt.Sprintf("0x%04x", pkt.UCPMethod))
		return nil
	}
```

- [ ] **Step 6: Add OpGetIP constant**

In `mocksbr/device.go`, extend the Op constants:

```go
const (
	OpDiscover Op = "discover"
	OpGet      Op = "get"
	OpSet      Op = "set"
	OpSave     Op = "save"
	OpReset    Op = "reset"
	OpGetIP    Op = "getip"
)
```

- [ ] **Step 7: Run tests to verify they pass**

```bash
go test -run TestGetIPHandler ./mocksbr/ -v
```

Expected: PASS for both subtests.

- [ ] **Step 8: Run full mocksbr suite**

```bash
go test ./mocksbr/...
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add mocksbr/device.go mocksbr/responses.go mocksbr/handlers.go mocksbr/integration_test.go
git commit -m "feat(mocksbr): handle get_ip requests with configurable IPs"
```

### Task 3.4: `getip` CLI subcommand

**Files:**
- Create: `cli/getip.go`
- Modify: `cli/cli.go`
- Modify: `cli/output.go`

- [ ] **Step 1: Add NetworkConfig output renderer**

Append to `cli/output.go`:

```go
// formatNetworkConfig writes IP / Subnet / Gateway lines, using "-"
// for any empty field.
func formatNetworkConfig(w io.Writer, nc udap.NetworkConfig) {
	fmt.Fprintf(w, "IP:      %s\n", ipOrDashCLI(nc.IP))
	fmt.Fprintf(w, "Subnet:  %s\n", ipOrDashCLI(nc.SubnetMask))
	fmt.Fprintf(w, "Gateway: %s\n", ipOrDashCLI(nc.Gateway))
}

func ipOrDashCLI(ip net.IP) string {
	if len(ip) == 0 || ip.IsUnspecified() {
		return "-"
	}
	return ip.String()
}
```

Add `"net"` to the imports of `cli/output.go`.

- [ ] **Step 2: Create cli/getip.go**

```go
package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/pflag"
)

func runGetIP(args []string, stdout, stderr io.Writer) error {
	fs := pflag.NewFlagSet("getip", pflag.ContinueOnError)
	fs.SetOutput(stderr)
	timeout := newDurationWithPlaceholder("DURATION", 5*time.Second)
	fs.Var(timeout, "timeout", "Operation timeout, e.g. 5s, 30s, 2m")
	verbose := fs.BoolP("verbose", "v", false, "Debug logging to stderr")
	if err := parseSubcommandFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return &ExitError{Code: 1, Err: fmt.Errorf("getip: expected exactly one MAC argument")}
	}
	mac, err := normalizeMAC(fs.Arg(0))
	if err != nil {
		return &ExitError{Code: 1, Err: err}
	}

	client, err := newClient(*verbose, stderr)
	if err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout.Value())
	defer cancel()
	stop := startProgress(stderr, "getip", timeout.Value())
	device, err := discoverAndFind(ctx, client, mac)
	if err != nil {
		stop()
		return err
	}
	nc, err := client.GetDeviceNetworkConfigWithContext(ctx, device)
	stop()
	if err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("get_ip failed for %s: %w", mac, err)}
	}
	formatNetworkConfig(stdout, nc)
	return nil
}
```

- [ ] **Step 3: Register subcommand in cli/cli.go**

In `cli/cli.go`, add a case to `dispatch`:

```go
func dispatch(cmd string, subArgs []string, stdout, syncErr io.Writer) error {
	switch cmd {
	case "discover":
		return runDiscover(subArgs, stdout, syncErr)
	case "info":
		return runInfo(subArgs, stdout, syncErr)
	case "read":
		return runRead(subArgs, stdout, syncErr)
	case "get":
		return runGet(subArgs, stdout, syncErr)
	case "set":
		return runSet(subArgs, stdout, syncErr)
	case "reboot":
		return runReboot(subArgs, stdout, syncErr)
	case "getip":
		return runGetIP(subArgs, stdout, syncErr)
	default:
		return &ExitError{Code: 1, Err: fmt.Errorf("unknown command: %s", cmd)}
	}
}
```

And update the `printUsage` block in the same file to add the `getip` line:

Before:
```
  reboot <mac>                   Reboot the device
```

After:
```
  reboot <mac>                   Reboot the device
  getip <mac>                    Query device IP / subnet / gateway via UCP get_ip
```

- [ ] **Step 4: Build to confirm it compiles**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add cli/getip.go cli/cli.go cli/output.go
git commit -m "feat(cli): add getip subcommand"
```

### Task 3.5: E2E tests for getip — happy path

**Files:**
- Create: `cli/e2e_getip_test.go`

- [ ] **Step 1: Write the failing test**

```go
package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"go-udap/mocksbr"
	"go-udap/udap"
)

func TestE2EGetIPHappyPath(t *testing.T) {
	network := mocksbr.NewNetwork(0, udap.NewNoOpLogger())
	t.Cleanup(network.Close)
	network.Add(mocksbr.DeviceConfig{
		MAC:        "00:04:20:00:00:01",
		IP:         "192.168.1.50",
		SubnetMask: "255.255.255.0",
		Gateway:    "192.168.1.1",
	})
	prev := newClient
	newClient = func(_ bool, _ io.Writer) (*udap.Client, error) {
		return udap.NewClientWithTransport(
			mocksbr.NewMockTransport(network),
			udap.NewNoOpLogger(),
		), nil
	}
	t.Cleanup(func() { newClient = prev })

	var outBuf, errBuf bytes.Buffer
	err := Run([]string{"getip", "00:04:20:00:00:01", "--timeout", "500ms"}, &outBuf, &errBuf)
	if err != nil {
		t.Fatalf("Run returned %v; stderr=%s", err, errBuf.String())
	}
	for _, want := range []string{"192.168.1.50", "255.255.255.0", "192.168.1.1"} {
		if !strings.Contains(outBuf.String(), want) {
			t.Errorf("stdout missing %q; got:\n%s", want, outBuf.String())
		}
	}
}

func TestE2EGetIPMissingMACIsExitCodeTwo(t *testing.T) {
	network := mocksbr.NewNetwork(1, udap.NewNoOpLogger())
	t.Cleanup(network.Close)
	prev := newClient
	newClient = func(_ bool, _ io.Writer) (*udap.Client, error) {
		return udap.NewClientWithTransport(
			mocksbr.NewMockTransport(network),
			udap.NewNoOpLogger(),
		), nil
	}
	t.Cleanup(func() { newClient = prev })

	var outBuf, errBuf bytes.Buffer
	err := Run([]string{"getip", "aa:bb:cc:dd:ee:ff", "--timeout", "200ms"}, &outBuf, &errBuf)
	if ExitCode(err) != 2 {
		t.Errorf("exit code %d, want 2 (device not found)", ExitCode(err))
	}
}

func TestE2EGetIPTimeoutWhenDeviceDropsRequest(t *testing.T) {
	network := mocksbr.NewNetwork(0, udap.NewNoOpLogger())
	t.Cleanup(network.Close)
	network.Add(mocksbr.DeviceConfig{
		MAC:       "00:04:20:00:00:01",
		DropGetIP: true,
	})
	prev := newClient
	newClient = func(_ bool, _ io.Writer) (*udap.Client, error) {
		return udap.NewClientWithTransport(
			mocksbr.NewMockTransport(network),
			udap.NewNoOpLogger(),
		), nil
	}
	t.Cleanup(func() { newClient = prev })

	var outBuf, errBuf bytes.Buffer
	err := Run([]string{"getip", "00:04:20:00:00:01", "--timeout", "200ms"}, &outBuf, &errBuf)
	if ExitCode(err) != 2 {
		t.Errorf("exit code %d, want 2 (timeout)", ExitCode(err))
	}
}
```

- [ ] **Step 2: Run tests to verify they pass**

```bash
go test -run TestE2EGetIP ./cli/ -v
```

Expected: PASS for all three. (Happy path uses the network we built in Task 3.3; timeout path uses DropGetIP.)

If failing, debug.

- [ ] **Step 3: Commit**

```bash
git add cli/e2e_getip_test.go
git commit -m "test(cli): e2e tests for getip subcommand"
```

### Task 3.6: E2E test for getip MethodError reply

**Files:**
- Modify: `cli/e2e_getip_test.go`

- [ ] **Step 1: Write the failing test**

mocksbr already has a `FailOn` knob that returns `MethodError`. Append:

```go
func TestE2EGetIPMethodErrorPropagatesMessage(t *testing.T) {
	network := mocksbr.NewNetwork(0, udap.NewNoOpLogger())
	t.Cleanup(network.Close)
	network.Add(mocksbr.DeviceConfig{
		MAC:    "00:04:20:00:00:01",
		FailOn: []mocksbr.Op{mocksbr.OpGetIP},
	})
	prev := newClient
	newClient = func(_ bool, _ io.Writer) (*udap.Client, error) {
		return udap.NewClientWithTransport(
			mocksbr.NewMockTransport(network),
			udap.NewNoOpLogger(),
		), nil
	}
	t.Cleanup(func() { newClient = prev })

	var outBuf, errBuf bytes.Buffer
	err := Run([]string{"getip", "00:04:20:00:00:01", "--timeout", "500ms"}, &outBuf, &errBuf)
	if ExitCode(err) != 2 {
		t.Errorf("exit code %d, want 2", ExitCode(err))
	}
	if err == nil || !strings.Contains(err.Error(), "mocksbr: configured to fail getip") {
		t.Errorf("error %v does not contain mocksbr's failure message", err)
	}
}
```

- [ ] **Step 2: Run test to verify it passes**

```bash
go test -run TestE2EGetIPMethodErrorPropagatesMessage ./cli/ -v
```

Expected: PASS. (mocksbr's existing FailOn machinery wraps the configured fail message in a TLV 0x03; our parseGetIPResponse error path extracts it.)

- [ ] **Step 3: Commit**

```bash
git add cli/e2e_getip_test.go
git commit -m "test(cli): e2e getip surfaces device MethodError message"
```

---

## Phase 4 — `discover --info` integration

### Task 4.1: Wire per-device get_ip into discover --info

**Files:**
- Modify: `cli/discover.go`
- Modify: `cli/output.go`

- [ ] **Step 1: Write the failing test (lives in cli/e2e_discover_info_getip_test.go)**

Create `cli/e2e_discover_info_getip_test.go`:

```go
package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"go-udap/mocksbr"
	"go-udap/udap"
)

func TestE2EDiscoverInfoIncludesNetworkConfig(t *testing.T) {
	network := mocksbr.NewNetwork(0, udap.NewNoOpLogger())
	t.Cleanup(network.Close)
	network.Add(mocksbr.DeviceConfig{
		MAC:        "00:04:20:00:00:01",
		IP:         "192.168.1.50",
		SubnetMask: "255.255.255.0",
		Gateway:    "192.168.1.1",
	})
	prev := newClient
	newClient = func(_ bool, _ io.Writer) (*udap.Client, error) {
		return udap.NewClientWithTransport(
			mocksbr.NewMockTransport(network),
			udap.NewNoOpLogger(),
		), nil
	}
	t.Cleanup(func() { newClient = prev })

	var outBuf, errBuf bytes.Buffer
	err := Run([]string{"discover", "--info", "--timeout", "500ms"}, &outBuf, &errBuf)
	if err != nil {
		t.Fatalf("Run returned %v; stderr=%s", err, errBuf.String())
	}
	for _, want := range []string{"192.168.1.50", "255.255.255.0", "192.168.1.1", "Subnet:", "Gateway:"} {
		if !strings.Contains(outBuf.String(), want) {
			t.Errorf("stdout missing %q; got:\n%s", want, outBuf.String())
		}
	}
}

func TestE2EDiscoverInfoPartialFailureRendersDashes(t *testing.T) {
	network := mocksbr.NewNetwork(0, udap.NewNoOpLogger())
	t.Cleanup(network.Close)
	network.Add(mocksbr.DeviceConfig{
		MAC:        "00:04:20:00:00:01",
		IP:         "10.0.0.1",
		SubnetMask: "255.0.0.0",
	})
	network.Add(mocksbr.DeviceConfig{
		MAC:       "00:04:20:00:00:02",
		DropGetIP: true,
	})
	prev := newClient
	newClient = func(_ bool, _ io.Writer) (*udap.Client, error) {
		return udap.NewClientWithTransport(
			mocksbr.NewMockTransport(network),
			udap.NewNoOpLogger(),
		), nil
	}
	t.Cleanup(func() { newClient = prev })

	var outBuf, errBuf bytes.Buffer
	err := Run([]string{"discover", "--info", "--timeout", "500ms"}, &outBuf, &errBuf)
	if ExitCode(err) != 0 {
		t.Errorf("exit code %d, want 0 (partial failures are soft)", ExitCode(err))
	}
	// Device 1 (answers): should show real values.
	if !strings.Contains(outBuf.String(), "10.0.0.1") {
		t.Errorf("expected 10.0.0.1 in output: %s", outBuf.String())
	}
	// Device 2 (drops get_ip): the block for it must contain at least one "-".
	// We confirm by counting "Subnet:" lines: both blocks should have one each.
	if got := strings.Count(outBuf.String(), "Subnet:"); got != 2 {
		t.Errorf("got %d Subnet: lines, want 2", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run "TestE2EDiscoverInfoIncludesNetworkConfig|TestE2EDiscoverInfoPartialFailureRendersDashes" ./cli/ -v
```

Expected: FAIL — `discover --info` doesn't query get_ip yet.

- [ ] **Step 3: Modify cli/discover.go to fire per-device get_ip**

Replace the loop in `runDiscover` (around line 49-58) with:

```go
	for i, d := range devices {
		if *info {
			if i > 0 {
				fmt.Fprintln(stdout)
			}
			formatDeviceInfo(stdout, d)
			nc, err := client.GetDeviceNetworkConfigWithContext(ctx, d)
			if err != nil {
				// Soft-fail: log and emit dashes so the table is consistent.
				fmt.Fprintf(stderr, "warning: get_ip failed for %s: %v\n", d.MAC, err)
				nc = udap.NetworkConfig{}
			}
			formatNetworkConfig(stdout, nc)
		} else {
			fmt.Fprintln(stdout, d.MAC)
		}
	}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test -run "TestE2EDiscoverInfoIncludesNetworkConfig|TestE2EDiscoverInfoPartialFailureRendersDashes" ./cli/ -v
```

Expected: PASS for both.

- [ ] **Step 5: Run full cli suite**

```bash
go test ./cli/...
```

Expected: PASS. Existing `discover --info` tests may need updating if they assert exact output; check `cli/e2e_discover_test.go`.

If any pre-existing test fails because it asserted exact output, update its expectation to include the new Network-config lines OR change the existing test to use a DropGetIP device (so its network block shows dashes) — whichever is least invasive.

- [ ] **Step 6: Commit**

```bash
git add cli/discover.go cli/e2e_discover_info_getip_test.go
# Plus any updates to existing tests
git commit -m "feat(cli): discover --info queries get_ip per device"
```

---

## Phase 5 — NetInterface + `interfaces` subcommand

### Task 5.1: NetInterface value object + directed-broadcast math

**Files:**
- Create: `udap/interfaces.go`
- Create: `udap/interfaces_test.go`

- [ ] **Step 1: Write the failing test**

```go
package udap

import (
	"net"
	"testing"
)

func TestComputeDirectedBroadcast(t *testing.T) {
	cases := []struct {
		name string
		addr net.IP
		mask net.IPMask
		want net.IP
	}{
		{"slash24", net.IPv4(192, 168, 1, 50), net.CIDRMask(24, 32), net.IPv4(192, 168, 1, 255)},
		{"slash16", net.IPv4(10, 1, 2, 3), net.CIDRMask(16, 32), net.IPv4(10, 1, 255, 255)},
		{"slash22", net.IPv4(172, 16, 5, 7), net.CIDRMask(22, 32), net.IPv4(172, 16, 7, 255)},
		{"slash30", net.IPv4(192, 168, 1, 5), net.CIDRMask(30, 32), net.IPv4(192, 168, 1, 7)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := computeDirectedBroadcast(tc.addr, tc.mask)
			if !got.Equal(tc.want) {
				t.Errorf("computeDirectedBroadcast = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEnumerateInterfacesAllPassFilter(t *testing.T) {
	ifs, err := EnumerateInterfaces()
	if err != nil {
		t.Skipf("EnumerateInterfaces error (likely permissions/CI): %v", err)
	}
	if len(ifs) == 0 {
		t.Skip("no usable interfaces on this host")
	}
	for _, ni := range ifs {
		if ni.Name == "" {
			t.Errorf("Name is empty")
		}
		if ni.Addr == nil {
			t.Errorf("%s has nil Addr", ni.Name)
		}
		if ni.Addr.To4() == nil {
			t.Errorf("%s Addr is not IPv4: %v", ni.Name, ni.Addr)
		}
		if ni.Broadcast == nil {
			t.Errorf("%s has nil Broadcast", ni.Name)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run "TestComputeDirectedBroadcast|TestEnumerateInterfacesAllPassFilter" ./udap/ -v
```

Expected: FAIL — symbols not defined.

- [ ] **Step 3: Create udap/interfaces.go**

```go
package udap

import (
	"fmt"
	"net"
)

// NetInterface is a context-local representation of a network interface
// usable for UDAP broadcast discovery. It translates from net.Interface
// keeping Go-stdlib terminology out of the udap domain — a small
// anti-corruption layer at the boundary.
type NetInterface struct {
	Name      string `json:"name"`
	Index     int    `json:"index"`
	Addr      net.IP `json:"addr"`
	Broadcast net.IP `json:"broadcast"`
}

// EnumerateInterfaces returns all interfaces usable for UDAP broadcast
// discovery. The filter is:
//
//   - FlagUp set
//   - FlagBroadcast set
//   - FlagLoopback NOT set
//   - At least one IPv4 address
//
// For each match, the first IPv4 address (and its mask) drives the
// computed directed-broadcast address (addr | ^mask).
//
// WireGuard / Tailscale interfaces are filtered out automatically
// because they don't carry FlagBroadcast — see the spec for rationale.
func EnumerateInterfaces() ([]NetInterface, error) {
	raw, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("net.Interfaces: %w", err)
	}
	out := make([]NetInterface, 0, len(raw))
	for _, iface := range raw {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagBroadcast == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipnet.IP.To4()
			if ip4 == nil {
				continue
			}
			out = append(out, NetInterface{
				Name:      iface.Name,
				Index:     iface.Index,
				Addr:      ip4,
				Broadcast: computeDirectedBroadcast(ip4, ipnet.Mask),
			})
			break // one IPv4 entry per interface is enough
		}
	}
	return out, nil
}

// computeDirectedBroadcast returns addr | ^mask, i.e. the subnet's
// directed-broadcast address. Pure arithmetic on the 4-byte IPv4
// representation — works the same on every platform.
func computeDirectedBroadcast(addr net.IP, mask net.IPMask) net.IP {
	ip4 := addr.To4()
	if ip4 == nil {
		return nil
	}
	// Normalise mask to 4 bytes too (CIDRMask may return 16).
	m := mask
	if len(m) == 16 {
		m = m[12:16]
	}
	out := make(net.IP, 4)
	for i := 0; i < 4; i++ {
		out[i] = ip4[i] | ^m[i]
	}
	return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test -run "TestComputeDirectedBroadcast|TestEnumerateInterfacesAllPassFilter" ./udap/ -v
```

Expected: PASS for `TestComputeDirectedBroadcast`. `TestEnumerateInterfacesAllPassFilter` either PASSes (on a host with at least one usable interface) or SKIPs.

- [ ] **Step 5: Commit**

```bash
git add udap/interfaces.go udap/interfaces_test.go
git commit -m "feat(udap): add NetInterface value object and EnumerateInterfaces"
```

### Task 5.2: `interfaces` subcommand

**Files:**
- Create: `cli/interfaces.go`
- Modify: `cli/cli.go`
- Modify: `cli/output.go`

- [ ] **Step 1: Add the table renderer**

Append to `cli/output.go`:

```go
// formatInterfacesTable writes a fixed-column table for NetInterfaces.
// If the slice is empty, writes nothing.
func formatInterfacesTable(w io.Writer, ifs []udap.NetInterface) {
	if len(ifs) == 0 {
		return
	}
	fmt.Fprintln(w, "NAME            INDEX  ADDRESS            BROADCAST")
	for _, ni := range ifs {
		fmt.Fprintf(w, "%-15s %-5d  %-18s %s\n", ni.Name, ni.Index, ni.Addr, ni.Broadcast)
	}
}
```

- [ ] **Step 2: Create cli/interfaces.go**

```go
package cli

import (
	"fmt"
	"io"

	"github.com/spf13/pflag"

	"go-udap/udap"
)

func runInterfaces(args []string, stdout, stderr io.Writer) error {
	fs := pflag.NewFlagSet("interfaces", pflag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := parseSubcommandFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return &ExitError{Code: 1, Err: fmt.Errorf("interfaces: takes no arguments")}
	}
	ifs, err := udap.EnumerateInterfaces()
	if err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("enumerate interfaces: %w", err)}
	}
	if len(ifs) == 0 {
		fmt.Fprintln(stderr, "no usable interfaces found")
		return nil
	}
	formatInterfacesTable(stdout, ifs)
	return nil
}
```

- [ ] **Step 3: Register subcommand**

In `cli/cli.go`'s `dispatch` switch, add:

```go
	case "interfaces":
		return runInterfaces(subArgs, stdout, syncErr)
```

And update `printUsage`:

```
  interfaces                     List network interfaces usable for discovery
```

(Insert after the existing `reboot` and `getip` lines.)

- [ ] **Step 4: Smoke-test it**

```bash
go build ./...
./go-udap interfaces
```

Expected: prints either a table of interfaces or "no usable interfaces found".

- [ ] **Step 5: Commit**

```bash
git add cli/interfaces.go cli/cli.go cli/output.go
git commit -m "feat(cli): add interfaces subcommand"
```

### Task 5.3: E2E smoke for interfaces subcommand

**Files:**
- Create: `cli/e2e_interfaces_test.go`

- [ ] **Step 1: Write the test**

```go
package cli

import (
	"bytes"
	"strings"
	"testing"

	"go-udap/udap"
)

func TestE2EInterfacesSubcommandSmoke(t *testing.T) {
	// EnumerateInterfaces is real-OS-state-dependent; this test runs
	// against the actual host. It should always exit 0; the output is
	// either a table or "no usable interfaces found" on stderr.
	ifs, err := udap.EnumerateInterfaces()
	if err != nil {
		t.Skipf("EnumerateInterfaces error: %v", err)
	}

	var outBuf, errBuf bytes.Buffer
	rerr := Run([]string{"interfaces"}, &outBuf, &errBuf)
	if ExitCode(rerr) != 0 {
		t.Errorf("exit code %d, want 0", ExitCode(rerr))
	}
	if len(ifs) > 0 {
		if !strings.Contains(outBuf.String(), "NAME") {
			t.Errorf("expected table header in stdout; got:\n%s", outBuf.String())
		}
	} else {
		if !strings.Contains(errBuf.String(), "no usable interfaces") {
			t.Errorf("expected 'no usable interfaces' on stderr; got:\n%s", errBuf.String())
		}
	}
}
```

- [ ] **Step 2: Run test**

```bash
go test -run TestE2EInterfacesSubcommandSmoke ./cli/ -v
```

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add cli/e2e_interfaces_test.go
git commit -m "test(cli): e2e smoke for interfaces subcommand"
```

---

## Phase 6 — `--interface NAME` single-interface mode

### Task 6.1: NewUDPTransportOnInterface

**Files:**
- Modify: `udap/transport.go`
- Test: `udap/transport_test.go` (extend if exists)

- [ ] **Step 1: Write the failing test**

Append to `udap/transport_test.go` (file already exists per repo listing):

```go
func TestNewUDPTransportOnInterfaceBindsToAddr(t *testing.T) {
	ifs, err := EnumerateInterfaces()
	if err != nil || len(ifs) == 0 {
		t.Skip("no usable interfaces")
	}
	iface := ifs[0]
	tr, err := NewUDPTransportOnInterface(iface, 0, NewNoOpLogger())
	if err != nil {
		t.Fatalf("NewUDPTransportOnInterface: %v", err)
	}
	defer tr.Close()
	addr := tr.LocalAddr().String()
	if !strings.Contains(addr, iface.Addr.String()) {
		t.Errorf("LocalAddr = %s, want it to contain %s", addr, iface.Addr)
	}
}
```

Add `"strings"` import to that test file if missing.

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run TestNewUDPTransportOnInterfaceBindsToAddr ./udap/ -v
```

Expected: FAIL — `NewUDPTransportOnInterface` not defined.

- [ ] **Step 3: Add the constructor to udap/transport.go**

Append to `udap/transport.go`:

```go
// NewUDPTransportOnInterface binds a UDP socket to the given
// interface's IPv4 address (so the kernel routes outbound packets out
// that NIC) and sends broadcasts to the interface's directed broadcast
// address rather than to 255.255.255.255. This is the fan-out building
// block used by MultiTransport, and also the single-interface mode
// used when --interface NAME is passed.
func NewUDPTransportOnInterface(iface NetInterface, port int, logger Logger) (*UDPTransport, error) {
	if iface.Addr == nil {
		return nil, fmt.Errorf("interface %s has no IPv4 address", iface.Name)
	}
	if iface.Broadcast == nil {
		return nil, fmt.Errorf("interface %s has no broadcast address", iface.Name)
	}
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", iface.Addr, port))
	if err != nil {
		return nil, fmt.Errorf("resolve UDP addr for %s: %w", iface.Name, err)
	}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return nil, fmt.Errorf("listen UDP on %s: %w", iface.Name, err)
	}
	enableBroadcast(conn, logger)
	logger.Debug("UDPTransport bound to interface",
		"interface", iface.Name, "address", conn.LocalAddr().String(),
		"broadcast", iface.Broadcast.String())
	return &UDPTransport{
		conn:        conn,
		logger:      logger,
		broadcastIP: iface.Broadcast,
	}, nil
}
```

The `UDPTransport` struct gains a `broadcastIP net.IP` field. Update the struct definition (around line 36):

```go
type UDPTransport struct {
	conn        *net.UDPConn
	logger      Logger
	broadcastIP net.IP // nil → fall back to 255.255.255.255 (default constructor)
}
```

Update `Send` (around line 58-68) to use it:

```go
func (t *UDPTransport) Send(packet []byte) error {
	dstStr := "255.255.255.255"
	if t.broadcastIP != nil {
		dstStr = t.broadcastIP.String()
	}
	dst, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", dstStr, Port))
	if err != nil {
		return fmt.Errorf("resolve broadcast addr: %w", err)
	}
	if _, err := t.conn.WriteToUDP(packet, dst); err != nil {
		return fmt.Errorf("UDP send: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run TestNewUDPTransportOnInterfaceBindsToAddr ./udap/ -v
```

Expected: PASS or SKIP (if no usable interfaces).

- [ ] **Step 5: Run full udap tests to confirm `NewUDPTransport` still works**

```bash
go test ./udap/...
```

Expected: PASS. (`NewUDPTransport` leaves `broadcastIP` nil, so Send falls back to 255.255.255.255 — original behaviour preserved.)

- [ ] **Step 6: Commit**

```bash
git add udap/transport.go udap/transport_test.go
git commit -m "feat(udap): add NewUDPTransportOnInterface for per-interface binding"
```

### Task 6.2: NewClientForInterface

**Files:**
- Modify: `udap/client.go`
- Test: append to `udap/client_test.go` (file exists)

- [ ] **Step 1: Write the failing test**

Append to `udap/client_test.go`:

```go
func TestNewClientForInterfaceRejectsUnknownName(t *testing.T) {
	_, err := NewClientForInterface("nonexistent-xyz-iface", NewNoOpLogger())
	if err == nil {
		t.Fatal("NewClientForInterface with nonexistent name returned nil error")
	}
}

func TestNewClientForInterfaceAcceptsKnownName(t *testing.T) {
	ifs, err := EnumerateInterfaces()
	if err != nil || len(ifs) == 0 {
		t.Skip("no usable interfaces")
	}
	c, err := NewClientForInterface(ifs[0].Name, NewNoOpLogger())
	if err != nil {
		t.Fatalf("NewClientForInterface(%q): %v", ifs[0].Name, err)
	}
	defer c.Close()
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run "TestNewClientForInterface" ./udap/ -v
```

Expected: FAIL — symbol undefined.

- [ ] **Step 3: Add constructor**

Append to `udap/client.go`:

```go
// NewClientForInterface constructs a Client whose UDP transport is
// bound to the given interface name's IPv4 address. Used by the CLI's
// --interface NAME flag. Errors if the interface does not exist, is
// down, lacks an IPv4 address, or is not broadcast-capable.
func NewClientForInterface(name string, logger Logger) (*Client, error) {
	ifs, err := EnumerateInterfaces()
	if err != nil {
		return nil, fmt.Errorf("enumerate interfaces: %w", err)
	}
	for _, iface := range ifs {
		if iface.Name == name {
			tr, err := NewUDPTransportOnInterface(iface, Port, logger)
			if err != nil {
				return nil, err
			}
			return NewClientWithTransport(tr, logger), nil
		}
	}
	return nil, fmt.Errorf("interface %q is not usable (must be up, broadcast-capable, with an IPv4 address)", name)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test -run "TestNewClientForInterface" ./udap/ -v
```

Expected: PASS (or SKIP for the accept case).

- [ ] **Step 5: Commit**

```bash
git add udap/client.go udap/client_test.go
git commit -m "feat(udap): add NewClientForInterface constructor"
```

### Task 6.3: CLI `--interface NAME` global flag

**Files:**
- Modify: `cli/cli.go`
- Modify: `cli/discover.go` (and any subcommand that uses `newClient`)

- [ ] **Step 1: Inspect newClient consumers**

```bash
grep -rn "newClient(" cli/
```

All subcommand files (discover, info, get, set, read, reboot, getip) call `newClient(*verbose, stderr)`. The seam signature is `func(verbose bool, stderr io.Writer) (*udap.Client, error)`. We need to thread the chosen interface into this call.

Approach: change `newClient` to a package-global `clientFactory` struct with both flags, and update every callsite to use it. Simpler alternative: keep `newClient` as-is but read interface flags from a package-global `globalState` set up by `Run`.

Use the simpler alternative — package globals.

- [ ] **Step 2: Write the failing test (CLI flag-parse rules)**

Create `cli/e2e_interface_flag_test.go`:

```go
package cli

import (
	"bytes"
	"testing"
)

func TestE2EInterfaceAndAllInterfacesMutuallyExclusive(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	err := Run([]string{"--interface", "eth0", "--all-interfaces", "discover"}, &outBuf, &errBuf)
	if ExitCode(err) != 1 {
		t.Errorf("exit code %d, want 1 (flag conflict)", ExitCode(err))
	}
}

func TestE2EInterfaceUnknownNameIsExitOne(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	err := Run([]string{"--interface", "definitely-not-a-real-interface", "discover", "--timeout", "100ms"}, &outBuf, &errBuf)
	if ExitCode(err) != 1 {
		t.Errorf("exit code %d, want 1 (unknown interface)", ExitCode(err))
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test -run TestE2EInterface ./cli/ -v
```

Expected: FAIL — flags not recognised.

- [ ] **Step 4: Add interface flag plumbing to cli/cli.go**

In `cli/cli.go`, add a package-global state struct (declare near the top after imports):

```go
// interfaceSelection captures the global --interface / --all-interfaces
// flags. The chosen mode determines which Client constructor newClient
// uses. Mutated only by Run before any subcommand executes.
type interfaceSelection struct {
	name string // empty unless --interface was set
	all  bool   // true if --all-interfaces was set
}

var currentInterfaceSelection interfaceSelection
```

Extend `globalFlagsBoolean` and `globalFlagsValue`:

```go
var (
	globalFlagsBoolean = map[string]bool{
		"-v":               true,
		"--verbose":        true,
		"--all-interfaces": true,
	}
	globalFlagsValue = map[string]bool{
		"--timeout":   true,
		"--interface": true,
	}
)
```

Update `Run` to extract the flags before dispatch. Insert near the top of `Run`, after `args = moveGlobalFlagsAfterSubcommand(args)`:

```go
	// Extract --interface / --all-interfaces from the moved-into-place
	// argv. moveGlobalFlagsAfterSubcommand has already validated the
	// shape (i.e. global flags are now positioned after args[0]).
	sel, remaining, err := extractInterfaceFlags(args)
	if err != nil {
		return err
	}
	if sel.name != "" && sel.all {
		return &ExitError{Code: 1, Err: fmt.Errorf("--interface and --all-interfaces are mutually exclusive")}
	}
	prevSel := currentInterfaceSelection
	currentInterfaceSelection = sel
	defer func() { currentInterfaceSelection = prevSel }()
	args = remaining
```

Add the helper:

```go
// extractInterfaceFlags scans args for --interface NAME and
// --all-interfaces (in either --foo=bar or --foo bar form), removes
// them, and returns the leftover argv plus the parsed selection.
func extractInterfaceFlags(args []string) (interfaceSelection, []string, error) {
	var sel interfaceSelection
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--all-interfaces":
			sel.all = true
		case strings.HasPrefix(a, "--interface="):
			sel.name = strings.TrimPrefix(a, "--interface=")
		case a == "--interface":
			if i+1 >= len(args) {
				return sel, nil, &ExitError{Code: 1, Err: fmt.Errorf("--interface requires a value")}
			}
			sel.name = args[i+1]
			i++
		default:
			out = append(out, a)
		}
	}
	return sel, out, nil
}
```

Update the `newClient` seam in `cli/discover.go` to consult the selection:

```go
var newClient = func(verbose bool, stderr io.Writer) (*udap.Client, error) {
	logger := udap.NewStructuredLoggerWith(stderr)
	if verbose {
		logger.SetLevel(udap.LogLevelDebug)
	} else {
		logger.SetLevel(udap.LogLevelWarn)
	}
	sel := currentInterfaceSelection
	switch {
	case sel.name != "":
		return udap.NewClientForInterface(sel.name, logger)
	case sel.all:
		return udap.NewClientForAllInterfaces(logger)
	default:
		return udap.NewClientWithLogger(logger)
	}
}
```

Note: `NewClientForAllInterfaces` doesn't exist yet — Phase 7 adds it. Leave the call here; Phase 7 will satisfy the compile.

Hmm — this introduces a compile-time gap. Solution: stub `NewClientForAllInterfaces` now in `udap/client.go` with a "not implemented yet" error, and replace its body in Phase 7. Add the stub:

```go
// NewClientForAllInterfaces constructs a Client whose UDP transport
// fans out to every usable interface returned by EnumerateInterfaces.
// Implemented in Phase 7 — currently returns an error.
//
// TODO(getip-phase-7): real implementation.
func NewClientForAllInterfaces(logger Logger) (*Client, error) {
	return nil, fmt.Errorf("--all-interfaces not yet implemented")
}
```

Acceptable because Phase 7 lands the real implementation, and the `--all-interfaces` test in Phase 6 only checks the mutual-exclusion error path, which doesn't exercise this stub.

Update `printUsage` to mention the new flags:

```
Global flags:
  --timeout DURATION  Operation timeout (default 5s)
  --interface NAME    Bind discovery to one network interface
  --all-interfaces    Broadcast on every usable interface (fan-out)
  --verbose, -v       Debug logging to stderr
  --version           Print version and exit
  --help, -h          Print this help
```

- [ ] **Step 5: Run test to verify it passes**

```bash
go test -run TestE2EInterface ./cli/ -v
```

Expected: PASS for both — mutual exclusion (exit 1) and unknown interface (exit 1).

- [ ] **Step 6: Run full cli suite to confirm no regression**

```bash
go test ./cli/...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add cli/cli.go cli/discover.go cli/e2e_interface_flag_test.go udap/client.go
git commit -m "feat(cli): add --interface NAME global flag with mutual-exclusion guard"
```

---

## Phase 7 — `--all-interfaces` and MultiTransport

### Task 7.1: MultiTransport — Send fans out

**Files:**
- Create: `udap/multi_transport.go`
- Create: `udap/multi_transport_test.go`

- [ ] **Step 1: Write the failing test**

```go
package udap

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeTransport is a controllable Transport for MultiTransport tests.
type fakeTransport struct {
	sent       [][]byte
	sendErr    error
	recvCh     chan recvOut
	closed     bool
}

type recvOut struct {
	pkt []byte
	src string
	err error
}

func newFakeTransport() *fakeTransport {
	return &fakeTransport{recvCh: make(chan recvOut, 8)}
}

func (f *fakeTransport) Send(p []byte) error {
	if f.sendErr != nil {
		return f.sendErr
	}
	f.sent = append(f.sent, p)
	return nil
}

func (f *fakeTransport) Recv(ctx context.Context) ([]byte, string, error) {
	select {
	case <-ctx.Done():
		return nil, "", ctx.Err()
	case ro, ok := <-f.recvCh:
		if !ok {
			return nil, "", errors.New("transport closed")
		}
		return ro.pkt, ro.src, ro.err
	}
}

func (f *fakeTransport) Close() error {
	if !f.closed {
		f.closed = true
		close(f.recvCh)
	}
	return nil
}

func TestMultiTransportSendFansOut(t *testing.T) {
	a, b := newFakeTransport(), newFakeTransport()
	mt := NewMultiTransport([]Transport{a, b}, NewNoOpLogger())
	defer mt.Close()

	if err := mt.Send([]byte{1, 2, 3}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(a.sent) != 1 || len(b.sent) != 1 {
		t.Errorf("expected each child to receive one send, got a=%d b=%d", len(a.sent), len(b.sent))
	}
}

func TestMultiTransportSendSucceedsIfAnyChildSucceeds(t *testing.T) {
	a, b := newFakeTransport(), newFakeTransport()
	a.sendErr = errors.New("child A broken")
	mt := NewMultiTransport([]Transport{a, b}, NewNoOpLogger())
	defer mt.Close()
	if err := mt.Send([]byte{1}); err != nil {
		t.Errorf("Send should succeed when at least one child succeeds, got %v", err)
	}
	if len(b.sent) != 1 {
		t.Errorf("expected B to receive the packet, got %d sends", len(b.sent))
	}
}

func TestMultiTransportSendFailsWhenAllChildrenFail(t *testing.T) {
	a, b := newFakeTransport(), newFakeTransport()
	a.sendErr = errors.New("A broken")
	b.sendErr = errors.New("B broken")
	mt := NewMultiTransport([]Transport{a, b}, NewNoOpLogger())
	defer mt.Close()
	if err := mt.Send([]byte{1}); err == nil {
		t.Error("Send should fail when all children fail")
	}
}

func TestMultiTransportRecvMergesReplies(t *testing.T) {
	a, b := newFakeTransport(), newFakeTransport()
	mt := NewMultiTransport([]Transport{a, b}, NewNoOpLogger())
	defer mt.Close()

	a.recvCh <- recvOut{pkt: []byte{0x0a}, src: "ifA"}
	b.recvCh <- recvOut{pkt: []byte{0x0b}, src: "ifB"}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	got := make(map[byte]bool)
	for i := 0; i < 2; i++ {
		pkt, _, err := mt.Recv(ctx)
		if err != nil {
			t.Fatalf("Recv: %v", err)
		}
		got[pkt[0]] = true
	}
	if !got[0x0a] || !got[0x0b] {
		t.Errorf("did not see both replies; got %v", got)
	}
}

func TestMultiTransportRecvCancelledByCtx(t *testing.T) {
	a := newFakeTransport()
	mt := NewMultiTransport([]Transport{a}, NewNoOpLogger())
	defer mt.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, _, err := mt.Recv(ctx)
	if err == nil {
		t.Error("Recv should return ctx error after deadline")
	}
}

func TestMultiTransportCloseClosesChildren(t *testing.T) {
	a, b := newFakeTransport(), newFakeTransport()
	mt := NewMultiTransport([]Transport{a, b}, NewNoOpLogger())
	if err := mt.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	if !a.closed || !b.closed {
		t.Errorf("expected both children closed; a=%v b=%v", a.closed, b.closed)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run TestMultiTransport ./udap/ -v
```

Expected: FAIL — `MultiTransport` not defined.

- [ ] **Step 3: Create udap/multi_transport.go**

```go
package udap

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// MultiTransport composes a set of child Transports. Send fans out to
// all of them; Recv merges replies through per-child goroutines into a
// shared channel. The Client doesn't care whether its transport is a
// single UDPTransport or a MultiTransport — both satisfy the Transport
// port.
//
// Used by --all-interfaces: one UDPTransport per usable interface,
// composed into a MultiTransport.
type MultiTransport struct {
	children []Transport
	logger   Logger

	startOnce sync.Once
	merged    chan multiTransportRecv
	stop      chan struct{}
	wg        sync.WaitGroup

	closeOnce sync.Once
	closeErr  error
}

type multiTransportRecv struct {
	pkt []byte
	src string
	err error
}

// NewMultiTransport constructs a MultiTransport composing the given
// children. The slice must contain at least one transport (callers
// should reject empty fan-out upstream so the error message can mention
// "no usable interfaces").
func NewMultiTransport(children []Transport, logger Logger) *MultiTransport {
	return &MultiTransport{
		children: children,
		logger:   logger,
		merged:   make(chan multiTransportRecv, 32),
		stop:     make(chan struct{}),
	}
}

// Send broadcasts the packet on every child. Returns success if at
// least one child succeeded; aggregated error only if every child
// failed. Per-child errors are logged at Warn.
func (m *MultiTransport) Send(packet []byte) error {
	var failures []string
	successes := 0
	for i, c := range m.children {
		if err := c.Send(packet); err != nil {
			m.logger.Warn("MultiTransport child send failed", "child", i, "error", err)
			failures = append(failures, err.Error())
			continue
		}
		successes++
	}
	if successes == 0 {
		return fmt.Errorf("all children failed: %v", failures)
	}
	return nil
}

// Recv returns the next packet from any child, or the context error if
// ctx is cancelled. Lazily starts one goroutine per child on first
// call.
func (m *MultiTransport) Recv(ctx context.Context) ([]byte, string, error) {
	m.startOnce.Do(m.spawnPumps)
	select {
	case <-ctx.Done():
		return nil, "", ctx.Err()
	case ro, ok := <-m.merged:
		if !ok {
			return nil, "", errors.New("multi transport closed")
		}
		if ro.err != nil {
			return nil, "", ro.err
		}
		return ro.pkt, ro.src, nil
	}
}

// spawnPumps starts one goroutine per child that forwards packets to
// m.merged until the stop channel closes.
func (m *MultiTransport) spawnPumps() {
	for i, c := range m.children {
		m.wg.Add(1)
		go m.pumpChild(i, c)
	}
}

func (m *MultiTransport) pumpChild(idx int, c Transport) {
	defer m.wg.Done()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// One watcher goroutine per pump that translates stop → ctx cancel.
	go func() {
		<-m.stop
		cancel()
	}()
	for {
		pkt, src, err := c.Recv(ctx)
		if err != nil {
			// On error (including ctx cancel from Close), pump exits.
			// Don't forward errors except as a debug log.
			select {
			case <-m.stop:
				// normal shutdown
			default:
				m.logger.Warn("MultiTransport child recv error",
					"child", idx, "error", err)
			}
			return
		}
		select {
		case m.merged <- multiTransportRecv{pkt: pkt, src: src}:
		case <-m.stop:
			return
		}
	}
}

// Close closes all children and signals the pump goroutines to exit.
// Returns the first non-nil child Close error.
func (m *MultiTransport) Close() error {
	m.closeOnce.Do(func() {
		close(m.stop)
		for _, c := range m.children {
			if err := c.Close(); err != nil && m.closeErr == nil {
				m.closeErr = err
			}
		}
		m.wg.Wait()
	})
	return m.closeErr
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test -run TestMultiTransport ./udap/ -v
```

Expected: PASS for all six subtests.

- [ ] **Step 5: Commit**

```bash
git add udap/multi_transport.go udap/multi_transport_test.go
git commit -m "feat(udap): add MultiTransport composite Transport"
```

### Task 7.2: NewClientForAllInterfaces — real implementation

**Files:**
- Modify: `udap/client.go`
- Test: append to `udap/client_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestNewClientForAllInterfacesErrorsWhenNoUsableInterfaces(t *testing.T) {
	// We can't reliably create a "no interfaces" condition on a real
	// host, so this test confirms only that the stub error is gone
	// (we now return a real client OR a real error from binding).
	ifs, _ := EnumerateInterfaces()
	if len(ifs) == 0 {
		t.Skip("can't test success path with zero interfaces")
	}
	c, err := NewClientForAllInterfaces(NewNoOpLogger())
	if err != nil {
		// May fail to bind to port 17784 (privileged on Linux); that's
		// OK — verify it isn't the stub error.
		if strings.Contains(err.Error(), "not yet implemented") {
			t.Errorf("got stub error, want real binding attempt")
		}
		return
	}
	c.Close()
}
```

Add `"strings"` to imports if not present.

- [ ] **Step 2: Run test to verify it fails**

```bash
go test -run TestNewClientForAllInterfacesErrorsWhenNoUsableInterfaces ./udap/ -v
```

Expected: FAIL — gets the "not yet implemented" stub error.

- [ ] **Step 3: Replace the stub with the real implementation**

In `udap/client.go`, replace the stub `NewClientForAllInterfaces` with:

```go
// NewClientForAllInterfaces constructs a Client whose UDP transport
// fans out to every usable interface returned by EnumerateInterfaces.
// Children that fail to bind are skipped with a Warn log; if no
// children succeed, returns an error.
func NewClientForAllInterfaces(logger Logger) (*Client, error) {
	ifs, err := EnumerateInterfaces()
	if err != nil {
		return nil, fmt.Errorf("enumerate interfaces: %w", err)
	}
	if len(ifs) == 0 {
		return nil, fmt.Errorf("no usable interfaces found")
	}
	children := make([]Transport, 0, len(ifs))
	for _, iface := range ifs {
		tr, err := NewUDPTransportOnInterface(iface, Port, logger)
		if err != nil {
			logger.Warn("skipping interface (bind failed)",
				"interface", iface.Name, "error", err)
			continue
		}
		children = append(children, tr)
	}
	if len(children) == 0 {
		return nil, fmt.Errorf("failed to bind on any usable interface")
	}
	return NewClientWithTransport(NewMultiTransport(children, logger), logger), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test -run TestNewClientForAllInterfacesErrorsWhenNoUsableInterfaces ./udap/ -v
```

Expected: PASS (real binding attempt — succeeds, or fails with a non-stub error).

- [ ] **Step 5: Commit**

```bash
git add udap/client.go udap/client_test.go
git commit -m "feat(udap): implement NewClientForAllInterfaces fan-out"
```

### Task 7.3: Post-implementation broadcast verification spike

**Files:** none (manual, like Phase 1)

- [ ] **Step 1: Rebuild**

```bash
task build
```

- [ ] **Step 2: tcpdump in one terminal**

```bash
sudo tcpdump -i any -n -nn 'udp port 17784'
```

- [ ] **Step 3: Run discovery with fan-out**

```bash
./go-udap --all-interfaces discover
```

- [ ] **Step 4: Observe**

Expected: outbound `udp.17784` packets visible on each usable interface (one packet per interface), with destination = each interface's directed-broadcast address (not `255.255.255.255`).

Append the observation to the bottom of `docs/superpowers/plans/2026-05-13-getip-hwrev-uuid-iface.md` under the existing `## Spike result` section. Commit:

```bash
git add docs/superpowers/plans/2026-05-13-getip-hwrev-uuid-iface.md
git commit -m "docs(plan): record post-implementation fan-out spike result"
```

---

## Final verification

### Task 8.1: Full test suite

- [ ] **Step 1: Run everything**

```bash
go test ./...
```

Expected: PASS for every package.

- [ ] **Step 2: Run linters / formatters**

```bash
task fmt && task lint
```

Expected: no diffs from `task fmt`; no vet errors from `task lint`.

- [ ] **Step 3: Build all targets**

```bash
task build:all
```

Expected: binaries for macOS, Linux (amd64/arm64), Windows.

### Task 8.2: Update README / CLAUDE.md if needed

- [ ] **Step 1: Inspect CLAUDE.md CLI Commands section**

```bash
grep -A 30 "CLI Commands" CLAUDE.md
```

- [ ] **Step 2: Add lines for getip and interfaces**

In `CLAUDE.md`, under "CLI Commands (when running the tool)", add:

```
- `go-udap getip <mac>` — Query device IP / subnet / gateway via UCP get_ip (UCP method 0x0002)
- `go-udap interfaces` — List network interfaces usable for discovery
```

Under "Global flags", add:

```
`--interface NAME` (restrict broadcasts to one interface), `--all-interfaces` (broadcast on every usable interface).
```

- [ ] **Step 3: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: document getip and interfaces subcommands + interface flags"
```

---

## Spike result

❯ sudo tcpdump -i any -n -nn 'udp port 17784'
tcpdump: data link type PKTAP
tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on any, link-type PKTAP (Apple DLT_PKTAP), snapshot length 524288 bytes
14:40:33.160298 IP 192.168.1.226.17784 > 255.255.255.255.17784: UDP, length 27
14:40:33.160300 IP 192.168.1.226.17784 > 255.255.255.255.17784: UDP, length 27
14:40:33.160304 IP 192.168.1.226.17784 > 255.255.255.255.17784: UDP, length 27
14:40:33.160660 IP 0.0.0.0.17784 > 255.255.255.255.17784: UDP, length 61
14:40:33.399624 IP 192.168.1.226.17784 > 255.255.255.255.17784: UDP, length 27
14:40:33.399626 IP 0.0.0.0.17784 > 255.255.255.255.17784: UDP, length 61
