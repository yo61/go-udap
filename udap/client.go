package udap

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
)

// Client handles UDAP protocol communication via an injected Transport.
//
// devicesMu guards devices: discovery may register devices concurrently
// with reads from CLI helpers (GetDevice/ListDevices/GetDevices), so
// without the lock that's a Go runtime data race.
type Client struct {
	transport Transport
	devicesMu sync.RWMutex
	devices   map[string]*Device
	sequence  uint32
	logger    Logger
}

// NewClient creates a new UDAP client bound to the standard UDAP port
// (17784) using the default structured logger.
func NewClient() (*Client, error) {
	return NewClientWithLogger(NewStructuredLogger())
}

// NewClientWithLogger creates a new UDAP client bound to the standard
// UDAP port (17784) with a custom logger.
func NewClientWithLogger(logger Logger) (*Client, error) {
	return newClientWithPort(Port, logger)
}

// newClientWithPort creates a UDAP client bound to the given UDP port.
// Port 0 lets the OS pick a free ephemeral port — used by tests so they
// don't collide with each other or with anything else holding port 17784.
func newClientWithPort(port int, logger Logger) (*Client, error) {
	tr, err := NewUDPTransport(port, logger)
	if err != nil {
		return nil, err
	}
	return NewClientWithTransport(tr, logger), nil
}

// NewClientWithTransport constructs a Client using an arbitrary Transport.
// Used by tests that want to inject a MockTransport (from the mocksbr
// package) for hermetic in-process testing.
//
// sequence starts at 0 because createUdapPacket uses
// atomic.AddUint32(_, 1), so the first packet's Sequence is 1.
func NewClientWithTransport(t Transport, logger Logger) *Client {
	return &Client{
		transport: t,
		devices:   make(map[string]*Device),
		logger:    logger,
	}
}

// Close releases the underlying transport resources.
func (c *Client) Close() error {
	return c.transport.Close()
}

// createUdapPacket creates the common UDAP packet header structure
// All UDAP messages share the same initial format up to UCPMethod.
// The sequence number is bumped atomically so concurrent Create*
// callers cannot race on the counter.
func (c *Client) createUdapPacket(dstMAC [6]byte, method uint16, flags uint8, broadcast bool) Packet {
	var dstBroadcast uint8
	if broadcast {
		dstBroadcast = 1
	}

	seq := atomic.AddUint32(&c.sequence, 1)

	return Packet{
		DstBroadcast: dstBroadcast,
		DstType:      AddrTypeETH, // Always use Ethernet addressing
		DstAddress:   dstMAC,
		SrcBroadcast: 0,                      // Source is never broadcast
		SrcType:      AddrTypeETH,            // Use ETH type like Lua implementation
		SrcAddress:   [MACAddressSize]byte{}, // All zeros for source
		Sequence:     uint16(seq),
		UDAPType:     TypeUCP, // Always 0xC001
		UCPFlags:     flags,
		UAPClass:     [4]byte{0x00, 0x01, 0x00, 0x01}, // Always UAP_CLASS_UCP
		UCPMethod:    method,
	}
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

// CreateGetDataPacket creates a UDAP GetData (0x0005) request packet.
//
// Wire format, validated against the Perl Net::UDAP reference
// implementation (perl_code.pcap frame 6):
//
//	[27-byte UDAP header, UCPMethod=0x0005]
//	[16 zero bytes — username]
//	[16 zero bytes — password]
//	[uint16 BE count of items]
//	[N × (uint16 BE NVRAM offset, uint16 BE length-to-read)]
//
// Parameter names not present in ConfigSettings are skipped with a warning.
// Items are sorted by offset for deterministic output, matching
// CreateSetDataPacket.
func (c *Client) CreateGetDataPacket(device *Device, params []string) ([]byte, error) {
	macBytes, err := c.parseMACAddress(device.MAC)
	if err != nil {
		return nil, err
	}

	packet := c.createUdapPacket(
		macBytes,
		MethodGetData, // 0x0005
		0x01,          // Request flag
		false,         // Not broadcast
	)

	items := make([]Parameter, 0, len(params))
	for _, name := range params {
		p, ok := ParameterByName(name)
		if !ok {
			c.logger.Warn("Unknown parameter skipped", "param", name, "device_mac", device.MAC)
			continue
		}
		items = append(items, p)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Offset < items[j].Offset
	})

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, packet)
	buf.Write(make([]byte, UsernameFieldSize))
	buf.Write(make([]byte, PasswordFieldSize))
	binary.Write(buf, binary.BigEndian, uint16(len(items)))
	for _, it := range items {
		binary.Write(buf, binary.BigEndian, it.Offset)
		binary.Write(buf, binary.BigEndian, it.Length)
	}

	return buf.Bytes(), nil
}

// parseMACAddress converts an "aa:bb:cc:dd:ee:ff" string to a [6]byte.
// Returns an error if the input doesn't parse to six hex bytes — prior
// versions silently returned an all-zeros MAC, which would unicast to
// 00:00:00:00:00:00 if a malformed device.MAC ever reached this code.
func (c *Client) parseMACAddress(mac string) ([6]byte, error) {
	var macBytes [6]byte
	n, err := fmt.Sscanf(mac, "%02x:%02x:%02x:%02x:%02x:%02x",
		&macBytes[0], &macBytes[1], &macBytes[2], &macBytes[3], &macBytes[4], &macBytes[5])
	if err != nil || n != 6 {
		return macBytes, fmt.Errorf("invalid MAC address %q", mac)
	}
	return macBytes, nil
}

