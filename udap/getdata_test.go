package udap

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// perlGetDataParams is the exact param list (in Perl-emitted order) that
// the Net::UDAP shell sends in a "get_data" request — see the captured
// session perl_code.pcap (frame 6). All 26 are NVRAM-resident parameters.
var perlGetDataParams = []string{
	"wireless_SSID", "wireless_wep_key", "secondary_dns", "hostname",
	"wireless_wpa_cipher", "lan_ip_mode", "squeezecenter_name",
	"wireless_wpa_psk", "server_address", "wireless_keylen",
	"wireless_region_id", "wireless_channel", "wireless_wep_key_2",
	"wireless_wep_key_3", "wireless_wep_on", "primary_dns",
	"wireless_wpa_on", "lan_gateway", "lan_network_address",
	"wireless_wep_key_1", "wireless_wpa_mode", "wireless_mode",
	"bridging", "lan_subnet_mask", "interface", "lms_address",
}

// TestCreateGetDataPacketWireFormat verifies CreateGetDataPacket emits the
// offset/length wire format that real SBRs accept (validated against the
// Perl Net::UDAP reference implementation, see perl_code.pcap frame 6).
//
// Format:
//
//	[27-byte UDAP header, UCPMethod=0x0005]
//	[16 zero bytes — username]
//	[16 zero bytes — password]
//	[uint16 BE count of items]
//	[N × (uint16 BE offset, uint16 BE length)]
//
// Earlier versions of go-udap mistakenly sent a TLV stream of parameter
// *names* here, which the device silently ignored — every read timed out.
func TestCreateGetDataPacketWireFormat(t *testing.T) {
	c := &Client{logger: NewNoOpLogger(), sequence: 1}
	device := &Device{MAC: "00:04:20:16:05:8f"}

	pkt := c.CreateGetDataPacket(device, perlGetDataParams)

	const userPassLen = UsernameFieldSize + PasswordFieldSize
	wantLen := UDAPHeaderSize + userPassLen + 2 + len(perlGetDataParams)*4
	if len(pkt) != wantLen {
		t.Fatalf("packet length: got %d, want %d", len(pkt), wantLen)
	}

	// Header check — method byte is the last 2 bytes.
	method := binary.BigEndian.Uint16(pkt[UDAPHeaderSize-2 : UDAPHeaderSize])
	if method != MethodGetData {
		t.Errorf("UCPMethod: got 0x%04x, want 0x%04x", method, MethodGetData)
	}

	// 32 zero bytes after header.
	for i := UDAPHeaderSize; i < UDAPHeaderSize+userPassLen; i++ {
		if pkt[i] != 0 {
			t.Errorf("byte %d in user/pass field is 0x%02x, want 0x00", i, pkt[i])
		}
	}

	// Count.
	count := binary.BigEndian.Uint16(pkt[UDAPHeaderSize+userPassLen:])
	if int(count) != len(perlGetDataParams) {
		t.Errorf("count: got %d, want %d", count, len(perlGetDataParams))
	}

	// Each (offset, length) pair must match ConfigSettings for that
	// param. Order isn't asserted here — sorted-by-offset order is the
	// implementation's choice, mirroring CreateSetDataPacket.
	pairs := pkt[UDAPHeaderSize+userPassLen+2:]
	got := make(map[uint16]uint16, count)
	for i := 0; i < int(count); i++ {
		ofs := binary.BigEndian.Uint16(pairs[i*4:])
		ln := binary.BigEndian.Uint16(pairs[i*4+2:])
		got[ofs] = ln
	}
	for _, name := range perlGetDataParams {
		want, ok := ParameterByName(name)
		if !ok {
			t.Fatalf("test bug: %q not in Parameters", name)
		}
		ln, present := got[want.Offset]
		if !present {
			t.Errorf("missing offset 0x%04x for %q", want.Offset, name)
			continue
		}
		if ln != want.Length {
			t.Errorf("%q: length at offset 0x%04x is %d, want %d",
				name, want.Offset, ln, want.Length)
		}
	}
}

