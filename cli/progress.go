package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const (
	progressBarWidth = 20
	progressTickRate = 100 * time.Millisecond
)

// startProgress draws a single-line progress bar to stderr that fills as
// `total` elapses. Returns a stop function the caller must invoke (defer
// works) to clear the line and tear down the goroutine.
//
// If stderr is not a TTY (e.g. piped to a log file, or a *bytes.Buffer
// in tests), startProgress is a no-op: the returned function is safe to
// call but does nothing, and no goroutine is spawned. This keeps log
// files clean and prevents stray escape sequences when the operator
// captures stderr.
func startProgress(stderr io.Writer, label string, total time.Duration) func() {
	f, ok := stderr.(*os.File)
	if !ok {
		return func() {}
	}
	st, err := f.Stat()
	if err != nil || (st.Mode()&os.ModeCharDevice) == 0 {
		return func() {}
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		start := time.Now()
		ticker := time.NewTicker(progressTickRate)
		defer ticker.Stop()
		for {
			drawProgressLine(f, label, time.Since(start), total)
			select {
			case <-ctx.Done():
				clearProgressLine(f)
				return
			case <-ticker.C:
			}
		}
	}()
	return func() {
		cancel()
		<-done
	}
}

func drawProgressLine(w io.Writer, label string, elapsed, total time.Duration) {
	pct := float64(elapsed) / float64(total)
	if pct > 1 {
		pct = 1
	}
	filled := int(pct * float64(progressBarWidth))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", progressBarWidth-filled)
	fmt.Fprintf(w, "\r%s: [%s] %3d%% (%.1fs/%.1fs)",
		label, bar, int(pct*100), elapsed.Seconds(), total.Seconds())
}

func clearProgressLine(w io.Writer) {
	fmt.Fprintf(w, "\r%s\r", strings.Repeat(" ", 80))
}
