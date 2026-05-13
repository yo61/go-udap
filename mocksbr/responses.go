package mocksbr

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"net"
	"strconv"

	"go-udap/udap"
)

// buildHeader constructs a Packet header for a response from device dev
// to the request whose header is req. The Sequence is echoed; addresses
// are swapped (Src→Dst, device's MAC→Src).
func buildHeader(req *udap.Packet, deviceMAC string, method uint16) udap.Packet {
	pkt := udap.Packet{
		DstBroadcast: 0,
		DstType:      udap.AddrTypeETH,
		DstAddress:   req.SrcAddress,
		SrcBroadcast: 0,
		SrcType:      udap.AddrTypeETH,
		SrcAddress:   parseMAC(deviceMAC),
		Sequence:     req.Sequence,
		UDAPType:     udap.TypeUCP,
		UCPFlags:     0x00,
		UAPClass:     [4]byte{0x00, 0x01, 0x00, 0x01},
		UCPMethod:    method,
	}
	return pkt
}

// parseMAC converts "aa:bb:cc:dd:ee:ff" to [6]byte. Invalid input
// produces all-zeros. validMAC must have been called by the caller.
func parseMAC(s string) [6]byte {
	var out [6]byte
	for i := range 6 {
		hi := hexNibble(s[i*3])
		lo := hexNibble(s[i*3+1])
		out[i] = (hi << 4) | lo
	}
	return out
}

func hexNibble(b byte) byte {
	switch {
	case b >= '0' && b <= '9':
		return b - '0'
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10
	}
	return 0
}

// encodeHeader serializes a Packet header into bytes.
func encodeHeader(pkt udap.Packet) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, pkt)
	return buf.Bytes()
}

// buildDiscoveryResponse constructs the discovery response for one
// device. Layout: 27-byte header + ordered TLV payload (state,
// device_id, hardware_rev, firmware_rev, device_type, device_name,
// and optionally uuid if configured). The captured fixtures
// discovery-factory.bin and discovery-configured.bin predate UUID
// support and therefore lack TLV 0x0d.
func (d *device) buildDiscoveryResponse(req *udap.Packet) []byte {
	hdr := buildHeader(req, d.cfg.MAC, req.UCPMethod)

	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, hdr)

	state := d.state()
	hostname := d.snapshotWorking()["hostname"]

	writeTLV(buf, 0x0c, []byte(state))
	writeTLV(buf, 0x0b, []byte(d.cfg.DeviceID))
	writeTLV(buf, 0x0a, []byte(d.cfg.Hardware))
	writeTLV(buf, 0x09, []byte(d.cfg.Firmware))
	writeTLV(buf, 0x03, []byte(d.cfg.Model))
	writeTLV(buf, 0x02, []byte(hostname))
	if d.cfg.UUID != "" && !d.cfg.SuppressDiscoveryUUID {
		writeTLV(buf, 0x0d, uuidWireBytes(d.cfg.UUID))
	}

	return buf.Bytes()
}

// writeTLV appends a single type-length-value entry. Length is one byte
// (uint8); writeTLV truncates values longer than 255 bytes (UDAP
// discovery TLVs never approach that).
func writeTLV(buf *bytes.Buffer, t byte, value []byte) {
	if len(value) > 255 {
		value = value[:255]
	}
	buf.WriteByte(t)
	buf.WriteByte(byte(len(value)))
	buf.Write(value)
}

// uuidWireBytes converts a hex-string UUID config into wire bytes. If
// the config isn't valid hex (e.g. "mock-sbr-001"), the bytes of the
// string are used directly; the client's hex.EncodeToString will then
// surface those bytes hex-encoded, which is harmless for tests.
func uuidWireBytes(uuid string) []byte {
	b, err := hex.DecodeString(uuid)
	if err != nil {
		return []byte(uuid)
	}
	return b
}