// CreateSetDataPacket creates a UDAP SetData packet using the correct Lua format
// Based on the createSetData function from the authoritative Lua implementation
func (c *Client) CreateSetDataPacket(device *Device, params map[string]string) ([]byte, error) {
	c.logger.Info("Creating SetData packet", "device_mac", device.MAC, "param_count", len(params))

	// Convert MAC address to bytes
	macBytes, err := c.parseMACAddress(device.MAC)
	if err != nil {
		return nil, err
	}

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
		value string
		Parameter
	}
	var paramList []paramEntry

	for name, value := range params {
		if p, ok := ParameterByName(name); ok {
			paramList = append(paramList, paramEntry{value: value, Parameter: p})
		} else {
			c.logger.Warn("Unknown parameter skipped", "param", name, "device_mac", device.MAC)
		}
	}

	// Sort parameters by offset for consistent ordering
	sort.Slice(paramList, func(i, j int) bool {
		return paramList[i].Offset < paramList[j].Offset
	})

	// Write number of parameters (2 bytes big-endian)
	binary.Write(buf, binary.BigEndian, uint16(len(paramList)))
	c.logger.Debug("Parameters sorted", "param_count", len(paramList), "device_mac", device.MAC)

	// Write each parameter using offset/length format
	for _, entry := range paramList {
		// Write offset (2 bytes big-endian)
		binary.Write(buf, binary.BigEndian, entry.Offset)

		// Write length (2 bytes big-endian)
		binary.Write(buf, binary.BigEndian, entry.Length)

		// Convert value based on parameter type
		var data []byte

		switch entry.Length {
		case 4:
			// All 4-byte parameters are IPv4 addresses. Reject anything
			// that doesn't parse — pre-fix this used to silently zero-fill,
			// so a typo like "192.168.1.x" would write 0.0.0.0 to NVRAM.
			ip := net.ParseIP(entry.value)
			if ip == nil {
				return nil, fmt.Errorf("param %q: cannot parse %q as IPv4 address", entry.Name, entry.value)
			}
			ip4 := ip.To4()
			if ip4 == nil {
				return nil, fmt.Errorf("param %q: %q is not an IPv4 address", entry.Name, entry.value)
			}
			data = []byte(ip4)
		case 1:
			val, err := strconv.ParseUint(entry.value, 10, 8)
			if err != nil {
				return nil, fmt.Errorf("param %q: %q is not a valid uint8: %w", entry.Name, entry.value, err)
			}
			data = []byte{byte(val)}
		case 2:
			val, err := strconv.ParseUint(entry.value, 10, 16)
			if err != nil {
				return nil, fmt.Errorf("param %q: %q is not a valid uint16: %w", entry.Name, entry.value, err)
			}
			data = make([]byte, 2)
			binary.BigEndian.PutUint16(data, uint16(val))
		default:
			// String data
			data = []byte(entry.value)
			if len(data) > int(entry.Length) {
				data = data[:entry.Length] // Truncate if too long
			}
		}

		// Pad with zeros to reach the required length
		padded := make([]byte, entry.Length)
		copy(padded, data)
		buf.Write(padded)

		c.logger.Debug("Parameter details",
			"param", entry.Name,
			"offset_hex", fmt.Sprintf("0x%04x", entry.Offset),
			"offset_dec", entry.Offset,
			"length", entry.Length,
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

	return packetBytes, nil
}

// CreateResetPacket creates a UDAP reset packet to restart the device.
func (c *Client) CreateResetPacket(device *Device) ([]byte, error) {
	macBytes, err := c.parseMACAddress(device.MAC)
	if err != nil {
		return nil, err
	}

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

	return buf.Bytes(), nil
}

// ListDevices returns a snapshot of currently-discovered devices.
func (c *Client) ListDevices() []*Device {
	c.devicesMu.RLock()
	defer c.devicesMu.RUnlock()
	devices := make([]*Device, 0, len(c.devices))
	for _, device := range c.devices {
		devices = append(devices, device)
	}
	return devices
}

// GetDevice returns a device by MAC address, or nil if not present.
func (c *Client) GetDevice(mac string) *Device {
	c.devicesMu.RLock()
	defer c.devicesMu.RUnlock()
	return c.devices[mac]
}

// GetDevices returns a snapshot copy of the devices map. Callers may
// mutate the returned map without affecting the client's internal
// state.
func (c *Client) GetDevices() map[string]*Device {
	c.devicesMu.RLock()
	defer c.devicesMu.RUnlock()
	out := make(map[string]*Device, len(c.devices))
	for k, v := range c.devices {
		out[k] = v
	}
	return out
}

// recordDevice stores a discovered device under its MAC. Used by the
// discovery listener; takes the write lock so it's safe to call
// concurrently with reads.
func (c *Client) recordDevice(d *Device) {
	c.devicesMu.Lock()
	c.devices[d.MAC] = d
	c.devicesMu.Unlock()
}
