package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go-udap/udap"
)

// normalizeMAC accepts MAC addresses written with colons, hyphens, or
// no separators (any case) and returns lowercase colon-separated form.
// Returns an error if the input is not a recognizable MAC.
//
// Hand-rolled byte parsing instead of regexp to avoid pulling in the
// regexp + regexp/syntax packages (~80KB) for two trivial format
// checks.
func normalizeMAC(in string) (string, error) {
	if in == "" {
		return "", fmt.Errorf("empty MAC address")
	}
	lower := strings.ToLower(in)
	withColons := strings.ReplaceAll(lower, "-", ":")
	if isLowerColonMAC(withColons) {
		return withColons, nil
	}
	noSep := strings.ReplaceAll(strings.ReplaceAll(lower, ":", ""), "-", "")
	if isLowerHex12(noSep) {
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

// isLowerColonMAC reports whether s is in `xx:xx:xx:xx:xx:xx` form
// where each x is a lowercase hex digit. The caller has already
// lower-cased the input, so we don't accept upper-case here.
func isLowerColonMAC(s string) bool {
	if len(s) != 17 {
		return false
	}
	for i := 0; i < 17; i++ {
		c := s[i]
		if i%3 == 2 {
			if c != ':' {
				return false
			}
			continue
		}
		if !isLowerHexByte(c) {
			return false
		}
	}
	return true
}

// isLowerHex12 reports whether s is exactly 12 lowercase hex digits.
func isLowerHex12(s string) bool {
	if len(s) != 12 {
		return false
	}
	for i := 0; i < 12; i++ {
		if !isLowerHexByte(s[i]) {
			return false
		}
	}
	return true
}

func isLowerHexByte(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')
}

// findPollInterval is how often discoverAndFind checks for the target MAC
// while discovery runs. Small enough to feel instant, large enough to
// avoid pointless CPU on the device map.
const findPollInterval = 50 * time.Millisecond

// discoverAndFind broadcasts a discovery and returns the device whose
// MAC matches as soon as it appears, cancelling discovery early
// instead of waiting for the full timeout. The caller's ctx is used
// directly so the discovery and the subsequent operation share one
// time budget — review finding #4. Returns an *ExitError with code 2
// if no matching device responds before ctx fires.
func discoverAndFind(ctx context.Context, client *udap.Client, mac string) (*udap.Device, error) {
	discoverCtx, cancelDiscover := context.WithCancel(ctx)
	defer cancelDiscover()

	discoverDone := make(chan error, 1)
	go func() {
		discoverDone <- client.DiscoverDevicesWithContext(discoverCtx)
	}()

	ticker := time.NewTicker(findPollInterval)
	defer ticker.Stop()
	for {
		if d := client.GetDevice(mac); d != nil {
			cancelDiscover()
			<-discoverDone
			return d, nil
		}
		select {
		case <-ticker.C:
			continue
		case err := <-discoverDone:
			if d := client.GetDevice(mac); d != nil {
				return d, nil
			}
			if err != nil && ctx.Err() == nil {
				return nil, &ExitError{Code: 2, Err: fmt.Errorf("discovery failed: %w", err)}
			}
			return nil, &ExitError{Code: 2, Err: fmt.Errorf("device %s not found before timeout", mac)}
		}
	}
}
