package udap

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Client handles UDAP protocol communication
type Client struct {
	conn     *net.UDPConn
	devices  map[string]*Device
	sequence uint32
	logger   Logger
}

// NewClient creates a new UDAP client
func NewClient() (*Client, error) {
	return NewClientWithLogger(NewStructuredLogger())
}

// NewClientWithLogger creates a new UDAP client with a custom logger,
// bound to the standard UDAP port (17784).
func NewClientWithLogger(logger Logger) (*Client, error) {
	return newClientWithPort(Port, logger)
}

// newClientWithPort creates a UDAP client bound to the given UDP port.
// Port 0 lets the OS pick a free ephemeral port — useful for tests so they
// don't collide with each other or with anything else holding port 17784.
func newClientWithPort(port int, logger Logger) (*Client, error) {
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return nil, err
	}

	enableBroadcast(conn, logger)

	logger.Debug("Created UDP socket", "address", conn.LocalAddr().String())

	return &Client{
		conn:     conn,
		devices:  make(map[string]*Device),
		sequence: 1,
		logger:   logger,
	}, nil
}

// Close closes the UDAP client connection
func (c *Client) Close() error {
	return c.conn.Close()
}

// createUdapPacket creates the common UDAP packet header structure
// All UDAP messages share the same initial format up to UCPMethod
func (c *Client) createUdapPacket(dstMAC [6]byte, method uint16, flags uint8, broadcast bool) Packet {
	var dstBroadcast uint8
	if broadcast {
		dstBroadcast = 1
	}

	packet := Packet{
		DstBroadcast: dstBroadcast,
		DstType:      AddrTypeETH, // Always use Ethernet addressing
		DstAddress:   dstMAC,
		SrcBroadcast: 0,                      // Source is never broadcast
		SrcType:      AddrTypeETH,            // Use ETH type like Lua implementation
		SrcAddress:   [MACAddressSize]byte{}, // All zeros for source
		Sequence:     uint16(c.sequence),
		UDAPType:     TypeUCP, // Always 0xC001
		UCPFlags:     flags,
		UAPClass:     [4]byte{0x00, 0x01, 0x00, 0x01}, // Always UAP_CLASS_UCP
		UCPMethod:    method,
	}

	c.sequence++
	return packet
}

// CreateDiscoveryPacket creates a standard UDAP discovery packet (method 0x0001)
func (c *Client) CreateDiscoveryPacket() []byte {
	// Standard discovery uses broadcast to all zeros MAC
	packet := c.createUdapPacket(
		[MACAddressSize]byte{}, // Broadcast MAC
		MethodDiscover,         // 0x0001
		FlagsDiscover,          // 0x01
		true,                   // Broadcast
	)

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, packet)
	return buf.Bytes()
}

// CreateAdvancedDiscoveryPacket creates an advanced UDAP discovery packet (method 0x0009)
func (c *Client) CreateAdvancedDiscoveryPacket() []byte {
	// Advanced discovery uses broadcast to all zeros MAC
	packet := c.createUdapPacket(
		[MACAddressSize]byte{}, // Broadcast MAC
		MethodAdvDisc,          // 0x0009
		FlagsDiscover,          // 0x01
		true,                   // Broadcast
	)

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, packet)
	return buf.Bytes()
}

// CreateGetDataPacket creates a UDAP GetData packet for retrieving parameters
func (c *Client) CreateGetDataPacket(device *Device, params []string) []byte {
	// Convert MAC address to bytes
	macBytes := c.parseMACAddress(device.MAC)

	packet := c.createUdapPacket(
		macBytes,
		MethodGetData, // 0x0005
		0x01,          // Request flag
		false,         // Not broadcast
	)

	// Create TLV data for parameters
	var tlvs []TLVData
	for _, param := range params {
		tlv := TLVData{
			Type:   TLVTypeParameterName, // Parameter name type
			Length: uint8(len(param)),
			Value:  []byte(param),
		}
		tlvs = append(tlvs, tlv)
	}

	// Encode packet header
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, packet)

	// Append TLV data
	buf.Write(EncodeTLV(tlvs))

	return buf.Bytes()
}

// parseMACAddress converts a MAC address string to a byte array
func (c *Client) parseMACAddress(mac string) [6]byte {
	var macBytes [6]byte
	fmt.Sscanf(mac, "%02x:%02x:%02x:%02x:%02x:%02x",
		&macBytes[0], &macBytes[1], &macBytes[2], &macBytes[3], &macBytes[4], &macBytes[5])
	return macBytes
}

