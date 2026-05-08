package udap

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
	"time"
)

func TestConstants(t *testing.T) {
	// Test protocol constants
	if Port != 17784 {
		t.Errorf("Expected Port to be 17784, got %d", Port)
	}

	if TypeUCP != 0xC001 {
		t.Errorf("Expected TypeUCP to be 0xC001, got 0x%04x", TypeUCP)
	}

	if MethodDiscover != 0x0001 {
		t.Errorf("Expected MethodDiscover to be 0x0001, got 0x%04x", MethodDiscover)
	}
}

func TestParametersIncludeEssentials(t *testing.T) {
	essentialParams := []string{
		"lan_ip_mode",
		"lan_network_address",
		"hostname",
		"wireless_SSID",
		"wireless_wpa_psk",
	}

	for _, name := range essentialParams {
		p, exists := ParameterByName(name)
		if !exists {
			t.Errorf("Essential parameter %s not found in Parameters", name)
			continue
		}
		if err := p.Validate(); err != nil {
			t.Errorf("Parameter %s failed validation: %v", name, err)
		}
	}
}

func TestParametersAreNonEmpty(t *testing.T) {
	if len(Parameters) == 0 {
		t.Error("Parameters should not be empty")
	}
	if len(ParameterNames()) != len(Parameters) {
		t.Errorf("ParameterNames length %d != Parameters length %d",
			len(ParameterNames()), len(Parameters))
	}
}

func TestParameterAliasesResolve(t *testing.T) {
	for alias := range parameterAliases {
		if _, ok := ParameterByName(alias); !ok {
			t.Errorf("alias %q does not resolve via ParameterByName", alias)
		}
	}
}

func TestTLVEncodeDecode(t *testing.T) {
	tests := []struct {
		name string
		tlvs []TLVData
	}{
		{
			name: "single TLV",
			tlvs: []TLVData{
				{Type: TLVTypeParameterName, Length: 4, Value: []byte("test")},
			},
		},
		{
			name: "multiple TLVs",
			tlvs: []TLVData{
				{Type: TLVTypeParameterName, Length: 8, Value: []byte("hostname")},
				{Type: TLVTypeParameterValue, Length: 7, Value: []byte("testbox")},
			},
		},
		{
			name: "empty TLV",
			tlvs: []TLVData{
				{Type: TLVTypeParameterName, Length: 0, Value: []byte{}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode
			encoded := EncodeTLV(tt.tlvs)

			// Decode
			decoded := DecodeTLV(encoded)

			// Compare
			if len(decoded) != len(tt.tlvs) {
				t.Fatalf("Expected %d TLVs, got %d", len(tt.tlvs), len(decoded))
			}

			for i, expected := range tt.tlvs {
				actual := decoded[i]
				if actual.Type != expected.Type {
					t.Errorf("TLV %d: expected type %d, got %d", i, expected.Type, actual.Type)
				}
				if actual.Length != expected.Length {
					t.Errorf("TLV %d: expected length %d, got %d", i, expected.Length, actual.Length)
				}
				if !bytes.Equal(actual.Value, expected.Value) {
					t.Errorf("TLV %d: expected value %v, got %v", i, expected.Value, actual.Value)
				}
			}
		})
	}
}

func TestTLVDecodeInvalidData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "truncated header",
			data: []byte{0x01}, // Only type, no length
		},
		{
			name: "truncated value",
			data: []byte{0x01, 0x05, 0x12, 0x34}, // Claims 5 bytes but only has 2
		},
		{
			name: "empty data",
			data: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tlvs := DecodeTLV(tt.data)
			// Should not panic, may return partial results or empty slice
			t.Logf("Decoded %d TLVs from invalid data", len(tlvs))
		})
	}
}

func TestParsePacket(t *testing.T) {
	tests := []struct {
		name           string
		data           []byte
		expectError    bool
		expectedMethod uint16
	}{
		{
			name: "standard UDAP packet",
			data: func() []byte {
				packet := Packet{
					DstBroadcast: 1,
					DstType:      AddrTypeETH,
					DstAddress:   [6]byte{0x00, 0x04, 0x20, 0x12, 0x34, 0x56},
					SrcBroadcast: 0,
					SrcType:      AddrTypeETH,
					SrcAddress:   [6]byte{},
					Sequence:     1,
					UDAPType:     TypeUCP,
					UCPFlags:     FlagsDiscover,
					UAPClass:     [4]byte{0x00, 0x01, 0x00, 0x01},
					UCPMethod:    MethodDiscover,
				}
				buf := new(bytes.Buffer)
				binary.Write(buf, binary.BigEndian, &packet)
				return buf.Bytes()
			}(),
			expectError:    false,
			expectedMethod: MethodDiscover,
		},
		{
			// This was previously accepted via a permissive Format-2
			// fallback that interpreted bytes 0-1 as UCPMethod for any
			// 4+-byte payload. That fallback was a workaround for a
			// since-fixed off-by-2 in UDAPHeaderSize. Real UDAP packets
			// are always >= 27 bytes; rejecting short ones is correct.
			name:        "short packet (no longer accepted as raw response)",
			data:        []byte{0x00, 0x01, 0x00, 0x00, 0x12, 0x34, 0x56, 0x78},
			expectError: true,
		},
		{
			name:        "too short packet",
			data:        []byte{0x01, 0x02},
			expectError: true,
		},
		{
			name:        "empty packet",
			data:        []byte{},
			expectError: true,
		},
		{
			// 27-byte buffer with the right size but UDAPType=0x0000,
			// which would be a non-UCP packet — should be rejected
			// rather than parsed as garbage.
			name:        "header-sized buffer but non-UCP UDAPType",
			data:        make([]byte, UDAPHeaderSize), // all zeros, UDAPType=0
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packet, _, err := ParsePacket(tt.data)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if packet == nil {
				t.Error("Expected packet but got nil")
				return
			}

			if packet.UCPMethod != tt.expectedMethod {
				t.Errorf("Expected method 0x%04x, got 0x%04x", tt.expectedMethod, packet.UCPMethod)
			}

			// Header-only packets produce a nil data slice; packets
			// carrying a TLV payload produce a non-empty one. The old
			// assertion ("data should not be nil") only ever passed
			// because of a header-size off-by-2 in ParsePacket that
			// returned 2 leftover header bytes as data — fixed now.
		})
	}
}

