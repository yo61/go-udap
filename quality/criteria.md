# Quality Criteria — go-udap

Testable checks evaluated before marking a task complete. `blocking`
criteria must pass; `warning` criteria are surfaced but do not stop the
work. See the global CLAUDE.md "Quality Gate" section for how these are
promoted, pruned, and maintained.

---

## Category: CLI error UX

## Criteria:

    - No error surfaced to the user leaks Go internals: no "context
      deadline exceeded", "context canceled", struct dumps, or raw
      "%!v(...)" from a device-facing command.
    - Every user-facing error names the operation and, where relevant,
      the device MAC and the actionable value (e.g. the --timeout used).
    - Error phrasing lives in the CLI layer; the udap package stays
      UI-agnostic (returns wrapped errors, not user prose).

## Severity: blocking

## Source: issue #110; decision 2026-07-21-cli-timeout-error-layer

## Last triggered: 2026-07-21 (#110 — timeout messages leaked the
context wrap chain)

---

## Category: Exit-code contract

## Criteria:

    - Exit 0 only on success; 1 for usage/validation errors; 2 for
      operation failures. No other codes.
    - Every non-happy-path branch returns an *ExitError with the correct
      code, and a test asserts that code.

## Severity: blocking

## Source: CLAUDE.md "Output ... Exit codes: 0 success, 1 usage error, 2 operation failure"

## Last triggered: never

---

## Category: Test determinism

## Criteria:

    - Timeout / failure behaviour is exercised via mocksbr injection
      knobs (DropGetData, DropGetIP, FailOn, Unreachable), not via
      wall-clock races where a fast machine and a slow machine disagree.
    - New tests pass under `go test -race ./...`.
    - Tests assert behaviour (output, exit code, wire effect), not
      implementation details that a refactor would break.

## Severity: blocking

## Source: CLAUDE.md Testing standards; #110 (DropGetData added to avoid a timing-based test)

## Last triggered: 2026-07-21 (#110 — added DropGetData knob rather than a Slow-timing test)

---

## Category: Protocol correctness

## Criteria:

    - All UDAP wire fields are big-endian; TLV encode/decode round-trips.
    - New wire behaviour is checked against a Net::UDAP reference capture
      or an explicit citation, not assumed.

## Severity: blocking

## Source: CLAUDE.md "Key Protocol Details"; udap/getdata_response.go provenance note

## Last triggered: never

---

## Category: Cross-platform

## Criteria:

    - Platform-specific socket behaviour lives in socket_{unix,darwin,linux,windows}.go
      behind a shared signature; no build breaks on any target in
      `task build:all`.
    - Features that cannot work on a platform (e.g. --bind-interface on
      Windows) fail with an explicit "not supported" message, not a
      silent no-op.

## Severity: warning

## Source: CLAUDE.md "Cross-Platform Support"

## Last triggered: never
