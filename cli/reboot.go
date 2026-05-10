package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/pflag"
)

func runReboot(args []string, stdout, stderr io.Writer) error {
	fs := pflag.NewFlagSet("reboot", pflag.ContinueOnError)
	fs.SetOutput(stderr)
	timeout := newDurationWithPlaceholder("DURATION", 5*time.Second)
	fs.Var(timeout, "timeout", "Operation timeout, e.g. 5s, 30s, 2m")
	verbose := fs.BoolP("verbose", "v", false, "Debug logging to stderr")
	if err := parseSubcommandFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return &ExitError{Code: 1, Err: fmt.Errorf("reboot: expected exactly one MAC argument")}
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

	ctx, cancel := context.WithTimeout(context.Background(), timeout.Value())
	defer cancel()
	stop := startProgress(stderr, "reboot", timeout.Value())
	defer stop()
	device, err := discoverAndFind(ctx, client, mac)
	if err != nil {
		return err
	}
	if err := client.ResetDeviceWithContext(ctx, device); err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("reboot failed: %w", err)}
	}
	return nil
}
