package udap

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

// UDAP Protocol Constants
// NOTE: All multi-byte fields use network byte order (big-endian) to match
// the working Net::UDAP Perl implementation
const (
	Port = 17784 // Correct UDAP port (0x4578)

	// Address Types
	AddrTypeETH = 0x01
	AddrTypeUDP = 0x02

	// UDAP Types
	TypeUCP = 0xC001 // UCP packets (0xC0, 0x01 in big endian/network order)

	// UCP Methods, per the authoritative Net::UDAP Constant.pm
	// (UCP_METHOD_*). Earlier comments here had two protocol-level
	// errors that are now fixed:
	//
	//   - 0x0002 was annotated "also used as data response". It isn't.
	//     get_ip is its own method that returns network-config TLVs
	//     (lan_ip_mode, lan_gateway, lan_subnet_mask, lan_network_address).
	//     GetData (0x0005) responses come back with method 0x0005, not
	//     0x0002. The misnomer alias MethodDataResp has been removed.
	//
	//   - 0x0008 was named MethodSetDataAck. It's actually
	//     UCP_METHOD_CREDENTIALS_ERROR per Net::UDAP — the device's way
	//     of rejecting a SetData when it didn't like the user/pass
	//     fields. SetData acks reuse the request's method (0x0006) on
	//     the response.
	MethodDiscover         = 0x0001
	MethodGetIP            = 0x0002
	MethodReset            = 0x0004
	MethodGetData          = 0x0005
	MethodSetData          = 0x0006
	MethodError            = 0x0007
	MethodCredentialsError = 0x0008
	MethodAdvDisc          = 0x0009

	// UCP Flags
	FlagsDiscover = 0x01

	// TLV Types
	TLVTypeParameterName  = 0x01 // Parameter name
	TLVTypeParameterValue = 0x02 // Parameter value
	TLVTypeErrorMessage   = 0x03 // Error message

	// Protocol Field Sizes
	MACAddressSize    = 6  // MAC address length in bytes
	UsernameFieldSize = 16 // Username field size in SetData packets
	PasswordFieldSize = 16 // Password field size in SetData packets
	UDAPHeaderSize    = 27 // Serialized size of the Packet struct (sum of fields, no padding)

	// Validation Limits
	MaxDeviceNameLength = 64    // Maximum device name length
	MaxTLVValueLength   = 255   // Maximum TLV value length (uint8 max)
	MaxConfigLength     = 256   // Maximum configuration parameter length
	MaxNVRAMOffset      = 65535 // Maximum NVRAM offset (uint16 max)

	// Common Network Values
	BroadcastIP = "255.255.255.255"
	LocalIP     = "0.0.0.0"
)

// Packet represents a UDAP packet with proper protocol structure
// Based on the Lua createUdap function from LMS-Community/squeezeplay
type Packet struct {
	DstBroadcast uint8   // Destination broadcast flag
	DstType      uint8   // Destination address type (0x01 for Ethernet)
	DstAddress   [6]byte // Destination MAC address
	SrcBroadcast uint8   // Source broadcast flag (should be 0)
	SrcType      uint8   // Source address type (0x01 for Ethernet)
	SrcAddress   [6]byte // Source MAC address
	Sequence     uint16  // Sequence number
	UDAPType     uint16  // UDAP packet type (0xC001 for UCP)
	UCPFlags     uint8   // UCP flags (0x01 for request)
	UAPClass     [4]byte // UAP class (0x00, 0x01, 0x00, 0x01)
	UCPMethod    uint16  // UCP method
}

