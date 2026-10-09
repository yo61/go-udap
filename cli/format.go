package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"go-udap/cli/output"
)

// Command annotations that decide where a format flag is accepted.
const (
	// annotationResult marks a Data command: it writes a Result.
	annotationResult = "go-udap/result"
	// annotationNoResult gives the reason a command writes no Result.
	// Commands below it in the tree inherit the reason.
	annotationNoResult = "go-udap/no-result"
)

const rootNoResultReason = "go-udap writes no output without --version or --build-info"

var (
	flagFormat output.Format
	flagJSON   bool
)

func init() {
	f := rootCmd.PersistentFlags()
	f.VarP(&flagFormat, "format", "o", "Output format: text, json or csv")
	f.BoolVar(&flagJSON, "json", false, "Shorthand for --format json")
	rootCmd.MarkFlagsMutuallyExclusive("json", "format")
	if err := rootCmd.RegisterFlagCompletionFunc("format", completeFormats); err != nil {
		panic(fmt.Sprintf("register format completion: %v", err))
	}
}

func completeFormats(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return output.Names(), cobra.ShellCompDirectiveNoFileComp
}

// selectedFormat resolves --json and --format. Read it in RunE: cobra
// rejects the two together only after PersistentPreRunE.
func selectedFormat() output.Format {
	if flagJSON {
		return output.JSON
	}
	return flagFormat
}

// renderResult writes r to the command's stdout in the selected format.
func renderResult(cmd *cobra.Command, r output.Result) error {
	if err := output.Render(cmd.OutOrStdout(), selectedFormat(), r); err != nil {
		return &ExitError{Code: exitFailure, Err: fmt.Errorf("write result: %w", err)}
	}
	return nil
}

// rejectFormatFlag fails when a format flag is given to a command that
// writes no Result. Help and shell completion are never rejected.
func rejectFormatFlag(cmd *cobra.Command) error {
	flag := changedFormatFlag(cmd)
	if flag == "" || isHelpOrCompletionRequest(cmd) || writesResult(cmd) {
		return nil
	}
	msg := fmt.Sprintf("%s: %s not supported", commandName(cmd), flag)
	if reason := noOutputReason(cmd); reason != "" {
		msg += " (" + reason + ")"
	}
	return &ExitError{Code: exitUsage, Err: errors.New(msg)}
}

func changedFormatFlag(cmd *cobra.Command) string {
	switch {
	case cmd.Flags().Changed("json"):
		return "--json"
	case cmd.Flags().Changed("format"):
		return "--format"
	default:
		return ""
	}
}

func isHelpOrCompletionRequest(cmd *cobra.Command) bool {
	switch cmd.Name() {
	case "help", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
		return true
	default:
		return false
	}
}

func writesResult(cmd *cobra.Command) bool {
	if !cmd.HasParent() {
		return flagVersion || flagBuildInfo
	}
	_, ok := cmd.Annotations[annotationResult]
	return ok
}

func commandName(cmd *cobra.Command) string {
	if !cmd.HasParent() {
		return cmd.Name()
	}
	return strings.TrimPrefix(cmd.CommandPath(), cmd.Root().Name()+" ")
}

func noOutputReason(cmd *cobra.Command) string {
	if !cmd.HasParent() {
		return rootNoResultReason
	}
	for c := cmd; c.HasParent(); c = c.Parent() {
		if reason, ok := c.Annotations[annotationNoResult]; ok {
			return reason
		}
	}
	return ""
}

// annotateCompletionCmd creates cobra's default completion command now,
// instead of during Execute, so it can carry a no-Result reason. Its RunE
// makes a bare `completion` run the hooks before printing help.
func annotateCompletionCmd() {
	rootCmd.InitDefaultCompletionCmd()
	for _, c := range rootCmd.Commands() {
		if c.Name() == "completion" {
			c.Annotations = map[string]string{annotationNoResult: "completion writes a shell script"}
			c.RunE = func(cmd *cobra.Command, _ []string) error { return cmd.Help() }
		}
	}
}
