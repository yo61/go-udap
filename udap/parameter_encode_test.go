package udap

import (
	"bytes"
	"testing"
)

// TestParameterEncodeUint8 exercises Length=1 NVRAM slots — single-byte
// numeric parameters like lan_ip_mode (0/1) and wireless_channel.
func TestParameterEncodeUint8(t *testing.T) {
	p := Parameter{Name: "lan_ip_mode", Offset: 4, Length: 1}

	cases := []struct {
		value   string
		want    []byte
		wantErr bool
	}{
		{"0", []byte{0x00}, false},
		{"1", []byte{0x01}, false},
		{"255", []byte{0xff}, false},
		{"256", nil, true},  // out of uint8 range
		{"-1", nil, true},   // negative not valid for ParseUint
		{"x", nil, true},    // non-numeric
		{"", nil, true},     // empty
		{"1abc", nil, true}, // partial-numeric — ParseUint is stricter than Sscanf
	}
	for _, tc := range cases {
		got, err := p.Encode(tc.value)
		if tc.wantErr {
			if err == nil {
				t.Errorf("Encode(%q) want error, got bytes %x", tc.value, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("Encode(%q) unexpected error: %v", tc.value, err)
			continue
		}
		if !bytes.Equal(got, tc.want) {
			t.Errorf("Encode(%q) = %x, want %x", tc.value, got, tc.want)
		}
	}
}

// TestParameterEncodeUint16 exercises Length=2 slots. No real UDAP
// parameter currently uses Length=2 (Parameters slice has only 1 and 4
// and various string lengths), but the encoder must still handle it
// because CreateSetDataPacket dispatches on Length.
func TestParameterEncodeUint16(t *testing.T) {
	p := Parameter{Name: "hypothetical_u16", Offset: 0, Length: 2}

	cases := []struct {
		value   string
		want    []byte
		wantErr bool
	}{
		{"0", []byte{0x00, 0x00}, false},
		{"1", []byte{0x00, 0x01}, false},
		{"256", []byte{0x01, 0x00}, false}, // big-endian
		{"65535", []byte{0xff, 0xff}, false},
		{"65536", nil, true}, // out of uint16 range
		{"-1", nil, true},
		{"abc", nil, true},
	}
	for _, tc := range cases {
		got, err := p.Encode(tc.value)
		if tc.wantErr {
			if err == nil {
				t.Errorf("Encode(%q) want error, got bytes %x", tc.value, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("Encode(%q) unexpected error: %v", tc.value, err)
			continue
		}
		if !bytes.Equal(got, tc.want) {
			t.Errorf("Encode(%q) = %x, want %x", tc.value, got, tc.want)
		}
	}
}

// TestParameterEncodeIPv4 exercises Length=4 slots — all IPv4 addresses
// per the Parameters table (server_address, lan_gateway, primary_dns,
// secondary_dns, lan_subnet_mask, lan_network_address, lms_address).
func TestParameterEncodeIPv4(t *testing.T) {
	p := Parameter{Name: "server_address", Offset: 71, Length: 4}

	cases := []struct {
		value   string
		want    []byte
		wantErr bool
	}{
		{"0.0.0.0", []byte{0, 0, 0, 0}, false},
		{"192.168.1.50", []byte{192, 168, 1, 50}, false},
		{"255.255.255.255", []byte{255, 255, 255, 255}, false},
		{"192.168.1", nil, true},       // incomplete
		{"192.168.1.x", nil, true},     // non-numeric octet
		{"192.168.1.256", nil, true},   // octet out of range
		{"::1", nil, true},             // IPv6 must be rejected
		{"2001:db8::1", nil, true},     // IPv6 must be rejected
		{"", nil, true},                // empty
		{"  192.168.1.1  ", nil, true}, // whitespace not trimmed
	}
	for _, tc := range cases {
		got, err := p.Encode(tc.value)
		if tc.wantErr {
			if err == nil {
				t.Errorf("Encode(%q) want error, got bytes %x", tc.value, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("Encode(%q) unexpected error: %v", tc.value, err)
			continue
		}
		if !bytes.Equal(got, tc.want) {
			t.Errorf("Encode(%q) = %x, want %x", tc.value, got, tc.want)
		}
	}
}

// TestParameterEncodeString covers the default (non-1/2/4) case:
// string-valued NVRAM slots like hostname (Length=33) or wireless_wpa_psk
// (Length=64). Strings shorter than Length are zero-padded; strings
// longer than Length are silently truncated, matching the historical
// CreateSetDataPacket behavior. (validateParameter rejects over-length
// strings at the CLI boundary; this encoder is the library-level fallback.)
func TestParameterEncodeStringPaddedAndTruncated(t *testing.T) {
	p := Parameter{Name: "hostname", Offset: 17, Length: 8}

	cases := []struct {
		value string
		want  []byte
	}{
		{"", []byte{0, 0, 0, 0, 0, 0, 0, 0}},
		{"a", []byte{'a', 0, 0, 0, 0, 0, 0, 0}},
		{"abcdefgh", []byte{'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h'}},
		// Truncated — preserves prior CreateSetDataPacket behavior.
		{"abcdefghIJK", []byte{'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h'}},
	}
	for _, tc := range cases {
		got, err := p.Encode(tc.value)
		if err != nil {
			t.Errorf("Encode(%q) unexpected error: %v", tc.value, err)
			continue
		}
		if !bytes.Equal(got, tc.want) {
			t.Errorf("Encode(%q) = %x, want %x", tc.value, got, tc.want)
		}
		if len(got) != int(p.Length) {
			t.Errorf("Encode(%q) returned %d bytes, want %d (Length)", tc.value, len(got), p.Length)
		}
	}
}

// TestParameterEncodeAllReturnsLengthBytes asserts the invariant that
// Encode always returns exactly p.Length bytes on success. This is what
// CreateSetDataPacket relies on — the wire format declares Length and
// then expects exactly that many bytes of value to follow.
func TestParameterEncodeReturnsExactLengthBytes(t *testing.T) {
	cases := []struct {
		p     Parameter
		value string
	}{
		{Parameter{Length: 1}, "0"},
		{Parameter{Length: 2}, "0"},
		{Parameter{Length: 4}, "0.0.0.0"},
		{Parameter{Length: 13}, "abc"},         // short string
		{Parameter{Length: 33}, "exact-len-x"}, // hostname-sized
	}
	for _, tc := range cases {
		got, err := tc.p.Encode(tc.value)
		if err != nil {
			t.Errorf("Encode(Length=%d, %q) unexpected error: %v", tc.p.Length, tc.value, err)
			continue
		}
		if len(got) != int(tc.p.Length) {
			t.Errorf("Encode(Length=%d, %q) returned %d bytes, want %d",
				tc.p.Length, tc.value, len(got), tc.p.Length)
		}
	}
}
