package udap

// ucpFlagsOffset is the byte index of UCPFlags within the serialized
// 27-byte UDAP packet header. Layout (sum of preceding field sizes):
//
//	DstBroadcast(1) + DstType(1) + DstAddress(6) +
//	SrcBroadcast(1) + SrcType(1) + SrcAddress(6) +
//	Sequence(2) + UDAPType(2) = 20.
const ucpFlagsOffset = 20

// isUDAPRequestPacket reports whether buf[:n] looks like a UDAP packet
// with the request bit set. The capture path uses this to skip our own
// kernel-looped broadcast: we send with UCPFlags=0x01 (request); real
// devices reply with UCPFlags=0x00 (response).
//
// Returns false (treat as not-a-request) if buf is too short to contain
// the UCPFlags byte — let the rest of the parser decide what to do.
func isUDAPRequestPacket(buf []byte, n int) bool {
	if n <= ucpFlagsOffset {
		return false
	}
	return buf[ucpFlagsOffset]&0x01 != 0
}
