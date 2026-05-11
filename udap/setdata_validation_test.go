package udap

import (
	"strings"
	"testing"
)

// TestCreateSetDataPacketRejectsInvalidIP locks in the fix for review
// finding #3. Pre-fix, an unparseable IPv4 value on a length-4 NVRAM
// parameter logged a warning and silently fell through to a zero-
// filled buffer, so the packet sent to the device wrote 0.0.0.0 to
// NVRAM. Post-fix, CreateSetDataPacket returns an error.
func TestCreateSetDataPacketRejectsInvalidIP(t *testing.T) {
	c := newTestClient(t)
	dev := &Device{MAC: MustParseMAC("00:04:20:00:00:01")}
	_, err := c.CreateSetDataPacket(dev, map[string]string{
		"server_address": "192.168.1.x",
	})
	if err == nil {
		t.Fatalf("expected error for invalid IP, got nil")
	}
	if !strings.Contains(err.Error(), "server_address") {
		t.Errorf("error %q should name the offending param", err)
	}
}

func TestCreateSetDataPacketRejectsInvalidUint8(t *testing.T) {
	c := newTestClient(t)
	dev := &Device{MAC: MustParseMAC("00:04:20:00:00:01")}
	_, err := c.CreateSetDataPacket(dev, map[string]string{
		"wireless_channel": "999",
	})
	if err == nil {
		t.Fatalf("expected error for out-of-range uint8, got nil")
	}
}

func TestCreateSetDataPacketRejectsInvalidUint16(t *testing.T) {
	c := newTestClient(t)
	dev := &Device{MAC: MustParseMAC("00:04:20:00:00:01")}
	_, err := c.CreateSetDataPacket(dev, map[string]string{
		"wireless_region_id": "999999",
	})
	if err == nil {
		t.Fatalf("expected error for out-of-range uint16, got nil")
	}
}

func TestCreateSetDataPacketAcceptsValidValues(t *testing.T) {
	c := newTestClient(t)
	dev := &Device{MAC: MustParseMAC("00:04:20:00:00:01")}
	pkt, err := c.CreateSetDataPacket(dev, map[string]string{
		"server_address": "10.0.0.1",
		"hostname":       "test-host",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkt) == 0 {
		t.Errorf("expected non-empty packet")
	}
}
