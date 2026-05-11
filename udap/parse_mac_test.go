package udap

import (
	"strings"
	"testing"
)

// TestCreateGetDataPacketRejectsZeroMAC, TestCreateSetDataPacketRejectsZeroMAC,
// and TestCreateResetPacketRejectsZeroMAC succeed the original
// "rejects-bad-MAC" suite. Pre-MAC-Value-Object, those tests fed
// strings like "not-a-mac" or "12:34" to Device.MAC and asserted that
// each Create*Packet refused to build a packet — the regression of
// concern was a silently-zero-filled DstAddress (review finding #8).
//
// With Device.MAC now a typed MAC value object, malformed strings are
// rejected at construction (ParseMAC errors), so the type system itself
// guards against most of the original failure modes. The one
// remaining reachable case is the zero-value MAC — an uninitialised
// Device — which the Create* methods still refuse to operate on so
// that &Device{} doesn't silently unicast to 00:00:00:00:00:00.

func TestCreateGetDataPacketRejectsZeroMAC(t *testing.T) {
	c := newTestClient(t)
	dev := &Device{} // zero-value MAC
	_, err := c.CreateGetDataPacket(dev, []string{"hostname"})
	if err == nil {
		t.Fatalf("expected error for zero MAC, got nil")
	}
	if !strings.Contains(err.Error(), "MAC") {
		t.Errorf("error %q should mention MAC", err)
	}
}

func TestCreateSetDataPacketRejectsZeroMAC(t *testing.T) {
	c := newTestClient(t)
	dev := &Device{}
	_, err := c.CreateSetDataPacket(dev, map[string]string{"hostname": "x"})
	if err == nil {
		t.Fatalf("expected error for zero MAC, got nil")
	}
}

func TestCreateResetPacketRejectsZeroMAC(t *testing.T) {
	c := newTestClient(t)
	dev := &Device{}
	_, err := c.CreateResetPacket(dev)
	if err == nil {
		t.Fatalf("expected error for zero MAC, got nil")
	}
}

func TestCreateGetDataPacketAcceptsValidMAC(t *testing.T) {
	c := newTestClient(t)
	dev := &Device{MAC: MustParseMAC("00:04:20:00:00:01")}
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
