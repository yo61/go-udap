package cli

// lookupIP returns the IPv4 address associated with mac in the host
// ARP cache, or "" if no entry exists or the OS lookup failed. Used
// by deviceFromMAC to populate Device.IP so the udap.Client can send
// the operation as unicast instead of broadcast — necessary on Wi-Fi
// networks where the AP suppresses UDP broadcasts to associated
// clients.
//
// Best-effort: never errors. An empty return leaves Device.IP unset
// and the operation falls back to the broadcast code path. The user
// can populate the ARP cache by pinging the device first.
//
// The actual platform-specific lookup is provided by arp_linux.go /
// arp_darwin.go / arp_windows.go via the arpLookup function below.
func lookupIP(mac string) string {
	ip, err := arpLookup(mac)
	if err != nil {
		return ""
	}
	return ip
}