// buildGetDataResponse constructs a GetData response (UCPMethod=0x0005)
// containing offset/length/value triples for each (offset, length) pair
// in the request payload. If d.cfg.Malformed is set, the response is
// post-processed into a deliberately broken shape.
func (d *device) buildGetDataResponse(req *udap.Packet, payload []byte) []byte {
	method := uint16(udap.MethodGetData)
	if d.cfg.Malformed == MalformedUnknownMethod {
		method = 0x9999
	}
	hdr := buildHeader(req, d.cfg.MAC, method)

	requested := parseGetDataRequest(payload)
	working := d.snapshotWorking()

	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, hdr)

	switch d.cfg.Malformed {
	case MalformedOversizedCount:
		// Promise 65535 items, write zero bodies. The client should
		// fail at the per-item bounds check.
		_ = binary.Write(buf, binary.BigEndian, uint16(0xFFFF))
		return buf.Bytes()
	case MalformedLengthExceedsPayload:
		// Promise 1 item, declare length=1000, write nothing.
		_ = binary.Write(buf, binary.BigEndian, uint16(1))
		_ = binary.Write(buf, binary.BigEndian, uint16(0)) // offset
		_ = binary.Write(buf, binary.BigEndian, uint16(1000))
		return buf.Bytes()
	}

	_ = binary.Write(buf, binary.BigEndian, uint16(len(requested)))
	for _, item := range requested {
		_ = binary.Write(buf, binary.BigEndian, item.offset)
		_ = binary.Write(buf, binary.BigEndian, item.length)
		buf.Write(encodeParamValue(item.length, working[item.name]))
	}
	return buf.Bytes()
}

// buildSetDataAck constructs the SetData/SaveData ack
// (UCPMethod=0x0006) with a 2-byte payload = number of params accepted.
func (d *device) buildSetDataAck(req *udap.Packet, accepted uint16) []byte {
	hdr := buildHeader(req, d.cfg.MAC, udap.MethodSetData)
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, hdr)
	_ = binary.Write(buf, binary.BigEndian, accepted)
	return buf.Bytes()
}

// buildResetAck constructs the Reset ack (UCPMethod=0x0004,
// header-only).
func (d *device) buildResetAck(req *udap.Packet) []byte {
	hdr := buildHeader(req, d.cfg.MAC, udap.MethodReset)
	return encodeHeader(hdr)
}

// buildErrorResponse constructs a MethodError (0x0007) response with a
// single TLVTypeErrorMessage payload. Used by the FailOn failure
// injection knob to simulate devices that reject a request.
func (d *device) buildErrorResponse(req *udap.Packet, message string) []byte {
	hdr := buildHeader(req, d.cfg.MAC, udap.MethodError)
	if len(message) > udap.MaxTLVValueLength {
		message = message[:udap.MaxTLVValueLength]
	}
	payload := udap.EncodeTLV([]udap.TLVData{{
		Type:   udap.TLVTypeErrorMessage,
		Length: uint8(len(message)),
		Value:  []byte(message),
	}})
	out := encodeHeader(hdr)
	return append(out, payload...)
}

// getDataItem represents one (offset, length, name) tuple decoded from
// a GetData request payload.
type getDataItem struct {
	offset uint16
	length uint16
	name   string
}

// parseGetDataRequest decodes a GetData request payload:
//
//	[16 bytes username][16 bytes password]
//	[uint16 BE count]
//	[count × (uint16 BE offset, uint16 BE length)]
//
// Items whose offset is unknown to udap.Parameters are returned with
// name="" so the response can still echo their requested offset/length
// (with a zero-filled value).
func parseGetDataRequest(payload []byte) []getDataItem {
	const credSize = 32 // username + password
	if len(payload) < credSize+2 {
		return nil
	}
	pos := credSize
	count := binary.BigEndian.Uint16(payload[pos:])
	pos += 2
	out := make([]getDataItem, 0, count)
	for i := 0; i < int(count); i++ {
		if pos+4 > len(payload) {
			break
		}
		ofs := binary.BigEndian.Uint16(payload[pos:])
		ln := binary.BigEndian.Uint16(payload[pos+2:])
		pos += 4
		out = append(out, getDataItem{
			offset: ofs,
			length: ln,
			name:   parameterByOffset(ofs),
		})
	}
	return out
}

// parameterByOffset returns the canonical parameter name for the given
// NVRAM offset, or "" if no parameter is registered at that offset.
func parameterByOffset(offset uint16) string {
	for _, p := range udap.Parameters {
		if p.Offset == offset {
			return p.Name
		}
	}
	return ""
}

// setDataItem represents one (offset, length, value, name) tuple
// decoded from a SetData request payload.
type setDataItem struct {
	offset uint16
	length uint16
	value  []byte
	name   string
}