// TestCreateGetDataPacketMatchesPerlPayload asserts the user/pass+count+
// (offset,length) payload portion of CreateGetDataPacket is byte-equal
// to the Perl Net::UDAP capture (frame 6 of perl_code.pcap), modulo
// ordering of the (offset, length) pairs.
func TestCreateGetDataPacketMatchesPerlPayload(t *testing.T) {
	fixture := filepath.Join("testdata", "captures", "getdata-request-26params.bin")
	perl, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if len(perl) != 165 {
		t.Fatalf("fixture: got %d bytes, want 165", len(perl))
	}

	c := &Client{logger: NewNoOpLogger(), sequence: 1}
	device := &Device{MAC: "00:04:20:16:05:8f"}
	got := c.CreateGetDataPacket(device, perlGetDataParams)

	if len(got) != len(perl) {
		t.Fatalf("packet length: got %d, want %d", len(got), len(perl))
	}

	// Compare the payload (everything after the 27-byte header).
	const userPassLen = UsernameFieldSize + PasswordFieldSize
	const countOff = UDAPHeaderSize + userPassLen
	if got[countOff] != perl[countOff] || got[countOff+1] != perl[countOff+1] {
		t.Errorf("count bytes: got %02x%02x, want %02x%02x",
			got[countOff], got[countOff+1], perl[countOff], perl[countOff+1])
	}

	gotPairs := extractPairs(got[countOff+2:])
	perlPairs := extractPairs(perl[countOff+2:])
	if len(gotPairs) != len(perlPairs) {
		t.Fatalf("pair count: got %d, want %d", len(gotPairs), len(perlPairs))
	}
	for ofs, ln := range perlPairs {
		if gotLn, ok := gotPairs[ofs]; !ok {
			t.Errorf("missing offset 0x%04x", ofs)
		} else if gotLn != ln {
			t.Errorf("offset 0x%04x: got length %d, want %d", ofs, gotLn, ln)
		}
	}
}

func extractPairs(b []byte) map[uint16]uint16 {
	out := make(map[uint16]uint16, len(b)/4)
	for i := 0; i+4 <= len(b); i += 4 {
		out[binary.BigEndian.Uint16(b[i:])] = binary.BigEndian.Uint16(b[i+2:])
	}
	return out
}

// TestParseGetDataResponseFromPerl loads the captured GetData response
// (perl_code.pcap frame 7) and verifies parseGetDataResponse decodes the
// payload into the expected param values.
//
// The response contains 26 params. We assert key values for params
// present in the Parameters table — anything Perl recognized that we
// don't is parsed as raw hex under a synthetic offset_NNN key.
func TestParseGetDataResponseFromPerl(t *testing.T) {
	fixture := filepath.Join("testdata", "captures", "getdata-response-26params.bin")
	raw, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if len(raw) != 387 {
		t.Fatalf("fixture: got %d bytes, want 387", len(raw))
	}

	pkt, data, err := ParsePacket(raw)
	if err != nil {
		t.Fatalf("ParsePacket: %v", err)
	}
	if pkt.UCPMethod != MethodGetData {
		t.Errorf("UCPMethod: got 0x%04x, want 0x%04x", pkt.UCPMethod, MethodGetData)
	}

	params, err := parseGetDataResponse(data)
	if err != nil {
		t.Fatalf("parseGetDataResponse: %v", err)
	}

	// Spot-check known values from the Perl decode (lines 32-124 of
	// perl_shell_session.txt for device 00:04:20:16:05:8f).
	want := map[string]string{
		"lan_ip_mode":         "1",
		"lan_subnet_mask":     "255.255.255.0",
		"lan_gateway":         "0.0.0.0",
		"lan_network_address": "0.0.0.0",
		"primary_dns":         "0.0.0.0",
		"secondary_dns":       "0.0.0.0",
		"server_address":      "0.0.0.0",
		"wireless_channel":    "6",
		"wireless_region_id":  "4",
		"wireless_keylen":     "0",
		"wireless_wpa_cipher": "3",
		"wireless_wpa_mode":   "1",
		"wireless_wpa_on":     "0",
		"wireless_wep_on":     "0",
		"wireless_mode":       "0",
		"bridging":            "0",
		"interface":           "128",
		"hostname":            "",
		"wireless_SSID":       "",
	}
	for k, v := range want {
		got, ok := params[k]
		if !ok {
			t.Errorf("param %q missing", k)
			continue
		}
		if got != v {
			t.Errorf("param %q: got %q, want %q", k, got, v)
		}
	}
}
