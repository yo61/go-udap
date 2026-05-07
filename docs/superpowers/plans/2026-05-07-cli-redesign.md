# CLI Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the readline-based interactive shell with single-shot CLI subcommands (`discover`, `info`, `read`, `get`, `set`, `save`, `reset`, `commit`) using GNU-style flags via `spf13/pflag`.

**Architecture:** New `cli/` package containing one file per subcommand plus shared helpers (INI parser, flag table for ~25 known params, source layering for `set`, output formatting, dispatcher). The `udap/` protocol package is unchanged except for two minor API additions: an exported `ValidateParameter` wrapper and changing the default logger destination from stdout to stderr. `main.go` becomes a thin dispatcher; readline is removed.

**Tech Stack:** Go 1.25, `github.com/spf13/pflag` (new), `golang.org/x/sys` (existing transitive), no other new dependencies.

**Spec:** [`docs/superpowers/specs/2026-05-07-cli-redesign-design.md`](../specs/2026-05-07-cli-redesign-design.md)

**Branch:** `robin/cli-redesign` (already created during spec commit)

---

## Task 1: Add `spf13/pflag` dependency

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Verify on the right branch**

Run: `git -C /Users/robin/code/github/robinbowes/go-udap branch --show-current`
Expected: `robin/cli-redesign`

- [ ] **Step 2: Add the dependency**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go get github.com/spf13/pflag@latest`
Expected: stdout shows `go: added github.com/spf13/pflag vX.Y.Z`; `go.mod` and `go.sum` updated.

- [ ] **Step 3: Verify go.mod**

Run: `grep pflag /Users/robin/code/github/robinbowes/go-udap/go.mod`
Expected: a line containing `github.com/spf13/pflag v` (any version).

- [ ] **Step 4: Confirm build still succeeds**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go build ./...`
Expected: exit code 0, no output.

- [ ] **Step 5: Commit**

```bash
cd /Users/robin/code/github/robinbowes/go-udap
git add go.mod go.sum
git commit -m "chore: add spf13/pflag dependency"
```

---

## Task 2: Change udap logger to write to stderr

The CLI must keep stdout machine-parseable. Logger currently writes to stdout (`udap/logger.go:56`).

**Files:**
- Modify: `udap/logger.go:56`
- Test: `udap/logger_test.go` (new)

- [ ] **Step 1: Write the failing test**

Create `/Users/robin/code/github/robinbowes/go-udap/udap/logger_test.go`:

```go
package udap

import (
	"bytes"
	"log"
	"os"
	"testing"
)

func TestNewStructuredLoggerWritesToStderr(t *testing.T) {
	l := NewStructuredLogger()
	if l.logger.Writer() != os.Stderr {
		t.Fatalf("expected logger writer to be os.Stderr, got %v", l.logger.Writer())
	}
}

func TestStructuredLoggerLogsMessageWithFields(t *testing.T) {
	var buf bytes.Buffer
	l := &StructuredLogger{
		level:  LogLevelInfo,
		logger: log.New(&buf, "", 0),
	}
	l.Info("hello", "k", "v")

	got := buf.String()
	if !bytes.Contains([]byte(got), []byte("hello")) {
		t.Fatalf("expected output to contain 'hello', got %q", got)
	}
	if !bytes.Contains([]byte(got), []byte("k=v")) {
		t.Fatalf("expected output to contain 'k=v', got %q", got)
	}
}
```

- [ ] **Step 2: Run the test, verify the writer test fails**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go test ./udap/ -run TestNewStructuredLoggerWritesToStderr -v`
Expected: FAIL — `expected logger writer to be os.Stderr, got &{...os.Stdout...}`.

- [ ] **Step 3: Change the logger destination**

Edit `udap/logger.go` line 56:

Replace:
```go
		logger: log.New(os.Stdout, "", 0),
```

With:
```go
		logger: log.New(os.Stderr, "", 0),
```

- [ ] **Step 4: Run the tests, verify both pass**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go test ./udap/ -run "TestNewStructuredLogger|TestStructuredLoggerLogs" -v`
Expected: PASS for both tests.

- [ ] **Step 5: Run full udap test suite to verify no regression**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go test ./udap/...`
Expected: PASS, no failures.

- [ ] **Step 6: Commit**

```bash
cd /Users/robin/code/github/robinbowes/go-udap
git add udap/logger.go udap/logger_test.go
git commit -m "feat(udap): write log output to stderr"
```

---

## Task 3: Export `ValidateParameter` from udap

The CLI needs to validate per-flag values without constructing a full `Device`. The package-private `validateParameter` (`udap/validation.go:38`) does exactly this; expose a thin wrapper.

**Files:**
- Modify: `udap/validation.go` (add exported wrapper near the unexported function)
- Test: `udap/validation_test.go` (new)

- [ ] **Step 1: Write the failing test**

Create `/Users/robin/code/github/robinbowes/go-udap/udap/validation_test.go`:

```go
package udap

import "testing"

func TestValidateParameterAcceptsKnownValid(t *testing.T) {
	if err := ValidateParameter("lan_ip_mode", "1"); err != nil {
		t.Fatalf("expected nil error for lan_ip_mode=1, got %v", err)
	}
}

func TestValidateParameterRejectsBadIP(t *testing.T) {
	if err := ValidateParameter("lan_network_address", "not.an.ip"); err == nil {
		t.Fatalf("expected error for invalid IP, got nil")
	}
}

func TestValidateParameterAcceptsUnknownParameter(t *testing.T) {
	// Matches the existing internal behavior: unknown params are not
	// rejected by validation; rejection happens at the CLI boundary.
	if err := ValidateParameter("not_a_real_param", "x"); err != nil {
		t.Fatalf("expected nil error for unknown param, got %v", err)
	}
}
```

- [ ] **Step 2: Run the test, verify it fails**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go test ./udap/ -run TestValidateParameter -v`
Expected: FAIL — `undefined: ValidateParameter`.

- [ ] **Step 3: Add the exported wrapper**

Edit `udap/validation.go`. Find the line `// validateParameter validates a configuration parameter based on its name and type` (around line 37) and immediately *before* it, insert:

```go
// ValidateParameter validates a single configuration parameter by name and value.
// Unknown parameter names are accepted (return nil); rejection of unknown names
// happens at the CLI boundary.
func ValidateParameter(name, value string) error {
	return validateParameter(name, value)
}

```

- [ ] **Step 4: Run the test, verify it passes**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go test ./udap/ -run TestValidateParameter -v`
Expected: PASS for all three test cases.

- [ ] **Step 5: Run full udap test suite**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go test ./udap/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
cd /Users/robin/code/github/robinbowes/go-udap
git add udap/validation.go udap/validation_test.go
git commit -m "feat(udap): export ValidateParameter wrapper"
```

---

## Task 4: Create `cli/` package with INI parser

**Files:**
- Create: `cli/config.go`
- Test: `cli/config_test.go`

- [ ] **Step 1: Write the failing tests**

Create `/Users/robin/code/github/robinbowes/go-udap/cli/config_test.go`:

```go
package cli

import (
	"strings"
	"testing"
)

func TestParseINIBasic(t *testing.T) {
	in := strings.NewReader("lan_ip_mode=1\nwireless_SSID=MyNet\n")
	got, err := ParseINI(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["lan_ip_mode"] != "1" {
		t.Errorf("lan_ip_mode: want 1, got %q", got["lan_ip_mode"])
	}
	if got["wireless_SSID"] != "MyNet" {
		t.Errorf("wireless_SSID: want MyNet, got %q", got["wireless_SSID"])
	}
}

func TestParseINICommentsAndBlanks(t *testing.T) {
	in := strings.NewReader("# hash comment\n; semicolon comment\n\n  lan_ip_mode = 1  \n")
	got, err := ParseINI(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["lan_ip_mode"] != "1" {
		t.Errorf("expected trimmed value 1, got %q", got["lan_ip_mode"])
	}
	if len(got) != 1 {
		t.Errorf("expected one entry, got %d", len(got))
	}
}

func TestParseINIRejectsMalformedLine(t *testing.T) {
	in := strings.NewReader("lan_ip_mode\n")
	_, err := ParseINI(in)
	if err == nil {
		t.Fatalf("expected error for line without =")
	}
}

func TestParseINIRejectsUnknownParameter(t *testing.T) {
	in := strings.NewReader("not_a_real_param=x\n")
	_, err := ParseINI(in)
	if err == nil {
		t.Fatalf("expected error for unknown parameter name")
	}
}

func TestParseINIRejectsInvalidValue(t *testing.T) {
	in := strings.NewReader("lan_network_address=not.an.ip\n")
	_, err := ParseINI(in)
	if err == nil {
		t.Fatalf("expected error for invalid IP value")
	}
}

func TestParseINIEmptyValueAllowed(t *testing.T) {
	// hostname is a string param with max length 33; empty is valid.
	in := strings.NewReader("hostname=\n")
	got, err := ParseINI(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["hostname"] != "" {
		t.Errorf("expected empty string, got %q", got["hostname"])
	}
}
```

