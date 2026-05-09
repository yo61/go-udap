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

func TestMoveGlobalFlagsAfterSubcommand(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "no flags",
			in:   []string{"read", "aa:bb:cc:dd:ee:ff"},
			want: []string{"read", "aa:bb:cc:dd:ee:ff"},
		},
		{
			name: "leading -v",
			in:   []string{"-v", "read", "aa:bb:cc:dd:ee:ff"},
			want: []string{"read", "-v", "aa:bb:cc:dd:ee:ff"},
		},
		{
			name: "leading --verbose",
			in:   []string{"--verbose", "read", "aa:bb:cc:dd:ee:ff"},
			want: []string{"read", "--verbose", "aa:bb:cc:dd:ee:ff"},
		},
		{
			name: "leading --timeout with separate value",
			in:   []string{"--timeout", "30s", "read", "aa:bb:cc:dd:ee:ff"},
			want: []string{"read", "--timeout", "30s", "aa:bb:cc:dd:ee:ff"},
		},
		{
			name: "leading --timeout=value",
			in:   []string{"--timeout=30s", "read", "aa:bb:cc:dd:ee:ff"},
			want: []string{"read", "--timeout=30s", "aa:bb:cc:dd:ee:ff"},
		},
		{
			name: "multiple leading flags",
			in:   []string{"-v", "--timeout", "30s", "read", "aa:bb:cc:dd:ee:ff"},
			want: []string{"read", "-v", "--timeout", "30s", "aa:bb:cc:dd:ee:ff"},
		},
		{
			name: "flags after subcommand stay put",
			in:   []string{"read", "-v", "aa:bb:cc:dd:ee:ff"},
			want: []string{"read", "-v", "aa:bb:cc:dd:ee:ff"},
		},
		{
			name: "unknown leading flag halts hoisting",
			in:   []string{"--frobnicate", "set", "aa:bb:cc:dd:ee:ff"},
			want: []string{"--frobnicate", "set", "aa:bb:cc:dd:ee:ff"},
		},
		{
			name: "no subcommand, only flag",
			in:   []string{"-v"},
			want: []string{"-v"},
		},
		{
			name: "leading -- terminator: args returned unchanged",
			in:   []string{"--", "read", "aa:bb:cc:dd:ee:ff"},
			want: []string{"--", "read", "aa:bb:cc:dd:ee:ff"},
		},
		{
			name: "leading global flag then -- terminator: no hoist",
			in:   []string{"-v", "--", "read", "aa:bb:cc:dd:ee:ff"},
			want: []string{"-v", "--", "read", "aa:bb:cc:dd:ee:ff"},
		},
		{
			name: "leading --timeout value then -- terminator: no hoist",
			in:   []string{"--timeout", "5s", "--", "read", "aa:bb:cc:dd:ee:ff"},
			want: []string{"--timeout", "5s", "--", "read", "aa:bb:cc:dd:ee:ff"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := moveGlobalFlagsAfterSubcommand(c.in)
			if len(got) != len(c.want) {
				t.Fatalf("len: got %d, want %d (%v vs %v)", len(got), len(c.want), got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("[%d]: got %q, want %q (full got=%v)", i, got[i], c.want[i], got)
				}
			}
		})
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
