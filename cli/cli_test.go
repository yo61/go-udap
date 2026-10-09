package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestExecutePrintsHelpWithNoArgs(t *testing.T) {
	t.Cleanup(resetFlagsForTesting)
	var stdout, stderr bytes.Buffer
	err := Execute(nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// Cobra's default --help output starts with the Long description (or
	// Short if no Long). Either way, "Usage:" appears somewhere — check
	// for it on either stream.
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "Usage:") {
		t.Errorf("expected usage in output, got stdout=%q stderr=%q",
			stdout.String(), stderr.String())
	}
}

func TestExecuteUnknownCommandIsUsageError(t *testing.T) {
	t.Cleanup(resetFlagsForTesting)
	var stdout, stderr bytes.Buffer
	err := Execute([]string{"flooble"}, &stdout, &stderr)
	if err == nil {
		t.Fatalf("expected error for unknown command")
	}
	if ExitCode(err) != 2 {
		t.Errorf("exit code %d, want 2 (usage error)", ExitCode(err))
	}
	if !strings.Contains(err.Error(), "flooble") {
		t.Errorf("expected error to mention %q, got %q", "flooble", err.Error())
	}
}

func TestExecuteVersionFlag(t *testing.T) {
	t.Cleanup(resetFlagsForTesting)
	var stdout, stderr bytes.Buffer
	err := Execute([]string{"--version"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(stdout.String(), "go-udap") {
		t.Errorf("expected version line on stdout, got %q", stdout.String())
	}
}

func TestVersionVariableIsOverridable(t *testing.T) {
	t.Cleanup(resetFlagsForTesting)
	original := Version
	t.Cleanup(func() { Version = original })
	Version = "test-1.2.3"

	var stdout, stderr bytes.Buffer
	if err := Execute([]string{"--version"}, &stdout, &stderr); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(stdout.String(), "test-1.2.3") {
		t.Errorf("expected version output to contain 'test-1.2.3', got %q",
			stdout.String())
	}
}

func TestExitCodeMapsErrors(t *testing.T) {
	if got := ExitCode(errors.New("plain")); got != 1 {
		t.Errorf("unclassified error should map to exit 1 (operation failure), got %d", got)
	}
	if got := ExitCode(nil); got != 0 {
		t.Errorf("nil error should map to exit 0, got %d", got)
	}
	if got := ExitCode(&ExitError{Code: 7}); got != 7 {
		t.Errorf("ExitError should preserve code, got %d", got)
	}
}

// TestRootSubcommandsHaveLongDescriptions guards against new subcommands
// shipping a bare DESCRIPTION section in the generated man pages. cmd/docs
// runs cobra/doc.GenManTree which falls back to Short when Long is empty,
// producing a one-line man page DESCRIPTION. Every visible subcommand
// should set Long.
func TestRootSubcommandsHaveLongDescriptions(t *testing.T) {
	for _, sub := range Root().Commands() {
		if sub.Hidden || sub.Name() == "help" || sub.Name() == "completion" {
			continue
		}
		if strings.TrimSpace(sub.Long) == "" {
			t.Errorf("subcommand %q has no Long description; man page will fall back to Short",
				sub.Name())
		}
	}
}
