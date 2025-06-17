package udap

import (
	"context"
	"fmt"
	"net"
	"sync"
	"syscall"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// DiscoverDevices discovers Squeezebox devices on the network using advanced discovery
func (c *Client) DiscoverDevices(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.DiscoverDevicesWithContext(ctx)
}

// DiscoverDevicesWithContext discovers devices using the provided context for cancellation
func (c *Client) DiscoverDevicesWithContext(ctx context.Context) error {
	return c.DiscoverDevicesAdvancedWithContext(ctx)
}

// DiscoverDevicesAdvanced uses advanced discovery (method 0x0009)
func (c *Client) DiscoverDevicesAdvanced(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.DiscoverDevicesAdvancedWithContext(ctx)
}

// DiscoverDevicesAdvancedWithContext uses advanced discovery with context
func (c *Client) DiscoverDevicesAdvancedWithContext(ctx context.Context) error {
	c.logger.Info("Starting advanced UDAP discovery", "method", "0x0009")
	return c.discoverWithMethodCtx(ctx, true)
}

// discoverWithMethodCtx performs discovery using the specified method with context
func (c *Client) discoverWithMethodCtx(ctx context.Context, advanced bool) error {
	return c.DiscoverDevicesWithRawCaptureCtx(ctx, advanced)
}

// DiscoverDevicesWithRawCapture uses UDP for sending and raw capture for receiving
func (c *Client) DiscoverDevicesWithRawCapture(timeout time.Duration, advanced bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.DiscoverDevicesWithRawCaptureCtx(ctx, advanced)
}

// DiscoverDevicesWithRawCaptureCtx uses UDP for sending and raw capture for receiving with context
func (c *Client) DiscoverDevicesWithRawCaptureCtx(ctx context.Context, advanced bool) error {
	// Create proper UDAP discovery packet
	var discoveryPacket []byte
	if advanced {
		discoveryPacket = c.CreateAdvancedDiscoveryPacket()
	} else {
		discoveryPacket = c.CreateDiscoveryPacket()
	}
	c.logger.Debug("Created discovery packet", "size_bytes", len(discoveryPacket), "hex", fmt.Sprintf("%x", discoveryPacket))

	// Start raw packet capture for responses
	c.logger.Info("Starting raw packet capture", "source_ip", "0.0.0.0")
	interfaces, err := pcap.FindAllDevs()
	if err != nil {
		c.logger.Warn("Could not find network interfaces for raw capture, falling back to UDP", "error", err)
		return c.DiscoverDevicesUDPCtx(ctx, advanced)
	}

	// Find a suitable interface (prefer en0, en1, etc.)
	var selectedInterface string
	for _, iface := range interfaces {
		if len(iface.Addresses) > 0 && (len(iface.Name) >= 3 && iface.Name[:2] == "en") {
			selectedInterface = iface.Name
			break
		}
	}

	if selectedInterface == "" {
		c.logger.Warn("No suitable network interface found, falling back to UDP only")
		return c.DiscoverDevicesUDPCtx(ctx, advanced)
	}

	c.logger.Info("Selected interface for raw packet capture", "interface", selectedInterface)

	// Start raw packet capture with proper lifecycle management
	responseChannel := make(chan *Device, 10)
	captureCtx, captureCancel := context.WithCancel(ctx)
	defer captureCancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.captureUDAPResponses(selectedInterface, responseChannel, captureCtx)
	}()

	// Give capture time to start
	time.Sleep(200 * time.Millisecond)

	// Send UDP broadcast
	broadcastAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("255.255.255.255:%d", Port))
	if err != nil {
		return fmt.Errorf("failed to resolve UDP broadcast address: %w", err)
	}

	broadcastConn, err := net.DialUDP("udp", nil, broadcastAddr)
	if err != nil {
		return fmt.Errorf("failed to create UDP broadcast socket: %w", err)
	}
	defer broadcastConn.Close()

	// Enable broadcast
	if file, err := broadcastConn.File(); err == nil {
		fd := int(file.Fd())
		err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
		file.Close()
		if err != nil {
			c.logger.Warn("Failed to enable SO_BROADCAST", "error", err)
		}
	}

	c.logger.Info("Sending discovery packet", "target", "255.255.255.255", "port", Port)
	_, err = broadcastConn.Write(discoveryPacket)
	if err != nil {
		return fmt.Errorf("failed to send UDP broadcast: %w", err)
	}

	c.logger.Info("Waiting for responses via raw packet capture")

	// Wait for responses or timeout
	deviceCount := 0

	for {
		select {
		case device := <-responseChannel:
			if device != nil {
				deviceCount++
				c.devices[device.MAC] = device
				c.logger.Info("Found device", "name", device.Name, "mac", device.MAC, "ip", device.IP)
			}

		case <-ctx.Done():
			c.logger.Info("Discovery timeout reached", "devices_found", deviceCount)
			captureCancel()
			wg.Wait() // Ensure goroutine cleanup
			return nil
		}
	}
}

