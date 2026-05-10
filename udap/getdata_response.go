package udap

import (
	"encoding/binary"
	"fmt"
	"strconv"
)

// configParamByOffset is a reverse index of Parameters, mapping NVRAM
// offset to the canonical parameter name.
var configParamByOffset = func() map[uint16]string {
	out := make(map[uint16]string, len(Parameters))
	for _, p := range Parameters {
		out[p.Offset] = p.Name
	}
	return out
}()

// parseGetDataResponse decodes the payload of a GetData (0x0005) response —
// everything after the 27-byte UDAP header.
//
// Wire format:
//
//	uint16 BE count
//	count × (uint16 BE offset, uint16 BE length, length × byte value)
//
// Each NVRAM offset is mapped back to its parameter name via the
// Parameters table. Values are formatted to round-trip through
// CreateSetDataPacket: 1- and 2-byte numerics as decimal, 4-byte values as
// dotted-quad IPv4, longer values as NUL-trimmed strings.
//
// Offsets that don't match any known parameter are recorded under a
// synthetic key "offset_<decimal>" with the bytes hex-encoded.
func parseGetDataResponse(data []byte) (map[string]string, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("getdata response: payload too short (%d bytes)", len(data))
	}
	count := binary.BigEndian.Uint16(data[:2])
	pos := 2
	// Clamp the map size hint to what the payload can actually hold:
	// each item header is 4 bytes, so at most (len(data)-2)/4 items can
	// fit. Without this, a crafted count=0xFFFF with a tiny body would
	// allocate a ~5 MB bucket array per response.
	out := make(map[string]string, min(int(count), (len(data)-2)/4))
	for i := 0; i < int(count); i++ {
		if pos+4 > len(data) {
			return nil, fmt.Errorf("getdata response: truncated header for item %d at offset %d", i, pos)
		}
		ofs := binary.BigEndian.Uint16(data[pos:])
		ln := binary.BigEndian.Uint16(data[pos+2:])
		pos += 4
		if pos+int(ln) > len(data) {
			return nil, fmt.Errorf(
				"getdata response: item %d (NVRAM offset %d, length %d) exceeds payload (%d bytes left)",
				i, ofs, ln, len(data)-pos)
		}
		value := data[pos : pos+int(ln)]
		pos += int(ln)

		name, known := configParamByOffset[ofs]
		if !known {
			out[fmt.Sprintf("offset_%d", ofs)] = fmt.Sprintf("%x", value)
			continue
		}
		out[name] = formatGetDataValue(value)
	}
	return out, nil
}

// formatGetDataValue renders a raw NVRAM value as a string that
// round-trips through CreateSetDataPacket's value-encoding logic.
func formatGetDataValue(value []byte) string {
	switch len(value) {
	case 1:
		return strconv.FormatUint(uint64(value[0]), 10)
	case 2:
		return strconv.FormatUint(uint64(binary.BigEndian.Uint16(value)), 10)
	case 4:
		return fmt.Sprintf("%d.%d.%d.%d", value[0], value[1], value[2], value[3])
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
