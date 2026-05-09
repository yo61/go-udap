package cli

import (
	"bytes"
	"testing"
	"time"
)

func TestStartProgressNoOpWhenStderrNotTTY(t *testing.T) {
	var buf bytes.Buffer
	stop := startProgress(&buf, "Test", 100*time.Millisecond)
	time.Sleep(50 * time.Millisecond)
	stop()
	if buf.Len() != 0 {
		t.Errorf("expected no output to non-TTY writer, got %q", buf.String())
	}
}

func TestStartProgressStopIsImmediate(t *testing.T) {
	var buf bytes.Buffer
	stop := startProgress(&buf, "Test", 5*time.Second)
	start := time.Now()
	stop()
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Errorf("stop() took %v; expected near-instant return", elapsed)
	}
}
