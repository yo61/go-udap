package udap

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"maps"
	"sort"
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
	retries   int
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

// SetRetries configures the number of retransmissions beyond the initial
// send. Negative values are silently clamped to 0. Default is 0 (no retries,
// just one send). Passing N means (N+1) total sends.
func (c *Client) SetRetries(n int) {
	if n < 0 {
		n = 0
	}
	c.retries = n
}

// sendRetried sends packet via the underlying transport, retransmitting
// c.retries additional times after the initial send. UDP send is
// fire-and-forget; aggregate success semantics: returns nil if any of
// the (c.retries + 1) attempts succeeded, returning the first error
// only when every attempt failed. There is no inter-send delay —
// matches squeezeplay's triple-send pattern (Net::UDAP applet's
// t_udapSend, SetupSqueezeboxApplet.lua:947-949).
func (c *Client) sendRetried(packet []byte) error {
	attempts := c.retries + 1
	var firstErr error
	successCount := 0
	for range attempts {
		if err := c.transport.Send(packet); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		} else {
			successCount++
		}
	}
	if successCount == 0 {
		return firstErr
	}
	return nil
}

// sendRetriedTo is the unicast counterpart of sendRetried: each
// attempt addresses dst directly instead of broadcasting. Same
// retry-and-aggregate semantics.
func (c *Client) sendRetriedTo(dst string, packet []byte) error {
	attempts := c.retries + 1
	var firstErr error
	successCount := 0
	for range attempts {
		if err := c.transport.SendUnicast(dst, packet); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		} else {
			successCount++
		}
	}
	if successCount == 0 {
		return firstErr
	}
	return nil
}

