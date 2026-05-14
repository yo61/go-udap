//go:build darwin

package cli

import (
	"os/exec"
	"regexp"
	"strings"
)

// arpLookup runs `arp -an` and returns the IPv4 paired with mac.
// macOS's arp tool strips leading zeros from each octet (`0:4:20`
// rather than `00:04:20`), so both sides are normalised before
// comparison.
//
// Example output line:
//
//	? (192.168.1.116) at 0:4:20:16:5:8f on en0 ifscope [ethernet]
//
// Returns "" with nil error if the MAC isn't in the cache; non-nil
// error only on exec failure (treated as miss by the caller).
var darwinARPRe = regexp.MustCompile(`\(([0-9.]+)\) at ([0-9a-fA-F:]+)`)

func arpLookup(mac string) (string, error) {
	out, err := exec.Command("arp", "-an").Output()
	if err != nil {
		return "", err
	}
	target := canonicalMAC(mac)
	for _, line := range strings.Split(string(out), "\n") {
		m := darwinARPRe.FindStringSubmatch(line)
		if len(m) != 3 {
			continue
		}
		if canonicalMAC(m[2]) == target {
			return m[1], nil
		}
	}
	return "", nil
}

// canonicalMAC zero-pads each colon-separated MAC octet to two hex
// digits and lower-cases the whole thing. "0:4:20:16:5:8f" →
// "00:04:20:16:05:8f". Returns the input unchanged if it doesn't
// look like six colon-separated octets.
func canonicalMAC(s string) string {
	parts := strings.Split(strings.ToLower(s), ":")
	if len(parts) != 6 {
		return s
	}
	for i, p := range parts {
		if len(p) == 1 {
			parts[i] = "0" + p
		}
	}
	return strings.Join(parts, ":")
}