// captureUDAPResponses captures UDP responses from 0.0.0.0 using raw packet capture
func (c *Client) captureUDAPResponses(interfaceName string, deviceChan chan<- *Device, ctx context.Context) {
	defer close(deviceChan)

	// Open interface for capturing
	handle, err := pcap.OpenLive(interfaceName, 1600, true, time.Millisecond*100)
	if err != nil {
		c.logger.Error("Error opening interface", "interface", interfaceName, "error", err)
		return
	}
	defer handle.Close()

	// Filter for UDP packets on port 17784 from 0.0.0.0
	filter := "udp src port 17784 and src host 0.0.0.0"
	err = handle.SetBPFFilter(filter)
	if err != nil {
		// Fallback to broader filter
		filter = "udp port 17784"
		err = handle.SetBPFFilter(filter)
		if err != nil {
			c.logger.Warn("Could not set BPF filter", "error", err)
		}
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	c.logger.Info("Started packet capture", "interface", interfaceName, "filter", filter)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Stopping packet capture")
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

			srcIP := ip.SrcIP.String()
			srcPort := udp.SrcPort

			// Check if this is a UDAP response from bootstrap device
			if srcIP == "0.0.0.0" && srcPort == 17784 {
				c.logger.Debug("Captured UDAP response", "source_ip", "0.0.0.0", "source_port", srcPort, "payload_bytes", len(udp.Payload))

				if len(udp.Payload) > 0 {
					// Parse as UDAP packet
					udapPacket, data, err := ParsePacket(udp.Payload)
					if err != nil {
						c.logger.Warn("Failed to parse UDAP packet", "error", err)
						continue
					}

					c.logger.Debug("Parsed UDAP packet", "type", fmt.Sprintf("0x%04x", udapPacket.UDAPType), "method", fmt.Sprintf("0x%04x", udapPacket.UCPMethod))

					// Check what type of response this is
					switch udapPacket.UCPMethod {
					case MethodDiscover, MethodDataResp, MethodAdvDisc:
						// This is a discovery response - parse device info
						device := c.parseDiscoveryResponse(data, srcIP, udapPacket)
						if device != nil {
							device.IP = "0.0.0.0"
							deviceChan <- device
						} else {
							c.logger.Warn("Could not parse device information from discovery response")
						}
					case MethodSetData:
						// This could be a save_data response (uses MethodSetData 0x0002) - parse configuration parameters
						c.logger.Info("Received SetData/save_data response", "action", "parsing_config_parameters")
						c.parseConfigResponse(data)
					default:
						c.logger.Debug("Received non-discovery response", "method", fmt.Sprintf("0x%04x", udapPacket.UCPMethod))
					}
				}
			}
		}
	}
}

// DiscoverDevicesUDP discovers devices using UDP broadcasts (Layer 3) - fallback method
func (c *Client) DiscoverDevicesUDP(timeout time.Duration, advanced bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.DiscoverDevicesUDPCtx(ctx, advanced)
}

// DiscoverDevicesUDPCtx discovers devices using UDP broadcasts with context
func (c *Client) DiscoverDevicesUDPCtx(ctx context.Context, advanced bool) error {
	// Create proper UDAP discovery packet
	var discoveryPacket []byte
	if advanced {
		discoveryPacket = c.CreateAdvancedDiscoveryPacket()
	} else {
		discoveryPacket = c.CreateDiscoveryPacket()
	}
	c.logger.Debug("Created UDP discovery packet", "size_bytes", len(discoveryPacket), "hex", fmt.Sprintf("%x", discoveryPacket))

	// Start listening BEFORE sending discovery packet to avoid missing quick responses
	c.logger.Info("Setting up response listener")
	deadline, ok := ctx.Deadline()
	if ok {
		c.conn.SetReadDeadline(deadline)
	}
	defer c.conn.SetReadDeadline(time.Time{})

	// Prepare broadcast socket
	broadcastAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("255.255.255.255:%d", Port))
	if err != nil {
		return fmt.Errorf("failed to resolve UDP broadcast address: %w", err)
	}

	broadcastConn, err := net.DialUDP("udp", nil, broadcastAddr)
	if err != nil {
		return fmt.Errorf("failed to create UDP broadcast socket: %w", err)
	}
	defer broadcastConn.Close()

	// Enable broadcast on the sending socket
	file, err := broadcastConn.File()
	if err == nil {
		fd := int(file.Fd())
		err = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
		file.Close()
		if err != nil {
			c.logger.Warn("Failed to enable SO_BROADCAST", "error", err)
		} else {
			c.logger.Debug("Successfully enabled SO_BROADCAST on send socket")
		}
	}

	// Start a goroutine to listen for responses with proper lifecycle management
	responseChannel := make(chan bool, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.logger.Debug("Started response listener goroutine")
		c.listenForResponses(responseChannel)
	}()

	// Give the listener a moment to start
	time.Sleep(100 * time.Millisecond)

	// Now send the discovery packet
	c.logger.Info("Broadcasting discovery packet", "target", "255.255.255.255", "port", Port)
	_, err = broadcastConn.Write(discoveryPacket)
	if err != nil {
		return fmt.Errorf("failed to send UDP broadcast: %w", err)
	}

	c.logger.Info("Sent UDP discovery packet", "target", "255.255.255.255", "port", Port)
	c.logger.Info("Waiting for responses")

	// Wait for responses or timeout
	select {
	case <-responseChannel:
		c.logger.Info("Response listener completed")
	case <-ctx.Done():
		c.logger.Info("Discovery timeout reached")
	}

	wg.Wait() // Ensure goroutine cleanup
	return nil
}

