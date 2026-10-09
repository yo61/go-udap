package cli

import (
	"bytes"
	"errors"
	"net"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"go-udap/udap"
)

// stubInterfaces replaces the enumerateInterfaces seam with a fixed list.
func stubInterfaces(t *testing.T, ifs []udap.NetInterface) {
	t.Helper()
	prev := enumerateInterfaces
	enumerateInterfaces = func() ([]udap.NetInterface, error) { return ifs, nil }
	t.Cleanup(func() { enumerateInterfaces = prev })
}

var twoInterfaces = []udap.NetInterface{
	{Name: "en0", Index: 4, Addr: net.IPv4(192, 168, 1, 10), Broadcast: net.IPv4(192, 168, 1, 255)},
	{Name: "utun3", Index: 17, Addr: net.IPv4(10, 8, 0, 2)},
}

func TestE2EInterfacesJSON(t *testing.T) {
	stubInterfaces(t, twoInterfaces)
	env := startMockEnv(t, 0)
	stdout, stderr, exitCode := env.runCLI(t, "interfaces", "-o", "json")
	if exitCode != 0 {
		t.Fatalf("exit code %d, want 0; stderr:\n%s", exitCode, stderr)
	}
	const want = `[{"name":"en0","index":4,"address":"192.168.1.10","broadcast":"192.168.1.255"},` +
		`{"name":"utun3","index":17,"address":"10.8.0.2","broadcast":null}]` + "\n"
	if stdout != want {
		t.Errorf("stdout:\n got %q\nwant %q", stdout, want)
	}
}

func TestE2EInterfacesTextIsUnchanged(t *testing.T) {
	stubInterfaces(t, twoInterfaces)
	env := startMockEnv(t, 0)
	for _, args := range [][]string{{"interfaces"}, {"interfaces", "--format", "text"}} {
		stdout, _, exitCode := env.runCLI(t, args...)
		if exitCode != 0 {
			t.Fatalf("%v: exit code %d, want 0", args, exitCode)
		}
		const want = "NAME            INDEX  ADDRESS            BROADCAST\n" +
			"en0             4      192.168.1.10       192.168.1.255\n" +
			"utun3           17     10.8.0.2           <nil>\n"
		if stdout != want {
			t.Errorf("%v stdout:\n got %q\nwant %q", args, stdout, want)
		}
	}
}

func TestE2EInterfacesCSV(t *testing.T) {
	stubInterfaces(t, twoInterfaces)
	env := startMockEnv(t, 0)
	stdout, _, exitCode := env.runCLI(t, "interfaces", "-o", "csv")
	if exitCode != 0 {
		t.Fatalf("exit code %d, want 0", exitCode)
	}
	const want = "name,index,address,broadcast\n" +
		"en0,4,192.168.1.10,192.168.1.255\n" +
		"utun3,17,10.8.0.2,\n"
	if stdout != want {
		t.Errorf("stdout:\n got %q\nwant %q", stdout, want)
	}
}

func TestE2EInterfacesEmptyResult(t *testing.T) {
	stubInterfaces(t, nil)
	env := startMockEnv(t, 0)
	cases := map[string]string{"text": "", "json": "[]\n", "csv": "name,index,address,broadcast\n"}
	for format, want := range cases {
		stdout, stderr, exitCode := env.runCLI(t, "interfaces", "-o", format)
		if exitCode != 0 {
			t.Fatalf("%s: exit code %d, want 0", format, exitCode)
		}
		if stdout != want {
			t.Errorf("%s stdout: got %q, want %q", format, stdout, want)
		}
		if stderr != "no usable interfaces found\n" {
			t.Errorf("%s stderr: got %q", format, stderr)
		}
	}
}

