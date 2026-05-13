package udap

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// DiscoverDevicesWithContext broadcasts a UDAP advanced-discovery
// (method 0x0009) request and collects responses until ctx is done.
// Discovered devices are stored on the Client and accessible via
// GetDevice / ListDevices / GetDevices.
func (c *Client) DiscoverDevicesWithContext(ctx context.Context) error {
	c.logger.Info("Starting UDAP discovery", "method", "0x0009")
	packet := c.CreateAdvancedDiscoveryPacket()
	if err := c.transport.Send(packet); err != nil {
		return fmt.Errorf("send discovery: %w", err)
	}
	c.logger.Debug("Sent discovery packet", "size", len(packet))

	for {
		reply, srcIP, err := c.transport.Recv(ctx)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("recv during discovery: %w", err)
		}
		c.handleDiscoveryReply(reply, srcIP)
	}
}

// handleDiscoveryReply parses one discovery reply and registers the
// device. Non-UCP packets and unparseable bytes are logged and skipped.
func (c *Client) handleDiscoveryReply(packetBytes []byte, srcIP string) {
	packet, data, err := ParsePacket(packetBytes)
	if err != nil {
		c.logger.Warn("Failed to parse discovery reply", "src_ip", srcIP, "error", err)
		return
	}
	if packet.UDAPType != TypeUCP {
		c.logger.Debug("Ignoring non-UCP packet during discovery", "src_ip", srcIP)
		return
	}
	device := c.parseDiscoveryResponse(data, srcIP, packet)
	if device == nil {
		c.logger.Warn("Discovery reply parsed but no device extracted", "src_ip", srcIP)
		return
	}
	c.recordDevice(device)
	c.logger.Info("Found device", "mac", device.MAC, "name", device.Name, "ip", device.IP)
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
	tlvUUID         = 0x0d
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
	// Real Squeezebox devices reply with AddrTypeETH and a 6-byte hardware
	// address. AddrTypeUDP is a Net::UDAP wire-spec constant for pseudo-
	// addressed devices, but no observed device actually uses it; the
	// previous code synthesised a "udp:<ip>" pseudo-MAC that no operation
	// downstream could actually use (Create*Packet would have failed to
	// parse it). With MAC promoted to a typed Value Object, the pseudo
	// path has no representable form — and removing it costs us nothing
	// that ever worked.
	if packet.SrcType != AddrTypeETH {
		c.logger.Warn("Ignoring discovery reply with non-Ethernet source type",
			"src_ip", ip, "src_type", fmt.Sprintf("0x%02x", packet.SrcType))
		return nil
	}

	device := &Device{
		MAC:        MAC(packet.SrcAddress),
		IP:         ip,
		LastSeen:   time.Now(),
		Parameters: make(map[string]string),
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
			device.HardwareRev = string(value)
		case tlvUUID:
			device.UUID = hex.EncodeToString(value)
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
