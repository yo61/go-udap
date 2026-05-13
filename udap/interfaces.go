package udap

import (
	"fmt"
	"net"
)

// NetInterface is a context-local representation of a network interface
// usable for UDAP broadcast discovery. It translates from net.Interface
// keeping Go-stdlib terminology out of the udap domain — a small
// anti-corruption layer at the boundary.
type NetInterface struct {
	Name      string `json:"name"`
	Index     int    `json:"index"`
	Addr      net.IP `json:"addr"`
	Broadcast net.IP `json:"broadcast"`
}

// EnumerateInterfaces returns all interfaces usable for UDAP broadcast
// discovery. The filter is:
//
//   - FlagUp set
//   - FlagBroadcast set
//   - FlagLoopback NOT set
//   - At least one IPv4 address
//
// For each match, the first IPv4 address (and its mask) drives the
// computed directed-broadcast address (addr | ^mask).
//
// WireGuard / Tailscale interfaces are filtered out automatically
// because they don't carry FlagBroadcast — see the spec for rationale.
func EnumerateInterfaces() ([]NetInterface, error) {
	raw, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("net.Interfaces: %w", err)
	}
	out := make([]NetInterface, 0, len(raw))
	for _, iface := range raw {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagBroadcast == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipnet.IP.To4()
			if ip4 == nil {
				continue
			}
			out = append(out, NetInterface{
				Name:      iface.Name,
				Index:     iface.Index,
				Addr:      ip4,
				Broadcast: computeDirectedBroadcast(ip4, ipnet.Mask),
			})
			break // one IPv4 entry per interface is enough
		}
	}
	return out, nil
}

// computeDirectedBroadcast returns addr | ^mask, i.e. the subnet's
// directed-broadcast address. Pure arithmetic on the 4-byte IPv4
// representation — works the same on every platform.
func computeDirectedBroadcast(addr net.IP, mask net.IPMask) net.IP {
	ip4 := addr.To4()
	if ip4 == nil {
		return nil
	}
	// Normalise mask to 4 bytes too (CIDRMask may return 16).
	m := mask
	if len(m) == 16 {
		m = m[12:16]
	}
	out := make(net.IP, 4)
	for i := 0; i < 4; i++ {
		out[i] = ip4[i] | ^m[i]
	}
	return out
}
