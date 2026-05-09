# Release automation — design spec

**Date:** 2026-05-09
**Status:** Draft, pending implementation plan

## Goal

Replace go-udap's manual versioning + manual-trigger build workflow with an automated release pipeline that:

1. Determines the next semver version from conventional-commits since the last tag.
2. Tags, releases, and publishes pre-built binaries on every code change merged to `main`.
3. Stamps the binary's `--version` output with the exact tag it was built from, so `go-udap --version` is always traceable to a specific release.
4. Mirrors the Synadia `terraform-azurerm-nats_node` reference workflow for the version/release-notes half (semantic-release + GitHub App for signed CHANGELOG.md commits).
5. Versions the `cmd/mocksbr` test binary the same way, sharing go-udap's tag space (no separate tag prefix today).

## Non-goals

- No goreleaser-driven Homebrew, scoop, or apt/deb packaging — that's a separate, later decision.
- No mocksbr release artifacts on the GitHub release page. mocksbr is a dev/test binary; only its source-built `--version` output is wired up for traceability.
- No separate tag space for mocksbr (e.g. `mocksbr/v*`). If mocksbr ever graduates to an installed tool, this can be added later without rework to go-udap.

## Architecture

Three workflows, each with a single responsibility:

```
.github/workflows/
  ci.yaml          PRs + main pushes. Tests on linux/macos/windows.
  release.yaml     Main pushes. semantic-release: tag + release + CHANGELOG.
  goreleaser.yaml  Tag pushes matching vX.Y.Z. Cross-compile + asset upload.
```

Trigger flow on a merged PR:

```
push → main
  ├─ ci.yaml        runs (matrix tests on ubuntu/macos/windows)
  └─ release.yaml   runs (semantic-release computes vX.Y.Z, commits CHANGELOG.md
                    via @semantic-release/git, creates the tag and release stub)
                       │
                       └─ tag push triggers goreleaser.yaml
                          (builds 5 binaries with -X go-udap/cli.Version=X.Y.Z,
                          uploads tar.gz/zip + SHA256SUMS to the existing release)
```

`ci.yaml` and `release.yaml` run in parallel (no `needs:` gate). Branch protection on `main` requiring CI to pass before merge is the gate that prevents broken code from reaching `release.yaml`. Adding an explicit `workflow_run` gate would only add latency.

## Components

### `cli/cli.go` — version variable

Change:
```go
const Version = "0.2.0"
```
to:
```go
// Version is the binary version string, surfaced by --version.
// Set at build time via -ldflags "-X go-udap/cli.Version=...".
// Defaults to "dev" for un-stamped local builds (e.g. go install).
var Version = "dev"
```

`const` → `var` because Go's `-X` linker flag only operates on package-level string variables.

### `cmd/mocksbr/main.go` — version variable

Add at package level:
```go
var Version = "dev"
```

Change the `*showVer` block from:
```go
fmt.Fprintln(stdout, "mocksbr (dev)")
```
to:
```go
fmt.Fprintf(stdout, "mocksbr %s\n", Version)
```

### `Taskfile.yml` — local builds inject version from git

Add `VERSION` var derived from `git describe`, fold into `LDFLAGS`:

```yaml
vars:
  BINARY_NAME: go-udap
  VERSION:
    sh: git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' || echo dev
  LDFLAGS: -s -w -X go-udap/cli.Version={{.VERSION}}
  BUILD_FLAGS: -ldflags="{{.LDFLAGS}}" -trimpath
```

