## Decision: Translate device-op timeout errors in the CLI handler layer, not the udap library leaf

When a device operation (`getip`, `read`, `get`, `set`) times out, the
plain-English message ("`<cmd>`: no reply from `<MAC>` within
`<timeout>`") is produced in the CLI command handlers via a shared
`cli.deviceOpError(op, mac, timeout, err)` helper. The `udap` package
keeps returning its raw wrapped error (`recv reply for <MAC>: %w`
around `context.DeadlineExceeded`); the CLI checks `errors.Is(err,
context.DeadlineExceeded)` through that wrap.

## Context: issue #110

Timeouts leaked Go's context model to the user:

    error: get_ip failed for <MAC>: recv reply for <MAC>: context deadline exceeded

Issue #110 proposed fixing this "at the leaf where `errors.Is(err,
context.DeadlineExceeded)` is true". Taken literally that leaf is
`udap.waitForDeviceReply`. Implemented in PR #168.

## Alternatives considered:

- **Translate in `udap.waitForDeviceReply` (the literal leaf):**
  rejected. The `udap` package is a reusable library; it should not
  hardcode CLI phrasing. It also cannot name the command the user typed
  (`getip` vs the wire method `get_ip`, `read` vs `get_data`) or the
  `--timeout` value (those live only in the CLI), and it would collide
  with `ResetDeviceWithContext`, which deliberately treats a deadline as
  *success* (the device rebooted before acking).
- **Rewrite the error string in each command inline (no helper):**
  rejected. Five call sites (getip, read, get, set-prelude, set-write)
  would duplicate the same `errors.Is` check and format string — past
  the rule-of-three, so a helper is warranted.

## Reasoning:

The command name and the `--timeout` value are CLI-layer facts; the
handler is the only place that has both. A single `deviceOpError` helper
centralises the deadline check and keeps the message format in one
place. `errors.Is` sees `context.DeadlineExceeded` through any wrap
depth, so the `udap` layer needs no change and non-deadline errors keep
their full wrapped chain (cause stays visible and `errors.Is`-able).

## Trade-offs accepted:

- The "within `<timeout>`" figure is the user's `--timeout`, not the
  residual budget the operation actually had. Discovery and the
  operation share one timeout context (review finding #4), so a slow
  discovery can leave the op a sliver of the budget. Reporting the value
  the user set matches their mental model; reporting the residual would
  be more confusing. Accepted.
- `context.Canceled` (Ctrl-C) is intentionally left untranslated — an
  aborted run is not a timeout.

## Supersedes: none. Establishes the pattern that issues #109 and #111
(which touch the same device-op error paths) should follow: user-facing
error phrasing belongs in the CLI layer, `udap` stays framework- and
UI-agnostic.
