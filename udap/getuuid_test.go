package udap

import (
	"testing"
)

func TestCreateGetUUIDPacketHeaderOnly(t *testing.T) {
	c := mustNewClient(t)
	defer c.Close()
	device := &Device{MAC: MAC([6]byte{0x00, 0x04, 0x20, 0x16, 0x05, 0x8f})}
	pkt, err := c.CreateGetUUIDPacket(device)
	if err != nil {
		t.Fatalf("CreateGetUUIDPacket: %v", err)
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
	if hdr.UCPMethod != MethodGetUUID {
		t.Errorf("UCPMethod = 0x%04x, want 0x%04x", hdr.UCPMethod, MethodGetUUID)
	}
	if hdr.DstBroadcast != 0 {
		t.Errorf("DstBroadcast = %d, want 0", hdr.DstBroadcast)
	}
	wantMAC := [6]byte{0x00, 0x04, 0x20, 0x16, 0x05, 0x8f}
	if hdr.DstAddress != wantMAC {
		t.Errorf("DstAddress = %x, want %x", hdr.DstAddress, wantMAC)
	}
}

func TestCreateGetUUIDPacketRejectsZeroMAC(t *testing.T) {
	c := mustNewClient(t)
	defer c.Close()
	device := &Device{MAC: MAC([6]byte{})}
	_, err := c.CreateGetUUIDPacket(device)
	if err == nil {
		t.Error("CreateGetUUIDPacket with zero MAC returned nil error")
	}
}

func TestParseGetUUIDResponseHappyPath(t *testing.T) {
	// TLV 0x0d, length 16, UUID = 0x00..0x0f.
	data := []byte{
		0x0d, 0x10,
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
	}
	got, err := parseGetUUIDResponse(data)
	if err != nil {
		t.Fatalf("parseGetUUIDResponse: %v", err)
	}
	want := "000102030405060708090a0b0c0d0e0f"
	if got != want {
		t.Errorf("UUID = %q, want %q", got, want)
	}
}

func TestParseGetUUIDResponseMissingUUIDTLVIsError(t *testing.T) {
	// Payload that has some unrelated TLV but no 0x0d.
	data := []byte{
		0x02, 0x04, 't', 'e', 's', 't',
	}
	_, err := parseGetUUIDResponse(data)
	if err == nil {
		t.Error("parseGetUUIDResponse with no UUID TLV should error")
	}
}

func TestParseGetUUIDResponseEmptyPayloadIsError(t *testing.T) {
	_, err := parseGetUUIDResponse(nil)
	if err == nil {
		t.Error("parseGetUUIDResponse(nil) should error")
	}
}

func TestParseGetUUIDResponseWrongLengthSkipsAndErrors(t *testing.T) {
	// TLV 0x0d declared as length 8 (wrong — UUID must be 16 bytes).
	// Parser should skip it and return "no UUID TLV" error.
	data := []byte{
		0x0d, 0x08,
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
	}
	_, err := parseGetUUIDResponse(data)
	if err == nil {
		t.Error("parseGetUUIDResponse with wrong-length UUID TLV should error (no valid UUID TLV)")
	}
}

func TestParseGetUUIDResponseMalformedLengthRunoffIsError(t *testing.T) {
	// One TLV declares length 16 but only provides 4 bytes. Parser should
	// break, not panic.
	data := []byte{
		0x0d, 0x10,
		0x00, 0x01, 0x02, 0x03,
	}
	_, err := parseGetUUIDResponse(data)
	if err == nil {
		t.Error("parseGetUUIDResponse with truncated payload should error")
	}
}
