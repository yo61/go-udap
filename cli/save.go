package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/pflag"
)

func runSave(args []string, stdout, stderr io.Writer) error {
	fs := pflag.NewFlagSet("save", pflag.ContinueOnError)
	fs.SetOutput(stderr)
	timeout := fs.Duration("timeout", 7*time.Second, "Operation timeout")
	verbose := fs.BoolP("verbose", "v", false, "Debug logging to stderr")
	if err := fs.Parse(args); err != nil {
		return &ExitError{Code: 1, Err: err}
	}
	if fs.NArg() != 1 {
		return &ExitError{Code: 1, Err: fmt.Errorf("save: expected exactly one MAC argument")}
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

	stop := startProgress(stderr, "save", *timeout)
	defer stop()
	device, err := discoverAndFind(client, mac, *timeout)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := client.SaveDeviceConfigWithContext(ctx, device); err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("save failed: %w", err)}
	}
	return nil
}
