package cli

import (
	"fmt"
	"io"
	"sync"
)

// stderrSync wraps an io.Writer (typically os.Stderr) so the progress
// bar and the structured logger can both write to it without
// interleaving. A mutex serializes all writes; in addition, when a log
// write happens while the bar is currently rendered, the bar's line is
// erased first. The bar's next tick will redraw on the new current line
// (the line below the log content), so subsequent log lines stack
// above the still-visible bar.
//
// stderrSync implements io.Writer so it can be passed to anything that
// expects one (the udap structured logger, fmt.Fprintln, etc.). The
// progress-bar goroutine uses barDraw / barClear instead of Write so
// the wrapper can track bar state.
type stderrSync struct {
	mu        sync.Mutex
	w         io.Writer
	barActive bool
}

// newStderrSync wraps w. Use the result as the stderr io.Writer
// throughout the CLI so all writes are serialized.
func newStderrSync(w io.Writer) *stderrSync {
	return &stderrSync{w: w}
}

// Write implements io.Writer. Erases an active bar line before writing
// log content so the two don't smash together on the same row.
func (s *stderrSync) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.barActive {
		fmt.Fprint(s.w, ansiEraseLine)
		s.barActive = false
	}
	return s.w.Write(p)
}

// barDraw renders the bar text. Marks the bar as active so subsequent
// log writes know to clear it.
func (s *stderrSync) barDraw(text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fmt.Fprint(s.w, text)
	s.barActive = true
}

// barClear erases the bar's line if one is active. Idempotent.
func (s *stderrSync) barClear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.barActive {
		fmt.Fprint(s.w, ansiEraseLine)
		s.barActive = false
	}
}

// underlying returns the wrapped writer. Used by startProgress to do
// TTY detection on the real *os.File without holding the mutex.
func (s *stderrSync) underlying() io.Writer {
	return s.w
}