// parseSetDataRequest decodes a SetData request payload:
//
//	[16 bytes username][16 bytes password]
//	[uint16 BE count]
//	[count × (uint16 BE offset, uint16 BE length, length × byte value)]
func parseSetDataRequest(payload []byte) []setDataItem {
	const credSize = 32
	if len(payload) < credSize+2 {
		return nil
	}
	pos := credSize
	count := binary.BigEndian.Uint16(payload[pos:])
	pos += 2
	out := make([]setDataItem, 0, count)
	for i := 0; i < int(count); i++ {
		if pos+4 > len(payload) {
			break
		}
		ofs := binary.BigEndian.Uint16(payload[pos:])
		ln := binary.BigEndian.Uint16(payload[pos+2:])
		pos += 4
		if pos+int(ln) > len(payload) {
			break
		}
		val := make([]byte, ln)
		copy(val, payload[pos:pos+int(ln)])
		pos += int(ln)
		out = append(out, setDataItem{
			offset: ofs,
			length: ln,
			value:  val,
			name:   parameterByOffset(ofs),
		})
	}
	return out
}

// encodeParamValue converts a string-form NVRAM value to its wire bytes,
// matching the encoding used by udap.Client.CreateSetDataPacket:
//   - length 1: decimal integer parsed as uint8
//   - length 2: decimal integer parsed as uint16 BE
//   - length 4: dotted-quad IPv4 → 4 bytes
//   - other: NUL-padded string, truncated to length
func encodeParamValue(length uint16, value string) []byte {
	out := make([]byte, length)
	switch length {
	case 1:
		if v, err := strconv.ParseUint(value, 10, 8); err == nil {
			out[0] = byte(v)
		}
	case 2:
		if v, err := strconv.ParseUint(value, 10, 16); err == nil {
			binary.BigEndian.PutUint16(out, uint16(v))
		}
	case 4:
		if ip := net.ParseIP(value); ip != nil {
			if ip4 := ip.To4(); ip4 != nil {
				copy(out, ip4)
				return out
			}
		}
	default:
		copy(out, []byte(value))
	}
	return out
}

// decodeParamValue is the inverse of encodeParamValue: bytes → canonical
// string form. Used by SetData handler to convert wire bytes into
// working memory entries.
func decodeParamValue(value []byte) string {
	switch len(value) {
	case 1:
		return strconv.FormatUint(uint64(value[0]), 10)
	case 2:
		return strconv.FormatUint(uint64(binary.BigEndian.Uint16(value)), 10)
	case 4:
		return net.IP(value).String()
	}
	end := len(value)
	for i, b := range value {
		if b == 0 {
			end = i
			break
		}
	}
	return string(value[:end])
}

// buildGetIPResponse constructs a get_ip reply (UCPMethod=0x0002) with
// TLV-encoded IP / SubnetMask / Gateway from DeviceConfig. Missing
// fields are encoded as zero IPv4 (0.0.0.0). The wire TLV codes match
// Net::UDAP: 0x05 IP_ADDR, 0x06 SUBNET_MASK, 0x07 GATEWAY_ADDR.
func (d *device) buildGetIPResponse(req *udap.Packet) []byte {
	hdr := buildHeader(req, d.cfg.MAC, udap.MethodGetIP)
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, hdr)
	writeIPTLV(buf, 0x05, d.cfg.IP)
	writeIPTLV(buf, 0x06, d.cfg.SubnetMask)
	writeIPTLV(buf, 0x07, d.cfg.Gateway)
	return buf.Bytes()
}

// writeIPTLV emits a 4-byte IPv4 TLV. Empty or unparseable inputs
// produce a 0.0.0.0 value.
func writeIPTLV(buf *bytes.Buffer, t byte, ipStr string) {
	out := []byte{0, 0, 0, 0}
	if ipStr != "" {
		if ip := net.ParseIP(ipStr); ip != nil {
			if ip4 := ip.To4(); ip4 != nil {
				out = ip4
			}
		}
	}
	buf.WriteByte(t)
	buf.WriteByte(0x04)
	buf.Write(out)
}

// buildGetUUIDResponse constructs a get_uuid reply (UCPMethod=0x000b)
// with the UUID as TLV 0x0d. The configured UUID string is converted
// via uuidWireBytes — a hex UUID like "deadbeefcafebabe1122334455667788"
// decodes to 16 raw bytes; a non-hex placeholder like "mock-sbr-001"
// is emitted as raw string bytes (length != 16, which will fail the
// client's UUID-length check — handy for exercising the fallback's
// error path).
func (d *device) buildGetUUIDResponse(req *udap.Packet) []byte {
	hdr := buildHeader(req, d.cfg.MAC, udap.MethodGetUUID)
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, hdr)
	writeTLV(buf, 0x0d, uuidWireBytes(d.cfg.UUID))
	return buf.Bytes()
}