func TestE2EFormatFlagUsageErrors(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		message string
	}{
		{"json with conflicting format", []string{"interfaces", "--json", "--format", "csv"},
			"if any flags in the group [json format] are set none of the others can be"},
		{"json with agreeing format", []string{"interfaces", "--json", "-o", "json"},
			"if any flags in the group [json format] are set none of the others can be"},
		{"uppercase format name", []string{"interfaces", "-o", "JSON"},
			`invalid argument "JSON" for "-o, --format" flag: want text, json, csv`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stubInterfaces(t, twoInterfaces)
			env := startMockEnv(t, 0)
			stdout, stderr, exitCode := env.runCLI(t, tc.args...)
			if exitCode != 2 {
				t.Errorf("exit code %d, want 2", exitCode)
			}
			if !strings.Contains(stderr, tc.message) {
				t.Errorf("stderr missing %q; got:\n%s", tc.message, stderr)
			}
			if stdout != "" {
				t.Errorf("stdout should be empty; got %q", stdout)
			}
		})
	}
}

func TestE2EVersionFormats(t *testing.T) {
	setVersion(t, "9.8.7")
	cases := map[string]string{
		"text": "go-udap 9.8.7\n",
		"json": `{"name":"go-udap","version":"9.8.7"}` + "\n",
		"csv":  "name,version\ngo-udap,9.8.7\n",
	}
	env := startMockEnv(t, 0)
	for format, want := range cases {
		stdout, stderr, exitCode := env.runCLI(t, "--version", "-o", format)
		if exitCode != 0 {
			t.Fatalf("%s: exit code %d, want 0; stderr:\n%s", format, exitCode, stderr)
		}
		if stdout != want {
			t.Errorf("%s stdout: got %q, want %q", format, stdout, want)
		}
	}
}

// stubBuildInfo replaces the readBuildInfo seam with fixed settings.
func stubBuildInfo(t *testing.T, settings ...debug.BuildSetting) {
	t.Helper()
	prev := readBuildInfo
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Settings: settings}, true
	}
	t.Cleanup(func() { readBuildInfo = prev })
}

func setVersion(t *testing.T, v string) {
	t.Helper()
	prev := Version
	Version = v
	t.Cleanup(func() { Version = prev })
}

func TestE2EBuildInfoWithoutVCSReportsVCSFieldsAsNull(t *testing.T) {
	setVersion(t, "9.8.7")
	stubBuildInfo(t,
		debug.BuildSetting{Key: "GOOS", Value: "linux"},
		debug.BuildSetting{Key: "GOARCH", Value: "arm64"})
	goVersion := runtime.Version()
	cases := map[string]string{
		"json": `{"name":"go-udap","version":"9.8.7","commit":null,"commit_time":null,` +
			`"modified":null,"go_version":"` + goVersion + `","os":"linux","arch":"arm64"}` + "\n",
		"csv": "name,version,commit,commit_time,modified,go_version,os,arch\n" +
			"go-udap,9.8.7,,,," + goVersion + ",linux,arm64\n",
		"text": "go-udap 9.8.7\n" +
			"Go version:  " + goVersion + "\n" +
			"OS:          linux\n" +
			"Arch:        arm64\n",
	}
	env := startMockEnv(t, 0)
	for format, want := range cases {
		stdout, stderr, exitCode := env.runCLI(t, "--build-info", "-o", format)
		if exitCode != 0 {
			t.Fatalf("%s: exit code %d, want 0; stderr:\n%s", format, exitCode, stderr)
		}
		if stdout != want {
			t.Errorf("%s stdout:\n got %q\nwant %q", format, stdout, want)
		}
	}
}

