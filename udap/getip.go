package udap

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
)

// CreateGetIPPacket builds a UCP_METHOD_GET_IP (0x0002) request — a
// 27-byte UDAP header addressed to device.MAC with no payload. The
// device replies with method=0x0002 and a TLV stream of network-config
// codes (0x05 IP, 0x06 SubnetMask, 0x07 Gateway).
//
// Reference: Net::UDAP MessageOut.pm — "nothing further to do for
// get_ip" once the header is built.
func (c *Client) CreateGetIPPacket(device *Device) ([]byte, error) {
	if device.MAC.IsZero() {
		return nil, fmt.Errorf("cannot build GetIP packet: device has zero MAC address")
	}
	packet := c.createUdapPacket(
		device.MAC.Bytes(),
		MethodGetIP, // 0x0002
		0x01,        // request flag
		false,       // unicast
	)
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, packet); err != nil {
		return nil, fmt.Errorf("encode GetIP header: %w", err)
	}
	return buf.Bytes(), nil
}

// GetDeviceNetworkConfigWithContext sends a get_ip request to device and
// parses the response. Soft-fails on missing or malformed TLVs (returns
// a NetworkConfig with zero-value fields for whichever pieces are
// missing); hard-fails on transport errors, context cancellation,
// MethodError, MethodCredentialsError, or unexpected reply methods.
func (c *Client) GetDeviceNetworkConfigWithContext(ctx context.Context, device *Device) (NetworkConfig, error) {
	packet, err := c.CreateGetIPPacket(device)
	if err != nil {
		return NetworkConfig{}, fmt.Errorf("build GetIP packet: %w", err)
	}
	if err := c.transport.Send(packet); err != nil {
		return NetworkConfig{}, fmt.Errorf("send GetIP: %w", err)
	}
	c.logger.Info("Sent GetIP request", "device_mac", device.MAC)

	respPacket, data, err := c.waitForDeviceReply(ctx, device)
	if err != nil {
		return NetworkConfig{}, err
	}

	switch respPacket.UCPMethod {
	case MethodGetIP:
		return parseGetIPResponse(data)
	case MethodError:
		if len(data) > 0 {
			for _, tlv := range DecodeTLV(data) {
				if tlv.Type == TLVTypeErrorMessage {
					return NetworkConfig{}, fmt.Errorf("device %s error: %s", device.MAC, string(tlv.Value))
				}
			}
		}
		return NetworkConfig{}, fmt.Errorf("device %s returned error response", device.MAC)
	case MethodCredentialsError:
		return NetworkConfig{}, fmt.Errorf("device %s rejected credentials", device.MAC)
	default:
		return NetworkConfig{}, fmt.Errorf("device %s: unexpected response method 0x%04x", device.MAC, respPacket.UCPMethod)
	}
}

// parseGetIPResponse decodes a get_ip reply payload. TLV format
// matches discovery: 1-byte code, 1-byte length, value bytes.
// Recognised codes:
//
//	0x05 UCP_CODE_IP_ADDR       (4 bytes IPv4)
//	0x06 UCP_CODE_SUBNET_MASK   (4 bytes IPv4 mask)
//	0x07 UCP_CODE_GATEWAY_ADDR  (4 bytes IPv4)
//
// Unknown codes are skipped. Wrong-length codes are skipped. The
// function never panics on malformed input; truncated TLVs cause the
// scan to stop where it is and return whatever was parsed up to that
// point.
func parseGetIPResponse(data []byte) (NetworkConfig, error) {
	var nc NetworkConfig
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
		case tlvIPAddr:
			if length == 4 {
				nc.IP = net.IPv4(value[0], value[1], value[2], value[3])
			}
		case tlvSubnetMask:
			if length == 4 {
				nc.SubnetMask = net.IPv4(value[0], value[1], value[2], value[3])
			}
		case tlvGatewayAddr:
			if length == 4 {
				nc.Gateway = net.IPv4(value[0], value[1], value[2], value[3])
			}
		}
	}
	return nc, nil
}

// get_ip / discovery TLV codes per Net::UDAP Constant.pm:
//
//	UCP_CODE_IP_ADDR       = 0x05
//	UCP_CODE_SUBNET_MASK   = 0x06
//	UCP_CODE_GATEWAY_ADDR  = 0x07
const (
	tlvIPAddr      = 0x05
	tlvSubnetMask  = 0x06
	tlvGatewayAddr = 0x07
)
