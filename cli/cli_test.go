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