func TestDevice(t *testing.T) {
	device := Device{
		MAC:      "00:04:20:12:34:56",
		IP:       "192.168.1.100",
		Name:     "Test Device",
		Model:    "Squeezebox Receiver",
		Firmware: "77",
		State:    "init",
		LastSeen: time.Now(),
		Parameters: map[string]string{
			"hostname":    "testbox",
			"lan_ip_mode": "1",
		},
	}

	// Test that device structure is properly initialized
	if device.MAC == "" {
		t.Error("Device MAC should not be empty")
	}

	if device.Parameters == nil {
		t.Error("Device Parameters should not be nil")
	}

	if len(device.Parameters) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(device.Parameters))
	}
}

func TestParameterStructure(t *testing.T) {
	p := Parameter{
		Name:   "test_param",
		Offset: 100,
		Length: 32,
		Help:   "Test parameter",
	}
	if p.Offset != 100 || p.Length != 32 {
		t.Errorf("field round-trip failed: %+v", p)
	}
	if got := p.FlagName(); got != "test-param" {
		t.Errorf("FlagName: got %q, want %q", got, "test-param")
	}
	if err := p.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}

func TestPacketStructure(t *testing.T) {
	packet := Packet{
		DstBroadcast: 1,
		DstType:      AddrTypeETH,
		DstAddress:   [6]byte{0x00, 0x04, 0x20, 0x12, 0x34, 0x56},
		SrcBroadcast: 0,
		SrcType:      AddrTypeETH,
		SrcAddress:   [6]byte{},
		Sequence:     42,
		UDAPType:     TypeUCP,
		UCPFlags:     FlagsDiscover,
		UAPClass:     [4]byte{0x00, 0x01, 0x00, 0x01},
		UCPMethod:    MethodDiscover,
	}

	// Test binary serialization
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, &packet)
	if err != nil {
		t.Fatalf("Failed to serialize packet: %v", err)
	}

	// Test that we get expected size. The Packet struct serializes via
	// encoding/binary to the sum of its field sizes (no padding):
	// 1+1+6+1+1+6+2+2+1+4+2 = 27 bytes.
	expectedSize := UDAPHeaderSize
	if buf.Len() != expectedSize {
		t.Errorf("Expected packet size %d, got %d", expectedSize, buf.Len())
	}

	// Test deserialization
	var packet2 Packet
	err = binary.Read(buf, binary.BigEndian, &packet2)
	if err != nil {
		t.Fatalf("Failed to deserialize packet: %v", err)
	}

	// Compare packets
	if !reflect.DeepEqual(packet, packet2) {
		t.Error("Serialized and deserialized packets do not match")
	}
}

func TestGetLocalIPs(t *testing.T) {
	localIPs := getLocalIPs()

	// Should have at least localhost entries
	if !localIPs["::"] {
		t.Error("Expected :: to be in local IPs")
	}

	if !localIPs["::1"] {
		t.Error("Expected ::1 to be in local IPs")
	}

	// Should have some actual network interfaces
	if len(localIPs) < 2 {
		t.Errorf("Expected at least 2 local IPs, got %d", len(localIPs))
	}
}

func TestTLVDataStructure(t *testing.T) {
	tlv := TLVData{
		Type:   TLVTypeParameterName,
		Length: 8,
		Value:  []byte("hostname"),
	}

	if tlv.Type != TLVTypeParameterName {
		t.Errorf("Expected type %d, got %d", TLVTypeParameterName, tlv.Type)
	}

	if tlv.Length != 8 {
		t.Errorf("Expected length 8, got %d", tlv.Length)
	}

	if string(tlv.Value) != "hostname" {
		t.Errorf("Expected value 'hostname', got '%s'", string(tlv.Value))
	}
}

func TestMethodConstants(t *testing.T) {
	// Test method constant values
	expectedMethods := map[string]uint16{
		"MethodDiscover": 0x0001,
		"MethodGetIP":    0x0002,
		"MethodReset":    0x0004,
		"MethodGetData":  0x0005,
		"MethodSetData":  0x0006,
		"MethodError":    0x0007,
		"MethodAdvDisc":  0x0009,
	}

	actualMethods := map[string]uint16{
		"MethodDiscover": MethodDiscover,
		"MethodGetIP":    MethodGetIP,
		"MethodReset":    MethodReset,
		"MethodGetData":  MethodGetData,
		"MethodSetData":  MethodSetData,
		"MethodError":    MethodError,
		"MethodAdvDisc":  MethodAdvDisc,
	}

	for name, expected := range expectedMethods {
		if actual, exists := actualMethods[name]; !exists {
			t.Errorf("Method constant %s not found", name)
		} else if actual != expected {
			t.Errorf("Method %s: expected 0x%04x, got 0x%04x", name, expected, actual)
		}
	}

	// Test method alias
	if MethodDataResp != MethodGetIP {
		t.Errorf("MethodDataResp should equal MethodGetIP (0x%04x), got 0x%04x", MethodGetIP, MethodDataResp)
	}
}
