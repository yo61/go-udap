package udap

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

// Regular expressions for validation
var (
	macRegex  = regexp.MustCompile(`^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`)
	uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
)

// isValidMAC validates MAC address format (XX:XX:XX:XX:XX:XX)
func isValidMAC(mac string) bool {
	return macRegex.MatchString(mac)
}

// isValidUUID validates UUID format
func isValidUUID(uuid string) bool {
	return uuid == "" || uuidRegex.MatchString(uuid)
}

// isValidIP validates IPv4 address format
func isValidIP(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	// Check if it's IPv4
	return parsedIP.To4() != nil
}

// validateParameter validates a configuration parameter based on its name and type
func validateParameter(name, value string) error {
	setting, exists := ConfigSettings[name]
	if !exists {
		// Unknown parameter, but we'll allow it
		return nil
	}

	switch setting.Length {
	case 1:
		// Single byte numeric value
		var val uint8
		if _, err := fmt.Sscanf(value, "%d", &val); err != nil {
			return fmt.Errorf("expected numeric value (0-255), got %q", value)
		}
	case 2:
		// Two byte numeric value
		var val uint16
		if _, err := fmt.Sscanf(value, "%d", &val); err != nil {
			return fmt.Errorf("expected numeric value (0-65535), got %q", value)
		}
	case 4:
		// IP address parameters
		if !isValidIP(value) {
			return fmt.Errorf("expected valid IPv4 address, got %q", value)
		}
	default:
		// String parameters
		if len(value) > int(setting.Length) {
			return fmt.Errorf("value too long (max %d chars), got %d", setting.Length, len(value))
		}
	}

	// Additional parameter-specific validation
	switch name {
	case "wireless_mode":
		// 0=infrastructure, 1=ad-hoc
		if value != "0" && value != "1" {
			return fmt.Errorf("must be 0 (infrastructure) or 1 (ad-hoc)")
		}
	case "wireless_channel":
		// Typically 1-11 in US, 1-13 in EU
		var ch int
		fmt.Sscanf(value, "%d", &ch)
		if ch < 1 || ch > 13 {
			return fmt.Errorf("must be between 1 and 13")
		}
	case "wireless_keylen":
		// 5 or 13 for WEP keys
		if value != "5" && value != "13" {
			return fmt.Errorf("must be 5 or 13 for WEP keys")
		}
	case "wireless_wpa_psk":
		// WPA PSK should be 8-63 characters
		if len(value) < 8 || len(value) > 63 {
			return fmt.Errorf("must be 8-63 characters")
		}
	case "wireless_SSID":
		// SSID should be 1-32 characters
		if len(value) < 1 || len(value) > 32 {
			return fmt.Errorf("must be 1-32 characters")
		}
	}

	return nil
}

// Validate checks if the Device struct contains valid data
func (d *Device) Validate() error {
	// Validate MAC address format
	if !isValidMAC(d.MAC) && !strings.HasPrefix(d.MAC, "udp:") {
		return fmt.Errorf("invalid MAC address format: %s", d.MAC)
	}

	// Validate IP address if not in bootstrap mode
	if d.IP != "" && d.IP != "0.0.0.0" && !isValidIP(d.IP) {
		return fmt.Errorf("invalid IP address: %s", d.IP)
	}

	// Validate UUID if provided
	if !isValidUUID(d.UUID) {
		return fmt.Errorf("invalid UUID format: %s", d.UUID)
	}

	// Validate name length
	if len(d.Name) > MaxDeviceNameLength {
		return fmt.Errorf("device name too long (max %d chars): %d", MaxDeviceNameLength, len(d.Name))
	}

	// Validate parameters
	for name, value := range d.Parameters {
		if err := validateParameter(name, value); err != nil {
			return fmt.Errorf("invalid parameter %s: %w", name, err)
		}
	}

	return nil
}

// Validate checks if the Packet struct contains valid data
func (p *Packet) Validate() error {
	// Validate address types
	if p.DstType != AddrTypeETH && p.DstType != AddrTypeUDP {
		return fmt.Errorf("invalid destination address type: 0x%02x", p.DstType)
	}
	if p.SrcType != AddrTypeETH && p.SrcType != AddrTypeUDP {
		return fmt.Errorf("invalid source address type: 0x%02x", p.SrcType)
	}

	// Validate broadcast flags
	if p.DstBroadcast > 1 {
		return fmt.Errorf("invalid destination broadcast flag: %d", p.DstBroadcast)
	}
	if p.SrcBroadcast > 1 {
		return fmt.Errorf("invalid source broadcast flag: %d", p.SrcBroadcast)
	}

	// Validate UDAP type
	if p.UDAPType != TypeUCP {
		return fmt.Errorf("invalid UDAP type: 0x%04x (expected 0x%04x)", p.UDAPType, TypeUCP)
	}

	// Validate UCP method
	switch p.UCPMethod {
	case MethodDiscover, MethodGetIP, MethodReset,
		MethodGetData, MethodSetData, MethodError,
		MethodAdvDisc:
		// Valid methods
	default:
		return fmt.Errorf("unknown UCP method: 0x%04x", p.UCPMethod)
	}

	// Validate UAP class
	expectedClass := [4]byte{0x00, 0x01, 0x00, 0x01}
	if p.UAPClass != expectedClass {
		return fmt.Errorf("invalid UAP class: %v (expected %v)", p.UAPClass, expectedClass)
	}

	return nil
}

// Validate checks if the TLVData struct contains valid data
func (t *TLVData) Validate() error {
	// Check if declared length matches actual value length
	if int(t.Length) != len(t.Value) {
		return fmt.Errorf("TLV length mismatch: declared %d, actual %d", t.Length, len(t.Value))
	}

	// Check for maximum length (uint8 max)
	if len(t.Value) > MaxTLVValueLength {
		return fmt.Errorf("TLV value too long: %d bytes (max %d)", len(t.Value), MaxTLVValueLength)
	}

	return nil
}

// Validate checks if the ConfigSetting struct contains valid data
func (c *ConfigSetting) Validate() error {
	// Check for reasonable offset (NVRAM typically < 64KB)
	if c.Offset > MaxNVRAMOffset {
		return fmt.Errorf("offset too large: %d", c.Offset)
	}

	// Check for reasonable length
	if c.Length == 0 {
		return fmt.Errorf("length cannot be zero")
	}
	if c.Length > MaxConfigLength {
		return fmt.Errorf("length too large: %d (max %d)", c.Length, MaxConfigLength)
	}

	return nil
}

// Validate checks if the PacketCaptureConfig struct contains valid data
func (p *PacketCaptureConfig) Validate() error {
	// Validate timeout
	if p.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}
	if p.Timeout > MaxTimeoutMinutes*60*time.Second {
		return fmt.Errorf("timeout too large (max %d minutes)", MaxTimeoutMinutes)
	}

	// Validate source IP if provided
	if p.SourceIP != "" && !isValidIP(p.SourceIP) {
		return fmt.Errorf("invalid source IP: %s", p.SourceIP)
	}

	// Validate source port
	if p.SourcePort > 65535 {
		return fmt.Errorf("invalid source port: %d", p.SourcePort)
	}

	return nil
}

// Validate checks if the Client struct is properly initialized
func (c *Client) Validate() error {
	if c.conn == nil {
		return fmt.Errorf("UDP connection not initialized")
	}

	if c.logger == nil {
		return fmt.Errorf("logger not initialized")
	}

	if c.devices == nil {
		return fmt.Errorf("devices map not initialized")
	}

	return nil
}
