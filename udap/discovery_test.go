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
