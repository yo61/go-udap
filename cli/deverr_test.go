package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// A bare context deadline becomes a plain-English message naming the
// command, the device, and the timeout the user asked for.
func TestDeviceOpErrorTranslatesDeadline(t *testing.T) {
	err := deviceOpError("getip", "00:04:20:00:00:01", 2*time.Second, context.DeadlineExceeded)

	if err.Code != 2 {
		t.Errorf("Code = %d, want 2", err.Code)
	}
	const want = "getip: no reply from 00:04:20:00:00:01 within 2s"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}

// The real error arrives wrapped (waitForDeviceReply wraps it as
// "recv reply for ...: %w"). errors.Is must still see the deadline
// through the wrap, and the leaked "context deadline exceeded" /
// "recv reply" internals must not reach the user.
func TestDeviceOpErrorTranslatesWrappedDeadline(t *testing.T) {
	wrapped := fmt.Errorf("recv reply for 00:04:20:00:00:01: %w", context.DeadlineExceeded)
	err := deviceOpError("read", "00:04:20:00:00:01", 500*time.Millisecond, wrapped)

	const want = "read: no reply from 00:04:20:00:00:01 within 500ms"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
	if s := err.Error(); strings.Contains(s, "context deadline exceeded") || strings.Contains(s, "recv reply") {
		t.Errorf("Error() leaks internals: %q", s)
	}
}

// A non-deadline error keeps its wrapped chain (so the underlying cause
// is still visible) and stays unwrappable via errors.Is.
func TestDeviceOpErrorPreservesNonDeadlineError(t *testing.T) {
	cause := errors.New("device rejected credentials")
	err := deviceOpError("set", "00:04:20:00:00:01", time.Second, cause)

	if err.Code != 2 {
		t.Errorf("Code = %d, want 2", err.Code)
	}
	const want = "set failed for 00:04:20:00:00:01: device rejected credentials"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
	if !errors.Is(err, cause) {
		t.Errorf("errors.Is(err, cause) = false, want true (chain broken)")
	}
}

// Ctrl-C (context.Canceled) is not a timeout; it must not be rewritten
// as "no reply within ...".
func TestDeviceOpErrorDoesNotTranslateCanceled(t *testing.T) {
	err := deviceOpError("get", "00:04:20:00:00:01", time.Second, context.Canceled)

	if strings.Contains(err.Error(), "no reply") {
		t.Errorf("Canceled rewritten as timeout: %q", err.Error())
	}
}
