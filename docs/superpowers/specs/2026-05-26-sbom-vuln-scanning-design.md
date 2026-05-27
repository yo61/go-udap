# SBOM generation and vulnerability scanning for go-udap

**Date:** 2026-05-26
**Status:** Implemented 2026-05-27 (PRs #95, #96, #97, #98; first shipped in v2.3.0)

## Summary

Add three independent layers of supply-chain hygiene to `go-udap`:

1. **Release-attached SBOMs** — GoReleaser invokes `syft` to emit a per-binary
   SBOM in both SPDX-JSON and CycloneDX-JSON formats. Files upload as release
   artifacts alongside the existing tarballs/zips so downstream consumers can
   verify exactly what's in the binary they downloaded.
2. **Active CI vulnerability scanning** — a new `security.yaml` workflow runs
   `govulncheck` (Go-native, reachability-aware) and `Grype` (broad CVE
   coverage against the syft SBOM) on every PR, every push to `main`, and on a
   daily cron. Findings appear in the GitHub Security tab via SARIF upload.
   HIGH/CRITICAL findings fail the job.
3. **Background dep patching** — `.github/dependabot.yml` opens grouped
   weekly PRs for `gomod` and `github-actions` ecosystems, with a 7-day
   cooldown to avoid pulling brand-new releases.

The three layers do not overlap meaningfully on false positives; each catches
a different failure mode (govulncheck = reachable Go CVEs, Grype = broader DB
coverage including non-Go-stdlib advisories, Dependabot = outdated deps that
haven't been disclosed yet).

## Motivation

- `go-udap` ships as a release binary distributed via GitHub Releases and a
  Homebrew tap. Downstream consumers have no machine-readable manifest of
  what's inside the binary, which blocks any consumer that performs SBOM-based
  compliance review (SPDX is ISO/IEC 5962:2021).
- The repo has zero active vulnerability scanning today. CI checks correctness
  (`go vet`, `go test`, `prek`) but not supply-chain exposure. A CVE in
  `cobra` or `pflag` would land silently.
- Most CVE detection happens *after* a release ships — disclosure timing is
  external to the project. A daily cron against `main` catches newly-disclosed
  CVEs in code that hasn't changed, with low runner cost (under a minute
  per job on a 3-dep project).

## Non-goals

- **Cosign signing / SLSA provenance** — distinct concern, tracked separately
  in [#93](https://github.com/yo61/go-udap/issues/93). See "Deferred:
  artifact signing and build provenance" below for the threat-model split
  and rationale.
- **Trivy** — overlaps Grype's coverage for a pure-Go CLI with three deps; not
  worth the second tool's runtime.
- **Committed SBOM file in repo** — generates churn on every dep bump for no
  consumer benefit beyond the release-attached SBOMs.
- **Auto-filed GitHub Issues on cron findings** — workflow failure plus email
  notification is sufficient for a low-traffic repo. Re-evaluate if cron-only
  findings prove easy to miss.
- **Blocking on LOW/MEDIUM severity** — produces noise without proportionate
  signal on a 3-dep project. HIGH/CRITICAL only.

### Deferred: artifact signing and build provenance

The SBOM tells consumers **what is in** the binary. It does not tell them
whether the file they downloaded is the one this project actually built, or
whether it was built from the source it claims. Those are two further
questions answered by separate mechanisms:

| Question | Mechanism | Tool |
|---|---|---|
| What is in this artifact? | SBOM | syft (in scope) |
| Was this artifact produced by the project, untampered in transit? | Artifact signature | **Cosign** (deferred) |
| Was this artifact built from the source it claims, on a trusted builder? | Build attestation | **SLSA provenance** (deferred) |

**Why deferred from this spec:**

- Each has its own threat model, its own consumer-side verification story
  (commands, tooling, docs), and its own UX cost. Bundling them with SBOM/
  scanning would inflate scope without sharpening any of the three.
- Cosign + SLSA both depend on OIDC plumbing in the GoReleaser workflow
  (`id-token: write`, `attestations: write` permissions, transparency-log
  uploads). The SBOM/scanning work does not. Keeping them separate keeps
  the GoReleaser change in this spec to a single self-contained `sboms:`
  block.
- Consumer verification needs documentation in `docs-site/` (a how-to per
  the Diataxis split already used there). That writing effort is non-trivial
  and lands more cleanly when paired with the signing work itself.

**Suggested order for the follow-up (per #93):**

1. **Cosign first** — simpler scope, satisfies the most common threat model
   (\"is this binary really from the project?\"). GoReleaser has a native
   `signs:` block that handles it in ~one config addition.
2. **SLSA provenance second** — reuses Cosign's OIDC + Sigstore plumbing;
   adds `actions/attest-build-provenance` as one workflow step. Low
   marginal cost once Cosign is in.

Both should sign **every release artifact**, including the SBOMs produced
by this spec — a signed SBOM is the actual goal; an unsigned SBOM is
informative but unverifiable. The work in this spec produces the SBOMs in a
state ready to be signed by the follow-up.

## References

- **syft** (`github.com/anchore/syft`) — SBOM generator, supports SPDX and
  CycloneDX. Used directly by GoReleaser's `sboms:` block and by
  `anchore/sbom-action` in CI.
- **govulncheck** (`golang.org/x/vuln/cmd/govulncheck`) — Go team's vuln
  scanner. Uses the Go vulnerability database. Reachability-aware: only flags
  CVEs in code paths your binary actually calls. SARIF output via
  `-format sarif`.
- **Grype** (`github.com/anchore/grype`) — Anchore's SBOM-based scanner.
  Sources advisories from NVD, GitHub Security Advisories, and ecosystem-
  specific feeds. Not reachability-aware. SARIF output via
  `anchore/scan-action`.
- **GoReleaser SBOMs** (`goreleaser.com/customization/sbom/`) — `sboms:` block
  invokes a generator (default `syft`) per artifact and uploads the result.
- **anchore/sbom-action** (`github.com/anchore/sbom-action`) — wraps syft for
  CI use; supports SARIF, SPDX, CycloneDX.
- **anchore/scan-action** (`github.com/anchore/scan-action`) — wraps Grype;
  `fail-build`, `severity-cutoff`, `output-format` inputs.
- **GitHub SARIF upload** (`github/codeql-action/upload-sarif`) — sends SARIF
  findings to the repository's Security tab.
- **Dependabot config** (`docs.github.com/en/code-security/dependabot/working-
  with-dependabot/dependabot-options-reference`) — `cooldown.default-days`,
  `groups.<name>.update-types` keys.

## Architecture

Four components, each isolated and independently maintainable:

| Component | File | Owner |
|---|---|---|
| Release SBOM | `.goreleaser.yaml` | Existing release pipeline |
| Active scanning | `.github/workflows/security.yaml` (new) | New workflow |
| Background patching | `.github/dependabot.yml` (modify; pre-existing pre-commit block stays) | GitHub-scheduled |
| Local repro | `Taskfile.yml` + `CLAUDE.md` | Contributors |

### Component 1 — Release-attached SBOMs

Modify `.goreleaser.yaml` to add an `sboms:` block. syft is pre-installed in
the `goreleaser-action` runner image, so no separate install step is needed.

```yaml
sboms:
  - id: spdx
    artifacts: archive
    documents:
      - "${artifact}.spdx.json"
    args: ["$artifact", "--output", "spdx-json=$document"]
  - id: cyclonedx
    artifacts: archive
    documents:
      - "${artifact}.cdx.json"
    args: ["$artifact", "--output", "cyclonedx-json=$document"]
```

`artifacts: archive` runs syft against each tarball/zip after `archives:`
runs, so the SBOM reflects the shipped artifact (binary + completions +
LICENSE + README). Two formats produced per archive:

- `go-udap_X.Y.Z_macos_arm64.tar.gz.spdx.json`
- `go-udap_X.Y.Z_macos_arm64.tar.gz.cdx.json`
- … (same for every platform tarball/zip)

Cost: two extra syft invocations per platform (~7 platforms post-format-
overrides) on each release. Sub-second each.

**Checksums:** GoReleaser already hashes archives into `SHA256SUMS`. SBOM
files will also need to land in `SHA256SUMS` for tamper-evidence. Confirm
behaviour: GoReleaser v2 includes SBOM artifacts in the checksum file by
default (`checksum.extra_files` not needed). Implementation plan verifies
this against a `--snapshot` run.

### Component 2 — Active CI vulnerability scanning

New workflow `.github/workflows/security.yaml`:

```yaml
name: Security

on:
  pull_request:
  push:
    branches: [main]
  schedule:
    - cron: '17 6 * * *'   # daily ~06:17 UTC; off-the-hour to dodge runner congestion
  workflow_dispatch:

permissions:
  contents: read

jobs:
  govulncheck:
    name: govulncheck
    runs-on: ubuntu-latest
    permissions:
      contents: read
      security-events: write   # SARIF upload
    steps:
      - uses: actions/checkout@<sha>          # v6
      - uses: actions/setup-go@<sha>          # v6
        with:
          go-version-file: go.mod
      - name: Install govulncheck
        run: go install golang.org/x/vuln/cmd/govulncheck@latest
      - name: Run govulncheck
        run: govulncheck -format sarif ./... > govulncheck.sarif
      - name: Upload SARIF
        if: always()
        uses: github/codeql-action/upload-sarif@<sha>
        with:
          sarif_file: govulncheck.sarif
          category: govulncheck

  sbom-scan:
    name: sbom-scan
    runs-on: ubuntu-latest
    permissions:
      contents: read
      security-events: write
    steps:
      - uses: actions/checkout@<sha>
      - name: Generate SBOM (syft)
        uses: anchore/sbom-action@<sha>
        with:
          path: .
          format: cyclonedx-json
          output-file: sbom.cdx.json
          upload-artifact: true
      - name: Scan SBOM (Grype)
        uses: anchore/scan-action@<sha>
        with:
          sbom: sbom.cdx.json
          fail-build: true
          severity-cutoff: high
          output-format: sarif
        id: grype
      - name: Upload SARIF
        if: always()
        uses: github/codeql-action/upload-sarif@<sha>
        with:
          sarif_file: ${{ steps.grype.outputs.sarif }}
          category: grype
```

All `uses:` references pin to full SHA with `# vX.Y.Z` comment per the
project's existing convention (see existing workflows). Implementation plan
resolves the actual SHAs at write time.

**Failure semantics:**

| Job | Fails on |
|---|---|
| `govulncheck` | Any finding (govulncheck is reachability-gated; a finding is always a real exposure) |
| `sbom-scan` | Grype finding at severity ≥ HIGH |

LOW/MEDIUM Grype findings appear in the Security tab via SARIF but do not
block. SARIF upload uses `if: always()` so findings surface even when the
job fails.

**Cron behaviour:** the same workflow runs against `main` daily. A workflow
failure produces a red badge and an email to repo admins — sufficient
signal for a low-traffic repo. If signal proves noisy or easy to miss, add
issue-on-failure later via `peter-evans/create-issue-from-file` or similar.

### Component 3 — Dependabot

Modify the existing `.github/dependabot.yml` (which today only covers the
`pre-commit` ecosystem) to add three more:

```yaml
version: 2
updates:
  # Pre-existing block — unchanged
  - package-ecosystem: pre-commit
    directory: /
    schedule:
      interval: weekly
    cooldown:
      default-days: 7
    groups:
      hooks:
        patterns: ["*"]

  - package-ecosystem: gomod
    directory: /
    schedule:
      interval: weekly
    cooldown:
      default-days: 7
    groups:
      gomod-minor-patch:
        update-types: [minor, patch]

  - package-ecosystem: github-actions
    directory: /
    schedule:
      interval: weekly
    cooldown:
      default-days: 7
    groups:
      actions-minor-patch:
        update-types: [minor, patch]

  - package-ecosystem: npm
    directory: /docs/site
    schedule:
      interval: weekly
    cooldown:
      default-days: 7
    groups:
      docs-site-minor-patch:
        update-types: [minor, patch]
```

Major-version bumps remain individual PRs (Dependabot's default when a major
update isn't covered by a group). The 7-day cooldown matches the project's
existing `prek auto-update --cooldown-days 7` configuration and the existing
`pre-commit` block's own cooldown — same posture, different package
managers.

The `docs/site/` subdirectory has its own pnpm-managed `package.json` (the
Astro docs site). The `npm` ecosystem covers it; Dependabot reads both
`package-lock.json` and `pnpm-lock.yaml`. Major Astro/framework bumps will
land as individual PRs that may need manual review.

### Component 4 — Local developer story

Add a `task security` target to `Taskfile.yml`:

```yaml
security:
  desc: Run govulncheck + grype locally (matches CI)
  cmds:
    - go run golang.org/x/vuln/cmd/govulncheck@latest ./...
    - |
      if command -v grype >/dev/null 2>&1; then
        grype dir:.
      else
        echo "grype not installed; skipping (brew install grype)" >&2
      fi
```

`govulncheck` uses `go run` so no global install required. `grype` is
optional locally — CI is authoritative.

Document the workflow briefly in `CLAUDE.md` (new "Security scanning"
subsection under "Common Commands") so contributors can repro CI findings
locally.

## Data flow

```
PR opened
   │
   ├─► CI (existing): test, lint, completion-smoke
   │
   └─► Security (new):
         │
         ├─► govulncheck ──► SARIF ──► Security tab (fails on any finding)
         │
         └─► syft → SBOM → grype ──► SARIF ──► Security tab (fails ≥ HIGH)

Daily cron @ 06:17 UTC
   └─► Security workflow against main (same jobs, same failure semantics)

Release tag pushed
   └─► GoReleaser (existing) + syft per-archive (new)
         └─► .spdx.json + .cdx.json uploaded with each tarball/zip
```

## Failure modes and mitigations

| Failure mode | Mitigation |
|---|---|
| New CVE disclosed against an existing dep over the weekend | Daily cron catches it within 24 h; Dependabot also opens an upgrade PR if a patched version exists |
| govulncheck DB lookup fails (rare; network blip) | Job fails → CI red. Re-run resolves. No false-positive blocking since the tool errored, not flagged a CVE. |
| Grype rate-limited against NVD | `anchore/scan-action` caches its DB; failures are rare. If observed, switch to `update-db: false` with a periodic DB-update job. |
| SBOM format consumer wants neither SPDX nor CycloneDX | Out of scope; both formats cover ≥99% of consumers. |
| Dependabot PR storms after a busy CVE week | Grouped minor/patch keeps it to one PR per ecosystem per week. Major bumps remain separate (rare). |
| GoReleaser SBOM block breaks on a new syft version | Pinned via the `goreleaser-action` image; syft upgrades land via Dependabot's `github-actions` ecosystem and are testable on a `--snapshot` run before tag. |

## Test plan

The implementation plan covers:

- **`security.yaml` PR smoke test** — open the PR that introduces the
  workflow; both jobs must run green against the current 3-dep project.
- **Inject a known CVE** — temporarily pin a `go.mod` dep to a vulnerable
  version, confirm govulncheck flags it (reachable) or Grype flags it
  (regardless of reachability). Revert before merge.
- **Cron schedule sanity** — confirm the cron expression is valid via
  GitHub's UI; the first scheduled run lands within 24 h of merge.
- **Dependabot config validation** — push to a branch, check that the
  Dependabot tab in repo settings parses the file without errors.
- **GoReleaser SBOM smoke** — `goreleaser release --snapshot --clean`
  locally produces `.spdx.json` and `.cdx.json` per archive. Verify the
  SBOM lists `cobra`, `pflag`, `mousetrap` and the Go runtime version.
- **SARIF surfacing** — after merge, confirm both `govulncheck` and `grype`
  categories appear under the Security tab → Code scanning alerts.

## Rollback

Each component is in a separate file and can be reverted independently:

| Component | Rollback |
|---|---|
| Release SBOMs | Revert the `sboms:` block in `.goreleaser.yaml`; next tag ships without SBOMs. No effect on existing releases. |
| Active scanning | Delete `.github/workflows/security.yaml`; CI returns to pre-change behaviour. |
| Dependabot | Delete `.github/dependabot.yml`; GitHub stops opening update PRs. Existing open Dependabot PRs are unaffected (can be closed manually). |

No data migrations, no state changes outside CI metadata.

## Open questions for implementation

- **Resolve action SHAs** — `anchore/sbom-action`, `anchore/scan-action`,
  `github/codeql-action/upload-sarif` need pinning to current stable SHAs
  at implementation time.
- **Confirm `checksum.extra_files` behaviour** — does GoReleaser v2's
  default `checksum:` block include SBOM artifacts? Verify via `--snapshot`
  before merging the GoReleaser change.
