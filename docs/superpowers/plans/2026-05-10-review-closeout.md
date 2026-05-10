# Review close-out — 2026-05-10

Brief on the audit + remediation work done in May 2026.
For pickup later: see "Phase 4 — pending" below.

## What shipped

10 patch releases between v1.3.1 and v1.3.9, all gated by failing-first
tests.

| Tag | Bug / change |
|-----|--------------|
| v1.3.0 | mocksbr Phase 3 failure-injection knobs (Unreachable, Slow, FailOn, Malformed, InjectReply) — #18 |
| v1.3.1 | sequence counter race fixed via atomic — #20 |
| v1.3.2 | parseMACAddress returns error instead of zero-MAC — #21 |
| v1.3.3 | ResetDeviceWithContext propagates MethodError — #25 |
| v1.3.4 | parseGetDataResponse map size hint clamped — #22 |
| v1.3.5 | one shared context across discoverAndFind + operation — #26 |
| v1.3.6 | set values validated at flag boundary + CreateSetDataPacket errors — #27 |
| v1.3.7 | SetDeviceConfigWithContext errors when prelude read fails — #28 |
| v1.3.8 | waitForDeviceReply pins source against device.IP — #29 |
| v1.3.9 | sequence wrap explicit + offset_NNN cleanup + reset doc — #31, #32, #33 |