- [ ] **Step 2: Run tests, verify they fail with package-not-found**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go test ./cli/...`
Expected: FAIL — `no Go files in .../cli` or `package cli; expected package name`.

- [ ] **Step 3: Create the implementation**

Create `/Users/robin/code/github/robinbowes/go-udap/cli/config.go`:

```go
package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"go-udap/udap"
)

// ParseINI parses an INI-style stream of key=value lines.
// Lines starting with # or ; are comments; blank lines are ignored.
// Whitespace around keys and values is trimmed. Values are not quoted.
// Unknown parameter names and values failing udap.ValidateParameter
// produce errors.
func ParseINI(r io.Reader) (map[string]string, error) {
	out := make(map[string]string)
	scanner := bufio.NewScanner(r)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
			continue
		}
		eq := strings.IndexByte(trimmed, '=')
		if eq < 0 {
			return nil, fmt.Errorf("line %d: missing '=' separator", lineNum)
		}
		key := strings.TrimSpace(trimmed[:eq])
		value := strings.TrimSpace(trimmed[eq+1:])
		if _, known := udap.ConfigSettings[key]; !known {
			return nil, fmt.Errorf("line %d: unknown parameter %q", lineNum, key)
		}
		if err := udap.ValidateParameter(key, value); err != nil {
			return nil, fmt.Errorf("line %d: %s: %w", lineNum, key, err)
		}
		out[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}
	return out, nil
}
```

- [ ] **Step 4: Run tests, verify they all pass**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go test ./cli/... -v`
Expected: PASS for all six tests in `TestParseINI*`.

- [ ] **Step 5: Commit**

```bash
cd /Users/robin/code/github/robinbowes/go-udap
git add cli/config.go cli/config_test.go
git commit -m "feat(cli): add INI parser for set source files"
```

---

## Task 5: Add CLI flag table for known parameters

**Files:**
- Create: `cli/params.go`
- Test: `cli/params_test.go`

- [ ] **Step 1: Write the failing tests**

Create `/Users/robin/code/github/robinbowes/go-udap/cli/params_test.go`:

```go
package cli

import (
	"testing"

	"go-udap/udap"
)

func TestParamFlagsCoverAllKnownParameters(t *testing.T) {
	flags := paramFlags()
	byUDAP := make(map[string]paramFlag, len(flags))
	for _, f := range flags {
		byUDAP[f.udapName] = f
	}
	for _, name := range udap.KnownParameters {
		if _, ok := byUDAP[name]; !ok {
			t.Errorf("missing flag table entry for known parameter %q", name)
		}
	}
}

func TestParamFlagsAllReferenceConfigSettings(t *testing.T) {
	for _, f := range paramFlags() {
		if _, ok := udap.ConfigSettings[f.udapName]; !ok {
			t.Errorf("flag table entry %q does not match any ConfigSettings key", f.udapName)
		}
	}
}

func TestParamFlagNamesAreHyphenatedLowercase(t *testing.T) {
	for _, f := range paramFlags() {
		for _, ch := range f.flagName {
			if ch == '_' || (ch >= 'A' && ch <= 'Z') {
				t.Errorf("flag name %q must be lowercase-with-hyphens (no underscores or uppercase)", f.flagName)
				break
			}
		}
	}
}

func TestParamFlagsHaveHelpText(t *testing.T) {
	for _, f := range paramFlags() {
		if f.help == "" {
			t.Errorf("flag table entry %q has no help text", f.udapName)
		}
	}
}
```

- [ ] **Step 2: Run tests, verify they fail with undefined symbols**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go test ./cli/ -run TestParamFlags -v`
Expected: FAIL — `undefined: paramFlags` and `undefined: paramFlag`.

- [ ] **Step 3: Create the implementation**

Create `/Users/robin/code/github/robinbowes/go-udap/cli/params.go`:

```go
package cli

// paramFlag maps a CLI flag name to its canonical UDAP parameter name and help text.
//   - udapName is used in protocol messages and INI files (e.g. "wireless_SSID").
//   - flagName is the CLI form, lowercase-with-hyphens (e.g. "wireless-ssid").
type paramFlag struct {
	udapName string
	flagName string
	help     string
}

// paramFlags returns the full table of CLI flags for known UDAP parameters.
// The table must stay in sync with udap.KnownParameters; cli/params_test.go
// asserts coverage in both directions.
func paramFlags() []paramFlag {
	return []paramFlag{
		{"lan_ip_mode", "lan-ip-mode", "0=static, 1=DHCP"},
		{"lan_network_address", "lan-network-address", "Static IPv4 address (e.g. 192.168.1.50)"},
		{"lan_subnet_mask", "lan-subnet-mask", "Subnet mask (e.g. 255.255.255.0)"},
		{"lan_gateway", "lan-gateway", "Default gateway IPv4 address"},
		{"hostname", "hostname", "Device hostname (max 33 chars)"},
		{"bridging", "bridging", "0=disabled, 1=enabled"},
		{"interface", "interface", "0=wireless, 1=wired (Ethernet)"},
		{"primary_dns", "primary-dns", "Primary DNS server IPv4 address"},
		{"secondary_dns", "secondary-dns", "Secondary DNS server IPv4 address"},
		{"server_address", "server-address", "Logitech Media Server IPv4 address"},
		{"lms_address", "lms-address", "Alternative LMS server IPv4 address"},
		{"wireless_mode", "wireless-mode", "0=infrastructure, 1=ad-hoc"},
		{"wireless_SSID", "wireless-ssid", "Wireless SSID (1-32 chars)"},
		{"wireless_channel", "wireless-channel", "Wireless channel (1-13)"},
		{"wireless_region_id", "wireless-region-id", "Wireless region identifier"},
		{"wireless_keylen", "wireless-keylen", "WEP key length: 5 or 13"},
		{"wireless_wep_key", "wireless-wep-key", "Primary WEP key"},
		{"wireless_wep_key_1", "wireless-wep-key-1", "WEP key slot 1"},
		{"wireless_wep_key_2", "wireless-wep-key-2", "WEP key slot 2"},
		{"wireless_wep_key_3", "wireless-wep-key-3", "WEP key slot 3"},
		{"wireless_wep_on", "wireless-wep-on", "0=disabled, 1=enabled"},
		{"wireless_wpa_cipher", "wireless-wpa-cipher", "WPA cipher type"},
		{"wireless_wpa_mode", "wireless-wpa-mode", "WPA mode"},
		{"wireless_wpa_on", "wireless-wpa-on", "0=disabled, 1=enabled"},
		{"wireless_wpa_psk", "wireless-wpa-psk", "WPA pre-shared key (8-63 chars)"},
	}
}
```

- [ ] **Step 4: Run tests, verify they pass**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go test ./cli/ -run TestParamFlags -v`
Expected: PASS for all four `TestParamFlags*` tests.

- [ ] **Step 5: Commit**

```bash
cd /Users/robin/code/github/robinbowes/go-udap
git add cli/params.go cli/params_test.go
git commit -m "feat(cli): add flag table for known UDAP parameters"
```

---

## Task 6: Source layering for `set`

The `set` subcommand merges values from up to three sources: `--config FILE` (or `--config -` for stdin), piped stdin (auto-detected), and CLI flags. CLI flags always win; if both `--config FILE` and piped stdin are supplied, the file wins and stdin is ignored with a warning on stderr.

**Files:**
- Create: `cli/source.go`
- Test: `cli/source_test.go`

- [ ] **Step 1: Write the failing tests**

Create `/Users/robin/code/github/robinbowes/go-udap/cli/source_test.go`:

```go
package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestMergeSourcesFileOnly(t *testing.T) {
	var warn bytes.Buffer
	in := sourceInputs{
		fileContent: strings.NewReader("lan_ip_mode=1\n"),
		fileLabel:   "test.conf",
	}
	got, err := mergeSources(in, &warn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["lan_ip_mode"] != "1" {
		t.Errorf("want lan_ip_mode=1, got %q", got["lan_ip_mode"])
	}
	if warn.Len() != 0 {
		t.Errorf("expected no warnings, got %q", warn.String())
	}
}

func TestMergeSourcesStdinOnly(t *testing.T) {
	var warn bytes.Buffer
	in := sourceInputs{
		stdinContent: strings.NewReader("hostname=foo\n"),
		stdinPiped:   true,
	}
	got, err := mergeSources(in, &warn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["hostname"] != "foo" {
		t.Errorf("want hostname=foo, got %q", got["hostname"])
	}
}

func TestMergeSourcesFlagsOnly(t *testing.T) {
	var warn bytes.Buffer
	in := sourceInputs{
		flags: map[string]string{"lan_ip_mode": "0"},
	}
	got, err := mergeSources(in, &warn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["lan_ip_mode"] != "0" {
		t.Errorf("want lan_ip_mode=0, got %q", got["lan_ip_mode"])
	}
}

func TestMergeSourcesFlagsOverrideFile(t *testing.T) {
	var warn bytes.Buffer
	in := sourceInputs{
		fileContent: strings.NewReader("lan_ip_mode=1\nhostname=base\n"),
		fileLabel:   "base.conf",
		flags:       map[string]string{"hostname": "override"},
	}
	got, err := mergeSources(in, &warn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["hostname"] != "override" {
		t.Errorf("want hostname=override, got %q", got["hostname"])
	}
	if got["lan_ip_mode"] != "1" {
		t.Errorf("want lan_ip_mode=1 (from file), got %q", got["lan_ip_mode"])
	}
}

func TestMergeSourcesFileWinsOverPipedStdinWithWarning(t *testing.T) {
	var warn bytes.Buffer
	in := sourceInputs{
		fileContent:  strings.NewReader("hostname=fromfile\n"),
		fileLabel:    "f.conf",
		stdinContent: strings.NewReader("hostname=fromstdin\n"),
		stdinPiped:   true,
	}
	got, err := mergeSources(in, &warn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["hostname"] != "fromfile" {
		t.Errorf("want hostname=fromfile, got %q", got["hostname"])
	}
	if !strings.Contains(warn.String(), "ignoring piped stdin") {
		t.Errorf("expected warning about ignoring stdin, got %q", warn.String())
	}
}

func TestMergeSourcesExplicitStdinViaConfigDash(t *testing.T) {
	// When fileContent is provided AND fileLabel is "-", that means
	// the caller resolved --config - to stdin; no warning expected.
	var warn bytes.Buffer
	in := sourceInputs{
		fileContent: strings.NewReader("hostname=fromstdin\n"),
		fileLabel:   "-",
	}
	got, err := mergeSources(in, &warn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["hostname"] != "fromstdin" {
		t.Errorf("want hostname=fromstdin, got %q", got["hostname"])
	}
	if warn.Len() != 0 {
		t.Errorf("expected no warnings for explicit --config -, got %q", warn.String())
	}
}

func TestMergeSourcesEmptyIsError(t *testing.T) {
	var warn bytes.Buffer
	in := sourceInputs{}
	_, err := mergeSources(in, &warn)
	if err == nil {
		t.Fatalf("expected error when no sources supply parameters")
	}
}
```

- [ ] **Step 2: Run tests, verify they fail**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go test ./cli/ -run TestMergeSources -v`
Expected: FAIL — `undefined: sourceInputs`, `undefined: mergeSources`.

- [ ] **Step 3: Create the implementation**

Create `/Users/robin/code/github/robinbowes/go-udap/cli/source.go`:

```go
package cli

import (
	"fmt"
	"io"
)

// sourceInputs collects the three possible parameter sources for `set`.
//   - fileContent: bytes from --config FILE (or --config - for stdin); fileLabel
//     identifies it for error messages ("-" if it was explicit stdin).
//   - stdinContent: piped stdin when no explicit --config was given.
//   - stdinPiped: true if stdin was piped (caller's responsibility to detect).
//   - flags: per-param values from CLI flags.
type sourceInputs struct {
	fileContent  io.Reader
	fileLabel    string
	stdinContent io.Reader
	stdinPiped   bool
	flags        map[string]string
}

// mergeSources combines parameter sources in layered order
// (file/stdin first, then CLI flags overlay) and returns the merged map.
// Warnings (e.g. piped stdin ignored when a --config FILE was given) are
// written to warn. Returns an error if no source supplies any parameters.
func mergeSources(in sourceInputs, warn io.Writer) (map[string]string, error) {
	merged := make(map[string]string)

	switch {
	case in.fileContent != nil:
		params, err := ParseINI(in.fileContent)
		if err != nil {
			label := in.fileLabel
			if label == "" {
				label = "config"
			}
			return nil, fmt.Errorf("%s: %w", label, err)
		}
		for k, v := range params {
			merged[k] = v
		}
		if in.stdinPiped && in.fileLabel != "-" {
			fmt.Fprintln(warn, "warning: --config supplied; ignoring piped stdin")
		}
	case in.stdinPiped && in.stdinContent != nil:
		params, err := ParseINI(in.stdinContent)
		if err != nil {
			return nil, fmt.Errorf("stdin: %w", err)
		}
		for k, v := range params {
			merged[k] = v
		}
	}

	for k, v := range in.flags {
		merged[k] = v
	}

	if len(merged) == 0 {
		return nil, fmt.Errorf("no parameters specified")
	}
	return merged, nil
}
```

- [ ] **Step 4: Run tests, verify they pass**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go test ./cli/ -run TestMergeSources -v`
Expected: PASS for all seven tests.

- [ ] **Step 5: Commit**

```bash
cd /Users/robin/code/github/robinbowes/go-udap
git add cli/source.go cli/source_test.go
git commit -m "feat(cli): layer config file, stdin, and flags for set"
```

---

## Task 7: Output formatting helpers

**Files:**
- Create: `cli/output.go`
- Test: `cli/output_test.go`

- [ ] **Step 1: Write the failing tests**

Create `/Users/robin/code/github/robinbowes/go-udap/cli/output_test.go`:

```go
package cli

import (
	"bytes"
	"strings"
	"testing"

	"go-udap/udap"
)

func TestFormatParamMapSortsByKey(t *testing.T) {
	var buf bytes.Buffer
	err := formatParamMap(&buf, map[string]string{
		"hostname":    "foo",
		"lan_ip_mode": "1",
		"interface":   "0",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := buf.String()
	want := "hostname=foo\ninterface=0\nlan_ip_mode=1\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatGetSingleValueIsBare(t *testing.T) {
	var buf bytes.Buffer
	err := formatGetResult(&buf, []string{"lan_ip_mode"}, map[string]string{"lan_ip_mode": "1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.String() != "1\n" {
		t.Errorf("got %q, want %q", buf.String(), "1\n")
	}
}

func TestFormatGetMultipleValuesIsKeyEqValue(t *testing.T) {
	var buf bytes.Buffer
	err := formatGetResult(&buf, []string{"lan_ip_mode", "hostname"}, map[string]string{
		"lan_ip_mode": "1",
		"hostname":    "foo",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "lan_ip_mode=1\n") || !strings.Contains(got, "hostname=foo\n") {
		t.Errorf("got %q, want both param=value lines", got)
	}
}

func TestFormatDeviceInfoLines(t *testing.T) {
	var buf bytes.Buffer
	d := &udap.Device{
		MAC:      "aa:bb:cc:dd:ee:ff",
		Name:     "Receiver",
		Model:    "SBR",
		Firmware: "77",
		IP:       "192.168.1.50",
		UUID:     "1234abcd-...",
	}
	formatDeviceInfo(&buf, d)
	got := buf.String()
	for _, want := range []string{"aa:bb:cc:dd:ee:ff", "Receiver", "SBR", "77", "192.168.1.50", "1234abcd-..."} {
		if !strings.Contains(got, want) {
			t.Errorf("info output missing %q; got %q", want, got)
		}
	}
}
```

- [ ] **Step 2: Run tests, verify they fail**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go test ./cli/ -run "TestFormat" -v`
Expected: FAIL — `undefined: formatParamMap`, `undefined: formatGetResult`, `undefined: formatDeviceInfo`.

- [ ] **Step 3: Create the implementation**

Create `/Users/robin/code/github/robinbowes/go-udap/cli/output.go`:

```go
package cli

import (
	"fmt"
	"io"
	"sort"

	"go-udap/udap"
)

// formatParamMap writes "key=value\n" lines to w, sorted by key.
// Used by `read` and multi-param `get`.
func formatParamMap(w io.Writer, m map[string]string) error {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if _, err := fmt.Fprintf(w, "%s=%s\n", k, m[k]); err != nil {
			return err
		}
	}
	return nil
}

