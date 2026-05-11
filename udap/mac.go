package udap

import (
	"fmt"
)

// MAC is the canonical value-object form of a 48-bit IEEE 802 hardware
// address. The codebase's wire and string layers historically passed
// MACs as plain strings ("aa:bb:cc:dd:ee:ff"), parsed and re-parsed at
// each boundary. MAC centralizes the parse/format/compare rules so that
// validation happens once at the boundary, and the type's existence
// then carries the guarantee through downstream code.
//
// This PR introduces the type and switches the protocol-internal hot
// paths (CreateGetDataPacket / CreateSetDataPacket / CreateResetPacket /
// waitForDeviceReply) to use it. A follow-up PR is expected to promote
// Device.MAC from string to MAC; MarshalText / UnmarshalText are
// implemented now so the JSON wire format is unchanged when that
// happens.
type MAC [MACAddressSize]byte

// ParseMAC parses a canonical colon-separated hex MAC string into the
// MAC value object. Accepts upper, lower, and mixed case; rejects any
// other separator, any length other than exactly 17 characters, any
// non-hex digit, and any leading or trailing garbage.
//
// Behavior matches the legacy isValidMAC + Client.parseMACAddress pair
// it replaces, with one tightening: fmt.Sscanf in the prior
// parseMACAddress could be tricked by trailing whitespace ("aa:bb:cc:
// dd:ee:ff ") into accepting input that isValidMAC then separately
// rejected. ParseMAC is strict in the same way isValidMAC was, so the
// two halves no longer disagree.
func ParseMAC(s string) (MAC, error) {
	var m MAC
	if len(s) != 17 {
		return m, fmt.Errorf("invalid MAC address %q: want 17 chars, got %d", s, len(s))
	}
	for i := range 6 {
		base := i * 3
		hi, ok := hexNibble(s[base])
		if !ok {
			return MAC{}, fmt.Errorf("invalid MAC address %q: non-hex digit at %d", s, base)
		}
		lo, ok := hexNibble(s[base+1])
		if !ok {
			return MAC{}, fmt.Errorf("invalid MAC address %q: non-hex digit at %d", s, base+1)
		}
		if i < 5 && s[base+2] != ':' {
			return MAC{}, fmt.Errorf("invalid MAC address %q: missing colon at %d", s, base+2)
		}
		m[i] = hi<<4 | lo
	}
	return m, nil
}

// hexNibble converts a single ASCII hex digit to its 0-15 value. Hand-rolled
// to avoid pulling in encoding/hex or strconv for a hot-path single-byte
// decode.
func hexNibble(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

// String returns the canonical "aa:bb:cc:dd:ee:ff" lowercase
// representation. This is the form ParseMAC accepts as canonical, the
// form Client.devices uses as a map key, and the form Device.MAC stores.
func (m MAC) String() string {
	const hex = "0123456789abcdef"
	buf := [17]byte{}
	for i := range 6 {
		buf[i*3] = hex[m[i]>>4]
		buf[i*3+1] = hex[m[i]&0x0f]
		if i < 5 {
			buf[i*3+2] = ':'
		}
	}
	return string(buf[:])
}

// Bytes returns the underlying [6]byte for the wire path —
// createUdapPacket needs a [6]byte for its DstAddress field.
func (m MAC) Bytes() [MACAddressSize]byte {
	return m
}

// IsZero reports whether m is the all-zeros MAC. createUdapPacket uses
// the zero MAC as the SrcAddress placeholder and as the broadcast
// DstAddress, so callers need a clean way to spot it.
func (m MAC) IsZero() bool {
	return m == MAC{}
}

// MarshalText / UnmarshalText make MAC encode as its canonical string
// in JSON, YAML, and any other encoding/text-based serializer. Added
// now so that a future PR can switch Device.MAC's field type from
// string to MAC without changing the JSON wire format.
func (m MAC) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

func (m *MAC) UnmarshalText(data []byte) error {
	parsed, err := ParseMAC(string(data))
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}
