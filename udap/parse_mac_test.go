package udap

import (
	"strings"
	"testing"
)

// TestCreateGetDataPacketRejectsBadMAC, TestCreateSetDataPacketRejectsBadMAC,
// and TestCreateResetPacketRejectsBadMAC lock in the fix for review
// finding #8 — parseMACAddress used to silently return an all-zeros
// MAC on Sscanf failure, so a malformed device.MAC produced a unicast
// packet aimed at 00:00:00:00:00:00. Each Create* method now refuses
// to build the packet and returns an error.

func TestCreateGetDataPacketRejectsBadMAC(t *testing.T) {
	c := newTestClient(t)
	dev := &Device{MAC: "not-a-mac"}
	_, err := c.CreateGetDataPacket(dev, []string{"hostname"})
	if err == nil {
		t.Fatalf("expected error for invalid MAC, got nil")
	}
	if !strings.Contains(err.Error(), "MAC") {
		t.Errorf("error %q should mention MAC", err)
	}
}

func TestCreateSetDataPacketRejectsBadMAC(t *testing.T) {
	c := newTestClient(t)
	dev := &Device{MAC: "12:34"}
	_, err := c.CreateSetDataPacket(dev, map[string]string{"hostname": "x"})
	if err == nil {
		t.Fatalf("expected error for invalid MAC, got nil")
	}
}

func TestCreateResetPacketRejectsBadMAC(t *testing.T) {
	c := newTestClient(t)
	dev := &Device{MAC: ""}
	_, err := c.CreateResetPacket(dev)
	if err == nil {
		t.Fatalf("expected error for invalid MAC, got nil")
	}
}

func TestCreateGetDataPacketAcceptsValidMAC(t *testing.T) {
	c := newTestClient(t)
	dev := &Device{MAC: "00:04:20:00:00:01"}
	got, err := c.CreateGetDataPacket(dev, []string{"hostname"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) == 0 {
		t.Errorf("expected non-empty packet")
	}
}

func newTestClient(t *testing.T) *Client {
	t.Helper()
	c, err := newClientWithPort(0, NewNoOpLogger())
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}
