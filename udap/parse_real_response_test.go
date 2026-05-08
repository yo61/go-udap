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
// https://github.com/robinbowes/go-udap (issue: "info doesn't return
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

	t.Logf("MAC=%q Name=%q Model=%q Firmware=%q UUID=%q",
		device.MAC, device.Name, device.Model, device.Firmware, device.UUID)

	if device.MAC != "00:04:20:16:05:8f" {
		t.Errorf("MAC: got %q, want 00:04:20:16:05:8f", device.MAC)
	}
	// Whatever Name is, it shouldn't be the default fallback if a real
	// TLV set it. Empty TLV in this capture means the device has no
	// configured name; but the parser overwrites empty with the
	// "Squeezebox Device" default, so we accept that here.
	if device.Name == "" {
		t.Errorf("Name unexpectedly empty (default fallback should fire)")
	}

	// The model TLV (0x03) clearly contains "squeezebox" — this is the
	// claim under test.
	if device.Model != "squeezebox" {
		t.Errorf("Model: got %q, want squeezebox", device.Model)
	}
}