// listenForResponses handles the response listening in a separate goroutine
func (c *Client) listenForResponses(done chan<- bool) {
	defer func() { done <- true }()

	buffer := make([]byte, 1024)
	responseCount := 0

	// Get our own IP addresses to filter out self-received packets
	localIPs := getLocalIPs()

	for {
		n, addr, err := c.conn.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				c.logger.Info("Discovery timeout reached", "responses_received", responseCount)
				break
			}
			c.logger.Error("Read error", "error", err)
			return
		}

		// Skip our own packets, but allow 0.0.0.0 (devices in bootstrap mode)
		if addr.IP.String() != "0.0.0.0" && localIPs[addr.IP.String()] {
			c.logger.Debug("Ignoring self-received packet", "source_ip", addr.IP.String())
			continue
		}

		responseCount++
		c.logger.Debug("Received response", "bytes", n, "source_ip", addr.IP.String(), "source_port", addr.Port, "hex", fmt.Sprintf("%x", buffer[:n]))

		packet, data, err := ParsePacket(buffer[:n])
		if err != nil {
			c.logger.Warn("Failed to parse packet", "source_ip", addr.IP.String(), "error", err, "raw_data", string(buffer[:n]))
			continue
		}

		c.logger.Debug("Parsed packet", "udap_type", fmt.Sprintf("0x%04x", packet.UDAPType), "ucp_method", fmt.Sprintf("0x%04x", packet.UCPMethod), "dst_type", packet.DstType, "src_type", packet.SrcType)

		// Check packet type and method
		switch {
		case packet.UDAPType == TypeUCP:
			// Check if it's a discovery response or contains device info
			device := c.parseDiscoveryResponse(data, addr.IP.String(), packet)
			if device != nil {
				c.devices[device.MAC] = device
				c.logger.Info("Found device", "name", device.Name, "mac", device.MAC, "ip", device.IP)
			} else {
				c.logger.Warn("UCP packet received but no device info parsed", "source_ip", addr.IP.String())
			}
		case packet.UCPMethod == MethodDiscover:
			c.logger.Debug("Received discovery packet from another UDAP client", "source_ip", addr.IP.String())
		default:
			c.logger.Debug("Received unexpected packet", "udap_type", fmt.Sprintf("0x%04x", packet.UDAPType), "ucp_method", fmt.Sprintf("0x%04x", packet.UCPMethod), "source_ip", addr.IP.String())
		}
	}
}

// parseDiscoveryResponse parses a discovery response and creates a Device
func (c *Client) parseDiscoveryResponse(data []byte, ip string, packet *Packet) *Device {
	device := &Device{
		IP:         ip,
		LastSeen:   time.Now(),
		Parameters: make(map[string]string),
	}

	// Extract MAC from source address if it's ETH type
	if packet.SrcType == AddrTypeETH {
		device.MAC = fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
			packet.SrcAddress[0], packet.SrcAddress[1], packet.SrcAddress[2],
			packet.SrcAddress[3], packet.SrcAddress[4], packet.SrcAddress[5])
	} else {
		// For UDP type, we might not have the MAC directly
		// Try to parse from data if available, otherwise use IP as identifier
		device.MAC = fmt.Sprintf("udp:%s", ip)
	}

	// Parse additional data if present (TLV format or other)
	if len(data) > 0 {
		// Try to parse TLV data with proper parsing
		offset := 0
		for offset < len(data) {
			if offset+2 >= len(data) {
				break
			}

			tagType := data[offset]
			length := data[offset+1]
			offset += 2

			if offset+int(length) > len(data) {
				break
			}

			value := data[offset : offset+int(length)]

			switch tagType {
			case 0x01: // MAC Address
				device.MAC = string(value)
			case 0x02: // Device Name
				device.Name = string(value)
			case 0x03: // Model
				device.Model = string(value)
			case 0x04: // Firmware Version
				device.Firmware = string(value)
			case 0x05: // UUID
				device.UUID = string(value)
			case 0x1a: // Possible device name (seen in packet)
				if device.Name == "" {
					device.Name = string(value)
				}
			case 0xad: // Another possible name field (seen in packet)
				if device.Name == "" {
					// Null-terminated string
					nullIndex := 0
					for i, b := range value {
						if b == 0 {
							nullIndex = i
							break
						}
					}
					if nullIndex > 0 {
						device.Name = string(value[:nullIndex])
					} else {
						device.Name = string(value)
					}
				}
			case 0xb7: // Another field (seen in packet)
				// This might be model or firmware
				nullIndex := 0
				for i, b := range value {
					if b == 0 {
						nullIndex = i
						break
					}
				}
				valueStr := string(value)
				if nullIndex > 0 {
					valueStr = string(value[:nullIndex])
				}
				if device.Model == "" && valueStr != "" {
					device.Model = valueStr
				}
			}

			offset += int(length)
		}
	}

	// Set default name if not provided
	if device.Name == "" {
		device.Name = "Squeezebox Device"
	}

	return device
}

