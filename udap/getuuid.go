package udap

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// CreateGetUUIDPacket builds a UCP_METHOD_GET_UUID (0x000b) request — a
// 27-byte UDAP header addressed to device.MAC with no payload. The device
// replies with method=0x000b and a TLV stream containing the UUID under
// code 0x0d (same code as the discovery TLV).
//
// Reference: Net::UDAP MessageOut.pm for the canonical wire shape;
// squeezeplay's createGetUUID (jive/net/Udap.lua) is the cross-check.
// Use this as a fallback when the discovery response's TLV 0x0d is
// missing or all-zeros — older firmware doesn't include UUID in
// adv_discover but answers get_uuid correctly.
func (c *Client) CreateGetUUIDPacket(device *Device) ([]byte, error) {
	if device.MAC.IsZero() {
		return nil, fmt.Errorf("cannot build GetUUID packet: device has zero MAC address")
	}
	packet := c.createUdapPacket(
		device.MAC.Bytes(),
		MethodGetUUID, // 0x000b
		0x01,          // request flag
		false,         // unicast
	)
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, packet); err != nil {
		return nil, fmt.Errorf("encode GetUUID header: %w", err)
	}
	return buf.Bytes(), nil
}

// GetDeviceUUIDWithContext sends a get_uuid request to device and returns
// the device's UUID as a lowercase hex string (e.g.
// "000102030405060708090a0b0c0d0e0f"). Hard-fails on transport errors,
// context cancellation, MethodError, MethodCredentialsError, or an
// unexpected reply method. Returns ("", error) if the response is missing
// TLV 0x0d entirely.
func (c *Client) GetDeviceUUIDWithContext(ctx context.Context, device *Device) (string, error) {
	packet, err := c.CreateGetUUIDPacket(device)
	if err != nil {
		return "", fmt.Errorf("build GetUUID packet: %w", err)
	}
	if err := c.sendRetried(packet); err != nil {
		return "", fmt.Errorf("send GetUUID: %w", err)
	}
	c.logger.Info("Sent GetUUID request", "device_mac", device.MAC)

	respPacket, data, err := c.waitForDeviceReply(ctx, device)
	if err != nil {
		return "", err
	}

	switch respPacket.UCPMethod {
	case MethodGetUUID:
		return parseGetUUIDResponse(data)
	case MethodError:
		if len(data) > 0 {
			for _, tlv := range DecodeTLV(data) {
				if tlv.Type == TLVTypeErrorMessage {
					return "", fmt.Errorf("device %s error: %s", device.MAC, string(tlv.Value))
				}
			}
		}
		return "", fmt.Errorf("device %s returned error response", device.MAC)
	case MethodCredentialsError:
		return "", fmt.Errorf("device %s rejected credentials", device.MAC)
	default:
		return "", fmt.Errorf("device %s: unexpected response method 0x%04x", device.MAC, respPacket.UCPMethod)
	}
}

// parseGetUUIDResponse decodes a get_uuid reply payload. The payload is a
// TLV stream (1-byte code, 1-byte length, value bytes). The recognised
// code is:
//
//	0x0d UCP_CODE_UUID (16 bytes raw UUID, hex-encoded for the return)
//
// Returns ("", error) if the response contains no TLV 0x0d. Unknown codes
// are skipped. Wrong-length 0x0d entries are skipped (and treated as
// missing).
func parseGetUUIDResponse(data []byte) (string, error) {
	for offset := 0; offset+2 <= len(data); {
		tagType := data[offset]
		length := int(data[offset+1])
		offset += 2
		if offset+length > len(data) {
			break
		}
		value := data[offset : offset+length]
		offset += length

		if tagType == tlvUUIDResp && length == 16 {
			return hex.EncodeToString(value), nil
		}
	}
	return "", fmt.Errorf("get_uuid response missing UUID TLV")
}

// get_uuid response TLV code per Net::UDAP Constant.pm:
//
//	UCP_CODE_UUID = 0x0d  (same code as in discovery responses)
//
// Named tlvUUIDResp here (rather than reusing the discovery-side tlvUUID
// constant from discovery.go) so the two parsers are self-contained. The
// values are equal.
const tlvUUIDResp = 0x0d
