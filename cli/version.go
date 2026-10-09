package cli

import (
	"fmt"
	"io"
	"runtime"
	"runtime/debug"
	"strconv"

	"github.com/spf13/cobra"
)

var (
	flagVersion   bool
	flagBuildInfo bool
)

// readBuildInfo is a seam for tests.
var readBuildInfo = debug.ReadBuildInfo

func init() {
	f := rootCmd.Flags()
	f.BoolVar(&flagVersion, "version", false, "Print version and exit")
	f.BoolVar(&flagBuildInfo, "build-info", false, "Print version and build metadata, then exit")
}

// runRoot handles the root command's own flags. With none of them set,
// a bare invocation prints help.
func runRoot(cmd *cobra.Command, _ []string) error {
	if flagVerbose && (flagBuildInfo || flagVersion) {
		flag := "--version"
		if flagBuildInfo {
			flag = "--build-info"
		}
		return &ExitError{Code: exitUsage, Err: fmt.Errorf("--verbose cannot be used with %s", flag)}
	}
	if flagBuildInfo {
		return renderResult(cmd, newBuildInfoResult())
	}
	if flagVersion {
		return renderResult(cmd, versionResult{Name: programName, Version: Version})
	}
	return cmd.Help()
}

// versionResult is the Result of --version.
type versionResult struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (r versionResult) WriteText(w io.Writer) error {
	_, err := fmt.Fprintf(w, "%s %s\n", r.Name, r.Version)
	return err
}

func (r versionResult) JSONValue() any { return r }

func (r versionResult) CSVHeader() []string { return []string{"name", "version"} }

func (r versionResult) CSVRows() [][]*string { return [][]*string{{&r.Name, &r.Version}} }

// buildInfoResult is the Result of --build-info.
type buildInfoResult struct {
	Name       string  `json:"name"`
	Version    string  `json:"version"`
	Commit     *string `json:"commit"`
	CommitTime *string `json:"commit_time"`
	Modified   *bool   `json:"modified"`
	GoVersion  string  `json:"go_version"`
	OS         *string `json:"os"`
	Arch       *string `json:"arch"`
}

func newBuildInfoResult() buildInfoResult {
	r := buildInfoResult{Name: programName, Version: Version, GoVersion: runtime.Version()}
	info, ok := readBuildInfo()
	if !ok {
		return r
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			r.Commit = new(s.Value)
		case "vcs.time":
			r.CommitTime = new(s.Value)
		case "vcs.modified":
			r.Modified = new(s.Value == "true")
		case "GOOS":
			r.OS = new(s.Value)
		case "GOARCH":
			r.Arch = new(s.Value)
		}
	}
	return r
}

type buildField struct {
	label string
	value *string
}

// fields lists the metadata after name and version, in output order.
func (r buildInfoResult) fields() []buildField {
	var modified *string
	if r.Modified != nil {
		modified = new(strconv.FormatBool(*r.Modified))
	}
	return []buildField{
		{"Commit", r.Commit},
		{"Commit time", r.CommitTime},
		{"Modified", modified},
		{"Go version", &r.GoVersion},
		{"OS", r.OS},
		{"Arch", r.Arch},
	}
}

func (r buildInfoResult) WriteText(w io.Writer) error {
	if err := (versionResult{Name: r.Name, Version: r.Version}).WriteText(w); err != nil {
		return err
	}
	for _, f := range r.fields() {
		if f.value == nil {
			continue
		}
		if _, err := fmt.Fprintf(w, "%-12s %s\n", f.label+":", *f.value); err != nil {
			return err
		}
	}
	return nil
}

func (r buildInfoResult) JSONValue() any { return r }

func (r buildInfoResult) CSVHeader() []string {
	return []string{"name", "version", "commit", "commit_time", "modified", "go_version", "os", "arch"}
}

func (r buildInfoResult) CSVRows() [][]*string {
	row := []*string{&r.Name, &r.Version}
	for _, f := range r.fields() {
		row = append(row, f.value)
	}
	return [][]*string{row}
}