Add a `build:mocksbr` task using `-X main.Version=...` (mocksbr's `Version` lives in `package main`, not `package cli`).

`sed 's/^v//'` strips the leading `v` from `git describe` output so local builds match GoReleaser's `{{.Version}}` format (which strips it too).

Output examples:

| Build state | `go-udap --version` |
|---|---|
| Built on tagged commit `v0.3.0` (release or local) | `go-udap 0.3.0` |
| 2 commits past v0.3.0 (local) | `go-udap 0.3.0-2-gabc1234` |
| Same, dirty working tree (local) | `go-udap 0.3.0-2-gabc1234-dirty` |
| Built outside git / no tags / `go install` | `go-udap dev` |

`task release` (existing, UPX-packs binaries locally) keeps working unchanged — UPX is a local-only convenience and is intentionally not used in CI/release builds (avoids antivirus false positives on Windows).

### `.releaserc.json` — semantic-release config

```json
{
  "branches": ["main"],
  "ci": false,
  "tagFormat": "v${version}",
  "plugins": [
    ["@semantic-release/commit-analyzer", { "preset": "conventionalcommits" }],
    ["@semantic-release/release-notes-generator", { "preset": "conventionalcommits" }],
    ["@semantic-release/changelog", {
      "changelogFile": "CHANGELOG.md",
      "changelogTitle": "# Changelog\n\nAll notable changes to this project will be documented in this file."
    }],
    ["@semantic-release/git", {
      "assets": ["CHANGELOG.md"],
      "message": "chore(release): version ${nextRelease.version} [skip ci]\n\n${nextRelease.notes}"
    }],
    ["@semantic-release/github", { "labels": false, "releasedLabels": false }]
  ]
}
```

Plugin order is load-bearing: `changelog` writes the file, `git` commits it back, `github` creates the release referencing the just-pushed tag.

### `.goreleaser.yaml` — build matrix and asset layout

```yaml
version: 2

before:
  hooks:
    - go mod tidy

builds:
  - id: go-udap
    main: .
    binary: go-udap
    env: [CGO_ENABLED=0]
    flags: [-trimpath]
    ldflags:
      - -s -w
      - -X go-udap/cli.Version={{.Version}}
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ignore:
      - { goos: windows, goarch: arm64 }

archives:
  - id: go-udap
    name_template: >-
      {{ .ProjectName }}_{{ .Version }}_
      {{- if eq .Os "darwin" }}macos
      {{- else }}{{ .Os }}{{ end }}_
      {{- if eq .Arch "amd64" }}x86_64
      {{- else }}{{ .Arch }}{{ end }}
    files: [LICENSE, README.md]
    format_overrides:
      - { goos: windows, format: zip }

checksum:
  name_template: 'SHA256SUMS'
  algorithm: sha256

changelog:
  disable: true
```

Release-page assets for tag `v0.3.0`:

```
go-udap_0.3.0_linux_x86_64.tar.gz
go-udap_0.3.0_linux_arm64.tar.gz
go-udap_0.3.0_macos_x86_64.tar.gz
go-udap_0.3.0_macos_arm64.tar.gz
go-udap_0.3.0_windows_x86_64.zip
SHA256SUMS
```

`changelog: disable: true` because semantic-release already produces release notes.

### `.github/workflows/ci.yaml`

```yaml
name: CI
on:
  pull_request:
  push:
    branches: [main]

jobs:
  test:
    strategy:
      fail-fast: false
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version-file: go.mod }
      - run: go vet ./...
      - run: go test -race ./...
```

Race detector on (`-race`) — `mocksbr/integration_test.go` exercises real loopback UDP and goroutines via `SpawnMock`, so race-free behaviour is part of what's being tested.

The three GitHub-hosted runners cover three of the five build targets natively (linux-amd64, darwin-arm64, windows-amd64). The other two (linux-arm64, darwin-amd64) get build-only coverage in `goreleaser.yaml` — they cross-compile cleanly or the release fails.

### `.github/workflows/release.yaml`

```yaml
name: Release
on:
  workflow_dispatch:
  push:
    branches: [main]
    paths:
      - '**/*.go'
      - 'go.mod'
      - 'go.sum'
      - '.github/workflows/release.yaml'
      - '.goreleaser.yaml'
      - '.releaserc.json'

jobs:
  release:
    name: Release
    runs-on: ubuntu-latest
    if: github.repository_owner == 'robinbowes'
    steps:
      - id: create_token
        uses: actions/create-github-app-token@v2
        with:
          app-id: ${{ secrets.SEMANTIC_RELEASE_APP_ID }}
          private-key: ${{ secrets.SEMANTIC_RELEASE_APP_PRIVATE_KEY }}
          owner: ${{ github.repository_owner }}
      - uses: actions/checkout@v4
        with:
          persist-credentials: false
          fetch-depth: 0
      - uses: cycjimmy/semantic-release-action@v4
        with:
          semantic_version: 23.0.2
          extra_plugins: |
            @semantic-release/changelog@6.0.3
            @semantic-release/git@10.0.1
            conventional-changelog-conventionalcommits@7.0.2
        env:
          GITHUB_TOKEN: ${{ steps.create_token.outputs.token }}
```

Path filters skip releases on doc-only or task-config-only changes. The `if: github.repository_owner == 'robinbowes'` guard prevents forks from running release machinery.

### `.github/workflows/goreleaser.yaml`

```yaml
name: GoReleaser
on:
  push:
    tags:
      # Full SemVer: vX.Y.Z, vX.Y.Z-prerelease, vX.Y.Z+build, vX.Y.Z-prerelease+build
      - 'v[0-9]+.[0-9]+.[0-9]+'
      - 'v[0-9]+.[0-9]+.[0-9]+-*'
      - 'v[0-9]+.[0-9]+.[0-9]+\+*'
      - 'v[0-9]+.[0-9]+.[0-9]+-*\+*'
permissions:
  contents: write
jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - uses: actions/setup-go@v5
        with: { go-version-file: go.mod }
      - uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

Uses the default `GITHUB_TOKEN` — only needs `contents: write` on the release that semantic-release just created. No GitHub App needed for this step.

### Files to remove

- `.github/workflows/build.yml` — superseded by `goreleaser.yaml`. The use case it covered (manual cross-platform binary builds) is now covered by either `task build:all` locally or by triggering `release.yaml` via `workflow_dispatch`.

## One-time setup prerequisites

Before the first release, the user must:

1. **Create a GitHub App** scoped to `robinbowes/go-udap`:
   - Permissions: `Contents: read & write`, `Pull requests: read & write`, `Metadata: read`
   - Install on the `go-udap` repository
   - Generate a private key
2. **Add repository secrets:**
   - `SEMANTIC_RELEASE_APP_ID` — the GitHub App's numeric ID
   - `SEMANTIC_RELEASE_APP_PRIVATE_KEY` — the full private-key PEM contents
3. **Configure branch protection on `main`:**
   - Require CI to pass before merge (gates broken code out of `release.yaml`)
   - Allow the GitHub App to push to `main` (so `@semantic-release/git` can land CHANGELOG.md commits)

Both secrets and branch protection are configured via the repo's GitHub settings UI — out of scope for what this spec ships in code.

## Cutover plan

Order matters. The goal is to make v0.2.0 the first release of the new system, so the workflow's traceability story is clean from the very first published binary.

1. **Set up the GitHub App and secrets** per "One-time setup prerequisites" above. Do this first so the workflow has what it needs to run on merge.

2. **Make all code and config changes on a feature branch** (single PR):
   - `cli/cli.go`: `const Version = "0.2.0"` → `var Version = "dev"`
   - `cmd/mocksbr/main.go`: add `var Version = "dev"`, change `--version` print to use `Version`
   - `Taskfile.yml`: add `VERSION` var sourced from `git describe`, fold into `LDFLAGS`, add `build:mocksbr` task
   - Add `.releaserc.json`
   - Add `.goreleaser.yaml`
   - Add `.github/workflows/ci.yaml`
   - Add `.github/workflows/release.yaml`
   - Add `.github/workflows/goreleaser.yaml`
   - Remove `.github/workflows/build.yml`

3. **Open the PR.** `ci.yaml` runs against the matrix; tests must pass on all three runners.

4. **Dry-run semantic-release locally** to verify the version it would compute when run on `main`:
   ```
   npx -p semantic-release -p @semantic-release/changelog -p @semantic-release/git \
     -p conventional-changelog-conventionalcommits semantic-release --dry-run --no-ci
   ```
   Expected against current state: v0.3.0 (v0.2.0 still exists, commits since are minor-bump). Expected after step 5: v0.2.0 (v0.1.0 is the last tag, commits since are minor-bump).

5. **Delete the existing v0.2.0 tag and GitHub release.** `git push origin :refs/tags/v0.2.0` and remove the release via the GitHub UI. After this, semantic-release will compute v0.2.0 as the next version on its next run.

6. **Merge the PR.** This triggers the first end-to-end run. Verify:
   - `release.yaml` ran successfully and committed `CHANGELOG.md` back to `main` under the GitHub App's identity.
   - The v0.2.0 tag and GitHub release exist.
   - `goreleaser.yaml` ran on the tag push and uploaded 5 archives + `SHA256SUMS`.
   - Each downloaded binary's `--version` output matches `0.2.0` exactly.

## Risks and open questions

- **GitHub App push to a protected `main`**: needs the App to be in the branch protection's "allowed actors" list. If overlooked, `@semantic-release/git` fails. Mitigation: verify in step 2 of cutover.
- **First release computes wrong version**: if commits between v0.1.0 and HEAD include a `feat!:` or `BREAKING CHANGE:` footer, semantic-release would compute v1.0.0 rather than v0.2.0. Mitigation: dry-run in step 7 catches this; if it happens, decide whether to accept v1.0.0 or amend the offending commit's footer before re-running.
- **Concurrent `release.yaml` and `ci.yaml` runs**: both trigger on push to main. If `ci.yaml` fails post-merge, `release.yaml` may still publish a broken release. Mitigation: branch protection requiring CI to pass *before* merge — the post-merge `ci.yaml` run is then a confirmation, not a gate.
- **Asset naming convention**: the spec uses GoReleaser's cleaned-up scheme (`linux_x86_64`, `macos_arm64`). The previous `build.yml` used the raw Go scheme (`linux-amd64`, `darwin-arm64`). If consumers have scripts that download by URL, the change is breaking. No known consumers today; flagged for awareness.
- **mocksbr cross-compilation in CI**: `goreleaser.yaml` does not build mocksbr. If a future commit breaks `cmd/mocksbr` compilation on linux-arm64 specifically, neither `ci.yaml` (no arm64 runner) nor `goreleaser.yaml` (mocksbr not in builds) catches it. Acceptable today: mocksbr is dev-only, and `task build:mocksbr` locally covers the common case. If this becomes a real risk, add a `mocksbr` build entry to `.goreleaser.yaml` with `archives: skip` so it cross-compiles for all 5 targets without producing release assets.

## Out of scope (for this spec)

- Homebrew tap / scoop bucket / apt repo publishing.
- Signed binaries (codesign for macOS, signtool for Windows).
- SBOM generation, provenance attestation, sigstore signing.
- Test-coverage reporting upload (Codecov etc.).
- Dependabot / Renovate auto-updates of action SHAs.

Each of the above is a one-or-two-line addition to the relevant workflow if/when needed; none affects the core architecture above.