// Device represents a discovered Squeezebox device. Fields are populated
// from the discovery-response TLVs (per Net::UDAP Constant.pm code map):
//
//	Name        ← TLV 0x02 device_name (the configured hostname)
//	Model       ← TLV 0x03 device_type + TLV 0x0b device_id, joined into a
//	              human label (e.g. "Squeezebox Receiver")
//	Firmware    ← TLV 0x09 firmware_rev (e.g. "77")
//	State       ← TLV 0x0c device_status (init / wait_slimserver / connected)
//	HardwareRev ← TLV 0x0a hardware_rev (opaque string, e.g. "0005")
//
// MAC and IP come from the UDAP packet header / UDP source address.
// MAC is the canonical value-object form: validated once at the point
// of construction (discovery, CLI input parsing, JSON unmarshal) and
// then carried by the type system throughout. JSON wire format is
// unchanged thanks to MAC.MarshalText / UnmarshalText.
type Device struct {
	MAC         MAC               `json:"mac"`
	IP          string            `json:"ip"`
	Name        string            `json:"name"`
	Model       string            `json:"model"`
	Firmware    string            `json:"firmware"`
	HardwareRev string            `json:"hardware_rev,omitempty"`
	UUID        string            `json:"uuid,omitempty"`
	State       string            `json:"state,omitempty"`
	LastSeen    time.Time         `json:"last_seen"`
	Parameters  map[string]string `json:"parameters"`
}

// TLVData represents a Type-Length-Value data structure
type TLVData struct {
	Type   uint8
	Length uint8
	Value  []byte
}

// EncodeTLV encodes a list of TLV data into a byte slice
func EncodeTLV(tlvs []TLVData) []byte {
	var buf bytes.Buffer
	for _, tlv := range tlvs {
		buf.WriteByte(tlv.Type)
		buf.WriteByte(tlv.Length)
		buf.Write(tlv.Value)
	}
	return buf.Bytes()
}

// DecodeTLV decodes a byte slice into TLV data structures
func DecodeTLV(data []byte) []TLVData {
	var tlvs []TLVData
	offset := 0

	for offset < len(data) {
		if offset+2 > len(data) {
			break
		}

		tlv := TLVData{
			Type:   data[offset],
			Length: data[offset+1],
		}
		offset += 2

		if offset+int(tlv.Length) > len(data) {
			break
		}

		tlv.Value = data[offset : offset+int(tlv.Length)]
		offset += int(tlv.Length)

		tlvs = append(tlvs, tlv)
	}

	return tlvs
}

// ParsePacket parses a UDAP packet header and returns the parsed
// header, the remaining payload bytes (or nil if the packet was
// header-only), and any decode error.
//
// Real UDAP packets are at least UDAPHeaderSize (27) bytes — the
// header is fixed-width — and the UDAPType field at bytes 18-19 must
// equal TypeUCP (0xC001). Anything shorter or with a non-UCP UDAPType
// is junk on our socket (mDNS leakage, stray broadcasts) and is
// rejected so callers don't try to interpret garbage.
//
// An earlier permissive Format-2 fallback parsed any 4+-byte payload
// by taking bytes 0-1 as a UCPMethod; that was a workaround for a
// since-fixed off-by-2 in UDAPHeaderSize and is no longer needed.
func ParsePacket(packetData []byte) (*Packet, []byte, error) {
	if len(packetData) < UDAPHeaderSize {
		return nil, nil, fmt.Errorf("packet too short: %d bytes (minimum %d)", len(packetData), UDAPHeaderSize)
	}

	var packet Packet
	if err := binary.Read(bytes.NewReader(packetData), binary.BigEndian, &packet); err != nil {
		return nil, nil, fmt.Errorf("decode UDAP header: %w", err)
	}
	if packet.UDAPType != TypeUCP {
		return nil, nil, fmt.Errorf("not a UDAP/UCP packet: UDAPType=0x%04x", packet.UDAPType)
	}

	var data []byte
	if len(packetData) > UDAPHeaderSize {
		data = packetData[UDAPHeaderSize:]
	}
	return &packet, data, nil
}

// getLocalIPs returns a map of all local IP addresses for filtering
func getLocalIPs() map[string]bool {
	localIPs := make(map[string]bool)
	if interfaces, err := net.Interfaces(); err == nil {
		for _, iface := range interfaces {
			if addrs, err := iface.Addrs(); err == nil {
				for _, addr := range addrs {
					if ipnet, ok := addr.(*net.IPNet); ok {
						localIPs[ipnet.IP.String()] = true
					}
				}
			}
		}
	}
	// Also add common IPv6 loopback/unspecified addresses
	localIPs["::"] = true
	localIPs["::1"] = true
	return localIPs
}