// formatGetResult writes the result of a `get` command. Single-param requests
// produce a bare value (one line, no key=); multi-param requests produce
// key=value lines (one per requested param, in request order).
func formatGetResult(w io.Writer, requested []string, values map[string]string) error {
	if len(requested) == 1 {
		_, err := fmt.Fprintf(w, "%s\n", values[requested[0]])
		return err
	}
	for _, k := range requested {
		if _, err := fmt.Fprintf(w, "%s=%s\n", k, values[k]); err != nil {
			return err
		}
	}
	return nil
}

// formatDeviceInfo writes a multi-line metadata block for one device.
// Used by `info` and by `discover --info`.
func formatDeviceInfo(w io.Writer, d *udap.Device) {
	fmt.Fprintf(w, "MAC:      %s\n", d.MAC)
	fmt.Fprintf(w, "Name:     %s\n", d.Name)
	fmt.Fprintf(w, "Model:    %s\n", d.Model)
	fmt.Fprintf(w, "Firmware: %s\n", d.Firmware)
	fmt.Fprintf(w, "IP:       %s\n", d.IP)
	fmt.Fprintf(w, "UUID:     %s\n", d.UUID)
}
```

- [ ] **Step 4: Run tests, verify they pass**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go test ./cli/ -run "TestFormat" -v`
Expected: PASS for all four `TestFormat*` tests.

- [ ] **Step 5: Commit**

```bash
cd /Users/robin/code/github/robinbowes/go-udap
git add cli/output.go cli/output_test.go
git commit -m "feat(cli): add output formatting helpers"
```

---

## Task 8: Dispatcher, global flags, exit codes, and stub subcommands

This task wires up the entry point so `main.go` can call `cli.Run(args)`. Subcommands themselves are stubbed; later tasks fill them in.

**Files:**
- Create: `cli/cli.go`
- Create: `cli/cli_test.go`

- [ ] **Step 1: Write the failing tests**

Create `/Users/robin/code/github/robinbowes/go-udap/cli/cli_test.go`:

```go
package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestRunPrintsHelpWithNoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run(nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Errorf("expected usage on stdout, got %q", stdout.String())
	}
}

func TestRunUnknownCommandIsExitCode1(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run([]string{"flooble"}, &stdout, &stderr)
	if err == nil {
		t.Fatalf("expected error for unknown command")
	}
	var ee *ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected *ExitError, got %T", err)
	}
	if ee.Code != 1 {
		t.Errorf("want exit code 1, got %d", ee.Code)
	}
}

func TestRunVersionFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run([]string{"--version"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(stdout.String(), "go-udap") {
		t.Errorf("expected version line on stdout, got %q", stdout.String())
	}
}

func TestExitCodeReturnsZeroForNonExitError(t *testing.T) {
	if got := ExitCode(errors.New("plain")); got != 2 {
		t.Errorf("plain error should map to exit 2, got %d", got)
	}
	if got := ExitCode(nil); got != 0 {
		t.Errorf("nil error should map to exit 0, got %d", got)
	}
	if got := ExitCode(&ExitError{Code: 7}); got != 7 {
		t.Errorf("ExitError should preserve code, got %d", got)
	}
}
```

- [ ] **Step 2: Run tests, verify they fail**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go test ./cli/ -run "TestRun|TestExitCode" -v`
Expected: FAIL — `undefined: Run`, `undefined: ExitError`, `undefined: ExitCode`.

- [ ] **Step 3: Create the implementation**

Create `/Users/robin/code/github/robinbowes/go-udap/cli/cli.go`:

```go
package cli

import (
	"errors"
	"fmt"
	"io"
	"time"
)

// Version is the binary version string, surfaced by --version.
// Updated manually for now; release tooling can wire this to the git tag later.
const Version = "0.2.0"

// ExitError carries a process exit code alongside a message.
// Use it from subcommand handlers to control go-udap's exit status.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error { return e.Err }

// ExitCode maps an error to a process exit code:
//   - nil           → 0
//   - *ExitError    → ee.Code
//   - any other err → 2 (operation failure)
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var ee *ExitError
	if errors.As(err, &ee) {
		return ee.Code
	}
	return 2
}

// globalFlags holds values that apply to every subcommand.
type globalFlags struct {
	timeout time.Duration
	verbose bool
}

// Run parses the given arguments and dispatches to the appropriate subcommand.
// stdout receives all command output; stderr receives logs and warnings.
func Run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stdout)
		return nil
	}

	switch args[0] {
	case "-h", "--help", "help":
		printUsage(stdout)
		return nil
	case "--version":
		fmt.Fprintf(stdout, "go-udap %s\n", Version)
		return nil
	}

	cmd := args[0]
	subArgs := args[1:]

	switch cmd {
	case "discover":
		return runDiscover(subArgs, stdout, stderr)
	case "info":
		return runInfo(subArgs, stdout, stderr)
	case "read":
		return runRead(subArgs, stdout, stderr)
	case "get":
		return runGet(subArgs, stdout, stderr)
	case "set":
		return runSet(subArgs, stdout, stderr)
	case "save":
		return runSave(subArgs, stdout, stderr)
	case "reset":
		return runReset(subArgs, stdout, stderr)
	case "commit":
		return runCommit(subArgs, stdout, stderr)
	default:
		return &ExitError{Code: 1, Err: fmt.Errorf("unknown command: %s", cmd)}
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, `go-udap — Squeezebox UDAP configuration tool

Usage:
  go-udap [global flags] <command> [args] [flags]

Commands:
  discover [--info]              Discover devices on the network
  info <mac>                     Show metadata for one device
  read <mac>                     Read all parameters from a device
  get <mac> <param> [<param>...] Read specific parameters
  set <mac> [--config FILE] [--<param> VALUE ...]
                                 Set parameters from any combination of
                                 --config FILE (or --config - for stdin),
                                 piped stdin, and per-param --flags.
  save <mac>                     Save current config to NVRAM
  reset <mac>                    Reboot the device
  commit <mac>                   Save then reset

Global flags:
  --timeout DURATION  Operation timeout (default 5s)
  --verbose, -v       Debug logging to stderr
  --version           Print version and exit
  --help, -h          Print this help`)
}
```

Now also create stub subcommand files so the package builds. Create `/Users/robin/code/github/robinbowes/go-udap/cli/stubs.go`:

```go
package cli

import (
	"fmt"
	"io"
)

// These stubs are replaced one per task in the following tasks.

func runDiscover(args []string, stdout, stderr io.Writer) error {
	return notImplemented("discover")
}
func runInfo(args []string, stdout, stderr io.Writer) error {
	return notImplemented("info")
}
func runRead(args []string, stdout, stderr io.Writer) error {
	return notImplemented("read")
}
func runGet(args []string, stdout, stderr io.Writer) error {
	return notImplemented("get")
}
func runSet(args []string, stdout, stderr io.Writer) error {
	return notImplemented("set")
}
func runSave(args []string, stdout, stderr io.Writer) error {
	return notImplemented("save")
}
func runReset(args []string, stdout, stderr io.Writer) error {
	return notImplemented("reset")
}
func runCommit(args []string, stdout, stderr io.Writer) error {
	return notImplemented("commit")
}

func notImplemented(name string) error {
	return &ExitError{Code: 2, Err: fmt.Errorf("%s: not implemented yet", name)}
}
```

- [ ] **Step 4: Run tests, verify they pass**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go test ./cli/ -run "TestRun|TestExitCode" -v`
Expected: PASS for all four tests.

- [ ] **Step 5: Run the entire cli test suite to confirm nothing regressed**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go test ./cli/...`
Expected: PASS, no failures.

- [ ] **Step 6: Commit**

```bash
cd /Users/robin/code/github/robinbowes/go-udap
git add cli/cli.go cli/cli_test.go cli/stubs.go
git commit -m "feat(cli): add dispatcher, global flags, exit codes"
```

---

## Task 9: Shared "discover and find device" helper

Every targeted subcommand (`info`, `read`, `get`, `set`, `save`, `reset`, `commit`) starts with a discovery, then locates the requested MAC. Extract this into one helper.

**Files:**
- Create: `cli/find.go`

Networking is exercised against a real device; the helper itself is small enough that an integration smoke test is sufficient (covered when subcommand tasks run end-to-end against hardware). For unit tests we cover the MAC normalization piece only, since it's pure logic.

- [ ] **Step 1: Write the failing test**

Create `/Users/robin/code/github/robinbowes/go-udap/cli/find_test.go`:

```go
package cli

import "testing"

