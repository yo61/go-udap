// Package mocksbr provides a software mock of a Squeezebox Receiver (SBR)
// for testing go-udap and the udap package without real hardware.
package mocksbr

import (
	"maps"
	"sync"
	"time"

	"go-udap/udap"
)

// factoryDefaults is the NVRAM contents of every freshly-instantiated
// virtual device. Values come from the FactoryDefault column of
// udap.Parameters, captured against a real Squeezebox Receiver after a
// hardware factory reset.
func factoryDefaults() map[string]string {
	out := make(map[string]string, len(udap.Parameters))
	for _, p := range udap.Parameters {
		out[p.Name] = p.FactoryDefault
	}
	return out
}

// DeviceConfig is the per-device knob set used by Network.Add and
// (eventually) cmd/mocksbr's --device flag.
type DeviceConfig struct {
	MAC      string // required; must be a valid MAC
	Name     string // optional; defaults to "Mock SBR <n>"
	Model    string // device_type TLV; defaults to "squeezebox"
	DeviceID string // device_id TLV (2-char hex); defaults to "07" (Receiver)
	Firmware string // firmware_rev TLV; defaults to "77"
	Hardware string // hardware_rev TLV; defaults to "0005"
	UUID     string // optional informational; defaults to "mock-sbr-<n>"

	// Network configuration reported by the get_ip operation
	// (UCP method 0x0002). All optional — empty values are emitted
	// as zero IPs (0.0.0.0) in the wire response.
	IP         string // e.g. "192.168.1.50"
	SubnetMask string // e.g. "255.255.255.0"
	Gateway    string // e.g. "192.168.1.1"

	// Phase 2/3 fields are present in the type so its public surface is
	// stable, but Phase 1 ignores them.
	NVRAM       map[string]string
	FailOn      []Op
	Slow        time.Duration
	Unreachable bool
	RebootDelay time.Duration

	// DropGetIP makes the device silently ignore get_ip requests
	// (for timeout testing).
	DropGetIP bool

	// Malformed selects a deliberately broken response shape used by
	// tests that exercise the client's error-handling path.
	Malformed MalformedMode
}

// MalformedMode selects a deliberately broken GetData response shape.
// Used by tests that exercise the client's error-handling path.
type MalformedMode int

const (
	// MalformedNone is the default — well-formed responses.
	MalformedNone MalformedMode = iota
	// MalformedOversizedCount declares count=65535 with no item bodies,
	// triggering the client's per-item bounds check.
	MalformedOversizedCount
	// MalformedLengthExceedsPayload writes one item header whose
	// declared length would extend past the payload, triggering the
	// client's "item exceeds payload" branch.
	MalformedLengthExceedsPayload
	// MalformedUnknownMethod replaces the response's UCPMethod with an
	// unrecognized value so the client takes its "unexpected method"
	// branch.
	MalformedUnknownMethod
)

// Op identifies a UDAP operation for failure-injection knobs (Phase 3).
type Op string

const (
	OpDiscover Op = "discover"
	OpGet      Op = "get"
	OpSet      Op = "set"
	OpSave     Op = "save"
	OpReset    Op = "reset"
	OpGetIP    Op = "getip"
)

// defaultRebootDelay is the post-Reset window during which the device
// drops every incoming packet, then reloads working memory from NVRAM.
// Real SBRs take ~10s; tests want fast iteration.
const defaultRebootDelay = 100 * time.Millisecond

// device is one virtual SBR. Internal type — the public surface is
// Network and DeviceConfig.
type device struct {
	mu             sync.Mutex
	cfg            DeviceConfig
	workingMemory  map[string]string
	nvram          map[string]string
	rebootDeadline time.Time // zero unless mid-reboot
}

// newDevice constructs a device in factory state. cfg.MAC must be set.
func newDevice(cfg DeviceConfig) *device {
	if cfg.Model == "" {
		cfg.Model = "squeezebox"
	}
	if cfg.DeviceID == "" {
		cfg.DeviceID = "07"
	}
	if cfg.Firmware == "" {
		cfg.Firmware = "77"
	}
	if cfg.Hardware == "" {
		cfg.Hardware = "0005"
	}
	if cfg.RebootDelay == 0 {
		cfg.RebootDelay = defaultRebootDelay
	}
	d := &device{cfg: cfg}
	d.workingMemory = factoryDefaults()
	d.nvram = factoryDefaults()
	if cfg.NVRAM != nil {
		for k, v := range cfg.NVRAM {
			d.workingMemory[k] = v
			d.nvram[k] = v
		}
	}
	// Set hostname from Name so discovery and GetData agree.
	if cfg.Name != "" {
		d.workingMemory["hostname"] = cfg.Name
		d.nvram["hostname"] = cfg.Name
	}
	return d
}

// applySet mutates working memory.
func (d *device) applySet(params map[string]string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	maps.Copy(d.workingMemory, params)
}

// applySave copies working memory to NVRAM.
func (d *device) applySave() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.nvram = cloneMap(d.workingMemory)
}

// applyReset reloads working memory from NVRAM. The reboot window is
// managed by the caller (Network), which serves the Reset packet first
// and then enters the deadline.
func (d *device) applyReset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.workingMemory = cloneMap(d.nvram)
}

// snapshotWorking returns a copy of working memory.
func (d *device) snapshotWorking() map[string]string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return cloneMap(d.workingMemory)
}

// startReboot records the post-Reset window during which incoming
// packets are silently dropped. Returns the time at which the device
// becomes responsive again.
func (d *device) startReboot() time.Time {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.rebootDeadline = time.Now().Add(d.cfg.RebootDelay)
	return d.rebootDeadline
}

// rebooting reports whether the device is currently in its post-Reset
// drop-all-packets window. Side effect: when the window has just
// expired, rebooting reloads working memory from NVRAM (a real device's
// "boot loaded the saved config" moment).
func (d *device) rebooting() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.rebootDeadline.IsZero() {
		return false
	}
	if time.Now().Before(d.rebootDeadline) {
		return true
	}
	d.rebootDeadline = time.Time{}
	d.workingMemory = cloneMap(d.nvram)
	return false
}

// failsOn reports whether the device is configured (via FailOn) to
// reject the given op with a MethodError response. SetData and SaveData
// share one wire method, so OpSet and OpSave are treated as aliases.
func (d *device) failsOn(op Op) bool {
	for _, configured := range d.cfg.FailOn {
		if configured == op {
			return true
		}
		if op == OpSet && configured == OpSave {
			return true
		}
		if op == OpSave && configured == OpSet {
			return true
		}
	}
	return false
}

// state returns the device_status TLV value reported in discovery.
// Factory-default devices report "init"; configured devices (anything
// with a non-empty hostname) report "wait_slimserver".
func (d *device) state() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.workingMemory["hostname"] != "" {
		return "wait_slimserver"
	}
	return "init"
}

func cloneMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	maps.Copy(out, m)
	return out
}
