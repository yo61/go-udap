package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"go-udap/udap"
)

func runRead(args []string, stdout, stderr io.Writer) error {
	fs := pflag.NewFlagSet("read", pflag.ContinueOnError)
	fs.SetOutput(stderr)
	timeout := newDurationWithPlaceholder("DURATION", 5*time.Second)
	fs.Var(timeout, "timeout", "Operation timeout, e.g. 5s, 30s, 2m")
	verbose := fs.BoolP("verbose", "v", false, "Debug logging to stderr")
	all := fs.BoolP("all", "a", false,
		"Include factory-default values and offset_NNN entries for unrecognized NVRAM offsets. Default: only print values changed from the factory defaults, so output round-trips cleanly through the set subcommand.")
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

	device, err := deviceFromMAC(mac)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout.Value())
	defer cancel()
	stop := startProgress(stderr, "read", timeout.Value())
	defer stop()
	if err := client.GetAllDeviceConfigWithContext(ctx, device); err != nil {
		return opError("read", mac, timeout.Value(), err)
	}
	stop()

	out := device.Parameters
	if !*all {
		out = filterReadOutput(out)
	}
	if err := formatParamMap(stdout, out); err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	return nil
}

// filterReadOutput trims a device-parameter map down to the entries
// that are interesting for backup/restore via `set`. Two classes of
// entries are dropped:
//
//  1. offset_NNN entries — the synthetic key parseGetDataResponse uses
//     for NVRAM offsets it couldn't reverse-map to a known parameter
//     (raw hex value; `set` would reject the unknown name).
//
//  2. Values matching the parameter's FactoryDefault — boring for
//     backup, and some (wireless_keylen=0, interface=128, empty
//     wireless_SSID) wouldn't even be accepted by `set`'s validation.
//
// Pass `read --all` to disable both filters and dump everything the
// device returned.
func filterReadOutput(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		if strings.HasPrefix(k, "offset_") {
			continue
		}
		if p, ok := udap.ParameterByName(k); ok && v == p.FactoryDefault {
			continue
		}
		out[k] = v
	}
	return out
}
