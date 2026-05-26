package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"go-udap/udap"
)

var interfacesCmd = &cobra.Command{
	Use:   "interfaces",
	Short: "List network interfaces usable for discovery",
	Args:  cobra.NoArgs,
	RunE:  runInterfaces,
}

func init() {
	rootCmd.AddCommand(interfacesCmd)
}

func runInterfaces(cmd *cobra.Command, _ []string) error {
	stdout := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()

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
