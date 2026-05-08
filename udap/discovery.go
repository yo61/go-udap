package udap

import (
	"context"
	"fmt"
	"net"
	"time"
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

// DiscoverDevicesWithRawCaptureCtx uses UDP for device discovery.
// This function delegates to DiscoverDevicesUDPCtx for cross-platform compatibility.
// The "raw capture" name is kept for API compatibility but now uses pure Go UDP networking.
func (c *Client) DiscoverDevicesWithRawCaptureCtx(ctx context.Context, advanced bool) error {
	c.logger.Info("Starting UDAP discovery", "advanced", advanced)
	return c.DiscoverDevicesUDPCtx(ctx, advanced)
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

	// Prepare broadcast address - send from c.conn so responses come back to the same socket
	broadcastAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", Port))
	if err != nil {
		return fmt.Errorf("failed to resolve UDP broadcast address: %w", err)
	}

	// Start listening BEFORE sending discovery packet to avoid missing quick responses
	c.logger.Info("Setting up response listener")

	// Use a done channel to signal the listener to stop
	doneChan := make(chan struct{})
	responseDone := make(chan bool, 1)

	go func() {
		c.logger.Debug("Started response listener goroutine")
		c.listenForResponsesWithCancel(responseDone, doneChan)
	}()

	// Give the listener a moment to start
	time.Sleep(100 * time.Millisecond)

	// Send the discovery packet from c.conn so responses come back to this socket
	c.logger.Info("Broadcasting discovery packet", "target", "255.255.255.255", "port", Port)
	_, err = c.conn.WriteToUDP(discoveryPacket, broadcastAddr)
	if err != nil {
		close(doneChan)
		return fmt.Errorf("failed to send UDP broadcast: %w", err)
	}

	c.logger.Info("Sent UDP discovery packet", "target", "255.255.255.255", "port", Port)
	c.logger.Info("Waiting for responses")

	// Wait for responses or timeout
	select {
	case <-responseDone:
		c.logger.Info("Response listener completed")
	case <-ctx.Done():
		c.logger.Info("Discovery timeout reached")
	}

	// Signal goroutine to stop
	close(doneChan)

	// Set a very short deadline to unblock any pending ReadFromUDP
	c.conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))

	// Wait for goroutine to confirm exit - this is critical to prevent
	// the listener from consuming packets meant for subsequent operations
	select {
	case <-responseDone:
		c.logger.Debug("Listener goroutine exited cleanly")
	case <-time.After(500 * time.Millisecond):
		c.logger.Warn("Listener goroutine did not exit in time")
	}

	// Reset deadline for future operations
	c.conn.SetReadDeadline(time.Time{})

	// Small delay to ensure socket is fully ready for next operation
	time.Sleep(50 * time.Millisecond)

	return nil
}

// listenForResponsesWithCancel handles response listening with cancellation support
func (c *Client) listenForResponsesWithCancel(done chan<- bool, cancel <-chan struct{}) {
	defer func() {
		select {
		case done <- true:
		default:
		}
	}()

	buffer := make([]byte, 1024)
	responseCount := 0

	// Get our own IP addresses to filter out self-received packets
	localIPs := getLocalIPs()
	c.logger.Debug("Local IPs for filtering", "count", len(localIPs))

	c.logger.Info("Listening for UDP responses on socket")

	// Set a short read deadline so we can check the cancel channel periodically
	for {
		// Check if we should stop
		select {
		case <-cancel:
			c.logger.Debug("Listener cancelled", "responses_received", responseCount)
			return
		default:
		}

		// Set a short deadline for each read so we can check cancel channel
		c.conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))

		n, addr, err := c.conn.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Check if we should stop on timeout
				select {
				case <-cancel:
					c.logger.Debug("Listener cancelled after timeout", "responses_received", responseCount)
					return
				default:
					// Continue waiting for more packets
					continue
				}
			}
			c.logger.Error("Read error", "error", err)
			return
		}

		c.logger.Info("Received UDP packet", "bytes", n, "source_ip", addr.IP.String(), "source_port", addr.Port)

		// Skip our own packets, but allow 0.0.0.0 (devices in bootstrap mode)
		if addr.IP.String() != "0.0.0.0" && localIPs[addr.IP.String()] {
			c.logger.Debug("Ignoring self-received packet", "source_ip", addr.IP.String())
			continue
		}

		responseCount++
		c.logger.Debug("Processing response", "bytes", n, "source_ip", addr.IP.String(), "source_port", addr.Port, "hex", fmt.Sprintf("%x", buffer[:n]))

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

// listenForResponses handles the response listening in a separate goroutine (legacy)
func (c *Client) listenForResponses(done chan<- bool) {
	defer func() { done <- true }()

	buffer := make([]byte, 1024)
	responseCount := 0

	// Get our own IP addresses to filter out self-received packets
	localIPs := getLocalIPs()
	c.logger.Debug("Local IPs for filtering", "count", len(localIPs))

	c.logger.Info("Listening for UDP responses on socket")

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

		c.logger.Info("Received UDP packet", "bytes", n, "source_ip", addr.IP.String(), "source_port", addr.Port)

		// Skip our own packets, but allow 0.0.0.0 (devices in bootstrap mode)
		if addr.IP.String() != "0.0.0.0" && localIPs[addr.IP.String()] {
			c.logger.Debug("Ignoring self-received packet", "source_ip", addr.IP.String())
			continue
		}

		responseCount++
		c.logger.Debug("Processing response", "bytes", n, "source_ip", addr.IP.String(), "source_port", addr.Port, "hex", fmt.Sprintf("%x", buffer[:n]))

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
