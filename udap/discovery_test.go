package udap

import "testing"

func mustNewClient(t *testing.T) *Client {
	t.Helper()
	c, err := NewClientWithLogger(NewNoOpLogger())
	if err != nil {
		t.Fatalf("NewClientWithLogger: %v", err)
	}
	return c
}

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