// CreateSetDataPacket creates a UDAP SetData packet using the correct Lua format
// Based on the createSetData function from the authoritative Lua implementation
func (c *Client) CreateSetDataPacket(device *Device, params map[string]string) []byte {
	c.logger.Info("Creating SetData packet", "device_mac", device.MAC, "param_count", len(params))

	// Convert MAC address to bytes
	macBytes := c.parseMACAddress(device.MAC)

	packet := c.createUdapPacket(
		macBytes,
		MethodSetData, // 0x0006
		0x01,          // Request flag
		false,         // Not broadcast
	)

	// Create payload following Lua createSetData format:
	// - 16 bytes username (zeros)
	// - 16 bytes password (zeros)
	// - 2 bytes number of parameters
	// - For each parameter: 2 bytes offset + 2 bytes length + data (padded to length)
	buf := new(bytes.Buffer)

	// Write packet header
	binary.Write(buf, binary.BigEndian, packet)

	// Write username field
	buf.Write(make([]byte, UsernameFieldSize))

	// Write password field
	buf.Write(make([]byte, PasswordFieldSize))

	// Build a list of parameters to write (with their settings)
	type paramEntry struct {
		name    string
		value   string
		setting ConfigSetting
	}
	var paramList []paramEntry

	for param, value := range params {
		if setting, exists := ConfigSettings[param]; exists {
			paramList = append(paramList, paramEntry{
				name:    param,
				value:   value,
				setting: setting,
			})
		} else {
			c.logger.Warn("Unknown parameter skipped", "param", param, "device_mac", device.MAC)
		}
	}

	// Sort parameters by offset for consistent ordering
	sort.Slice(paramList, func(i, j int) bool {
		return paramList[i].setting.Offset < paramList[j].setting.Offset
	})

	// Write number of parameters (2 bytes big-endian)
	binary.Write(buf, binary.BigEndian, uint16(len(paramList)))
	c.logger.Debug("Parameters sorted", "param_count", len(paramList), "device_mac", device.MAC)

	// Write each parameter using offset/length format
	for _, entry := range paramList {
		// Write offset (2 bytes big-endian)
		binary.Write(buf, binary.BigEndian, entry.setting.Offset)

		// Write length (2 bytes big-endian)
		binary.Write(buf, binary.BigEndian, entry.setting.Length)

		// Convert value based on parameter type
		var data []byte

		switch entry.setting.Length {
		case 4:
			// Check if this is an IP address parameter (all 4-byte parameters are IP addresses)
			// Parse IP address and convert to 4 bytes
			ip := net.ParseIP(entry.value)
			if ip != nil {
				ip = ip.To4()
				if ip != nil {
					data = []byte(ip)
				} else {
					c.logger.Warn("Invalid IPv4 address", "param", entry.name, "value", entry.value, "device_mac", device.MAC)
					data = make([]byte, 4) // Use zeros
				}
			} else {
				c.logger.Warn("Could not parse IP address", "param", entry.name, "value", entry.value, "device_mac", device.MAC)
				data = make([]byte, 4) // Use zeros
			}
		case 1:
			// Single byte numeric values - convert string to integer
			if val, err := strconv.ParseUint(entry.value, 10, 8); err == nil {
				data = []byte{byte(val)}
			} else {
				c.logger.Warn("Invalid numeric value", "param", entry.name, "value", entry.value, "type", "uint8", "device_mac", device.MAC)
				data = []byte{0} // Use zero as fallback
			}
		case 2:
			// Two byte numeric values - convert string to integer
			if val, err := strconv.ParseUint(entry.value, 10, 16); err == nil {
				data = make([]byte, 2)
				binary.BigEndian.PutUint16(data, uint16(val))
			} else {
				c.logger.Warn("Invalid numeric value", "param", entry.name, "value", entry.value, "type", "uint16", "device_mac", device.MAC)
				data = make([]byte, 2) // Use zeros as fallback
			}
		default:
			// String data
			data = []byte(entry.value)
			if len(data) > int(entry.setting.Length) {
				data = data[:entry.setting.Length] // Truncate if too long
			}
		}

		// Pad with zeros to reach the required length
		padded := make([]byte, entry.setting.Length)
		copy(padded, data)
		buf.Write(padded)

		c.logger.Debug("Parameter details",
			"param", entry.name,
			"offset_hex", fmt.Sprintf("0x%04x", entry.setting.Offset),
			"offset_dec", entry.setting.Offset,
			"length", entry.setting.Length,
			"value", entry.value)
	}

	// Log the complete packet for debugging
	packetBytes := buf.Bytes()
	c.logger.Debug("SetData packet details",
		"total_bytes", len(packetBytes),
		"header_hex", fmt.Sprintf("%x", packetBytes[:min(UDAPHeaderSize, len(packetBytes))]),
		"username_hex", func() string {
			start, end := UDAPHeaderSize, min(UDAPHeaderSize+UsernameFieldSize, len(packetBytes))
			if end > start {
				return fmt.Sprintf("%x", packetBytes[start:end])
			}
			return ""
		}(),
		"password_hex", func() string {
			start, end := UDAPHeaderSize+UsernameFieldSize, min(UDAPHeaderSize+UsernameFieldSize+PasswordFieldSize, len(packetBytes))
			if end > start {
				return fmt.Sprintf("%x", packetBytes[start:end])
			}
			return ""
		}(),
		"param_data_hex", func() string {
			payloadStart := UDAPHeaderSize + UsernameFieldSize + PasswordFieldSize
			if len(packetBytes) > payloadStart {
				return fmt.Sprintf("%x", packetBytes[payloadStart:])
			}
			return ""
		}())

	return packetBytes
}

