package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/pflag"

	"go-udap/udap"
)

func runGet(args []string, stdout, stderr io.Writer) error {
	fs := pflag.NewFlagSet("get", pflag.ContinueOnError)
	fs.SetOutput(stderr)
	timeout := newDurationWithPlaceholder("DURATION", 5*time.Second)
	fs.Var(timeout, "timeout", "Operation timeout, e.g. 5s, 30s, 2m")
	verbose := fs.BoolP("verbose", "v", false, "Debug logging to stderr")
	if err := parseSubcommandFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return &ExitError{Code: 1, Err: fmt.Errorf("get: expected MAC and at least one parameter name")}
	}
	mac, err := normalizeMAC(fs.Arg(0))
	if err != nil {
		return &ExitError{Code: 1, Err: err}
	}
	params := fs.Args()[1:]
	for _, p := range params {
		if _, ok := udap.ParameterByName(p); !ok {
			return &ExitError{Code: 1, Err: fmt.Errorf("get: unknown parameter %q", p)}
		}
	}

	client, err := newClient(*verbose, stderr)
	if err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	defer client.Close()

	stop := startProgress(stderr, "get", timeout.Value())
	defer stop()
	device, err := discoverAndFind(client, mac, timeout.Value())
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout.Value())
	defer cancel()
	values, err := client.GetDeviceConfigWithContext(ctx, device, params)
	if err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("get failed: %w", err)}
	}
	stop()
	if err := formatGetResult(stdout, params, values); err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	return nil
}
