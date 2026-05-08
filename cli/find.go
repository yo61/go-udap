package cli

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go-udap/udap"
)

var macColons = regexp.MustCompile(`^[0-9a-f]{2}(:[0-9a-f]{2}){5}$`)
var macHex = regexp.MustCompile(`^[0-9a-f]{12}$`)

// normalizeMAC accepts MAC addresses written with colons, hyphens, or
// no separators (any case) and returns lowercase colon-separated form.
// Returns an error if the input is not a recognizable MAC.
func normalizeMAC(in string) (string, error) {
	if in == "" {
		return "", fmt.Errorf("empty MAC address")
	}
	lower := strings.ToLower(in)
	withColons := strings.ReplaceAll(lower, "-", ":")
	if macColons.MatchString(withColons) {
		return withColons, nil
	}
	noSep := strings.ReplaceAll(strings.ReplaceAll(lower, ":", ""), "-", "")
	if macHex.MatchString(noSep) {
		var out strings.Builder
		for i := 0; i < 12; i += 2 {
			if i > 0 {
				out.WriteByte(':')
			}
			out.WriteString(noSep[i : i+2])
		}
		return out.String(), nil
	}
	return "", fmt.Errorf("invalid MAC address: %q", in)
}

// discoverAndFind broadcasts a discovery, waits up to timeout, and returns
// the device whose MAC matches. Returns an *ExitError with code 2 if not
// found within the timeout.
func discoverAndFind(client *udap.Client, mac string, timeout time.Duration) (*udap.Device, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := client.DiscoverDevicesWithContext(ctx); err != nil {
		return nil, &ExitError{Code: 2, Err: fmt.Errorf("discovery failed: %w", err)}
	}
	if d := client.GetDevice(mac); d != nil {
		return d, nil
	}
	return nil, &ExitError{Code: 2, Err: fmt.Errorf("device %s not found within %s", mac, timeout)}
}
