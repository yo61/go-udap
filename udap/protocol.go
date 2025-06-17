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

	// UCP Methods - Based on Lua implementation
	MethodDiscover = 0x0001 // Discovery method
	MethodGetIP    = 0x0002 // Get IP method (also used as data response)
	MethodReset    = 0x0004 // Reset method
	MethodGetData  = 0x0005 // Get data method
	MethodSetData  = 0x0006 // Set data method
	MethodError    = 0x0007 // Error method
	MethodAdvDisc  = 0x0009 // Advanced discovery method

	// Method aliases for backward compatibility
	MethodDataResp = MethodGetIP // GetIP (0x0002) is used for data responses

	// UCP Flags
	FlagsDiscover = 0x01
)

// ConfigSetting represents a configuration parameter with its NVRAM offset and length
type ConfigSetting struct {
	Offset uint16
	Length uint16
}

// ConfigSettings maps parameter names to their NVRAM offsets and lengths
// Based on the authoritative Lua implementation from LMS-Community/squeezeplay
var ConfigSettings = map[string]ConfigSetting{
	"lan_ip_mode":         {4, 1},
	"lan_network_address": {5, 4},
	"lan_subnet_mask":     {9, 4},
	"lan_gateway":         {13, 4},
	"hostname":            {17, 33},
	"bridging":            {50, 1},
	"interface":           {52, 1},
	"primary_dns":         {59, 4},
	"secondary_dns":       {67, 4},
	"server_address":      {71, 4},
	"lms_address":         {79, 4},
	"wireless_mode":       {173, 1},
	"wireless_SSID":       {183, 33},
	"wireless_channel":    {216, 1},
	"wireless_region_id":  {218, 1},
	"wireless_keylen":     {220, 1},
	"wireless_wep_key":    {222, 13},
	"wireless_wep_key_1":  {235, 13},
	"wireless_wep_key_2":  {248, 13},
	"wireless_wep_key_3":  {261, 13},
	"wireless_wep_on":     {274, 1},
	"wireless_wpa_cipher": {275, 1},
	"wireless_wpa_mode":   {276, 1},
	"wireless_wpa_on":     {277, 1},
	"wireless_wpa_psk":    {278, 64},
	// Alternative parameter names used by some tools
	// "slimserver_address":    {79, 4}, // Alias for lms_address
	// "squeezecenter_address": {79, 4}, // Alias for lms_address
	// Shortened names for wireless params
	// "mode":       {173, 1},
	// "SSID":       {183, 33},
	// "channel":    {216, 1},
	// "region_id":  {218, 1},
	// "keylen":     {220, 1},
	// "wep_key":    {222, 13},
	// "wep_key_1":  {235, 13},
	// "wep_key_2":  {248, 13},
	// "wep_key_3":  {261, 13},
	// "wep_on":     {274, 1},
	// "wpa_cipher": {275, 1},
	// "wpa_mode":   {276, 1},
	// "wpa_on":     {277, 1},
	// "wpa_psk":    {278, 64},
}

// KnownParameters is a list of all known UDAP parameters
var KnownParameters = []string{
	"lan_ip_mode",
	"lan_network_address",
	"lan_subnet_mask",
	"lan_gateway",
	"hostname",
	"bridging",
	"interface",
	"primary_dns",
	"secondary_dns",
	"server_address",
	"lms_address",
	"wireless_mode",
	"wireless_SSID",
	"wireless_channel",
	"wireless_region_id",
	"wireless_keylen",
	"wireless_wep_key",
	"wireless_wep_key_1",
	"wireless_wep_key_2",
	"wireless_wep_key_3",
	"wireless_wep_on",
	"wireless_wpa_cipher",
	"wireless_wpa_mode",
	"wireless_wpa_on",
	"wireless_wpa_psk",
}

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

// Device represents a discovered Squeezebox device
type Device struct {
	MAC        string            `json:"mac"`
	IP         string            `json:"ip"`
	Name       string            `json:"name"`
	Model      string            `json:"model"`
	Firmware   string            `json:"firmware"`
	UUID       string            `json:"uuid"`
	LastSeen   time.Time         `json:"last_seen"`
	Parameters map[string]string `json:"parameters"` // Stores all device parameters
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

// ParsePacket parses a UDAP packet and returns packet structure
func ParsePacket(packetData []byte) (*Packet, []byte, error) {
	if len(packetData) < 4 {
		return nil, nil, fmt.Errorf("packet too short: %d bytes", len(packetData))
	}

	// Check for different packet formats
	// Format 1: Standard UDAP packet (starts with destination broadcast flag)
	if len(packetData) >= 25 {
		var packet Packet
		buf := bytes.NewReader(packetData)
		err := binary.Read(buf, binary.BigEndian, &packet)
		if err == nil {
			// Check if this looks like a valid UDAP packet
			if packet.UDAPType == TypeUCP || packet.DstType <= 2 || packet.SrcType <= 2 {
				headerSize := 25
				var data []byte
				if len(packetData) > headerSize {
					data = packetData[headerSize:]
				}
				return &packet, data, nil
			}
		}
	}

	// Format 2: Raw response packet (like we're seeing in the capture)
	// Starts with method indicator (00 01 = discovery response, 00 02 = set response, etc)
	if len(packetData) >= 4 {
		method := binary.BigEndian.Uint16(packetData[0:2])
		// Create a minimal packet structure for raw responses
		packet := &Packet{
			UCPMethod: method,
			UDAPType:  TypeUCP,
			DstType:   AddrTypeUDP,
			SrcType:   AddrTypeUDP,
		}

		// Skip the method bytes and return the rest as data
		var data []byte
		if len(packetData) > 4 {
			data = packetData[4:]
		}

		// Note: This is in a package-level function, would need logger parameter to use structured logging
		// For now, keeping the printf statement as it's used in packet parsing
		return packet, data, nil
	}

	return nil, nil, fmt.Errorf("unrecognized packet format: length=%d bytes", len(packetData))
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
