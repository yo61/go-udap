package cli

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// deviceOpError renders a failure from a device-targeted operation as an
// *ExitError (exit code 2). A context deadline is translated into a
// plain-English "no reply from <mac> within <timeout>" message so the
// user never sees Go's "context deadline exceeded" wrap chain. Every
// other error keeps its wrapped chain, so the underlying cause stays
// visible and unwrappable via errors.Is.
//
// context.Canceled (Ctrl-C) is deliberately not translated: an aborted
// run is not a timeout, and reporting "within <timeout>" would misstate
// how long the user actually waited.
func deviceOpError(op, mac string, timeout time.Duration, err error) *ExitError {
	if errors.Is(err, context.DeadlineExceeded) {
		return &ExitError{Code: 2, Err: fmt.Errorf("%s: no reply from %s within %s", op, mac, timeout)}
	}
	return &ExitError{Code: 2, Err: fmt.Errorf("%s failed for %s: %w", op, mac, err)}
}
