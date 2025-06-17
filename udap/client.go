package udap

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sort"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
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

// NewClientWithLogger creates a new UDAP client with a custom logger
func NewClientWithLogger(logger Logger) (*Client, error) {
	// Listen on all interfaces for UDP port 17784
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", Port))
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	// Enable broadcast reception on the listening socket
	file, err := conn.File()
	if err == nil {
		defer file.Close() // Ensure file is always closed
		fd := int(file.Fd())
		// Enable SO_BROADCAST for receiving broadcast packets
		err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
		if err != nil {
			logger.Warn("Failed to enable socket option", "option", "SO_BROADCAST", "socket", "listening", "error", err)
		} else {
			logger.Debug("Socket option enabled", "option", "SO_BROADCAST", "socket", "listening")
		}
		// Enable SO_REUSEADDR to allow multiple listeners
		err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
		if err != nil {
			logger.Warn("Failed to enable socket option", "option", "SO_REUSEADDR", "socket", "listening", "error", err)
		}
	}

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
		"header_hex", fmt.Sprintf("%x", packetBytes[:min(25, len(packetBytes))]),
		"username_hex", func() string {
			start, end := 25, min(41, len(packetBytes))
			if end > start {
				return fmt.Sprintf("%x", packetBytes[start:end])
			}
			return ""
		}(),
		"password_hex", func() string {
			start, end := 41, min(57, len(packetBytes))
			if end > start {
				return fmt.Sprintf("%x", packetBytes[start:end])
			}
			return ""
		}(),
		"param_data_hex", func() string {
			if len(packetBytes) > 57 {
				return fmt.Sprintf("%x", packetBytes[57:])
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
	Filter     string        // BPF filter (defaults to "udp port 17784")
	SourceIP   string        // Expected source IP (defaults to "0.0.0.0")
	SourcePort uint16        // Expected source port (defaults to 17784)
}

// PacketCaptureResult contains the result of packet capture
type PacketCaptureResult struct {
	Payload []byte
	SrcIP   string
	SrcPort uint16
}

// capturePacketWithContext performs packet capture with proper lifecycle management
func (c *Client) capturePacketWithContext(ctx context.Context, config PacketCaptureConfig) (*PacketCaptureResult, error) {
	// Set defaults
	if config.Filter == "" {
		config.Filter = "udp port 17784"
	}
	if config.SourceIP == "" {
		config.SourceIP = "0.0.0.0"
	}
	if config.SourcePort == 0 {
		config.SourcePort = 17784
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

	// Find suitable network interface
	interfaces, err := pcap.FindAllDevs()
	if err != nil {
		c.logger.Warn("Could not find network interfaces", "error", err)
		return nil, fmt.Errorf("failed to find network interfaces: %w", err)
	}

	var selectedInterface string
	for _, iface := range interfaces {
		if len(iface.Addresses) > 0 && (len(iface.Name) >= 3 && iface.Name[:2] == "en") {
			selectedInterface = iface.Name
			break
		}
	}

	if selectedInterface == "" {
		c.logger.Warn("No suitable network interface found")
		return nil, fmt.Errorf("no suitable network interface found")
	}

	// Open interface for capturing
	handle, err := pcap.OpenLive(selectedInterface, 1600, true, time.Millisecond*100)
	if err != nil {
		c.logger.Error("Error opening interface", "interface", selectedInterface, "error", err)
		return nil, fmt.Errorf("failed to open interface %s: %w", selectedInterface, err)
	}
	defer handle.Close()

	// Set BPF filter
	err = handle.SetBPFFilter(config.Filter)
	if err != nil {
		c.logger.Warn("Could not set BPF filter", "filter", config.Filter, "error", err)
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	c.logger.Info("Started packet capture", "interface", selectedInterface, "purpose", config.Purpose, "filter", config.Filter)

	// Use buffered channels to prevent goroutine leaks
	resultChan := make(chan *PacketCaptureResult, 1)
	errorChan := make(chan error, 1)

	// Start packet capture in goroutine with proper cleanup
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(resultChan)
		defer close(errorChan)

		for {
			select {
			case <-captureCtx.Done():
				errorChan <- captureCtx.Err()
				return
			case packet := <-packetSource.Packets():
				if packet == nil {
					continue
				}

				// Parse IP layer
				ipLayer := packet.Layer(layers.LayerTypeIPv4)
				if ipLayer == nil {
					continue
				}
				ip := ipLayer.(*layers.IPv4)

				// Parse UDP layer
				udpLayer := packet.Layer(layers.LayerTypeUDP)
				if udpLayer == nil {
					continue
				}
				udp := udpLayer.(*layers.UDP)

				// Check if this matches our criteria
				if ip.SrcIP.String() == config.SourceIP && uint16(udp.SrcPort) == config.SourcePort && len(udp.Payload) > 0 {
					c.logger.Info("Found matching packet", "purpose", config.Purpose, "bytes", len(udp.Payload))
					result := &PacketCaptureResult{
						Payload: make([]byte, len(udp.Payload)),
						SrcIP:   ip.SrcIP.String(),
						SrcPort: uint16(udp.SrcPort),
					}
					copy(result.Payload, udp.Payload)
					resultChan <- result
					return
				}
			}
		}
	}()

	// Wait for result or timeout/cancellation
	select {
	case result := <-resultChan:
		if result != nil {
			wg.Wait() // Ensure goroutine cleanup
			return result, nil
		}
		wg.Wait()
		return nil, fmt.Errorf("no packet captured")
	case err := <-errorChan:
		wg.Wait() // Ensure goroutine cleanup
		return nil, err
	case <-captureCtx.Done():
		wg.Wait() // Ensure goroutine cleanup
		return nil, captureCtx.Err()
	}
}
