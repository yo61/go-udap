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
	if nc.IP != nil || nc.SubnetMask != nil || nc.Gateway != nil {
		t.Errorf("got %+v, want all zero IPs", nc)
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