// parseConfigResponse parses configuration data from save_data or config responses
func (c *Client) parseConfigResponse(data []byte) {
	if len(data) == 0 {
		return
	}

	c.logger.Debug("Parsing config response data", "bytes", len(data))

	// Parse TLV data for configuration parameters
	offset := 0
	paramCount := 0

	for offset < len(data) {
		if offset+2 >= len(data) {
			break
		}

		tagType := data[offset]
		length := data[offset+1]
		offset += 2

		if offset+int(length) > len(data) {
			c.logger.Warn("TLV parsing error", "tag", fmt.Sprintf("0x%02x", tagType), "length", length, "offset", offset, "reason", "would exceed data bounds")
			break
		}

		value := data[offset : offset+int(length)]

		// Parse configuration parameter based on tag
		paramName := ""
		paramValue := ""

		switch tagType {
		case 0x09: // Network mask
			if length == 4 {
				paramName = "lan_netmask"
				paramValue = fmt.Sprintf("%d.%d.%d.%d", value[0], value[1], value[2], value[3])
			}
		case 0x1a: // Some field (length 17 in packet)
			paramName = "unknown_0x1a"
			paramValue = string(value)
		case 0x32: // Some field
			paramName = "unknown_0x32"
			paramValue = string(value)
		case 0x34: // Some field
			paramName = "unknown_0x34"
			paramValue = string(value)
		case 0x4f: // IP address field
			if length == 4 {
				paramName = "lan_ip_address"
				paramValue = fmt.Sprintf("%d.%d.%d.%d", value[0], value[1], value[2], value[3])
			}
		case 0x47, 0x05, 0x53, 0x3b: // Various fields
			paramName = fmt.Sprintf("unknown_0x%02x", tagType)
			if length == 4 {
				paramValue = fmt.Sprintf("%d.%d.%d.%d", value[0], value[1], value[2], value[3])
			} else {
				paramValue = string(value)
			}
		case 0xad: // Device hostname/name (from packet - this was the wireless PSK)
			paramName = "wireless_wpa_psk"
			// Handle null-terminated string
			nullIndex := len(value)
			for i, b := range value {
				if b == 0 {
					nullIndex = i
					break
				}
			}
			paramValue = string(value[:nullIndex])
		case 0xb7: // SSID field (from packet - this was SlimNet)
			paramName = "wireless_SSID"
			// Handle null-terminated string
			nullIndex := len(value)
			for i, b := range value {
				if b == 0 {
					nullIndex = i
					break
				}
			}
			paramValue = string(value[:nullIndex])
		case 0xf8, 0xeb, 0xd8, 0x43, 0xda: // Other fields seen in packet
			paramName = fmt.Sprintf("unknown_0x%02x", tagType)
			if length <= 8 {
				// Short fields - might be numeric
				paramValue = fmt.Sprintf("0x%x", value)
			} else {
				// Longer fields - try as string
				nullIndex := len(value)
				for i, b := range value {
					if b == 0 {
						nullIndex = i
						break
					}
				}
				if nullIndex > 0 {
					paramValue = string(value[:nullIndex])
				} else {
					paramValue = fmt.Sprintf("0x%x", value)
				}
			}
		default:
			paramName = fmt.Sprintf("unknown_0x%02x", tagType)
			if length <= 4 {
				paramValue = fmt.Sprintf("0x%x", value)
			} else {
				paramValue = string(value)
			}
		}

		if paramName != "" {
			paramCount++
			c.logger.Debug("Parsed config parameter", "name", paramName, "value", paramValue)
		}

		offset += int(length)
	}

	c.logger.Info("Parsed configuration parameters", "count", paramCount)
}