// sendForDevice picks unicast over broadcast when the device's IP is
// known (e.g. populated from the host ARP cache before the operation).
// Unicast bypasses Wi-Fi AP broadcast suppression that silently drops
// UDP broadcasts to associated clients on many residential APs. Falls
// back to broadcast when device.IP is empty, preserving the historical
// behaviour for unconfigured devices and ARP-cache-miss cases.
func (c *Client) sendForDevice(device *Device, packet []byte) error {
	if device.IP != "" {
		c.logger.Debug("Sending via unicast", "dst", device.IP, "mac", device.MAC)
		return c.sendRetriedTo(device.IP, packet)
	}
	return c.sendRetried(packet)
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
		// The wire field is uint16 by spec; the internal counter is
		// uint32 so it never overflows in practice. The mask wraps
		// modulo 65536 explicitly — a uint32→uint16 cast does the
		// same arithmetic, but spelling it out documents the intent
		// and makes any future "drop the cast and call atomic.Add
		// directly" regression a compile error rather than a silent
		// truncation.
		Sequence:  uint16(seq & 0xFFFF),
		UDAPType:  TypeUCP, // Always 0xC001
		UCPFlags:  flags,
		UAPClass:  [4]byte{0x00, 0x01, 0x00, 0x01}, // Always UAP_CLASS_UCP
		UCPMethod: method,
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
	if device.MAC.IsZero() {
		return nil, fmt.Errorf("cannot build GetData packet: device has zero MAC address")
	}

	packet := c.createUdapPacket(
		device.MAC.Bytes(),
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

// CreateSetDataPacket creates a UDAP SetData packet using the correct Lua format
// Based on the createSetData function from the authoritative Lua implementation
func (c *Client) CreateSetDataPacket(device *Device, params map[string]string) ([]byte, error) {
	c.logger.Info("Creating SetData packet", "device_mac", device.MAC, "param_count", len(params))

	if device.MAC.IsZero() {
		return nil, fmt.Errorf("cannot build SetData packet: device has zero MAC address")
	}

	packet := c.createUdapPacket(
		device.MAC.Bytes(),
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

	// Write each parameter using offset/length format. Parameter.Encode
	// owns the wire encoding (uint8/uint16/IPv4/string) and guarantees
	// exactly entry.Length bytes on success — no separate padding step.
	for _, entry := range paramList {
		data, err := entry.Encode(entry.value)
		if err != nil {
			return nil, fmt.Errorf("param %q: %w", entry.Name, err)
		}

		binary.Write(buf, binary.BigEndian, entry.Offset)
		binary.Write(buf, binary.BigEndian, entry.Length)
		buf.Write(data)

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
	if device.MAC.IsZero() {
		return nil, fmt.Errorf("cannot build Reset packet: device has zero MAC address")
	}

	// Reset uses the MethodReset (0x0004) not MethodError
	packet := c.createUdapPacket(
		device.MAC.Bytes(),
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

// ListDevices returns a snapshot slice of currently-discovered devices.
// The slice is a fresh allocation, but each *Device points at the same
// underlying value held by the client; callers that mutate fields on a
// returned device (e.g. GetAllDeviceConfigWithContext writing through
// device.Parameters) update the client's view. Do not retain a returned
// *Device past the lifetime of the Client.
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
// The returned *Device aliases the client's internal entry — see
// ListDevices for the mutation contract.
func (c *Client) GetDevice(mac string) *Device {
	c.devicesMu.RLock()
	defer c.devicesMu.RUnlock()
	return c.devices[mac]
}

// GetDevices returns a snapshot copy of the devices map. The map itself
// is freshly allocated, so callers may add or remove keys without
// affecting the client; but each *Device value aliases the client's
// internal entry, so mutating fields on a returned device updates the
// client's view (see ListDevices for the same contract).
func (c *Client) GetDevices() map[string]*Device {
	c.devicesMu.RLock()
	defer c.devicesMu.RUnlock()
	out := make(map[string]*Device, len(c.devices))
	maps.Copy(out, c.devices)
	return out
}

// recordDevice stores a discovered device under its MAC. Used by the
// discovery listener; takes the write lock so it's safe to call
// concurrently with reads.
//
// The map key is d.MAC.String() rather than d.MAC directly so that
// CLI lookup paths (GetDevice(mac string)) keep working with the
// canonical "aa:bb:..." form callers already have. Promoting the map
// key type to MAC would force every caller to ParseMAC at the
// boundary; the current arrangement keeps that boundary at the CLI's
// normalizeMAC layer instead.
func (c *Client) recordDevice(d *Device) {
	c.devicesMu.Lock()
	c.devices[d.MAC.String()] = d
	c.devicesMu.Unlock()
}

// NewClientForInterface constructs a Client whose UDP transport is
// bound to the given interface name's IPv4 address. Used by the CLI's
// --interface NAME flag. Errors if the interface does not exist, is
// down, lacks an IPv4 address, or is not broadcast-capable.
func NewClientForInterface(name string, logger Logger) (*Client, error) {
	ifs, err := EnumerateInterfaces()
	if err != nil {
		return nil, fmt.Errorf("enumerate interfaces: %w", err)
	}
	for _, iface := range ifs {
		if iface.Name == name {
			tr, err := NewUDPTransportOnInterface(iface, Port, logger)
			if err != nil {
				return nil, err
			}
			return NewClientWithTransport(tr, logger), nil
		}
	}
	return nil, fmt.Errorf("interface %q is not usable (must be up, broadcast-capable, with an IPv4 address)", name)
}

// NewClientForAllInterfaces constructs a Client whose UDP transport
// fans out to every usable interface returned by EnumerateInterfaces.
// Children that fail to bind are skipped with a Warn log; if no
// children succeed, returns an error.
func NewClientForAllInterfaces(logger Logger) (*Client, error) {
	ifs, err := EnumerateInterfaces()
	if err != nil {
		return nil, fmt.Errorf("enumerate interfaces: %w", err)
	}
	if len(ifs) == 0 {
		return nil, fmt.Errorf("no usable interfaces found")
	}
	children := make([]Transport, 0, len(ifs))
	for _, iface := range ifs {
		tr, err := NewUDPTransportOnInterface(iface, Port, logger)
		if err != nil {
			logger.Warn("skipping interface (bind failed)",
				"interface", iface.Name, "error", err)
			continue
		}
		children = append(children, tr)
	}
	if len(children) == 0 {
		return nil, fmt.Errorf("failed to bind on any usable interface")
	}
	return NewClientWithTransport(NewMultiTransport(children, logger), logger), nil
}
