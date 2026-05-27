# SBOM + vulnerability scanning Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add three-layer supply-chain hygiene to `go-udap` — release-attached SBOMs, active CI vulnerability scanning (govulncheck + Grype), and expanded Dependabot coverage.

**Architecture:** Four files touched, each isolated. Land in five PRs, smallest-blast-radius first: Dependabot → local Taskfile → security workflow → GoReleaser SBOM block → CLAUDE.md doc. Each PR is independently revertable.

**Tech Stack:** GoReleaser v2 (existing), syft (via `anchore/sbom-action` and via GoReleaser's bundled syft), Grype (via `anchore/scan-action`), `govulncheck` (`golang.org/x/vuln/cmd/govulncheck`), GitHub Actions, GitHub Dependabot (gomod / github-actions / npm / pre-commit ecosystems).

**Reference:** `docs/superpowers/specs/2026-05-26-sbom-vuln-scanning-design.md`

---

## File map

| File | Action | Owner task |
|---|---|---|
| `.github/dependabot.yml` | modify (add gomod, github-actions, npm; keep pre-commit) | Task 1 |
| `Taskfile.yml` | modify (add `security` task) | Task 2 |
| `CLAUDE.md` | modify (document local repro + CI scanning) | Task 2 |
| `.github/workflows/security.yaml` | create | Task 3 |
| `.goreleaser.yaml` | modify (add `sboms:` block) | Task 4 |

## Pinned action SHAs (resolved 2026-05-27)

The security workflow uses these SHAs. Look up fresh if more than a few days have passed since this plan was written:

| Action | Version | SHA |
|---|---|---|
| `actions/checkout` | v6.0.2 | `de0fac2e4500dabe0009e67214ff5f5447ce83dd` |
| `actions/setup-go` | v6.4.0 | `4a3601121dd01d1626a1e23e37211e3254c1c06c` |
| `anchore/sbom-action` | v0.24.0 | `e22c389904149dbc22b58101806040fa8d37a610` |
| `anchore/scan-action` | v7.4.0 | `e1165082ffb1fe366ebaf02d8526e7c4989ea9d2` |
| `github/codeql-action/upload-sarif` | v3.36.0 | `03e4368ac7daa2bd82b3e85262f3bf87ee112f57` |

Look up commands (paste into Bash if regenerating):

```bash
gh api repos/anchore/sbom-action/releases/latest --jq '.tag_name' \
  | xargs -I{} gh api repos/anchore/sbom-action/git/refs/tags/{} --jq '.object.sha'
```

---

## Task 1: Expand Dependabot coverage

**Goal:** Add `gomod`, `github-actions`, and `npm` (for `docs/site/`) ecosystems to the existing `.github/dependabot.yml`, preserving the pre-existing `pre-commit` block.

**Files:**
- Modify: `.github/dependabot.yml`

**Branch:** `chore/dependabot-expand-ecosystems`

- [ ] **Step 1: Create a feature branch from main**

```bash
git checkout main && git pull --ff-only
git checkout -b chore/dependabot-expand-ecosystems
```

Expected: `Switched to a new branch 'chore/dependabot-expand-ecosystems'`

- [ ] **Step 2: Rewrite `.github/dependabot.yml`**

Replace the entire contents of `.github/dependabot.yml` with:

```yaml
version: 2
updates:
  # Pre-existing: pre-commit hook updates (don't remove)
  - package-ecosystem: "pre-commit"
    directory: "/"
    schedule:
      interval: "weekly"
    cooldown:
      default-days: 7
    groups:
      hooks:
        patterns: ["*"]

  # Go module updates
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
    cooldown:
      default-days: 7
    groups:
      gomod-minor-patch:
        update-types: ["minor", "patch"]

  # GitHub Actions updates (SHA-pinned actions get tag-tracking PRs)
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
    cooldown:
      default-days: 7
    groups:
      actions-minor-patch:
        update-types: ["minor", "patch"]

  # docs/site is pnpm-managed; Dependabot auto-detects pnpm-lock.yaml
  - package-ecosystem: "npm"
    directory: "/docs/site"
    schedule:
      interval: "weekly"
    cooldown:
      default-days: 7
    groups:
      docs-site-minor-patch:
        update-types: ["minor", "patch"]
```

- [ ] **Step 3: Validate YAML syntax**

Run:

```bash
python3 -c 'import yaml,sys; yaml.safe_load(open(".github/dependabot.yml"))' && echo OK
```

Expected: `OK`

- [ ] **Step 4: Validate Dependabot schema with a permissive checker**

If `actionlint` is on PATH it skips dependabot.yml; the authoritative check is GitHub's parser on push. Push and rely on the Dependabot tab as the validator (Step 7).

Local sanity check — confirm each ecosystem name is one of the supported values:

```bash
grep 'package-ecosystem:' .github/dependabot.yml | sort -u
```

Expected output (exactly these four lines):

```
  - package-ecosystem: "github-actions"
  - package-ecosystem: "gomod"
  - package-ecosystem: "npm"
  - package-ecosystem: "pre-commit"
```

- [ ] **Step 5: Commit**

```bash
git add .github/dependabot.yml
git commit -m "ci: expand dependabot to gomod, github-actions, npm

Adds three ecosystems alongside the existing pre-commit block:
gomod (root), github-actions (root), npm (docs/site/, pnpm-managed).
Weekly cadence with 7-day cooldown matches the pre-commit block.
Minor/patch grouped per ecosystem; majors remain individual PRs."
```

- [ ] **Step 6: Push and open PR**

```bash
git push -u origin chore/dependabot-expand-ecosystems
gh pr create --fill
```

- [ ] **Step 7: Verify GitHub parses the config after merge**

After the PR merges, navigate to:

```
https://github.com/yo61/go-udap/network/updates
```

Expected: four ecosystems listed (`pre-commit`, `gomod`, `github-actions`, `npm`), no parse errors. The first scheduled run lands within ~24 hours.

If GitHub flags a parse error, revert the merge commit and re-iterate.

---

## Task 2: Local repro — Taskfile `security` target + CLAUDE.md doc

**Goal:** Give contributors a one-liner that runs the same scans CI runs, and document it.

**Files:**
- Modify: `Taskfile.yml` (add `security` task)
- Modify: `CLAUDE.md` (add "Security scanning" subsection under "Common Commands")

**Branch:** `chore/local-security-task`

- [ ] **Step 1: Create branch**

```bash
git checkout main && git pull --ff-only
git checkout -b chore/local-security-task
```

- [ ] **Step 2: Add `security` task to `Taskfile.yml`**

Append this task block to `Taskfile.yml` (place it after the `mutate:` task, which is currently the last entry — preserve trailing newline):

```yaml
  security:
    desc: 'Run govulncheck + grype locally (matches CI). govulncheck is
      Go-native and reachability-aware; grype scans an in-process SBOM
      for broader CVE coverage. Both are advisory locally — CI is
      authoritative. grype is optional (install with `brew install grype`).'
    cmds:
      - go run golang.org/x/vuln/cmd/govulncheck@latest ./...
      - |
        if command -v grype >/dev/null 2>&1; then
          grype dir:.
        else
          echo "grype not installed; skipping (brew install grype)" >&2
        fi
```

- [ ] **Step 3: Verify the task runs locally**

Run:

```bash
task security
```

Expected: `govulncheck` runs, prints either "No vulnerabilities found." or a finding list. If `grype` is installed, it runs next; otherwise the skip message prints to stderr. Exit code 0 unless govulncheck flags a finding.

If govulncheck finds vulnerabilities in `cobra`/`pflag`/`mousetrap`, stop and surface them — that's the system working.

- [ ] **Step 4: Add "Security scanning" subsection to `CLAUDE.md`**

In `CLAUDE.md`, locate the "Common Commands" section's `### Using Task (Recommended)` code block. Append a `task security` line in alphabetical/logical order with the others (after `task run`, before `task dev`):

```
task security           # Run govulncheck + grype locally (matches CI)
```

Then add a new top-level subsection at the end of "Common Commands" (after the existing `### Manual Commands` block):

```markdown
### Security scanning

CI runs `govulncheck` and `grype` against every PR, every push to `main`, and on a daily cron (`.github/workflows/security.yaml`). To reproduce locally:

```bash
task security
```

`govulncheck` (Go-native, reachability-aware) is run via `go run` so no install is needed. `grype` is optional locally — install with `brew install grype`. CI is authoritative; the local target is for quick iteration on dep upgrades.

SBOMs are produced two ways:
- **Per release:** `.goreleaser.yaml` emits SPDX-JSON and CycloneDX-JSON per archive (uploaded as release artifacts).
- **Per CI run:** `security.yaml` produces a CycloneDX SBOM artifact for Grype to scan.
```

(The triple-backtick fences inside the new subsection are literal — they ship in CLAUDE.md.)

- [ ] **Step 5: Commit**

```bash
git add Taskfile.yml CLAUDE.md
git commit -m "chore: add 'task security' for local vuln scan

Wraps govulncheck (always) and grype (if installed). Matches the
checks the new security.yaml workflow runs in CI. Documents the
workflow + local repro under 'Common Commands' in CLAUDE.md."
```

- [ ] **Step 6: Push and open PR**

```bash
git push -u origin chore/local-security-task
gh pr create --fill
```

Note: this PR documents `security.yaml` *before* the workflow exists. That's fine for ordering — Task 3 lands the workflow file itself. The doc copy is forward-referencing; merge order between Task 2 and Task 3 doesn't matter (both can be open simultaneously).

---

## Task 3: Active CI scanning — `security.yaml` workflow

**Goal:** Add the GitHub Actions workflow that runs `govulncheck` and Grype on every PR, push, and daily cron. Findings upload to the GitHub Security tab as SARIF.

**Files:**
- Create: `.github/workflows/security.yaml`

**Branch:** `ci/security-scanning-workflow`

- [ ] **Step 1: Create branch**

```bash
git checkout main && git pull --ff-only
git checkout -b ci/security-scanning-workflow
```

- [ ] **Step 2: Create `.github/workflows/security.yaml`**

Write this exact content:

```yaml
name: Security

# Three-layer supply-chain scanning per
# docs/superpowers/specs/2026-05-26-sbom-vuln-scanning-design.md.
#
# govulncheck job: Go-native, reachability-aware. A finding means the
# CVE is in a code path the binary actually calls. Fails the job on
# any finding.
#
# sbom-scan job: syft produces a CycloneDX SBOM (also a workflow
# artifact), grype scans it. Fails the job on severity >= HIGH.
#
# Both jobs upload SARIF to the GitHub Security tab so findings
# surface alongside CodeQL-style results.

on:
  pull_request:
  push:
    branches: [main]
  schedule:
    # Daily ~06:17 UTC — off the hour to dodge runner congestion.
    # Catches newly-disclosed CVEs against existing deps.
    - cron: '17 6 * * *'
  workflow_dispatch:

permissions:
  contents: read

jobs:
  govulncheck:
    name: govulncheck
    runs-on: ubuntu-latest
    permissions:
      contents: read
      security-events: write  # SARIF upload to Security tab
    steps:
      - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd  # v6.0.2
        with:
          persist-credentials: false

      - uses: actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c  # v6.4.0
        with:
          go-version-file: go.mod

      - name: Install govulncheck
        run: go install golang.org/x/vuln/cmd/govulncheck@latest

      - name: Run govulncheck (SARIF)
        id: govulncheck
        # govulncheck exits non-zero on findings; we still want SARIF
        # uploaded in that case. `continue-on-error` lets us upload,
        # then re-fail the job in the final step.
        continue-on-error: true
        run: govulncheck -format sarif ./... > govulncheck.sarif

      - name: Upload SARIF
        if: always()
        uses: github/codeql-action/upload-sarif@03e4368ac7daa2bd82b3e85262f3bf87ee112f57  # v3.36.0
        with:
          sarif_file: govulncheck.sarif
          category: govulncheck

      - name: Fail job if govulncheck found something
        if: steps.govulncheck.outcome == 'failure'
        run: |
          echo "::error::govulncheck reported one or more vulnerabilities. See SARIF upload above and the Security tab for details."
          exit 1

  sbom-scan:
    name: sbom-scan
    runs-on: ubuntu-latest
    permissions:
      contents: read
      security-events: write
    steps:
      - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd  # v6.0.2
        with:
          persist-credentials: false

      - name: Generate SBOM (syft)
        uses: anchore/sbom-action@e22c389904149dbc22b58101806040fa8d37a610  # v0.24.0
        with:
          path: .
          format: cyclonedx-json
          output-file: sbom.cdx.json
          upload-artifact: true
          upload-artifact-retention: 30

      - name: Scan SBOM (Grype)
        id: grype
        uses: anchore/scan-action@e1165082ffb1fe366ebaf02d8526e7c4989ea9d2  # v7.4.0
        with:
          sbom: sbom.cdx.json
          fail-build: true
          severity-cutoff: high
          output-format: sarif

      - name: Upload SARIF
        if: always() && steps.grype.outputs.sarif != ''
        uses: github/codeql-action/upload-sarif@03e4368ac7daa2bd82b3e85262f3bf87ee112f57  # v3.36.0
        with:
          sarif_file: ${{ steps.grype.outputs.sarif }}
          category: grype
```

- [ ] **Step 3: Lint the workflow with actionlint (if installed)**

```bash
command -v actionlint >/dev/null && actionlint .github/workflows/security.yaml || echo "actionlint not installed; skipping"
```

Expected: no output (clean), or skip message. If actionlint reports issues, fix them — most likely they're real bugs.

- [ ] **Step 4: Validate YAML syntax**

```bash
python3 -c 'import yaml; yaml.safe_load(open(".github/workflows/security.yaml"))' && echo OK
```

Expected: `OK`

- [ ] **Step 5: Run zizmor if installed (action security audit)**

```bash
command -v zizmor >/dev/null && zizmor .github/workflows/security.yaml || echo "zizmor not installed; skipping"
```

Expected: no findings, or skip message. If zizmor reports `dangerous-triggers` or `unpinned-uses`, fix before committing.

- [ ] **Step 6: Commit**

```bash
git add .github/workflows/security.yaml
git commit -m "ci: add security workflow (govulncheck + grype)

Two parallel jobs on PR / push to main / daily cron / manual dispatch:
- govulncheck: Go-native, reachability-aware. Fails on any finding.
- sbom-scan: syft -> CycloneDX SBOM -> grype. Fails at severity >= HIGH.

Both upload SARIF to the GitHub Security tab. Action SHAs pinned with
version comments per existing repo convention."
```

- [ ] **Step 7: Push and open PR**

```bash
git push -u origin ci/security-scanning-workflow
gh pr create --fill
```

- [ ] **Step 8: Watch the PR's checks**

```bash
gh pr checks --watch
```

Expected: `Security / govulncheck` and `Security / sbom-scan` both pass. The new workflow runs on the PR itself because it's defined in the PR.

If `govulncheck` fails: the project has a real (reachable) vulnerability — surface it, do not bypass. Triage in a separate PR (likely a dep bump).

If `sbom-scan` fails on HIGH: same — real exposure, separate triage PR.

If either fails for tooling reasons (e.g., Anchore DB download timeout), re-run. If persistent, file an issue.

- [ ] **Step 9: After merge, verify Security tab surfaces results**

Navigate to:

```
https://github.com/yo61/go-udap/security/code-scanning
```

Expected: two tools listed under "Tool" filter — `govulncheck` and `grype`. If no findings, the categories appear with zero alerts; that's the desired state.

- [ ] **Step 10: Verify the daily cron fires within 24 h**

After merge, wait for the next scheduled run. Check:

```bash
gh run list --workflow=security.yaml --event=schedule --limit 1
```

Expected: a run exists with `event: schedule`. If after 26 hours nothing has fired, check repo activity — GitHub disables scheduled workflows in repos with no recent commits (>60 days inactivity); this repo is active so it shouldn't apply.

---

## Task 4: GoReleaser `sboms:` block

**Goal:** Have each release ship per-archive SBOMs in both SPDX-JSON and CycloneDX-JSON.

**Files:**
- Modify: `.goreleaser.yaml`

**Branch:** `feat/goreleaser-sboms`

- [ ] **Step 1: Create branch**

```bash
git checkout main && git pull --ff-only
git checkout -b feat/goreleaser-sboms
```

- [ ] **Step 2: Confirm GoReleaser and syft are installed locally**

```bash
goreleaser --version && syft --version
```

If either is missing:

```bash
brew install goreleaser/tap/goreleaser syft
```

- [ ] **Step 3: Add `sboms:` block to `.goreleaser.yaml`**

Open `.goreleaser.yaml` and add this block immediately **after** the `checksum:` block (and before `release:`):

```yaml
# Per-archive SBOMs in SPDX-JSON and CycloneDX-JSON. syft is invoked
# once per archive per format. Files upload alongside the archive in
# the GitHub release and are hashed into SHA256SUMS by the existing
# checksum block (verify via --snapshot if changing this).
#
# Design ref: docs/superpowers/specs/2026-05-26-sbom-vuln-scanning-design.md
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

- [ ] **Step 4: Validate config**

```bash
goreleaser check
```

Expected: `config is valid` (or equivalent green status). If errors, fix before continuing.

- [ ] **Step 5: Snapshot release locally**

```bash
goreleaser release --snapshot --clean --skip=publish
```

Expected: builds complete; `dist/` is populated.

- [ ] **Step 6: Verify SBOMs were produced**

```bash
ls -1 dist/*.spdx.json dist/*.cdx.json
```

Expected: one `.spdx.json` and one `.cdx.json` per archive (currently 7 archives post-format-overrides: linux/darwin/windows × amd64/arm64 minus windows-arm64).

Inspect one SBOM to confirm deps are listed:

```bash
jq '.packages[] | select(.name | test("cobra|pflag|mousetrap"))' dist/*.spdx.json | head -40
```

Expected: entries for `github.com/spf13/cobra`, `github.com/spf13/pflag`, `github.com/inconshreveable/mousetrap` with version strings.

- [ ] **Step 7: Verify `SHA256SUMS` includes the SBOM files**

```bash
grep -E '\.(spdx|cdx)\.json$' dist/SHA256SUMS
```

Expected: lines for each SBOM file.

If SBOMs are **not** in `SHA256SUMS`, fall back to extending the checksum block. Add to `.goreleaser.yaml` immediately after the existing `checksum:` block contents:

```yaml
  extra_files:
    - glob: ./dist/*.spdx.json
    - glob: ./dist/*.cdx.json
```

Re-run `goreleaser release --snapshot --clean --skip=publish` and re-verify.

- [ ] **Step 8: Clean up snapshot output**

```bash
rm -rf dist/
```

(Do not commit `dist/`; it's already in `.gitignore` — verify if uncertain with `git status`.)

- [ ] **Step 9: Commit**

```bash
git add .goreleaser.yaml
git commit -m "feat: emit per-archive SBOMs (spdx + cyclonedx)

GoReleaser's sboms block invokes syft once per archive per format.
Files upload to the GitHub release alongside the tarballs/zips and
are covered by the existing checksum block.

Two formats so downstream consumers don't have to convert: SPDX
(ISO/IEC 5962:2021, common compliance ask) and CycloneDX (OWASP,
common security-tool ask)."
```

- [ ] **Step 10: Push and open PR**

```bash
git push -u origin feat/goreleaser-sboms
gh pr create --fill
```

- [ ] **Step 11: After merge, smoke-test on the next release**

When release-please opens its next Release PR and you merge it, the GoReleaser run that follows should attach SBOM files to the release. Verify on the release page (look for `*.spdx.json` and `*.cdx.json` files alongside the archives).

If the release goes out and SBOMs are missing, file a follow-up — do not panic-revert, since unaffected machinery (binaries, checksums, homebrew cask) succeeded.

---

## Task 5: Spec / plan housekeeping

**Goal:** Once the four implementation PRs (Tasks 1–4) have merged, mark this plan and the spec as done.

**Files:**
- Modify: `docs/superpowers/specs/2026-05-26-sbom-vuln-scanning-design.md` (status header)
- Modify: `docs/superpowers/plans/2026-05-27-sbom-vuln-scanning.md` (this file — status note at top)

**Branch:** `docs/sbom-scanning-mark-done`

- [ ] **Step 1: Create branch**

```bash
git checkout main && git pull --ff-only
git checkout -b docs/sbom-scanning-mark-done
```

- [ ] **Step 2: Update spec status**

In `docs/superpowers/specs/2026-05-26-sbom-vuln-scanning-design.md`, change:

```markdown
**Status:** Draft — awaiting implementation plan
```

to:

```markdown
**Status:** Implemented (2026-MM-DD; PRs #NN, #NN, #NN, #NN)
```

Fill in the actual merge date and PR numbers from Tasks 1–4.

- [ ] **Step 3: Append a status footer to this plan**

At the top of `docs/superpowers/plans/2026-05-27-sbom-vuln-scanning.md`, immediately under the `# SBOM + vulnerability scanning Implementation Plan` heading, insert:

```markdown
> **Status:** Implemented 2026-MM-DD. Tasks 1–4 merged in PRs #NN, #NN, #NN, #NN. Follow-up (Cosign + SLSA) tracked in #93.
```

- [ ] **Step 4: Commit**

```bash
git add docs/superpowers/specs/2026-05-26-sbom-vuln-scanning-design.md \
        docs/superpowers/plans/2026-05-27-sbom-vuln-scanning.md
git commit -m "docs: mark sbom + scanning spec and plan as implemented"
```

- [ ] **Step 5: Push and open PR**

```bash
git push -u origin docs/sbom-scanning-mark-done
gh pr create --fill
```

---

## Cross-task verification (manual, after Tasks 1–4 merged)

Run this after all four implementation PRs are merged to main:

- [ ] **Cron schedule has fired**

```bash
gh run list --workflow=security.yaml --event=schedule --limit 1
```

Expected: one run within last 24 hours.

- [ ] **Dependabot has scanned at least one ecosystem**

Visit `https://github.com/yo61/go-udap/network/updates` — each of the four ecosystems shows a "Last checked" timestamp within the last week.

- [ ] **Security tab shows no surprise alerts**

Visit `https://github.com/yo61/go-udap/security/code-scanning` — filter by Tool. Both `govulncheck` and `grype` are listed. Triage any open alerts (most will be `LOW`/`MEDIUM` Grype findings that don't fail CI).

- [ ] **Next release ships SBOMs**

After the next release-please PR merges and GoReleaser runs, the release page shows `.spdx.json` and `.cdx.json` files per archive.

---

## Rollback (per component)

| Failure | Rollback |
|---|---|
| Task 1 — Dependabot config breaks | `git revert` the merge commit; GitHub re-parses on the next push and reverts to pre-existing pre-commit-only state. Open Dependabot PRs from the new ecosystems can be closed manually. |
| Task 2 — Taskfile/CLAUDE.md docs | Trivial revert; no runtime effect. |
| Task 3 — security workflow noisy or buggy | `git revert` the merge commit; CI returns to pre-change behaviour. Existing Security-tab findings remain (GitHub doesn't auto-clear) but become stale. Alternatively, delete the workflow file in a follow-up PR. |
| Task 4 — GoReleaser SBOM block | `git revert` the merge commit. The release that already shipped keeps its SBOMs; future releases ship without them. No downstream consumer breakage (SBOM is additive). |

---

## What this plan deliberately does NOT do

These are spec-level non-goals re-stated for the implementer:

- **No Cosign signing.** Tracked in [#93](https://github.com/yo61/go-udap/issues/93). The SBOMs land in a state ready to be signed by the follow-up.
- **No SLSA build provenance.** Same issue.
- **No Trivy.** Redundant with Grype for a 3-dep CLI.
- **No committed SBOM file in the repo.** Release-attached + CI artifact covers it.
- **No issue-on-failure for cron findings.** Workflow failure + email is enough for a low-traffic repo.
- **No blocking on LOW/MEDIUM Grype severities.** Noisy. Re-evaluate later if posture shifts.
