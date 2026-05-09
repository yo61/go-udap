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
