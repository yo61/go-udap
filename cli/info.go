package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/spf13/pflag"
)

func runInfo(args []string, stdout, stderr io.Writer) error {
	fs := pflag.NewFlagSet("info", pflag.ContinueOnError)
	fs.SetOutput(stderr)
	timeout := fs.Duration("timeout", 5*time.Second, "Discovery timeout")
	verbose := fs.BoolP("verbose", "v", false, "Debug logging to stderr")
	if err := fs.Parse(args); err != nil {
		return &ExitError{Code: 1, Err: err}
	}
	if fs.NArg() != 1 {
		return &ExitError{Code: 1, Err: fmt.Errorf("info: expected exactly one MAC argument")}
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

	stop := startProgress(stderr, "info", *timeout)
	device, err := discoverAndFind(client, mac, *timeout)
	stop()
	if err != nil {
		return err
	}
	formatDeviceInfo(stdout, device)
	return nil
}
