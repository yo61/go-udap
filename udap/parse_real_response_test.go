package udap

import (
	"encoding/hex"
	"testing"
)

// TestParseRealSBRDiscoveryResponse uses a real captured discovery
// response (firmware 77, factory state) to verify parseDiscoveryResponse
// extracts the device fields correctly.
//
// The capture came from the cli-redesign branch test session; see
// https://github.com/yo61/go-udap (issue: "info doesn't return
// all info"). 2 SBRs on the LAN; both responded identically apart from
// MAC. The 34-byte TLV payload contains 6 TLVs:
//
//	0c 04 "init"           type=0x0c
//	0b 02 "07"             type=0x0b
//	0a 04 "0005"           type=0x0a
//	09 02 "77"             type=0x09 (firmware-version-shaped)
//	03 0a "squeezebox"     type=0x03 (Model)
//	02 00                  type=0x02 (Name, empty in factory state)
func TestParseRealSBRDiscoveryResponse(t *testing.T) {
	raw, err := hex.DecodeString(
		"0001000000000000000100042016058f0001c001000001000100090c04696e69740b0230370a043030303509023737030a73717565657a65626f780200",
	)
	if err != nil {
		t.Fatalf("hex decode: %v", err)
	}
	if len(raw) != 61 {
		t.Fatalf("expected 61 bytes, got %d", len(raw))
	}

	packet, data, err := ParsePacket(raw)
	if err != nil {
		t.Fatalf("ParsePacket: %v", err)
	}
	if got := len(data); got != 34 {
		t.Errorf("expected 34-byte TLV payload, got %d", got)
	}

	c := &Client{logger: NewNoOpLogger()}
	device := c.parseDiscoveryResponse(data, "0.0.0.0", packet)
	if device == nil {
		t.Fatal("parseDiscoveryResponse returned nil")
	}

	t.Logf("MAC=%q Name=%q Model=%q Firmware=%q State=%q",
		device.MAC, device.Name, device.Model, device.Firmware, device.State)

	if device.MAC != "00:04:20:16:05:8f" {
		t.Errorf("MAC: got %q, want 00:04:20:16:05:8f", device.MAC)
	}
	// Empty TLV 0x02 in this capture means the device has no configured
	// hostname; parser substitutes "Squeezebox Device" so the user sees
	// something — accept that here.
	if device.Name == "" {
		t.Errorf("Name unexpectedly empty (default fallback should fire)")
	}
	// Model is computed: device_type (TLV 0x03 "squeezebox") + device_id
	// (TLV 0x0b "07") → "Squeezebox Receiver" via productNameByID lookup.
	if device.Model != "Squeezebox Receiver" {
		t.Errorf("Model: got %q, want Squeezebox Receiver", device.Model)
	}
	if device.Firmware != "77" {
		t.Errorf("Firmware: got %q, want 77", device.Firmware)
	}
	if device.State != "init" {
		t.Errorf("State: got %q, want init", device.State)
	}
}
