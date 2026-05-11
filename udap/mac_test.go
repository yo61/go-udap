package udap

import (
	"encoding/json"
	"testing"
)

// TestParseMACAcceptsCanonicalLowercase locks in the canonical wire form
// used everywhere in this codebase: six lowercase hex pairs separated
// by colons. Matches what discovery extracts from packet SrcAddress and
// what device.MAC stores.
func TestParseMACAcceptsCanonicalLowercase(t *testing.T) {
	m, err := ParseMAC("00:04:20:16:05:8f")
	if err != nil {
		t.Fatalf("ParseMAC: %v", err)
	}
	want := MAC{0x00, 0x04, 0x20, 0x16, 0x05, 0x8f}
	if m != want {
		t.Errorf("ParseMAC bytes = %x, want %x", m, want)
	}
}

// TestParseMACIsCaseInsensitive accepts upper, lower, and mixed case —
// matches isValidMAC's prior behavior. Input from the CLI is normalized
// to lowercase, but the parser must accept either form so that pasted
// vendor strings work too.
func TestParseMACIsCaseInsensitive(t *testing.T) {
	cases := []string{
		"AA:BB:CC:DD:EE:FF",
		"aa:bb:cc:dd:ee:ff",
		"Aa:Bb:Cc:Dd:Ee:Ff",
	}
	want := MAC{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	for _, in := range cases {
		m, err := ParseMAC(in)
		if err != nil {
			t.Errorf("ParseMAC(%q): %v", in, err)
			continue
		}
		if m != want {
			t.Errorf("ParseMAC(%q) = %x, want %x", in, m, want)
		}
	}
}

// TestParseMACRejectsMalformed covers the failure surface — these are
// the inputs isValidMAC currently rejects and that parseMACAddress
// (the soon-to-be-removed Client method) returns an error for.
func TestParseMACRejectsMalformed(t *testing.T) {
	bad := []string{
		"",
		"aa:bb:cc:dd:ee",                  // too short
		"aa:bb:cc:dd:ee:ff:00",            // too long
		"aa-bb-cc-dd-ee-ff",               // wrong separator
		"aa:bb:cc:dd:ee:zz",               // non-hex digit
		"aa:bb:cc:dd:ee:f",                // last octet 1 char
		"aaa:bb:cc:dd:ee:ff",              // first octet 3 chars
		"aa:bb:cc:dd:ee:ff ",              // trailing whitespace
		" aa:bb:cc:dd:ee:ff",              // leading whitespace
		"aa:bb:cc:dd:ee:ff\n",             // trailing newline
		"00:04:20:16:05:8f:extra-payload", // suffix garbage
	}
	for _, in := range bad {
		if _, err := ParseMAC(in); err == nil {
			t.Errorf("ParseMAC(%q) = nil error, want error", in)
		}
	}
}

// TestMACString roundtrips canonical lowercase form. This is the form
// used as the map key in Client.devices and the value stored in
// Device.MAC, so the renderer must produce exactly the same shape that
// ParseMAC accepts as canonical.
func TestMACString(t *testing.T) {
	m := MAC{0x00, 0x04, 0x20, 0x16, 0x05, 0x8f}
	if got, want := m.String(), "00:04:20:16:05:8f"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestMACStringRoundtripsThroughParseMAC(t *testing.T) {
	inputs := []string{
		"00:04:20:16:05:8f",
		"ff:ff:ff:ff:ff:ff",
		"00:00:00:00:00:00",
	}
	for _, in := range inputs {
		m, err := ParseMAC(in)
		if err != nil {
			t.Fatalf("ParseMAC(%q): %v", in, err)
		}
		if got := m.String(); got != in {
			t.Errorf("ParseMAC(%q).String() = %q, want %q", in, got, in)
		}
	}
}

// TestMACIsZero distinguishes the all-zeros broadcast/source-placeholder
// MAC from real device MACs. createUdapPacket uses a zero MAC as the
// SrcAddress placeholder and as the broadcast destination for
// discovery, so callers need a clean way to spot it.
func TestMACIsZero(t *testing.T) {
	var zero MAC
	if !zero.IsZero() {
		t.Error("zero-value MAC.IsZero() = false, want true")
	}
	nonZero, err := ParseMAC("00:00:00:00:00:01")
	if err != nil {
		t.Fatalf("ParseMAC: %v", err)
	}
	if nonZero.IsZero() {
		t.Error("non-zero MAC.IsZero() = true, want false")
	}
}

// TestMACBytes exposes the underlying [6]byte for the wire-packing path
// (createUdapPacket needs a [6]byte for DstAddress).
func TestMACBytes(t *testing.T) {
	m, err := ParseMAC("00:04:20:16:05:8f")
	if err != nil {
		t.Fatalf("ParseMAC: %v", err)
	}
	want := [6]byte{0x00, 0x04, 0x20, 0x16, 0x05, 0x8f}
	if got := m.Bytes(); got != want {
		t.Errorf("Bytes() = %x, want %x", got, want)
	}
}

// TestMACJSONRoundtrip pins the wire format for Device.MAC's JSON
// representation. Even though this PR keeps Device.MAC as string,
// MarshalText / UnmarshalText are added now so a future PR can switch
// Device.MAC to MAC without breaking any consumer that reads or writes
// device JSON.
func TestMACJSONRoundtrip(t *testing.T) {
	m, err := ParseMAC("00:04:20:16:05:8f")
	if err != nil {
		t.Fatalf("ParseMAC: %v", err)
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got, want := string(data), `"00:04:20:16:05:8f"`; got != want {
		t.Errorf("Marshal = %s, want %s", got, want)
	}
	var back MAC
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if back != m {
		t.Errorf("Unmarshal = %x, want %x", back, m)
	}
}

func TestMACUnmarshalTextRejectsMalformed(t *testing.T) {
	var m MAC
	if err := m.UnmarshalText([]byte("not-a-mac")); err == nil {
		t.Errorf("UnmarshalText(not-a-mac) = nil, want error")
	}
}
