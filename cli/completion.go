package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"go-udap/udap"
)

// completionStateDir is a seam for tests; production uses os.TempDir.
var completionStateDir = os.TempDir

// completionStatePath returns the per-shell-session state file path. The
// PPID scopes it to the invoking shell so two terminal sessions don't
// collide.
func completionStatePath() string {
	return filepath.Join(completionStateDir(),
		fmt.Sprintf("go-udap-complete-%d", os.Getppid()))
}

// nextMACTimeout picks the discovery timeout for completeMACs based on
// the previous attempt within the freshness window.
//
//   - File missing or mtime older than 10 s → cold tier (500 ms).
//   - File fresh AND last attempt returned 0 devices → retry tier
//     (defaultTimeout, currently 2 s).
//   - File fresh AND last attempt returned >0 devices → stay cold; the
//     network is responsive, no need to escalate.
func nextMACTimeout() time.Duration {
	const cold = 500 * time.Millisecond
	info, err := os.Stat(completionStatePath())
	if err != nil || time.Since(info.ModTime()) > 10*time.Second {
		return cold
	}
	data, err := os.ReadFile(completionStatePath())
	if err != nil {
		return cold
	}
	var n int
	if _, err := fmt.Sscanf(string(data), "%d", &n); err != nil {
		return cold
	}
	if n == 0 {
		return defaultTimeout
	}
	return cold
}

// recordMACAttempt writes the result count of the just-completed MAC
// completion attempt. Errors are swallowed — completion still works on
// the cold tier if state-writes fail.
func recordMACAttempt(count int) {
	_ = os.WriteFile(completionStatePath(),
		[]byte(strconv.Itoa(count)), 0o600)
}

// newClientForCompletion wraps newClient but discards all logger output
// so completion subprocesses don't leak warnings to the user's prompt.
var newClientForCompletion = func(_ *cobra.Command) (*udap.Client, error) {
	return newClient(false, io.Discard)
}

// completeMACs is the ValidArgsFunction for info/read/get/set/reboot/getip.
// It runs a short-timeout UDAP discovery and returns "<MAC>\t<Name>" entries
// for Cobra to render. The first tab uses a 500 ms budget; if that returns
// empty, a second tab within 10 s uses defaultTimeout (the global --timeout
// default, currently 2 s).
func completeMACs(cmd *cobra.Command, args []string, _ string) (
	[]string, cobra.ShellCompDirective,
) {
	if len(args) >= 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	parent := cmd.Context()
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, nextMACTimeout())
	defer cancel()
	client, err := newClientForCompletion(cmd)
	if err != nil {
		recordMACAttempt(0)
		return nil, cobra.ShellCompDirectiveError
	}
	defer client.Close()
	_ = client.DiscoverDevicesWithContext(ctx)
	devices := client.ListDevices()
	out := make([]string, 0, len(devices))
	for _, d := range devices {
		if d.Name != "" {
			out = append(out, fmt.Sprintf("%s\t%s", d.MAC, d.Name))
		} else {
			out = append(out, d.MAC.String())
		}
	}
	recordMACAttempt(len(out))
	return out, cobra.ShellCompDirectiveNoFileComp
}

// completeInterfaces is the flag-value completer for --bind-interface.
// Uses udap.EnumerateInterfaces() — instant, local, no network.
func completeInterfaces(_ *cobra.Command, _ []string, _ string) (
	[]string, cobra.ShellCompDirective,
) {
	ifs, err := udap.EnumerateInterfaces()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	out := make([]string, 0, len(ifs))
	for _, i := range ifs {
		out = append(out, fmt.Sprintf("%s\t%s", i.Name, i.Addr))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

// completeParameterNames is the ValidArgsFunction for `get <mac> <param>...`.
// First positional arg (MAC) delegates to completeMACs; subsequent args
// complete from udap.Parameters, filtering out names already on the line.
func completeParameterNames(cmd *cobra.Command, args []string, toComplete string) (
	[]string, cobra.ShellCompDirective,
) {
	if len(args) == 0 {
		return completeMACs(cmd, args, toComplete)
	}
	out := make([]string, 0, len(udap.Parameters))
	for _, p := range udap.Parameters {
		if slices.Contains(args[1:], p.Name) {
			continue
		}
		out = append(out, fmt.Sprintf("%s\t%s", p.Name, p.Help))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

// dumpCompletionsCmd is hidden; GoReleaser's before-hook invokes it to
// produce the three completion scripts that ship in the release archive.
var dumpCompletionsCmd = &cobra.Command{
	Use:    "__dump-completions <out-dir>",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := args[0]
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		if err := rootCmd.GenBashCompletionFileV2(
			filepath.Join(dir, "go-udap.bash"), true); err != nil {
			return err
		}
		if err := rootCmd.GenZshCompletionFile(
			filepath.Join(dir, "_go-udap")); err != nil {
			return err
		}
		return rootCmd.GenFishCompletionFile(
			filepath.Join(dir, "go-udap.fish"), true)
	},
}

func init() {
	rootCmd.AddCommand(dumpCompletionsCmd)
}
