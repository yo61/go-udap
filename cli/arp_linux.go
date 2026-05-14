//go:build linux

package cli

import (
	"bufio"
	"os"
	"strings"
)

// arpLookup reads the kernel's ARP cache from /proc/net/arp and
// returns the IPv4 paired with mac. Returns "" with nil error when
// the MAC isn't in the cache; non-nil error only on file-open
// failures (treated as miss by the caller).
//
// /proc/net/arp format (space-separated, single header row):
//
//	IP address       HW type     Flags       HW address            Mask     Device
//	192.168.1.116    0x1         0x2         00:04:20:16:05:8f     *        wlan0
func arpLookup(mac string) (string, error) {
	f, err := os.Open("/proc/net/arp")
	if err != nil {
		return "", err
	}
	defer f.Close()

	target := strings.ToLower(mac)
	scanner := bufio.NewScanner(f)
	first := true
	for scanner.Scan() {
		if first {
			first = false
			continue
		}
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}
		if strings.ToLower(fields[3]) == target {
			return fields[0], nil
		}
	}
	return "", scanner.Err()
}
