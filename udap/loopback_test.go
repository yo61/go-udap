package udap

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func buildHeader(t *testing.T, flags uint8) []byte {
	t.Helper()
	pkt := Packet{
		DstBroadcast: 0,
		DstType:      AddrTypeETH,
		DstAddress:   [6]byte{0, 4, 0x20, 0x16, 5, 0x8f},
		SrcBroadcast: 0,
		SrcType:      AddrTypeETH,
		Sequence:     1,
		UDAPType:     TypeUCP,
		UCPFlags:     flags,
		UAPClass:     [4]byte{0, 1, 0, 1},
		UCPMethod:    MethodGetData,
	}
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, pkt); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}

func TestIsUDAPRequestPacket(t *testing.T) {
	t.Run("request flag set", func(t *testing.T) {
		b := buildHeader(t, 0x01)
		if !isUDAPRequestPacket(b, len(b)) {
			t.Fatalf("expected request packet to be detected")
		}
	})
	t.Run("response flag clear", func(t *testing.T) {
		b := buildHeader(t, 0x00)
		if isUDAPRequestPacket(b, len(b)) {
			t.Fatalf("response packet wrongly detected as request")
		}
	})
	t.Run("packet too short", func(t *testing.T) {
		b := []byte{1, 2, 3}
		if isUDAPRequestPacket(b, len(b)) {
			t.Fatalf("short packet wrongly classified as request")
		}
	})
}
