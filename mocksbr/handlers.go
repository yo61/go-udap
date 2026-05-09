package mocksbr

import (
	"fmt"
	"strings"

	"go-udap/udap"
)

// Receive feeds an inbound packet to the matching device(s) and returns
// zero or more reply packets.
//
//   - Discovery requests (broadcast) fan out to every device.
//   - Unicast requests (GetData, SetData, Reset) target one device by
//     destination MAC. Unknown MACs produce no replies.
//   - Devices in their post-Reset reboot window silently drop packets.
func (n *Network) Receive(packetBytes []byte) [][]byte {
	pkt, payload, err := udap.ParsePacket(packetBytes)
	if err != nil {
		n.logger.Debug("mocksbr: ignoring unparseable packet", "error", err)
		return nil
	}

	switch pkt.UCPMethod {
	case udap.MethodDiscover, udap.MethodAdvDisc:
		return n.handleDiscovery(pkt)
	case udap.MethodGetData:
		return n.dispatchUnicast(pkt, func(d *device) []byte {
			return d.buildGetDataResponse(pkt, payload)
		})
	case udap.MethodSetData:
		return n.dispatchUnicast(pkt, func(d *device) []byte {
			return d.handleSetData(pkt, payload)
		})
	case udap.MethodReset:
		return n.dispatchUnicast(pkt, func(d *device) []byte {
			ack := d.buildResetAck(pkt)
			d.startReboot()
			d.applyReset()
			return ack
		})
	default:
		n.logger.Debug("mocksbr: unhandled UCPMethod",
			"method", fmt.Sprintf("0x%04x", pkt.UCPMethod))
		return nil
	}
}

// handleDiscovery returns one discovery response per device, in the
// stable insertion order used by NewNetwork/Add. Devices in their
// reboot window are skipped.
func (n *Network) handleDiscovery(pkt *udap.Packet) [][]byte {
	n.mu.Lock()
	macs := append([]string(nil), n.order...)
	n.mu.Unlock()

	replies := make([][]byte, 0, len(macs))
	for _, mac := range macs {
		d := n.devices[mac]
		if d.rebooting() {
			continue
		}
		replies = append(replies, d.buildDiscoveryResponse(pkt))
	}
	return replies
}

// dispatchUnicast looks up the target device by destination MAC and
// invokes handler if the device is present and not rebooting.
func (n *Network) dispatchUnicast(pkt *udap.Packet, handler func(*device) []byte) [][]byte {
	mac := strings.ToLower(formatMAC(pkt.DstAddress))

	n.mu.Lock()
	d, ok := n.devices[mac]
	n.mu.Unlock()
	if !ok {
		n.logger.Debug("mocksbr: no device for MAC", "mac", mac)
		return nil
	}
	if d.rebooting() {
		n.logger.Debug("mocksbr: device rebooting, dropping packet", "mac", mac)
		return nil
	}
	reply := handler(d)
	if reply == nil {
		return nil
	}
	return [][]byte{reply}
}

// handleSetData parses the request, applies recognized params to working
// memory, and returns a SetData ack with payload = count of params
// accepted. SetData and SaveData are the same wire method (0x0006); the
// mock copies working memory → NVRAM on every set so that subsequent
// Reset reloads observe the most recent values, matching real-SBR
// behavior on the test bench.
func (d *device) handleSetData(req *udap.Packet, payload []byte) []byte {
	items := parseSetDataRequest(payload)
	updates := make(map[string]string, len(items))
	for _, item := range items {
		if item.name == "" {
			continue
		}
		updates[item.name] = decodeParamValue(item.value)
	}
	d.applySet(updates)
	d.applySave()
	return d.buildSetDataAck(req, uint16(len(items)))
}

// formatMAC renders a [6]byte MAC as lowercase aa:bb:cc:dd:ee:ff.
func formatMAC(b [6]byte) string {
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		b[0], b[1], b[2], b[3], b[4], b[5])
}
