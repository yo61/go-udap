package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/pflag"
)

func runRead(args []string, stdout, stderr io.Writer) error {
	fs := pflag.NewFlagSet("read", pflag.ContinueOnError)
	fs.SetOutput(stderr)
	timeout := newDurationWithPlaceholder("DURATION", 5*time.Second)
	fs.Var(timeout, "timeout", "Operation timeout, e.g. 5s, 30s, 2m")
	verbose := fs.BoolP("verbose", "v", false, "Debug logging to stderr")
	includeUnknown := fs.Bool("include-unknown", false,
		"Include offset_NNN entries for NVRAM offsets the device returned but our parameter table doesn't recognize (raw hex; not round-trippable through `set`)")
	if err := parseSubcommandFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return &ExitError{Code: 1, Err: fmt.Errorf("read: expected exactly one MAC argument")}
	}
	mac, err := normalizeMAC(fs.Arg(0))
	if err != nil {
		return &ExitError{Code: 1, Err: err}
	}

	client, err := newClient(*verbose, stderr)
	if err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	defer client.Close()

	stop := startProgress(stderr, "read", timeout.Value())
	defer stop()
	device, err := discoverAndFind(client, mac, timeout.Value())
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout.Value())
	defer cancel()
	if err := client.GetAllDeviceConfigWithContext(ctx, device); err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("read failed: %w", err)}
	}
	stop()

	out := device.Parameters
	if !*includeUnknown {
		out = filterUnknownOffsets(device.Parameters)
	}
	if err := formatParamMap(stdout, out); err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	return nil
}

// filterUnknownOffsets drops entries whose key looks like the synthetic
// "offset_<decimal>" form that parseGetDataResponse uses for NVRAM
// offsets it couldn't reverse-map to a known parameter name. Those
// entries are raw hex and don't round-trip through `set` (which would
// reject the unknown name), so by default we hide them — pass
// --include-unknown to see them.
func filterUnknownOffsets(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		if strings.HasPrefix(k, "offset_") {
			continue
		}
		out[k] = v
	}
	return out
}
