package udap

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"
)

// waitForDeviceReply blocks on transport.Recv until it receives a
// packet matching device, or until ctx is cancelled. A reply matches
// when both:
//
//   - the in-payload SrcAddress equals device.MAC, AND
//   - the transport-reported source (an IP for UDPTransport) equals
//     device.IP, when device.IP is set.
//
// The transport-source check (review finding #6) makes a forged UDAP
// packet from a LAN attacker spoofing the target's MAC ineffective —
// the reply has to arrive from the address discovery learned. When
// device.IP is empty (e.g. the caller bypassed discovery), the check
// falls back to MAC-only matching for backward compatibility.
func (c *Client) waitForDeviceReply(ctx context.Context, device *Device) (*Packet, []byte, error) {
	want := device.MAC
	for {
		reply, src, err := c.transport.Recv(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("recv reply for %s: %w", want, err)
		}
		packet, data, perr := ParsePacket(reply)
		if perr != nil {
			c.logger.Warn("ignoring unparseable reply", "error", perr)
			continue
		}
		gotMAC := fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
			packet.SrcAddress[0], packet.SrcAddress[1], packet.SrcAddress[2],
			packet.SrcAddress[3], packet.SrcAddress[4], packet.SrcAddress[5])
		if gotMAC != want {
			c.logger.Debug("ignoring reply from different device", "from", gotMAC, "want", want)
			continue
		}
		if device.IP != "" && src != device.IP {
			c.logger.Warn("ignoring reply with mismatched source",
				"mac", gotMAC, "src", src, "expected", device.IP)
			continue
		}
		return packet, data, nil
	}
}

// GetDeviceConfigWithContext reads the named parameters from a device.
// Sends a UCP_METHOD_GET_DATA (0x0005) request and decodes the matching
// offset/length/value response.
func (c *Client) GetDeviceConfigWithContext(ctx context.Context, device *Device, params []string) (map[string]string, error) {
	packet, err := c.CreateGetDataPacket(device, params)
	if err != nil {
		return nil, fmt.Errorf("build GetData packet: %w", err)
	}
	if err := c.transport.Send(packet); err != nil {
		return nil, fmt.Errorf("send GetData: %w", err)
	}
	c.logger.Info("Sent GetData request", "device_mac", device.MAC, "param_count", len(params))

	respPacket, data, err := c.waitForDeviceReply(ctx, device)
	if err != nil {
		return nil, err
	}

	switch respPacket.UCPMethod {
	case MethodGetData:
		parsed, perr := parseGetDataResponse(data)
		if perr != nil {
			return nil, fmt.Errorf("decode GetData response from %s: %w", device.MAC, perr)
		}
		return parsed, nil
	case MethodError:
		return nil, fmt.Errorf("device %s returned error response", device.MAC)
	default:
		return nil, fmt.Errorf("device %s: unexpected response method 0x%04x", device.MAC, respPacket.UCPMethod)
	}
}

// GetAllDeviceConfigWithContext retrieves all known parameters using the
// caller-supplied context for cancellation/timeout.
//
// Stale `offset_NNN` synthetic keys from any previous call are cleared
// before the new response is merged. parseGetDataResponse emits those
// keys for NVRAM offsets unknown to the udap.Parameters table, and
// without this cleanup a long-running consumer calling GetAll
// repeatedly would let device.Parameters grow without bound across
// firmware variations.
func (c *Client) GetAllDeviceConfigWithContext(ctx context.Context, device *Device) error {
	c.logger.Info("Reading all device parameters", "device_mac", device.MAC)

	config, err := c.GetDeviceConfigWithContext(ctx, device, ParameterNames())
	if err != nil {
		return fmt.Errorf("failed to read device parameters: %w", err)
	}

	if device.Parameters == nil {
		device.Parameters = make(map[string]string)
	}
	for k := range device.Parameters {
		if strings.HasPrefix(k, "offset_") {
			delete(device.Parameters, k)
		}
	}
	maps.Copy(device.Parameters, config)

	c.logger.Info("Read parameters from device", "param_count", len(config), "device_mac", device.MAC)
	return nil
}