func TestE2EBuildInfoWithVCS(t *testing.T) {
	setVersion(t, "9.8.7")
	stubBuildInfo(t,
		debug.BuildSetting{Key: "vcs.revision", Value: "abc123"},
		debug.BuildSetting{Key: "vcs.time", Value: "2026-10-09T12:00:00Z"},
		debug.BuildSetting{Key: "vcs.modified", Value: "true"},
		debug.BuildSetting{Key: "GOOS", Value: "darwin"},
		debug.BuildSetting{Key: "GOARCH", Value: "amd64"})
	goVersion := runtime.Version()
	env := startMockEnv(t, 0)

	stdout, _, exitCode := env.runCLI(t, "--build-info", "--json")
	if exitCode != 0 {
		t.Fatalf("json: exit code %d, want 0", exitCode)
	}
	wantJSON := `{"name":"go-udap","version":"9.8.7","commit":"abc123",` +
		`"commit_time":"2026-10-09T12:00:00Z","modified":true,"go_version":"` + goVersion +
		`","os":"darwin","arch":"amd64"}` + "\n"
	if stdout != wantJSON {
		t.Errorf("json stdout:\n got %q\nwant %q", stdout, wantJSON)
	}

	stdout, _, exitCode = env.runCLI(t, "--version", "--build-info")
	if exitCode != 0 {
		t.Fatalf("text: exit code %d, want 0", exitCode)
	}
	wantText := "go-udap 9.8.7\n" +
		"Commit:      abc123\n" +
		"Commit time: 2026-10-09T12:00:00Z\n" +
		"Modified:    true\n" +
		"Go version:  " + goVersion + "\n" +
		"OS:          darwin\n" +
		"Arch:        amd64\n"
	if stdout != wantText {
		t.Errorf("text stdout:\n got %q\nwant %q", stdout, wantText)
	}
}

func TestE2EVerboseWithVersionFlagsIsUsageError(t *testing.T) {
	cases := map[string][]string{
		"--version":    {"--version", "--verbose"},
		"--build-info": {"-v", "--build-info"},
	}
	for flag, args := range cases {
		t.Run(flag, func(t *testing.T) {
			env := startMockEnv(t, 0)
			stdout, stderr, exitCode := env.runCLI(t, args...)
			if exitCode != 2 {
				t.Errorf("exit code %d, want 2", exitCode)
			}
			want := "--verbose cannot be used with " + flag
			if !strings.Contains(stderr, want) {
				t.Errorf("stderr missing %q; got:\n%s", want, stderr)
			}
			if stdout != "" {
				t.Errorf("stdout should be empty; got %q", stdout)
			}
		})
	}
}

// addUnannotatedCommand adds a subcommand with neither result
// annotation, as a newly added command would start out.
func addUnannotatedCommand(t *testing.T) {
	t.Helper()
	cmd := &cobra.Command{
		Use:  "unannotated",
		RunE: func(*cobra.Command, []string) error { return nil },
	}
	rootCmd.AddCommand(cmd)
	t.Cleanup(func() { rootCmd.RemoveCommand(cmd) })
}

func TestE2EFormatFlagRejectedWhereThereIsNoResult(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		message string
	}{
		{"reboot", []string{"reboot", "00:04:20:00:00:01", "--json"},
			"reboot: --json not supported (reboot writes no output)"},
		{"reboot with --format text", []string{"reboot", "00:04:20:00:00:01", "-o", "text"},
			"reboot: --format not supported (reboot writes no output)"},
		{"bare root", []string{"--json"},
			"go-udap: --json not supported (go-udap writes no output without --version or --build-info)"},
		{"completion script", []string{"completion", "bash", "--format", "csv"},
			"completion bash: --format not supported (completion writes a shell script)"},
		{"completion parent", []string{"completion", "--json"},
			"completion: --json not supported (completion writes a shell script)"},
		{"command without an annotation", []string{"unannotated", "--json"},
			"unannotated: --json not supported"},
	}
	addUnannotatedCommand(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env := startMockEnv(t, 1)
			stdout, stderr, exitCode := env.runCLI(t, tc.args...)
			if exitCode != 2 {
				t.Errorf("exit code %d, want 2", exitCode)
			}
			if stderr != "error: "+tc.message+"\n" {
				t.Errorf("stderr:\n got %q\nwant %q", stderr, "error: "+tc.message+"\n")
			}
			if stdout != "" {
				t.Errorf("stdout should be empty; got %q", stdout)
			}
		})
	}
}

