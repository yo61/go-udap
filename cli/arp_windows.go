//go:build windows

package cli

import (
	"os/exec"
	"regexp"
	"strings"
)

// arpLookup runs `arp -a` and returns the IPv4 paired with mac.
// Windows uses hyphen-separated MACs in the output and groups entries
// by interface header lines.
//
// Example block:
//
//	Interface: 192.168.1.241 --- 0xc
//	  Internet Address      Physical Address      Type
//	  192.168.1.116         00-04-20-16-05-8f     dynamic
//
// Returns "" with nil error if the MAC isn't in the cache; non-nil
// error only on exec failure (treated as miss by the caller).
var windowsARPRe = regexp.MustCompile(`^\s*(\d+\.\d+\.\d+\.\d+)\s+([0-9a-fA-F-]+)`)

func arpLookup(mac string) (string, error) {
	out, err := exec.Command("arp", "-a").Output()
	if err != nil {
		return "", err
	}
	target := strings.ToLower(mac)
	for _, line := range strings.Split(string(out), "\r\n") {
		m := windowsARPRe.FindStringSubmatch(line)
		if len(m) != 3 {
			continue
		}
		// Windows uses xx-xx-...; normalise to colon form for compare.
		candidate := strings.ToLower(strings.ReplaceAll(m[2], "-", ":"))
		if candidate == target {
			return m[1], nil
		}
	}
	return "", nil
}
