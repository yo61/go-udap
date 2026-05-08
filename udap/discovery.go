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

	// Use a done channel to signal the listener to stop. listenerExited
	// is closed (not sent-to) so it can be read repeatedly from both
	// selects below — closed-channel reads return immediately and
	// idempotently.
	doneChan := make(chan struct{})
	listenerExited := make(chan struct{})
	internalDone := make(chan bool, 1) // dummy; preserves listener's existing API

	go func() {
		defer close(listenerExited)
		c.logger.Debug("Started response listener goroutine")
		c.listenForResponsesWithCancel(internalDone, doneChan)
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

	// Wait for either listener exit (it doesn't normally exit on its
	// own — this branch fires only if it crashed) or ctx done.
	select {
	case <-listenerExited:
		c.logger.Info("Response listener completed")
	case <-ctx.Done():
		c.logger.Info("Discovery timeout reached")
	}

	// Signal listener to stop, then unblock any in-flight ReadFromUDP
	// so it can see the cancel signal on the next loop iteration.
	close(doneChan)
	c.conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))

	// Block until the listener has actually returned. This is critical:
	// the previous 500ms-timeout shortcut here could let cleanup proceed
	// while the listener was still alive on the shared socket, causing
	// it to consume packets meant for the next operation (GetData/SetData
	// responses). The wait is bounded by SetReadDeadline above (~50ms);
	// if it hangs longer than that we have a deeper bug worth knowing
	// about, not worth papering over with a timeout.
	<-listenerExited
	c.logger.Debug("Listener goroutine exited cleanly")

	c.conn.SetReadDeadline(time.Time{})
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

// Discovery-response TLV codes, per Net::UDAP Constant.pm
// (UCP_CODE_* constants). The codes are 1-byte; values are
// length-prefixed bytes.
const (
	tlvDeviceName   = 0x02
	tlvDeviceType   = 0x03
	tlvFirmwareRev  = 0x09
	tlvHardwareRev  = 0x0a
	tlvDeviceID     = 0x0b
	tlvDeviceStatus = 0x0c
)

// productNameByID maps a device_id (TLV 0x0b, sent as a 2-char ASCII
// hex string e.g. "07") to its human-readable product name. Source:
// Lugi/squeezeplay device tables; Receiver is the one we've physically
// captured ("07"). Other entries are for completeness — we'd need
// captures from each model to verify their device_id at the wire.
var productNameByID = map[string]string{
	"02": "Squeezebox 2",
	"03": "Squeezebox 3",
	"04": "Transporter",
	"05": "SoftSqueeze",
	"06": "Squeezebox Boom",
	"07": "Squeezebox Receiver",
	"08": "Squeezebox Touch",
	"09": "Squeezebox Radio",
	"0a": "Squeezebox Controller",
	"0b": "Squeezeslave",
}

// parseDiscoveryResponse parses a discovery response and creates a
// Device. TLV codes are per Net::UDAP Constant.pm — see the const
// block above.
func (c *Client) parseDiscoveryResponse(data []byte, ip string, packet *Packet) *Device {
	device := &Device{
		IP:         ip,
		LastSeen:   time.Now(),
		Parameters: make(map[string]string),
	}

	if packet.SrcType == AddrTypeETH {
		device.MAC = fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
			packet.SrcAddress[0], packet.SrcAddress[1], packet.SrcAddress[2],
			packet.SrcAddress[3], packet.SrcAddress[4], packet.SrcAddress[5])
	} else {
		device.MAC = fmt.Sprintf("udp:%s", ip)
	}

	var deviceType, deviceID string

	for offset := 0; offset+2 <= len(data); {
		tagType := data[offset]
		length := int(data[offset+1])
		offset += 2
		if offset+length > len(data) {
			break
		}
		value := data[offset : offset+length]
		offset += length

		switch tagType {
		case tlvDeviceName:
			device.Name = string(value)
		case tlvDeviceType:
			deviceType = string(value)
		case tlvFirmwareRev:
			device.Firmware = string(value)
		case tlvDeviceID:
			deviceID = string(value)
		case tlvDeviceStatus:
			device.State = string(value)
		case tlvHardwareRev:
			// Not surfaced today; recorded for future use.
		default:
			c.logger.Debug("unknown discovery TLV", "tag", fmt.Sprintf("0x%02x", tagType), "len", length)
		}
	}

	device.Model = combineModel(deviceType, deviceID)

	if device.Name == "" {
		device.Name = "Squeezebox Device"
	}
	return device
}

// combineModel renders a friendly Model string from the device_type
// (TLV 0x03, e.g. "squeezebox") and device_id (TLV 0x0b, e.g. "07").
// Falls back gracefully if either is missing.
func combineModel(deviceType, deviceID string) string {
	product, known := productNameByID[deviceID]
	switch {
	case known:
		return product
	case deviceType != "" && deviceID != "":
		return fmt.Sprintf("%s (id=%s)", deviceType, deviceID)
	case deviceType != "":
		return deviceType
	default:
		return ""
	}
}
