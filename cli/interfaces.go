package cli

import (
	"fmt"
	"io"

	"github.com/spf13/pflag"

	"go-udap/udap"
)

func runInterfaces(args []string, stdout, stderr io.Writer) error {
	fs := pflag.NewFlagSet("interfaces", pflag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := parseSubcommandFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return &ExitError{Code: 1, Err: fmt.Errorf("interfaces: takes no arguments")}
	}
	ifs, err := udap.EnumerateInterfaces()
	if err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("enumerate interfaces: %w", err)}
	}
	if len(ifs) == 0 {
		fmt.Fprintln(stderr, "no usable interfaces found")
		return nil
	}
	formatInterfacesTable(stdout, ifs)
	return nil
}
