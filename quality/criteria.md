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

---

## Category: Supply-chain scan findings

## Criteria:

    - Every entry in docs/site/pnpm-workspace.yaml overrides names the
      advisory (or advisories) it mitigates and the version that fixes
      them. A package already listed there is a likely repeat offender,
      not a solved problem: `pkg@<X: ^X` is a floor, not a ceiling, and
      does nothing once X itself is found vulnerable.
    - Before trusting a post-fix clean scan, the PRE-fix tree is scanned
      and the finding confirmed to reproduce, and `grype db status`
      matches the DB build the failing CI run logged. Grype scans with a
      stale DB and reports "No vulnerabilities found" after only a WARN,
      so a clean result on its own is not evidence.
    - govulncheck findings are verified with the toolchain pinned
      (`GOTOOLCHAIN=goX.Y.Z`), never the local default. CI installs the
      exact go.mod directive, so a newer local toolchain masks the
      alerts CI reports.
    - When a stdlib alert is open in the Security tab, go.mod's go
      directive is moved to a release that fixes it.

## Severity: warning

## Source: PR #213 (fast-uri re-pin + Go 1.27.1); recurring across #115, #129, #171, #179, #188

## Last triggered: 2026-10-09 (sharp GHSA-wq5f-xc86-pv6w landed on
0.35.4, above the existing `^0.35.0` floor — repeat offender again)

---

## Category: Generated packaging

## Criteria:

    - A change to how the Homebrew cask is generated is verified by an
      actual `goreleaser release --snapshot` render, then `brew style
      --fix` and `brew style` over the rendered file — never by reading
      `.goreleaser.yaml` and reasoning about the output. GoReleaser
      templates every `homebrew_casks` string field, so literal `{{...}}`
      in a cask stanza fails the release unless escaped.
    - Any stanza the cask relies on is proven to *execute*, not merely to
      parse: install the cask, and confirm a deliberately broken variant
      fails. `brew style` passing says nothing about runtime.
    - Homebrew cask cops and DSL deprecations move independently of this
      repo. When a `brew` command warns about the cask, fix the generator
      in `.goreleaser.yaml` as well as the tap's checked-in copy —
      regenerating on the next release leaves the tap's daily sweep red
      until then.

## Severity: warning

## Source: decision 2026-09-05-cask-postflight-steps (postflight →
postflight_steps deprecation)

## Last triggered: 2026-09-05 (`postflight` deprecated; `brew style --fix`
could not autocorrect it, so the tap's daily sweep failed)
