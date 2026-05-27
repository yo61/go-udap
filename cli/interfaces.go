package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"go-udap/udap"
)

var interfacesCmd = &cobra.Command{
	Use:   "interfaces",
	Short: "List network interfaces usable for discovery",
	Long: `Print a table of local network interfaces that satisfy the filter
go-udap applies to discovery: up, broadcast-capable, has an IPv4
address, and not a loopback.

Useful for picking a value for the global --bind-interface flag on
multi-homed hosts. The Broadcast column is informational only — UDAP
discovery always targets the limited broadcast 255.255.255.255 so
unconfigured devices can hear it.`,
	Args: cobra.NoArgs,
	RunE: runInterfaces,
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