// CreateResetPacket creates a UDAP reset packet to restart the device
func (c *Client) CreateResetPacket(device *Device) []byte {
	// Convert MAC address to bytes
	macBytes := c.parseMACAddress(device.MAC)

	// Reset uses the MethodReset (0x0004) not MethodError
	packet := c.createUdapPacket(
		macBytes,
		MethodReset, // 0x0004 - Reset method from Lua implementation
		0x01,        // Request flag
		false,       // Not broadcast - send directly to device
	)

	// Encode packet header
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, packet)

	// Based on Lua createReset, no additional payload is needed
	// The reset command is just the header with MethodReset

	return buf.Bytes()
}

// CreateSaveDataPacket creates a UDAP SaveData packet using the correct Lua format
// This is an alias for CreateSetDataPacket since save_data uses the SetData method
func (c *Client) CreateSaveDataPacket(device *Device, allParams map[string]string) []byte {
	// Save data uses the same format as SetData with method 0x0006
	return c.CreateSetDataPacket(device, allParams)
}

// ListDevices returns a list of discovered devices
func (c *Client) ListDevices() []*Device {
	devices := make([]*Device, 0, len(c.devices))
	for _, device := range c.devices {
		devices = append(devices, device)
	}
	return devices
}

// GetDevice returns a device by MAC address
func (c *Client) GetDevice(mac string) *Device {
	return c.devices[mac]
}

// GetDevices returns the devices map
func (c *Client) GetDevices() map[string]*Device {
	return c.devices
}

// PacketCaptureConfig configures packet capture behavior
type PacketCaptureConfig struct {
	Purpose    string        // Description of what this capture is for
	Timeout    time.Duration // Timeout for the entire operation
	SourceIP   string        // Expected source IP (empty string accepts any IP including 0.0.0.0)
	SourcePort uint16        // Expected source port (defaults to 17784)
}

// PacketCaptureResult contains the result of packet capture
type PacketCaptureResult struct {
	Payload []byte
	SrcIP   string
	SrcPort uint16
}

// getActiveNetworkInterface returns a suitable network interface name for packet operations.
// This replaces pcap.FindAllDevs() with pure Go networking.
func getActiveNetworkInterface() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get network interfaces: %w", err)
	}

	// First pass: prefer "en" (macOS) or "eth" (Linux) interfaces
	for _, iface := range interfaces {
		// Skip down or loopback interfaces
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// Prefer standard interface prefixes
		if strings.HasPrefix(iface.Name, "en") || strings.HasPrefix(iface.Name, "eth") {
			addrs, err := iface.Addrs()
			if err == nil && len(addrs) > 0 {
				return iface.Name, nil
			}
		}
	}

	// Fallback: return first active interface with addresses
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err == nil && len(addrs) > 0 {
			return iface.Name, nil
		}
	}

	return "", fmt.Errorf("no suitable network interface found")
}