func TestNormalizeMAC(t *testing.T) {
	cases := []struct {
		in, want string
		ok       bool
	}{
		{"AA:BB:CC:DD:EE:FF", "aa:bb:cc:dd:ee:ff", true},
		{"aa:bb:cc:dd:ee:ff", "aa:bb:cc:dd:ee:ff", true},
		{"aa-bb-cc-dd-ee-ff", "aa:bb:cc:dd:ee:ff", true},
		{"aabbccddeeff", "aa:bb:cc:dd:ee:ff", true},
		{"not a mac", "", false},
		{"aa:bb:cc:dd:ee", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, err := normalizeMAC(c.in)
		if (err == nil) != c.ok {
			t.Errorf("normalizeMAC(%q): ok=%v, err=%v", c.in, c.ok, err)
			continue
		}
		if got != c.want {
			t.Errorf("normalizeMAC(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run the test, verify it fails**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go test ./cli/ -run TestNormalizeMAC -v`
Expected: FAIL — `undefined: normalizeMAC`.

- [ ] **Step 3: Create the implementation**

Create `/Users/robin/code/github/robinbowes/go-udap/cli/find.go`:

```go
package cli

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go-udap/udap"
)

var macColons = regexp.MustCompile(`^[0-9a-f]{2}(:[0-9a-f]{2}){5}$`)
var macHex = regexp.MustCompile(`^[0-9a-f]{12}$`)

// normalizeMAC accepts MAC addresses written with colons, hyphens, or
// no separators (any case) and returns lowercase colon-separated form.
// Returns an error if the input is not a recognizable MAC.
func normalizeMAC(in string) (string, error) {
	if in == "" {
		return "", fmt.Errorf("empty MAC address")
	}
	lower := strings.ToLower(in)
	withColons := strings.ReplaceAll(lower, "-", ":")
	if macColons.MatchString(withColons) {
		return withColons, nil
	}
	noSep := strings.ReplaceAll(strings.ReplaceAll(lower, ":", ""), "-", "")
	if macHex.MatchString(noSep) {
		var out strings.Builder
		for i := 0; i < 12; i += 2 {
			if i > 0 {
				out.WriteByte(':')
			}
			out.WriteString(noSep[i : i+2])
		}
		return out.String(), nil
	}
	return "", fmt.Errorf("invalid MAC address: %q", in)
}

// discoverAndFind broadcasts a discovery, waits up to timeout, and returns
// the device whose MAC matches. Returns an *ExitError with code 2 if not
// found within the timeout.
func discoverAndFind(client *udap.Client, mac string, timeout time.Duration) (*udap.Device, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := client.DiscoverDevicesWithContext(ctx); err != nil {
		return nil, &ExitError{Code: 2, Err: fmt.Errorf("discovery failed: %w", err)}
	}
	if d := client.GetDevice(mac); d != nil {
		return d, nil
	}
	return nil, &ExitError{Code: 2, Err: fmt.Errorf("device %s not found within %s", mac, timeout)}
}
```

- [ ] **Step 4: Run the test, verify it passes**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go test ./cli/ -run TestNormalizeMAC -v`
Expected: PASS.

- [ ] **Step 5: Verify the package still builds**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go build ./cli/...`
Expected: exit code 0, no output.

- [ ] **Step 6: Commit**

```bash
cd /Users/robin/code/github/robinbowes/go-udap
git add cli/find.go cli/find_test.go
git commit -m "feat(cli): add MAC normalization and device-find helper"
```

---

## Task 10: Implement `discover` subcommand

**Files:**
- Modify: `cli/stubs.go` (remove `runDiscover` stub)
- Create: `cli/discover.go`

- [ ] **Step 1: Write the implementation**

Create `/Users/robin/code/github/robinbowes/go-udap/cli/discover.go`:

```go
package cli

import (
	"context"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/spf13/pflag"

	"go-udap/udap"
)

func runDiscover(args []string, stdout, stderr io.Writer) error {
	fs := pflag.NewFlagSet("discover", pflag.ContinueOnError)
	fs.SetOutput(stderr)
	timeout := fs.Duration("timeout", 5*time.Second, "Discovery timeout")
	verbose := fs.BoolP("verbose", "v", false, "Debug logging to stderr")
	info := fs.Bool("info", false, "Also print metadata per device")
	if err := fs.Parse(args); err != nil {
		return &ExitError{Code: 1, Err: err}
	}

	client, err := newClient(*verbose)
	if err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := client.DiscoverDevicesWithContext(ctx); err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("discovery failed: %w", err)}
	}

	devices := client.ListDevices()
	sort.Slice(devices, func(i, j int) bool { return devices[i].MAC < devices[j].MAC })

	for i, d := range devices {
		if *info {
			if i > 0 {
				fmt.Fprintln(stdout)
			}
			formatDeviceInfo(stdout, d)
		} else {
			fmt.Fprintln(stdout, d.MAC)
		}
	}
	return nil
}

// newClient constructs a udap.Client; verbose controls log level.
func newClient(verbose bool) (*udap.Client, error) {
	logger := udap.NewStructuredLogger()
	if verbose {
		logger.SetLevel(udap.LogLevelDebug)
	} else {
		logger.SetLevel(udap.LogLevelWarn)
	}
	return udap.NewClientWithLogger(logger)
}
```

- [ ] **Step 2: Remove the stub**

Edit `cli/stubs.go`. Delete the `runDiscover` function (the four lines beginning `func runDiscover(...)` through the closing `}`).

- [ ] **Step 3: Verify the package builds**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go build ./cli/...`
Expected: exit code 0, no output.

- [ ] **Step 4: Verify all existing tests still pass**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go test ./cli/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /Users/robin/code/github/robinbowes/go-udap
git add cli/discover.go cli/stubs.go
git commit -m "feat(cli): implement discover subcommand"
```

---

## Task 11: Implement `info` subcommand

**Files:**
- Create: `cli/info.go`
- Modify: `cli/stubs.go` (remove `runInfo` stub)

- [ ] **Step 1: Write the implementation**

Create `/Users/robin/code/github/robinbowes/go-udap/cli/info.go`:

```go
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

	client, err := newClient(*verbose)
	if err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	defer client.Close()

	device, err := discoverAndFind(client, mac, *timeout)
	if err != nil {
		return err
	}
	formatDeviceInfo(stdout, device)
	return nil
}
```

- [ ] **Step 2: Remove the stub**

Edit `cli/stubs.go` and delete the `runInfo` function.

- [ ] **Step 3: Verify build**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go build ./cli/...`
Expected: exit code 0.

- [ ] **Step 4: Verify all tests still pass**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go test ./cli/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /Users/robin/code/github/robinbowes/go-udap
git add cli/info.go cli/stubs.go
git commit -m "feat(cli): implement info subcommand"
```

---

## Task 12: Implement `read` subcommand

**Files:**
- Create: `cli/read.go`
- Modify: `cli/stubs.go` (remove `runRead`)

- [ ] **Step 1: Write the implementation**

Create `/Users/robin/code/github/robinbowes/go-udap/cli/read.go`:

```go
package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/spf13/pflag"
)

func runRead(args []string, stdout, stderr io.Writer) error {
	fs := pflag.NewFlagSet("read", pflag.ContinueOnError)
	fs.SetOutput(stderr)
	timeout := fs.Duration("timeout", 5*time.Second, "Operation timeout")
	verbose := fs.BoolP("verbose", "v", false, "Debug logging to stderr")
	if err := fs.Parse(args); err != nil {
		return &ExitError{Code: 1, Err: err}
	}
	if fs.NArg() != 1 {
		return &ExitError{Code: 1, Err: fmt.Errorf("read: expected exactly one MAC argument")}
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

	device, err := discoverAndFind(client, mac, *timeout)
	if err != nil {
		return err
	}
	if err := client.GetAllDeviceConfig(device); err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("read failed: %w", err)}
	}
	if err := formatParamMap(stdout, device.Parameters); err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	return nil
}
```

- [ ] **Step 2: Remove the stub**

Edit `cli/stubs.go` and delete the `runRead` function.

- [ ] **Step 3: Verify build and tests**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go build ./cli/... && go test ./cli/...`
Expected: build succeeds, tests pass.

- [ ] **Step 4: Commit**

```bash
cd /Users/robin/code/github/robinbowes/go-udap
git add cli/read.go cli/stubs.go
git commit -m "feat(cli): implement read subcommand"
```

---

## Task 13: Implement `get` subcommand

**Files:**
- Create: `cli/get.go`
- Modify: `cli/stubs.go` (remove `runGet`)

- [ ] **Step 1: Write the implementation**

Create `/Users/robin/code/github/robinbowes/go-udap/cli/get.go`:

```go
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
	timeout := fs.Duration("timeout", 5*time.Second, "Operation timeout")
	verbose := fs.BoolP("verbose", "v", false, "Debug logging to stderr")
	if err := fs.Parse(args); err != nil {
		return &ExitError{Code: 1, Err: err}
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
		if _, ok := udap.ConfigSettings[p]; !ok {
			return &ExitError{Code: 1, Err: fmt.Errorf("get: unknown parameter %q", p)}
		}
	}

	client, err := newClient(*verbose)
	if err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	defer client.Close()

	device, err := discoverAndFind(client, mac, *timeout)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	values, err := client.GetDeviceConfigWithContext(ctx, device, params)
	if err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("get failed: %w", err)}
	}
	if err := formatGetResult(stdout, params, values); err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	return nil
}
```

- [ ] **Step 2: Remove the stub**

Edit `cli/stubs.go` and delete the `runGet` function.

- [ ] **Step 3: Verify build and tests**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go build ./cli/... && go test ./cli/...`
Expected: success.

- [ ] **Step 4: Commit**

```bash
cd /Users/robin/code/github/robinbowes/go-udap
git add cli/get.go cli/stubs.go
git commit -m "feat(cli): implement get subcommand"
```

---

## Task 14: Implement `set` subcommand

This is the most involved subcommand: it builds the per-param flag table at parse time, detects piped stdin, optionally reads `--config FILE` (or `-` for stdin), merges sources, validates, then sends.

**Files:**
- Create: `cli/set.go`
- Modify: `cli/stubs.go` (remove `runSet`)

- [ ] **Step 1: Write the implementation**

Create `/Users/robin/code/github/robinbowes/go-udap/cli/set.go`:

```go
package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/pflag"
)

func runSet(args []string, stdout, stderr io.Writer) error {
	fs := pflag.NewFlagSet("set", pflag.ContinueOnError)
	fs.SetOutput(stderr)
	timeout := fs.Duration("timeout", 5*time.Second, "Operation timeout")
	verbose := fs.BoolP("verbose", "v", false, "Debug logging to stderr")
	configPath := fs.String("config", "", "Read parameters from FILE (use - for stdin)")

	// Register a string flag for every known UDAP parameter.
	pf := paramFlags()
	for _, p := range pf {
		fs.String(p.flagName, "", p.help)
	}

	if err := fs.Parse(args); err != nil {
		return &ExitError{Code: 1, Err: err}
	}
	if fs.NArg() != 1 {
		return &ExitError{Code: 1, Err: fmt.Errorf("set: expected exactly one MAC argument")}
	}
	mac, err := normalizeMAC(fs.Arg(0))
	if err != nil {
		return &ExitError{Code: 1, Err: err}
	}

	// Collect per-param flag values that were actually set.
	flagValues := make(map[string]string)
	for _, p := range pf {
		if !fs.Changed(p.flagName) {
			continue
		}
		v, err := fs.GetString(p.flagName)
		if err != nil {
			return &ExitError{Code: 1, Err: err}
		}
		flagValues[p.udapName] = v
	}

	// Resolve --config (file path, "-" for stdin, or unset).
	var fileContent io.Reader
	var fileLabel string
	switch {
	case *configPath == "-":
		fileContent = os.Stdin
		fileLabel = "-"
	case *configPath != "":
		f, err := os.Open(*configPath)
		if err != nil {
			return &ExitError{Code: 1, Err: fmt.Errorf("open config: %w", err)}
		}
		defer f.Close()
		fileContent = f
		fileLabel = *configPath
	}

	// Detect piped stdin (only consulted when no --config was given).
	stdinPiped := isStdinPiped()
	var stdinContent io.Reader
	if stdinPiped {
		stdinContent = os.Stdin
	}

	merged, err := mergeSources(sourceInputs{
		fileContent:  fileContent,
		fileLabel:    fileLabel,
		stdinContent: stdinContent,
		stdinPiped:   stdinPiped,
		flags:        flagValues,
	}, stderr)
	if err != nil {
		return &ExitError{Code: 1, Err: err}
	}

	client, err := newClient(*verbose)
	if err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	defer client.Close()

	device, err := discoverAndFind(client, mac, *timeout)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := client.SetDeviceConfigWithContext(ctx, device, merged); err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("set failed: %w", err)}
	}

	// Echo what we sent for confirmation, sorted.
	if err := formatParamMap(stdout, merged); err != nil {
		return &ExitError{Code: 2, Err: err}
	}
	return nil
}

// isStdinPiped returns true when stdin is not a terminal (i.e. data is piped
// or redirected from a file). False if stdin is interactive or unavailable.
func isStdinPiped() bool {
	st, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	if (st.Mode() & os.ModeCharDevice) != 0 {
		return false
	}
	return true
}
```

- [ ] **Step 2: Remove the stub**

Edit `cli/stubs.go` and delete the `runSet` function.

- [ ] **Step 3: Verify build and tests**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go build ./cli/... && go test ./cli/...`
Expected: success.

- [ ] **Step 4: Manual sanity-check the help**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go run . set --help 2>&1 | head -30`
Expected: stderr lists `--config`, `--lan-ip-mode`, `--wireless-ssid`, `--timeout`, etc.

- [ ] **Step 5: Commit**

```bash
cd /Users/robin/code/github/robinbowes/go-udap
git add cli/set.go cli/stubs.go
git commit -m "feat(cli): implement set subcommand with layered sources"
```

---

## Task 15: Implement `save`, `reset`, and `commit` subcommands

These three are structurally identical (parse args, find device, call one udap method); group them.

**Files:**
- Create: `cli/save.go`
- Create: `cli/reset.go`
- Create: `cli/commit.go`
- Modify: `cli/stubs.go` (remove the three stubs and the now-unused `notImplemented` if no stubs remain)

- [ ] **Step 1: Write `cli/save.go`**

Create `/Users/robin/code/github/robinbowes/go-udap/cli/save.go`:

```go
package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/pflag"
)

func runSave(args []string, stdout, stderr io.Writer) error {
	fs := pflag.NewFlagSet("save", pflag.ContinueOnError)
	fs.SetOutput(stderr)
	timeout := fs.Duration("timeout", 7*time.Second, "Operation timeout")
	verbose := fs.BoolP("verbose", "v", false, "Debug logging to stderr")
	if err := fs.Parse(args); err != nil {
		return &ExitError{Code: 1, Err: err}
	}
	if fs.NArg() != 1 {
		return &ExitError{Code: 1, Err: fmt.Errorf("save: expected exactly one MAC argument")}
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

	device, err := discoverAndFind(client, mac, *timeout)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := client.SaveDeviceConfigWithContext(ctx, device); err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("save failed: %w", err)}
	}
	return nil
}
```

- [ ] **Step 2: Write `cli/reset.go`**

Create `/Users/robin/code/github/robinbowes/go-udap/cli/reset.go`:

```go
package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/pflag"
)

func runReset(args []string, stdout, stderr io.Writer) error {
	fs := pflag.NewFlagSet("reset", pflag.ContinueOnError)
	fs.SetOutput(stderr)
	timeout := fs.Duration("timeout", 5*time.Second, "Operation timeout")
	verbose := fs.BoolP("verbose", "v", false, "Debug logging to stderr")
	if err := fs.Parse(args); err != nil {
		return &ExitError{Code: 1, Err: err}
	}
	if fs.NArg() != 1 {
		return &ExitError{Code: 1, Err: fmt.Errorf("reset: expected exactly one MAC argument")}
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

	device, err := discoverAndFind(client, mac, *timeout)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	if err := client.ResetDeviceWithContext(ctx, device); err != nil {
		return &ExitError{Code: 2, Err: fmt.Errorf("reset failed: %w", err)}
	}
	return nil
}
```

- [ ] **Step 3: Write `cli/commit.go`**

Create `/Users/robin/code/github/robinbowes/go-udap/cli/commit.go`:

```go
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
```

- [ ] **Step 4: Delete `cli/stubs.go`**

After this task, every subcommand has a real implementation, so the entire stubs file (including `notImplemented`) is unused. Delete it:

Run: `rm /Users/robin/code/github/robinbowes/go-udap/cli/stubs.go`

- [ ] **Step 5: Verify build and tests**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go build ./cli/... && go test ./cli/...`
Expected: success.

- [ ] **Step 6: Commit**

```bash
cd /Users/robin/code/github/robinbowes/go-udap
git add cli/
git commit -m "feat(cli): implement save, reset, and commit subcommands"
```

---

## Task 16: Replace `main.go` and remove readline

**Files:**
- Modify: `main.go` (gut and rewrite)
- Modify: `go.mod`, `go.sum` (run `go mod tidy`)

- [ ] **Step 1: Rewrite main.go**

Replace the entire contents of `/Users/robin/code/github/robinbowes/go-udap/main.go` with:

```go
package main

import (
	"fmt"
	"os"

	"go-udap/cli"
)

func main() {
	err := cli.Run(os.Args[1:], os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
	}
	os.Exit(cli.ExitCode(err))
}
```

- [ ] **Step 2: Run go mod tidy to drop readline**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go mod tidy`
Expected: stdout silent or shows removal of `chzyer/readline`.

- [ ] **Step 3: Verify chzyer/readline is gone from go.mod**

Run: `grep -c chzyer /Users/robin/code/github/robinbowes/go-udap/go.mod || true`
Expected: `0` (no matches).

- [ ] **Step 4: Build the binary**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go build -o go-udap .`
Expected: exit code 0; produces `./go-udap`.

- [ ] **Step 5: Smoke-test help and version**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && ./go-udap`
Expected: stdout shows the usage block; exit code 0.

Run: `cd /Users/robin/code/github/robinbowes/go-udap && ./go-udap --version`
Expected: stdout shows `go-udap 0.2.0`; exit code 0.

Run: `cd /Users/robin/code/github/robinbowes/go-udap && ./go-udap flooble; echo "exit=$?"`
Expected: stderr shows "error: unknown command: flooble"; `exit=1`.

- [ ] **Step 6: Run all tests one more time**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && go test ./...`
Expected: PASS for all tests.

- [ ] **Step 7: Commit**

```bash
cd /Users/robin/code/github/robinbowes/go-udap
git add main.go go.mod go.sum
git commit -m "refactor: replace interactive shell with CLI dispatcher"
```

---

## Task 17: Update Taskfile

**Files:**
- Modify: `Taskfile.yml` (`run` and `dev` tasks)

- [ ] **Step 1: Update `run` task**

Edit `Taskfile.yml` lines 97-101. Replace:

```yaml
  run:
    desc: Build and run the application
    cmds:
      - task: build
      - ./{{.BINARY_NAME}}
```

With:

```yaml
  run:
    desc: Build and print help (CLI is single-shot; pass args explicitly)
    cmds:
      - task: build
      - ./{{.BINARY_NAME}} --help
```

- [ ] **Step 2: Update `dev` task**

Edit `Taskfile.yml` lines 103-106. Replace:

```yaml
  dev:
    desc: Run without building (for development)
    cmds:
      - go run main.go
```

With:

```yaml
  dev:
    desc: Run without building; pass CLI args via -- (e.g. task dev -- discover)
    cmds:
      - go run . {{.CLI_ARGS}}
```

- [ ] **Step 3: Verify the tasks work**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && task run`
Expected: builds binary, prints usage to stdout.

Run: `cd /Users/robin/code/github/robinbowes/go-udap && task dev -- --version`
Expected: prints `go-udap 0.2.0`.

- [ ] **Step 4: Commit**

```bash
cd /Users/robin/code/github/robinbowes/go-udap
git add Taskfile.yml
git commit -m "chore: update run/dev tasks for CLI-first model"
```

---

## Task 18: Update README

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Rewrite the README**

Replace the entire contents of `/Users/robin/code/github/robinbowes/go-udap/README.md` with:

````markdown
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/robinbowes/go-udap)

# Squeezebox UDAP Configuration Tool

A command-line tool for discovering and configuring Squeezebox devices on your network using the UDAP (Universal Device Access Protocol).

## Overview

This tool allows you to:
- Discover Squeezebox devices on your local network
- Configure network settings (IP address, gateway, DNS)
- Configure wireless settings (SSID, WPA/WEP keys)
- Set the Logitech Media Server (LMS) address
- Save configuration to device persistent storage
- Reset devices to apply new configuration

The tool is single-shot and command-line driven; every operation is one
invocation. There is no interactive shell.

## Installation

### Pre-built Binaries

Download the latest release for your platform from the [Releases](https://github.com/robinbowes/go-udap/releases) page.

### Build from Source

```bash
git clone https://github.com/robinbowes/go-udap.git
cd go-udap
go build -o go-udap .
```

See [DEVELOPMENT.md](DEVELOPMENT.md) for detailed build instructions and cross-compilation.

## Usage

```
go-udap [global flags] <command> [args] [flags]
```

### Commands

| Command | Description |
|---------|-------------|
| `discover [--info]` | Discover devices; print MAC addresses (or full metadata with `--info`) |
| `info <mac>` | Show metadata for one device |
| `read <mac>` | Read all parameters from a device |
| `get <mac> <param> [<param>...]` | Read specific parameters |
| `set <mac> [--config FILE] [--<param> VALUE ...]` | Set parameters from any combination of `--config FILE` (or `--config -` for stdin), piped stdin, and per-param flags |
| `save <mac>` | Save current config to NVRAM |
| `reset <mac>` | Reboot the device |
| `commit <mac>` | Save then reset (combined) |

### Global flags

| Flag | Default | Purpose |
|---|---|---|
| `--timeout DURATION` | `5s` | Operation timeout |
| `--verbose, -v` | off | Debug logging to stderr |
| `--version` | — | Print version and exit |
| `--help, -h` | — | Print help |

### Output

Command output is on **stdout**; logs and warnings are on **stderr**. This
keeps stdout machine-parseable.

- `discover` — one MAC per line.
- `discover --info` — multi-line metadata block per device.
- `read` — `param=value` lines, sorted by name.
- `get <mac> <param>` (single) — bare value.
- `get <mac> <p1> <p2>` (multi) — `param=value` lines in request order.

### Examples

Discover devices on the LAN:

```bash
go-udap discover
# 00:04:20:16:06:02
```

Show full metadata:

```bash
go-udap discover --info
```

Read all parameters and save them as a backup:

```bash
go-udap read 00:04:20:16:06:02 > backup.conf
```

Configure a device for DHCP on wireless with WPA2:

```bash
go-udap set 00:04:20:16:06:02 \
  --interface 0 --lan-ip-mode 1 \
  --wireless-ssid SlimNet --wireless-wpa-on 1 --wireless-wpa-mode 2 \
  --wireless-wpa-psk 'shared-secret' \
  --server-address 192.168.1.250
go-udap commit 00:04:20:16:06:02
```

Apply a saved config file (and override one value at the CLI):

```bash
go-udap set 00:04:20:16:06:02 --config backup.conf --hostname new-name
go-udap commit 00:04:20:16:06:02
```

Pipe parameters from stdin (here-string or here-doc):

```bash
go-udap set 00:04:20:16:06:02 <<< "lan_ip_mode=1
wireless_SSID=foo"

go-udap set 00:04:20:16:06:02 <<EOF
interface=1
lan_ip_mode=0
lan_network_address=192.168.1.50
lan_subnet_mask=255.255.255.0
lan_gateway=192.168.1.1
EOF
```

Get a single value for use in a script:

```bash
ip=$(go-udap get 00:04:20:16:06:02 lan_network_address)
```

### Config file format

INI-style: one `key=value` per line; `#` and `;` start comments; blank
lines ignored. Keys must be canonical UDAP parameter names (e.g.
`wireless_SSID`, not `wireless-ssid`). The format matches `read` output, so
round-tripping works without conversion.

```ini
# Network
interface=1
lan_ip_mode=1

# Wireless
wireless_SSID=MyNet
wireless_wpa_on=1
wireless_wpa_psk=secret
```

## Configuration Parameters

### Network

| Parameter | Type | Description |
|-----------|------|-------------|
| `lan_ip_mode` | Integer (0-1) | 0 = Static IP, 1 = DHCP |
| `lan_network_address` | IPv4 | Static IP address |
| `lan_subnet_mask` | IPv4 | Subnet mask |
| `lan_gateway` | IPv4 | Default gateway |
| `primary_dns` | IPv4 | Primary DNS |
| `secondary_dns` | IPv4 | Secondary DNS |
| `hostname` | String (max 33) | Device hostname |
| `bridging` | Integer (0-1) | Enable/disable bridging |
| `interface` | Integer (0-1) | 0 = Wireless, 1 = Wired |

### Server

| Parameter | Type | Description |
|-----------|------|-------------|
| `server_address` | IPv4 | LMS address |
| `lms_address` | IPv4 | Alternative LMS address |
| `squeezecenter_address` | IPv4 | Alias for `server_address` |
| `slimserver_address` | IPv4 | Alias for `server_address` |

### Wireless

| Parameter | Type | Description |
|-----------|------|-------------|
| `wireless_mode` | Integer (0-1) | 0 = Infrastructure, 1 = Ad-hoc |
| `wireless_SSID` | String (1-32) | Network name |
| `wireless_channel` | Integer (1-13) | Channel |
| `wireless_region_id` | Integer | Region |

### Wireless security — WEP

| Parameter | Type | Description |
|-----------|------|-------------|
| `wireless_wep_on` | Integer (0-1) | Enable/disable WEP |
| `wireless_keylen` | Integer (5/13) | WEP key length |
| `wireless_wep_key` | String | Primary WEP key |
| `wireless_wep_key_1`..`_3` | String | WEP key slots 1-3 |

### Wireless security — WPA/WPA2

| Parameter | Type | Description |
|-----------|------|-------------|
| `wireless_wpa_on` | Integer (0-1) | Enable/disable WPA |
| `wireless_wpa_mode` | Integer | WPA mode |
| `wireless_wpa_cipher` | Integer | WPA cipher |
| `wireless_wpa_psk` | String (8-63) | WPA pre-shared key |

### Factory reset

Factory reset is **not** exposed by the protocol. Perform it on the device
itself: hold the front button for ~6 seconds until it blinks fast red.
See https://wiki.lyrion.org/index.php/SBRFrontButtonAndLED.

## Troubleshooting

### No devices found

- Ensure the device is powered on and on the same network segment.
- Devices in bootstrap mode (unconfigured) report IP `0.0.0.0` and are still
  reachable via broadcast.
- Make sure UDP port 17784 is not blocked by a firewall.

### Configuration not applying

- After `set`, run `save` to persist, then `reset` to reboot.
- Or use `commit` to save and reset in one step.

### Permission errors

- Binding to UDP 17784 typically does not require root, but on some platforms
  you may need elevated privileges if the port is otherwise restricted.

## License

MIT License — see [LICENSE](LICENSE) for details.

You must retain the copyright notice and license in any copies or substantial
portions of the software.
````

- [ ] **Step 2: Commit**

```bash
cd /Users/robin/code/github/robinbowes/go-udap
git add README.md
git commit -m "docs: rewrite README for CLI-first usage"
```

---

## Task 19: Update CLAUDE.md and DEVELOPMENT.md

**Files:**
- Modify: `CLAUDE.md` (the "CLI Commands" section)
- Modify: `DEVELOPMENT.md` (any shell-specific text)

- [ ] **Step 1: Update CLAUDE.md "CLI Commands" section**

Edit `/Users/robin/code/github/robinbowes/go-udap/CLAUDE.md`. Find the section starting `## CLI Commands (when running the tool)` and replace it (and its bulleted list) with:

```markdown
## CLI Commands (when running the tool)

The tool is single-shot CLI; every operation is one invocation. There is no
interactive shell.

- `go-udap discover [--info]` — Discover devices; MACs only, or full metadata with `--info`
- `go-udap info <mac>` — Show metadata for one device
- `go-udap read <mac>` — Read all parameters from a device
- `go-udap get <mac> <param> [<param>...]` — Read specific parameters
- `go-udap set <mac> [--config FILE] [--<param> VALUE ...]` — Set parameters from file, piped stdin, and/or per-param flags (CLI flags win)
- `go-udap save <mac>` — Save current config to NVRAM
- `go-udap reset <mac>` — Reboot the device
- `go-udap commit <mac>` — Save then reset

Global flags: `--timeout DURATION` (default 5s), `--verbose`/`-v`, `--version`, `--help`/`-h`.

Output is on stdout; logs and warnings on stderr. Exit codes: 0 success,
1 usage error, 2 operation failure.
```

- [ ] **Step 2: Scan DEVELOPMENT.md**

Run: `grep -n -i "shell\|readline\|interactive\|REPL" /Users/robin/code/github/robinbowes/go-udap/DEVELOPMENT.md || true`
If the grep returns matches, edit each line to remove or rephrase the shell-specific content. If it returns nothing, no changes needed.

- [ ] **Step 3: Commit**

```bash
cd /Users/robin/code/github/robinbowes/go-udap
git add CLAUDE.md DEVELOPMENT.md
git commit -m "docs: update CLAUDE.md and DEVELOPMENT.md for CLI-first model"
```

(If DEVELOPMENT.md was unchanged, drop it from `git add` — the commit will only contain CLAUDE.md.)

---

## Task 20: Final verification

**Files:** none modified; this task only runs verification commands.

- [ ] **Step 1: Format check**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && task fmt`
Expected: no output, exit code 0.

- [ ] **Step 2: Lint (go vet)**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && task lint`
Expected: no output, exit code 0.

- [ ] **Step 3: Test suite**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && task test`
Expected: all `cli/` and `udap/` tests pass.

- [ ] **Step 4: Build for current platform**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && task build`
Expected: produces `./go-udap`, no errors.

- [ ] **Step 5: Build for all platforms**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && task build:all`
Expected: produces all four platform binaries with no errors.

- [ ] **Step 6: Smoke-test the binary**

Run: `cd /Users/robin/code/github/robinbowes/go-udap && ./go-udap --help` — usage on stdout
Run: `cd /Users/robin/code/github/robinbowes/go-udap && ./go-udap --version` — `go-udap 0.2.0`
Run: `cd /Users/robin/code/github/robinbowes/go-udap && ./go-udap set --help 2>&1 | grep -c -- '--'` — should report >= 27 flags (25 params + --config + --timeout + --verbose; pflag may add more)

- [ ] **Step 7: Confirm chzyer/readline is fully gone**

Run: `grep -r chzyer /Users/robin/code/github/robinbowes/go-udap/go.mod /Users/robin/code/github/robinbowes/go-udap/go.sum || echo "clean"`
Expected: `clean`.

- [ ] **Step 8: Check git status is clean**

Run: `git -C /Users/robin/code/github/robinbowes/go-udap status --short`
Expected: empty output (or only untracked build artifacts like `go-udap`, `go-udap.exe`, `go-udap-linux-*`).

- [ ] **Step 9: Hardware validation (manual, if a real SBR is available)**

Run against a real Squeezebox Receiver on the network:

```bash
./go-udap discover
./go-udap discover --info
./go-udap info <mac-from-discover>
./go-udap read <mac>
./go-udap get <mac> hostname
./go-udap get <mac> hostname interface
./go-udap set <mac> --hostname go-udap-test
./go-udap commit <mac>
```

Verify each command produces the expected output and the device responds.

- [ ] **Step 10: Push the branch and open a PR**

(Only when everything above passes. Confirm with the user before opening the PR.)

```bash
git -C /Users/robin/code/github/robinbowes/go-udap push -u origin robin/cli-redesign
gh pr create --title "Replace interactive shell with single-shot CLI" --body "$(cat <<'EOF'
## Summary

- Replaces the readline-based interactive shell with single-shot subcommands (`discover`, `info`, `read`, `get`, `set`, `save`, `reset`, `commit`).
- Adds GNU-style `--<param>` flags for all 25 known UDAP parameters via `spf13/pflag`.
- `set` accepts layered sources: `--config FILE` (or `--config -`), piped stdin (auto-detected), and per-param CLI flags. CLI flags win.
- Drops the `chzyer/readline` dependency.

See `docs/superpowers/specs/2026-05-07-cli-redesign-design.md` for the full design.

## Test plan

- [ ] `go test ./...` passes
- [ ] `task build:all` succeeds
- [ ] Manual: `discover`, `info`, `read`, `get`, `set`, `commit` against a real Squeezebox Receiver

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

---

## Plan self-review notes

- **Spec coverage:** every section of the spec has at least one task — global flags & dispatcher (Task 8), exit codes (Task 8), I/O streams (Task 2 logger + Task 8 stdout/stderr), commands (Tasks 10-15), targeting/discovery preamble (Task 9 + each subcommand), source layering (Task 6 + Task 14), INI format (Task 4), per-param flags (Task 5 + Task 14), file layout (Tasks 4-15 in aggregate), removed/added deps (Tasks 1, 16), udap unchanged (verified by Task 20 step 3 running existing udap tests), supporting files (Tasks 17-19).
- **Type consistency:** `paramFlag`/`paramFlags()` defined in Task 5, used in Task 14. `sourceInputs`/`mergeSources` defined in Task 6, used in Task 14. `ExitError`/`ExitCode`/`Run` defined in Task 8, used by every subcommand task and Task 16. `normalizeMAC`/`discoverAndFind` defined in Task 9, used by Tasks 11-15. `formatParamMap`/`formatGetResult`/`formatDeviceInfo` defined in Task 7, used by Tasks 10-13. `newClient` defined in Task 10, used by Tasks 11-15. `udap.ValidateParameter` added in Task 3, used in Task 4. Logger destination changed in Task 2, relied on implicitly by every subcommand task.
- **No placeholders:** every implementation step contains complete code; every test step includes the test code and the run command with expected output.