// SetDeviceConfigWithContext writes the named parameters to a device.
// Read-modify-write: omitted params would clobber neighbouring NVRAM
// regions with zeros, so the client first reads the device's current
// values and merges the caller's overrides on top. If that prelude
// read fails, the whole operation aborts — the previous warn-and-
// continue path produced exactly the partial write the read was
// supposed to prevent.
func (c *Client) SetDeviceConfigWithContext(ctx context.Context, device *Device, config map[string]string) error {
	if len(device.Parameters) == 0 {
		c.logger.Info("Device parameters not loaded, reading current configuration")
		if err := c.GetAllDeviceConfigWithContext(ctx, device); err != nil {
			return fmt.Errorf("read current parameters before set: %w", err)
		}
	}

	allParams := make(map[string]string, len(device.Parameters)+len(config))
	maps.Copy(allParams, device.Parameters)
	maps.Copy(allParams, config)

	packet, err := c.CreateSetDataPacket(device, allParams)
	if err != nil {
		return fmt.Errorf("build SetData packet: %w", err)
	}
	if err := c.transport.Send(packet); err != nil {
		return fmt.Errorf("send SetData: %w", err)
	}
	c.logger.Info("Sent SetData request", "device_mac", device.MAC, "total_params", len(allParams))

	respPacket, data, err := c.waitForDeviceReply(ctx, device)
	if err != nil {
		return err
	}

	switch respPacket.UCPMethod {
	case MethodSetData, MethodGetData, MethodGetIP:
		// Apply caller's overrides to the cached aggregate only after
		// the device has acknowledged the write. Mutating earlier left
		// device.Parameters showing values that were never persisted to
		// NVRAM if the round-trip failed.
		maps.Copy(device.Parameters, config)
		c.logger.Info("Device acknowledged configuration change", "method", fmt.Sprintf("0x%04x", respPacket.UCPMethod))
		return nil
	case MethodError:
		if len(data) > 0 {
			tlvs := DecodeTLV(data)
			for _, tlv := range tlvs {
				if tlv.Type == TLVTypeErrorMessage {
					return fmt.Errorf("device %s error: %s", device.MAC, string(tlv.Value))
				}
			}
		}
		return fmt.Errorf("device %s returned error response", device.MAC)
	case MethodCredentialsError:
		return fmt.Errorf("device %s rejected credentials", device.MAC)
	default:
		return fmt.Errorf("device %s: unexpected response method 0x%04x", device.MAC, respPacket.UCPMethod)
	}
}

// ResetDeviceWithContext sends a UCP_METHOD_RESET (0x0004) command. The
// device may reboot before sending an ack, so a context-cancellation
// error is treated as success. A MethodError reply is surfaced as an
// error rather than being misclassified as a successful ack.
func (c *Client) ResetDeviceWithContext(ctx context.Context, device *Device) error {
	packet, err := c.CreateResetPacket(device)
	if err != nil {
		return fmt.Errorf("build Reset packet: %w", err)
	}
	if err := c.transport.Send(packet); err != nil {
		return fmt.Errorf("send Reset: %w", err)
	}
	c.logger.Info("Sent Reset", "device_mac", device.MAC)

	respPacket, data, err := c.waitForDeviceReply(ctx, device)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			c.logger.Info("No reset acknowledgment; device may have reset immediately")
			return nil
		}
		// Non-context errors here (transport-level, packet-parse,
		// etc.) are returned as-is. UDP doesn't have connection
		// teardown, so a successful reset never breaks the local
		// socket; any non-context error from Recv indicates a real
		// problem the caller should surface, not a silently-
		// successful reboot.
		return err
	}

	switch respPacket.UCPMethod {
	case MethodReset:
		c.logger.Info("Device acknowledged reset")
		return nil
	case MethodError:
		if len(data) > 0 {
			for _, tlv := range DecodeTLV(data) {
				if tlv.Type == TLVTypeErrorMessage {
					return fmt.Errorf("device %s rejected reset: %s", device.MAC, string(tlv.Value))
				}
			}
		}
		return fmt.Errorf("device %s rejected reset", device.MAC)
	default:
		return fmt.Errorf("device %s: unexpected response method 0x%04x", device.MAC, respPacket.UCPMethod)
	}
}