// capturePacketWithContext performs UDP packet capture using standard Go networking.
// This replaces the previous pcap-based implementation for cross-platform compatibility.
func (c *Client) capturePacketWithContext(ctx context.Context, config PacketCaptureConfig) (*PacketCaptureResult, error) {
	// Set defaults
	if config.SourcePort == 0 {
		config.SourcePort = Port
	}
	if config.Purpose == "" {
		config.Purpose = "packet capture"
	}

	// Create context with timeout if specified
	captureCtx := ctx
	if config.Timeout > 0 {
		var cancel context.CancelFunc
		captureCtx, cancel = context.WithTimeout(ctx, config.Timeout)
		defer cancel()
	}

	// Create a dedicated listening socket on the UDAP port
	// Use udp4 explicitly for consistency with the main client socket
	listenAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("0.0.0.0:%d", Port))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve listen address for %s: %w", config.Purpose, err)
	}

	conn, err := net.ListenUDP("udp4", listenAddr)
	if err != nil {
		// If we can't create a new listener (port in use), use the existing client connection
		c.logger.Debug("Using existing connection for capture", "reason", err.Error(), "purpose", config.Purpose)
		return c.capturePacketFromExistingConn(captureCtx, config)
	}
	defer conn.Close()

	// Enable socket options for proper broadcast reception
	enableBroadcastSimple(conn, c.logger)

	// Set read deadline from context
	if deadline, ok := captureCtx.Deadline(); ok {
		conn.SetReadDeadline(deadline)
	}

	c.logger.Info("Started UDP capture", "purpose", config.Purpose, "port", Port)

	buffer := make([]byte, 2048)
	for {
		select {
		case <-captureCtx.Done():
			return nil, fmt.Errorf("capture timeout for %s: %w", config.Purpose, captureCtx.Err())
		default:
			n, srcAddr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					return nil, fmt.Errorf("capture timeout for %s: no matching packet received", config.Purpose)
				}
				return nil, fmt.Errorf("read error during %s: %w", config.Purpose, err)
			}

			srcIP := srcAddr.IP.String()
			srcPort := uint16(srcAddr.Port)

			// Filter by source criteria
			// Accept if: no specific source IP required OR source IP matches OR source is 0.0.0.0 (bootstrap mode)
			ipMatches := config.SourceIP == "" || srcIP == config.SourceIP || srcIP == "0.0.0.0"
			portMatches := config.SourcePort == 0 || srcPort == config.SourcePort

			if ipMatches && portMatches && n > 0 {
				c.logger.Info("Found matching packet", "purpose", config.Purpose, "bytes", n, "src_ip", srcIP, "src_port", srcPort)

				result := &PacketCaptureResult{
					Payload: make([]byte, n),
					SrcIP:   srcIP,
					SrcPort: srcPort,
				}
				copy(result.Payload, buffer[:n])
				return result, nil
			}
		}
	}
}

// flushStalePackets reads and discards any pending packets from the socket buffer.
// This prevents stale responses from previous operations being mistaken for new responses.
func (c *Client) flushStalePackets() {
	buffer := make([]byte, 2048)
	flushedCount := 0

	// Set a very short deadline to quickly drain any pending packets
	c.conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
	defer c.conn.SetReadDeadline(time.Time{})

	for {
		n, addr, err := c.conn.ReadFromUDP(buffer)
		if err != nil {
			// Timeout or other error means no more pending packets
			break
		}
		flushedCount++
		c.logger.Debug("Flushed stale packet", "bytes", n, "src_ip", addr.IP.String())
	}

	if flushedCount > 0 {
		c.logger.Debug("Flushed stale packets from buffer", "count", flushedCount)
	}
}

// capturePacketFromExistingConn captures a packet using the client's existing UDP connection.
// This is used as a fallback when a dedicated capture socket cannot be created.
func (c *Client) capturePacketFromExistingConn(ctx context.Context, config PacketCaptureConfig) (*PacketCaptureResult, error) {
	// First, flush any stale packets from the buffer by reading with a very short timeout
	c.flushStalePackets()

	// Set read deadline from context
	if deadline, ok := ctx.Deadline(); ok {
		c.conn.SetReadDeadline(deadline)
	}
	defer c.conn.SetReadDeadline(time.Time{})

	c.logger.Info("Started UDP capture on existing connection", "purpose", config.Purpose, "port", Port)
	c.logger.Debug("Capture filter", "expected_source_ip", config.SourceIP, "expected_source_port", config.SourcePort)

	buffer := make([]byte, 2048)
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("capture timeout for %s: %w", config.Purpose, ctx.Err())
		default:
			n, srcAddr, err := c.conn.ReadFromUDP(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					return nil, fmt.Errorf("capture timeout for %s: no matching packet received", config.Purpose)
				}
				return nil, fmt.Errorf("read error during %s: %w", config.Purpose, err)
			}

			srcIP := srcAddr.IP.String()
			srcPort := uint16(srcAddr.Port)

			c.logger.Debug("Capture received packet", "purpose", config.Purpose, "bytes", n, "src_ip", srcIP, "src_port", srcPort)

			// Filter by source criteria
			// Accept if: no specific source IP required OR source IP matches OR source is 0.0.0.0 (bootstrap mode)
			ipMatches := config.SourceIP == "" || srcIP == config.SourceIP || srcIP == "0.0.0.0"
			portMatches := config.SourcePort == 0 || srcPort == config.SourcePort

			if !ipMatches {
				c.logger.Debug("Packet filtered out", "reason", "IP mismatch", "expected", config.SourceIP, "got", srcIP)
				continue
			}
			if !portMatches {
				c.logger.Debug("Packet filtered out", "reason", "port mismatch", "expected", config.SourcePort, "got", srcPort)
				continue
			}

			if n > 0 {
				c.logger.Info("Found matching packet", "purpose", config.Purpose, "bytes", n, "src_ip", srcIP, "src_port", srcPort)

				result := &PacketCaptureResult{
					Payload: make([]byte, n),
					SrcIP:   srcIP,
					SrcPort: srcPort,
				}
				copy(result.Payload, buffer[:n])
				return result, nil
			}
		}
	}
}
