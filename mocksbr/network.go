package mocksbr

import (
	"fmt"
	"strings"
	"sync"

	"go-udap/udap"
)

// Network is one or more virtual SBR devices sharing a single inbound
// packet queue, dispatched by destination MAC.
type Network struct {
	mu      sync.Mutex
	logger  udap.Logger
	devices map[string]*device // keyed by lowercase MAC
	order   []string           // MACs in insertion order, for deterministic fan-out
}

// NewNetwork constructs a Network of n auto-generated virtual devices.
// Auto-generated identities use deterministic MACs starting at
// 00:04:20:00:00:01.
func NewNetwork(n int, logger udap.Logger) *Network {
	if logger == nil {
		logger = udap.NewNoOpLogger()
	}
	net := &Network{
		logger:  logger,
		devices: make(map[string]*device, n),
		order:   make([]string, 0, n),
	}
	for i := 1; i <= n; i++ {
		cfg := autoConfig(i)
		mac := strings.ToLower(cfg.MAC)
		net.devices[mac] = newDevice(cfg)
		net.order = append(net.order, mac)
	}
	return net
}

// Add appends one explicitly-configured device. Returns the assigned MAC.
func (n *Network) Add(cfg DeviceConfig) (string, error) {
	if !validMAC(cfg.MAC) {
		return "", fmt.Errorf("invalid MAC: %q", cfg.MAC)
	}
	mac := strings.ToLower(cfg.MAC)
	cfg.MAC = mac

	n.mu.Lock()
	defer n.mu.Unlock()
	if _, exists := n.devices[mac]; exists {
		return "", fmt.Errorf("duplicate MAC: %s", mac)
	}
	n.devices[mac] = newDevice(cfg)
	n.order = append(n.order, mac)
	return mac, nil
}

// Close releases per-device resources. Phase 1 has no resources to
// release; the method exists so the public API stays stable.
func (n *Network) Close() error {
	return nil
}

// RemoveAuto removes the idx'th auto-generated device (1-indexed) from
// the network and returns its MAC, or "" if no device exists at that
// position. Used by cmd/mocksbr to apply --device overrides on top of
// the --devices N auto-generated baseline.
func (n *Network) RemoveAuto(idx int) string {
	n.mu.Lock()
	defer n.mu.Unlock()
	if idx < 1 || idx > len(n.order) {
		return ""
	}
	mac := n.order[idx-1]
	delete(n.devices, mac)
	n.order = append(n.order[:idx-1], n.order[idx:]...)
	return mac
}

// validMAC returns true for canonical xx:xx:xx:xx:xx:xx hex MACs.
// Hand-rolled to avoid pulling in the regexp package.
func validMAC(s string) bool {
	if len(s) != 17 {
		return false
	}
	for i, ch := range s {
		switch i % 3 {
		case 0, 1:
			if !isHex(byte(ch)) {
				return false
			}
		case 2:
			if ch != ':' {
				return false
			}
		}
	}
	return true
}

func isHex(b byte) bool {
	switch {
	case b >= '0' && b <= '9':
		return true
	case b >= 'a' && b <= 'f':
		return true
	case b >= 'A' && b <= 'F':
		return true
	}
	return false
}
