package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// ANSI Erase in Line (CSI n K) with n=2 clears the entire current line
// without moving the cursor. We follow it with \r to land at col 0.
// This avoids the "write 80 spaces" hazard, where on an exactly-80-column
// terminal the fill wraps to a new line before the trailing \r can return,
// leaving a stranded blank line above the next stdout output.
const ansiEraseLine = "\033[2K\r"

const (
	progressBarWidth   = 20
	progressTickRate   = 100 * time.Millisecond
	progressStartDelay = 500 * time.Millisecond
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
//
// The bar does not appear for the first progressStartDelay (500ms) so
// quick operations — single-device commands typically return in <100ms —
// don't flash a bar on and off. If the operation finishes before the
// delay elapses, no output is ever written.
func startProgress(stderr io.Writer, label string, total time.Duration) func() {
	// stderr should be a *stderrSync (cli.Run wraps every subcommand's
	// stderr in one). Unwrap to get the underlying writer so we can
	// TTY-detect; if anything else is passed (tests, or a future caller
	// that bypasses Run), we degrade to the plain *os.File path.
	var sink barSink
	var f *os.File
	switch ww := stderr.(type) {
	case *stderrSync:
		osFile, ok := ww.underlying().(*os.File)
		if !ok {
			return func() {}
		}
		f = osFile
		sink = ww
	case *os.File:
		f = ww
		sink = directSink{f}
	default:
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
		select {
		case <-ctx.Done():
			return
		case <-time.After(progressStartDelay):
		}
		ticker := time.NewTicker(progressTickRate)
		defer ticker.Stop()
		for {
			sink.barDraw(renderProgressLine(label, time.Since(start), total))
			select {
			case <-ctx.Done():
				sink.barClear()
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

// barSink is the small interface startProgress needs from its stderr
// destination. Both *stderrSync (synchronized with the logger) and
// directSink (plain stderr, no synchronization) satisfy it.
type barSink interface {
	barDraw(text string)
	barClear()
}

// directSink writes the bar straight to a writer with no synchronization.
// Used as a fallback when startProgress is called outside cli.Run's
// stderrSync wrapping.
type directSink struct{ w io.Writer }

func (d directSink) barDraw(text string) { fmt.Fprint(d.w, text) }
func (d directSink) barClear()           { fmt.Fprint(d.w, ansiEraseLine) }

func renderProgressLine(label string, elapsed, total time.Duration) string {
	pct := float64(elapsed) / float64(total)
	if pct > 1 {
		pct = 1
	}
	filled := int(pct * float64(progressBarWidth))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", progressBarWidth-filled)
	return fmt.Sprintf("\r%s: [%s] %3d%% (%.1fs/%.1fs)",
		label, bar, int(pct*100), elapsed.Seconds(), total.Seconds())
}