Every original review finding (#1–#10) is closed by a test that failed
without the fix. See `git log --oneline main` for the full sequence.

## Infrastructure now in place

Test net (top to bottom):

- **Subprocess / Layer-3 e2e** (`mocksbr/testhelper/spawn_test.go`) —
  builds `cmd/mocksbr` and drives real UDP round-trips for GetData,
  SetData, Reset, unknown-method, malformed-packet.
- **In-process e2e** (`cli/e2e_*_test.go`) — `cli.Run()` against
  MockTransport for every CLI command.
- **Failure injection** (`mocksbr/failure_injection_test.go`) —
  Unreachable / Slow / FailOn / Malformed / InjectReply cover the
  client's error-handling paths.
- **Mutation testing** — `task mutate` runs go-mutesting on
  udap/getdata_response.go. 98.6% mutation score on a clean run.

Workflow gates:

- **Default Branch ruleset** (in repo settings, no PR) requires green
  CI on the merge result before the merge button enables. semantic-
  release App is in `bypass_actors` so it can push the chore(release)
  commit + tag without deadlocking on the same gate.
- **Release workflow** (`.github/workflows/release.yaml`) is
  `workflow_run`-triggered on a successful CI run, never on a raw
  push — replaced #24's previous race that let v1.3.2 ship from a
  vet-failing tree.
- **prek hooks in CI** (`.github/workflows/ci.yaml::lint`) catches
  gofmt / goimports / trailing-whitespace drift that previously only
  fired on local `git commit`.

## Phase 4 — pending: capture `getdata-response.bin` from real hardware

The mocksbr Phase 1 plan called for a captured-from-real-hardware
fixture for byte-for-byte comparison of the mock's GetData response,
analogous to the existing `discovery-factory.bin`,
`discovery-configured.bin`, and `reset-ack.bin` fixtures. It was
deferred because it requires a Squeezebox Receiver on the LAN.

### Pre-flight

- A Squeezebox Receiver (or other UDAP device) on the LAN.
- The device should be **factory-default** so its NVRAM matches
  mocksbr's `factoryDefaults()`. A `go-udap reboot <mac>` after a
  hard factory-reset is the cleanest starting state.
- `tshark` (or `tcpdump`) and a `go-udap` build on the same machine.

### Capture

```bash
# In one terminal — capture all UDAP traffic.
sudo tshark -i <iface> -w /tmp/udap-capture.pcap -f 'udp port 17784'

# In another terminal — read the device.
go-udap discover  # confirm the MAC
go-udap read --all <mac>  # this issues the GetData

# Stop tshark (Ctrl-C).
```

### Extract the GetData response frame

The capture will contain (in order):

1. discovery broadcast (request, frame N)
2. discovery reply (frame N+1)
3. GetData request from go-udap (frame N+2)
4. **GetData response from the device (frame N+3)** ← this is what we want

```bash
# Inspect the capture; identify the GetData response by UCPMethod=0x0005
# in the response frame (bytes [25:27] of the UDP payload, big-endian).
tshark -r /tmp/udap-capture.pcap -V

# Extract the UDP payload of the chosen frame as raw bytes.
tshark -r /tmp/udap-capture.pcap -Y 'frame.number == <N+3>' \
  -T fields -e data.data | xxd -r -p > mocksbr/testdata/captures/getdata-response.bin
```

Verify size: a 26-param GetData response is 165 bytes header+payload;
exact size depends on what params the device returned. Compare to
`udap/testdata/captures/getdata-response-26params.bin` (which already
exists and is 387 bytes) for shape.

### Add the test

Mirror the existing pattern in `mocksbr/fixture_test.go`:

```go
// TestGetDataResponseMatchesFixture reproduces the captured GetData
// response: 27-byte header, UCPMethod=0x0005, 32 bytes user/pass
// (zeros), uint16 BE count, count × (offset, length, value) tuples.
func TestGetDataResponseMatchesFixture(t *testing.T) {
    want := readFixture(t, "getdata-response.bin")

    net := NewNetwork(0, udap.NewNoOpLogger())
    mac, err := net.Add(DeviceConfig{MAC: "<mac-from-capture>"})
    if err != nil { t.Fatalf("Add: %v", err) }

    // Burn discovery sequences to align with the captured Sequence
    // (real client probably sent discovery first, then GetData=seq 2).
    c := udap.NewClientWithTransport(NewMockTransport(net), udap.NewNoOpLogger())
    defer c.Close()
    _ = c.CreateAdvancedDiscoveryPacket() // burns sequence=1
    dev := &udap.Device{MAC: mac}
    getReq, err := c.CreateGetDataPacket(dev, udap.ParameterNames())
    if err != nil { t.Fatalf("CreateGetDataPacket: %v", err) }

    replies := net.Receive(getReq)
    if len(replies) != 1 {
        t.Fatalf("expected 1 reply, got %d", len(replies))
    }
    if !bytes.Equal(replies[0], want) {
        t.Errorf("GetData response mismatch\n got %x\nwant %x", replies[0], want)
    }
}
```

If the test fails on the first run, the most likely causes are:

- Sequence number mismatch — adjust burn count to match the captured
  Sequence field.
- Source MAC mismatch — the captured device's MAC must match the one
  passed to `net.Add`.
- NVRAM mismatch — the captured device must be factory-default for
  the values to match `factoryDefaults()`. If you can't get a clean
  factory state, capture the GetData of a *known-configured* device
  and seed the mocksbr device's NVRAM via `DeviceConfig.NVRAM`
  (Phase 2 feature; not yet wired into `cmd/mocksbr` flags but the
  field exists on the struct).

### Effort

~30 min on your end with hardware; ~15 min for me to wire up the
test once you commit the bin file.

## Other deferred items

- **`actionlint` stale metadata for `client-id:`** — the action
  `actions/create-github-app-token@v3` accepts the `client-id`
  input but `actionlint`'s embedded metadata doesn't list it. False
  positive on local runs only (CI doesn't run actionlint). Will
  resolve when actionlint upstreams a metadata refresh.

- **`@semantic-release/github` migration** — currently the release
  flow uses `@semantic-release/changelog` + `@semantic-release/git`
  to commit `CHANGELOG.md` back to main. Works because the
  semantic-release App bypasses the ruleset. Switching to
  `@semantic-release/github` (creates a GitHub Release with
  auto-generated tag) would make the CHANGELOG auto-commit
  unnecessary; the release notes live on the GitHub Release page
  instead. Lower priority — current setup is working.
