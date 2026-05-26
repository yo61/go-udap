package cli

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestCompletionStatePath_ScopedByPPID(t *testing.T) {
	got := completionStatePath()
	ppid := strconv.Itoa(os.Getppid())
	if !strings.Contains(got, ppid) {
		t.Errorf("completionStatePath() = %q, want contains PPID %q", got, ppid)
	}
}

func TestNextMACTimeout_ColdWhenNoState(t *testing.T) {
	dir := t.TempDir()
	prev := completionStateDir
	completionStateDir = func() string { return dir }
	t.Cleanup(func() { completionStateDir = prev })

	got := nextMACTimeout()
	if got != 500*time.Millisecond {
		t.Errorf("nextMACTimeout() = %v, want 500ms (cold)", got)
	}
}

func TestNextMACTimeout_ColdWhenStateStale(t *testing.T) {
	dir := t.TempDir()
	prev := completionStateDir
	completionStateDir = func() string { return dir }
	t.Cleanup(func() { completionStateDir = prev })

	path := completionStatePath()
	if err := os.WriteFile(path, []byte("0"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-15 * time.Second)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}

	got := nextMACTimeout()
	if got != 500*time.Millisecond {
		t.Errorf("nextMACTimeout() = %v, want 500ms (stale)", got)
	}
}

func TestNextMACTimeout_RetryWhenRecentEmpty(t *testing.T) {
	dir := t.TempDir()
	prev := completionStateDir
	completionStateDir = func() string { return dir }
	t.Cleanup(func() { completionStateDir = prev })

	path := completionStatePath()
	if err := os.WriteFile(path, []byte("0"), 0o600); err != nil {
		t.Fatal(err)
	}

	got := nextMACTimeout()
	if got != defaultTimeout {
		t.Errorf("nextMACTimeout() = %v, want defaultTimeout %v (retry tier)", got, defaultTimeout)
	}
}

func TestNextMACTimeout_ColdWhenRecentNonempty(t *testing.T) {
	dir := t.TempDir()
	prev := completionStateDir
	completionStateDir = func() string { return dir }
	t.Cleanup(func() { completionStateDir = prev })

	path := completionStatePath()
	if err := os.WriteFile(path, []byte("3"), 0o600); err != nil {
		t.Fatal(err)
	}

	got := nextMACTimeout()
	if got != 500*time.Millisecond {
		t.Errorf("nextMACTimeout() = %v, want 500ms (recent-nonempty stays cold)", got)
	}
}

func TestRecordMACAttempt_WritesCount(t *testing.T) {
	dir := t.TempDir()
	prev := completionStateDir
	completionStateDir = func() string { return dir }
	t.Cleanup(func() { completionStateDir = prev })

	recordMACAttempt(5)

	data, err := os.ReadFile(completionStatePath())
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}
	if string(data) != "5" {
		t.Errorf("state file = %q, want %q", string(data), "5")
	}
}

func TestRecordMACAttempt_TolerantOfReadOnlyDir(t *testing.T) {
	prev := completionStateDir
	completionStateDir = func() string {
		return filepath.Join(t.TempDir(), "does-not-exist", "deeper")
	}
	t.Cleanup(func() { completionStateDir = prev })

	// Should not panic; should not propagate error.
	recordMACAttempt(7)
}

func TestCompleteInterfaces_ReturnsLocalInterfaces(t *testing.T) {
	cmd := &cobra.Command{Use: "fake"}
	out, directive := completeInterfaces(cmd, nil, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want NoFileComp", directive)
	}
	if len(out) == 0 {
		t.Skip("no usable interfaces on this host; skipping")
	}
	for _, entry := range out {
		if !strings.Contains(entry, "\t") {
			t.Errorf("entry %q missing tab separator (expected Name\\tAddr)", entry)
		}
	}
}

func TestCompleteParameterNames_FiltersAlreadyListed(t *testing.T) {
	cmd := &cobra.Command{Use: "get"}
	args := []string{"aa:bb:cc:dd:ee:ff", "server_address"}
	out, directive := completeParameterNames(cmd, args, "")

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %v, want NoFileComp", directive)
	}
	for _, entry := range out {
		name := strings.SplitN(entry, "\t", 2)[0]
		if name == "server_address" {
			t.Errorf("server_address should be filtered out, got %q in results", entry)
		}
	}
	if len(out) == 0 {
		t.Error("expected non-empty parameter list after filtering one entry")
	}
}

func TestCompleteParameterNames_EmptyArgsDelegatesToMACs(t *testing.T) {
	cmd := &cobra.Command{Use: "get"}
	_, directive := completeParameterNames(cmd, nil, "")
	if directive != cobra.ShellCompDirectiveNoFileComp && directive != cobra.ShellCompDirectiveError {
		t.Errorf("directive = %v, want NoFileComp or Error", directive)
	}
}
