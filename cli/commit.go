package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/pflag"
)

func runCommit(args []string, stdout, stderr io.Writer) error {
	fs := pflag.NewFlagSet("commit", pflag.ContinueOnError)
	fs.SetOutput(stderr)
	timeout := fs.Duration("timeout", 10*time.Second, "Operation timeout")
	verbose := fs.BoolP("verbose", "v", false, "Debug logging to stderr")
	if err := fs.Parse(args); err != nil {
		return &ExitError{Code: 1, Err: err}
	}
	if fs.NArg() != 1 {
		return &ExitError{Code: 1, Err: fmt.Errorf("commit: expected exactly one MAC argument")}
	}
	mac, err := normalizeMAC(fs.Arg(0))
	if err != nil {
		return &ExitError{Code: 1, Err: err}
	}

	client, err := newClient(*verbose)
	if err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	defer client.Close()

	stop := startProgress(stderr, "commit", *timeout)
	defer stop()
	device, err := discoverAndFind(client, mac, *timeout)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := client.SaveDeviceConfigWithContext(ctx, device); err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("commit (save) failed: %w", err)}
	}
	if err := client.ResetDeviceWithContext(ctx, device); err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("commit (reset) failed: %w", err)}
	}
	return nil
}