func TestE2EHelpWinsOverFormatFlag(t *testing.T) {
	for _, args := range [][]string{
		{"read", "--json", "--help"},
		{"reboot", "-o", "csv", "-h"},
		{"help", "read", "--json"},
	} {
		env := startMockEnv(t, 0)
		stdout, stderr, exitCode := env.runCLI(t, args...)
		if exitCode != 0 {
			t.Errorf("%v: exit code %d, want 0; stderr:\n%s", args, exitCode, stderr)
		}
		if !strings.Contains(stdout, "Usage:") {
			t.Errorf("%v: expected help on stdout; got:\n%s", args, stdout)
		}
	}
}

func TestE2EShellCompletionOffersFormatNames(t *testing.T) {
	env := startMockEnv(t, 0)
	stdout, stderr, exitCode := env.runCLI(t, cobra.ShellCompRequestCmd, "interfaces", "-o", "")
	if exitCode != 0 {
		t.Fatalf("exit code %d, want 0; stderr:\n%s", exitCode, stderr)
	}
	for _, name := range []string{"text", "json", "csv"} {
		if !strings.Contains(stdout, name+"\n") {
			t.Errorf("completion missing %q; got:\n%s", name, stdout)
		}
	}
}

func TestE2EShellCompletionIsNotRejectedByFormatFlag(t *testing.T) {
	env := startMockEnv(t, 0)
	stdout, stderr, exitCode := env.runCLI(t, cobra.ShellCompRequestCmd, "--json", "")
	if exitCode != 0 {
		t.Fatalf("exit code %d, want 0; stderr:\n%s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "interfaces") {
		t.Errorf("expected subcommand completions; got:\n%s", stdout)
	}
}

func TestE2EBuildInfoUnavailableReportsOnlyNameVersionAndGoVersion(t *testing.T) {
	setVersion(t, "9.8.7")
	prev := readBuildInfo
	readBuildInfo = func() (*debug.BuildInfo, bool) { return nil, false }
	t.Cleanup(func() { readBuildInfo = prev })
	env := startMockEnv(t, 0)
	stdout, stderr, exitCode := env.runCLI(t, "--build-info", "--json")
	if exitCode != 0 {
		t.Fatalf("exit code %d, want 0; stderr:\n%s", exitCode, stderr)
	}
	want := `{"name":"go-udap","version":"9.8.7","commit":null,"commit_time":null,` +
		`"modified":null,"go_version":"` + runtime.Version() + `","os":null,"arch":null}` + "\n"
	if stdout != want {
		t.Errorf("stdout:\n got %q\nwant %q", stdout, want)
	}
}

func TestE2EInterfacesEnumerationFailureIsOperationFailure(t *testing.T) {
	prev := enumerateInterfaces
	enumerateInterfaces = func() ([]udap.NetInterface, error) {
		return nil, errors.New("netlink down")
	}
	t.Cleanup(func() { enumerateInterfaces = prev })
	env := startMockEnv(t, 0)
	stdout, stderr, exitCode := env.runCLI(t, "interfaces", "--json")
	if exitCode != 1 {
		t.Errorf("exit code %d, want 1", exitCode)
	}
	if !strings.Contains(stderr, "enumerate interfaces: netlink down") {
		t.Errorf("stderr missing the cause; got:\n%s", stderr)
	}
	if stdout != "" {
		t.Errorf("stdout should be empty; got %q", stdout)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("broken pipe") }

func TestE2EStdoutWriteFailureIsOperationFailure(t *testing.T) {
	stubInterfaces(t, twoInterfaces)
	t.Cleanup(resetFlagsForTesting)
	var stderr bytes.Buffer
	err := Execute([]string{"interfaces", "--json"}, failingWriter{}, &stderr)
	if ExitCode(err) != 1 {
		t.Errorf("exit code %d, want 1", ExitCode(err))
	}
	if err == nil || !strings.Contains(err.Error(), "write result: broken pipe") {
		t.Errorf("error = %v, want it to mention the write failure", err)
	}
}
