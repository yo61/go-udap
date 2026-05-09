# Release Automation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace go-udap's manual versioning + manual-trigger build workflow with automated semantic-release tagging and GoReleaser-driven cross-platform binary publishing, with binary `--version` output stamped from the release tag.

**Architecture:** Three GitHub Actions workflows — `ci.yaml` (PR + main tests on linux/macos/windows), `release.yaml` (semantic-release: tag, CHANGELOG, GitHub release), `goreleaser.yaml` (tag-triggered cross-compile + asset upload). Binary `Version` lives in source as a `var` defaulting to `"dev"`, overridden at build time via `-ldflags -X` from `git describe` (Taskfile) or `{{.Version}}` (GoReleaser).

**Tech Stack:** Go 1.26, Task (Taskfile.yml), semantic-release 23.x with conventional-commits preset, GoReleaser v2, GitHub Actions.

**Spec reference:** `docs/superpowers/specs/2026-05-09-release-automation-design.md`

**Branch:** All work happens on `robin/release-automation` (already created and contains the spec).

---

## Prerequisites

Before Task 1, ensure local tooling needed for verification is installed. These are one-time installs.

- [ ] **P1: Install `actionlint` and `goreleaser`**

```bash
brew install actionlint goreleaser
```

Verify:
```bash
command -v actionlint goreleaser
```
Expected: both paths print, no errors.

`shellcheck`, `task`, `go`, `npx` are already present.

---

## Task 1: Wire `Version` variable in `cli` package

**Files:**
- Modify: `cli/cli.go:13-15` (change `const Version` to `var Version`)
- Modify: `cli/cli_test.go` (add overridability test)

- [ ] **Step 1: Add a failing test that proves `Version` is overridable**

Append to `cli/cli_test.go`:

```go
func TestVersionVariableIsOverridable(t *testing.T) {
	original := Version
	t.Cleanup(func() { Version = original })
	Version = "test-1.2.3"

	var stdout, stderr bytes.Buffer
	if err := Run([]string{"--version"}, &stdout, &stderr); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(stdout.String(), "test-1.2.3") {
		t.Errorf("expected version output to contain 'test-1.2.3', got %q", stdout.String())
	}
}
```

- [ ] **Step 2: Run the test to confirm it fails to compile**

```bash
go test ./cli/ -run TestVersionVariableIsOverridable
```
Expected: compile error — "cannot assign to Version (declared const)".

- [ ] **Step 3: Change `const Version` to `var Version` with `"dev"` default**

In `cli/cli.go`, replace lines 13–15:

```go
// Version is the binary version string, surfaced by --version.
// Updated manually for now; release tooling can wire this to the git tag later.
const Version = "0.2.0"
```

with:

```go
// Version is the binary version string, surfaced by --version.
// Set at build time via -ldflags "-X go-udap/cli.Version=...".
// Defaults to "dev" for un-stamped local builds (e.g. `go install`).
var Version = "dev"
```

- [ ] **Step 4: Run all cli tests**

```bash
go test -race ./cli/
```
Expected: PASS, including `TestVersionVariableIsOverridable` and existing `TestRunVersionFlag`.

- [ ] **Step 5: Commit**

```bash
git add cli/cli.go cli/cli_test.go
git commit -m "refactor(cli): make Version a var so release builds can stamp it

const → var so -ldflags -X can override the value at build time. Default
changed from \"0.2.0\" to \"dev\" so un-stamped builds (e.g. go install)
clearly identify themselves as dev builds."
```

---

## Task 2: Wire `Version` variable in `cmd/mocksbr`

**Files:**
- Modify: `cmd/mocksbr/main.go` (add `var Version`, change `--version` print)

- [ ] **Step 1: Add `var Version = "dev"` at package level**

In `cmd/mocksbr/main.go`, immediately after `var errUsage = errors.New("usage error")` (around line 42), add:

```go
// Version is the binary version string, surfaced by --version.
// Set at build time via -ldflags "-X main.Version=...".
// Defaults to "dev" for un-stamped local builds.
var Version = "dev"
```

- [ ] **Step 2: Update the `--version` print to use `Version`**

In `cmd/mocksbr/main.go`, replace:

```go
	if *showVer {
		fmt.Fprintln(stdout, "mocksbr (dev)")
		return nil
	}
```

with:

```go
	if *showVer {
		fmt.Fprintf(stdout, "mocksbr %s\n", Version)
		return nil
	}
```

- [ ] **Step 3: Verify default build prints `"dev"`**

```bash
go build -o /tmp/mocksbr ./cmd/mocksbr && /tmp/mocksbr --version
```
Expected output: `mocksbr dev`

- [ ] **Step 4: Verify ldflag injection works**

```bash
go build -ldflags "-X main.Version=test-1.2.3" -o /tmp/mocksbr ./cmd/mocksbr && /tmp/mocksbr --version
```
Expected output: `mocksbr test-1.2.3`

- [ ] **Step 5: Run the full test suite to confirm nothing else broke**

```bash
go test -race ./...
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add cmd/mocksbr/main.go
git commit -m "refactor(cmd/mocksbr): make Version a var so release builds can stamp it

Mirrors the cli/cli.go change: var Version = \"dev\" overrideable via
-ldflags -X main.Version=... at build time. mocksbr shares go-udap's
git tag space (no separate tag prefix today)."
```

---

## Task 3: Update `Taskfile.yml` to inject version from git

**Files:**
- Modify: `Taskfile.yml` (add `VERSION` var, fold into `LDFLAGS`, add `build:mocksbr` task)

- [ ] **Step 1: Replace the `vars:` block**

In `Taskfile.yml`, replace lines 3–6:

```yaml
vars:
  BINARY_NAME: go-udap
  LDFLAGS: -s -w
  BUILD_FLAGS: -ldflags="{{.LDFLAGS}}" -trimpath
```

with:

```yaml
vars:
  BINARY_NAME: go-udap
  VERSION:
    sh: git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' || echo dev
  LDFLAGS: -s -w -X go-udap/cli.Version={{.VERSION}}
  MOCKSBR_LDFLAGS: -s -w -X main.Version={{.VERSION}}
  BUILD_FLAGS: -ldflags="{{.LDFLAGS}}" -trimpath
  MOCKSBR_BUILD_FLAGS: -ldflags="{{.MOCKSBR_LDFLAGS}}" -trimpath
```

`VERSION` is sourced once at task-graph evaluation; `MOCKSBR_LDFLAGS` exists because mocksbr's `Version` lives in `package main` (not `package cli`) so the `-X` target differs.

- [ ] **Step 2: Add a `build:mocksbr` task**

In `Taskfile.yml`, append after the `build:linux-arm64` task block (before `build:all`):

```yaml
  build:mocksbr:
    desc: Build mocksbr binary for current platform with version stamped
    cmds:
      - go build {{.MOCKSBR_BUILD_FLAGS}} -o mocksbr ./cmd/mocksbr
    sources:
      - "**/*.go"
    generates:
      - mocksbr
```

- [ ] **Step 3: Run `task build` and verify version output**

```bash
task build && ./go-udap --version
```
Expected output (current branch is past v0.2.0):
```
go-udap 0.2.0-N-gXXXXXXX[-dirty]
```
(N is commit count past v0.2.0; XXXXXXX is short hash; `-dirty` only if working tree has uncommitted changes.)

- [ ] **Step 4: Run `task build:mocksbr` and verify**

```bash
task build:mocksbr && ./mocksbr --version
```
Expected output: same version format prefixed with `mocksbr `.

- [ ] **Step 5: Verify `go install` / non-Taskfile builds still default to `"dev"`**

```bash
go build -o /tmp/go-udap-plain . && /tmp/go-udap-plain --version
```
Expected output: `go-udap dev`

This confirms the source default kicks in when `-ldflags` aren't supplied.

- [ ] **Step 6: Commit**

```bash
git add Taskfile.yml
git commit -m "build(taskfile): inject version into binaries via git describe

task build / task build:mocksbr now stamp Version at link time using
\`git describe --tags --always --dirty\`, so local --version output
matches the nearest tag plus commit-distance suffix. Source default
remains \"dev\" for builds that bypass the Taskfile (e.g. go install)."
```

---

## Task 4: Add `.releaserc.json` (semantic-release config)

**Files:**
- Create: `.releaserc.json`

- [ ] **Step 1: Create `.releaserc.json`**

Write to repo root:

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

- [ ] **Step 2: Validate JSON parses**

```bash
python3 -c 'import json; json.load(open(".releaserc.json"))' && echo OK
```
Expected: `OK`.

- [ ] **Step 3: Dry-run semantic-release locally to see the proposed version**

```bash
GITHUB_TOKEN=fake-token-for-dry-run \
  npx -p semantic-release@23 \
      -p @semantic-release/changelog@6 \
      -p @semantic-release/git@10 \
      -p conventional-changelog-conventionalcommits@7 \
      semantic-release --dry-run --no-ci 2>&1 | tail -40
```
Expected: log lines including `The next release version is X.Y.Z` (likely `0.3.0` since v0.2.0 currently exists). The fake token is fine — `--dry-run` doesn't push to GitHub. Errors about `GITHUB_TOKEN` invalidity at the publish step are expected and ignored; we only care about the analysis output.

Record the version semantic-release computed; it confirms the config parses correctly.

- [ ] **Step 4: Commit**

```bash
git add .releaserc.json
git commit -m "build: add semantic-release config

Conventional-commits preset; tagFormat v\${version}; commits CHANGELOG.md
back to main via @semantic-release/git; creates GitHub release via
@semantic-release/github. Mirrors the Synadia reference workflow."
```

---

## Task 5: Add `.goreleaser.yaml` (cross-compile + asset config)

**Files:**
- Create: `.goreleaser.yaml`
- Modify: `.gitignore` (add `dist/` for GoReleaser output)

- [ ] **Step 1: Create `.goreleaser.yaml`**

Write to repo root:

```yaml
version: 2

before:
  hooks:
    - go mod tidy

builds:
  - id: go-udap
    main: .
    binary: go-udap
    env:
      - CGO_ENABLED=0
    flags:
      - -trimpath
    ldflags:
      - -s -w
      - -X go-udap/cli.Version={{.Version}}
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ignore:
      - goos: windows
        goarch: arm64

archives:
  - id: go-udap
    name_template: >-
      {{ .ProjectName }}_{{ .Version }}_
      {{- if eq .Os "darwin" }}macos
      {{- else }}{{ .Os }}{{ end }}_
      {{- if eq .Arch "amd64" }}x86_64
      {{- else }}{{ .Arch }}{{ end }}
    files:
      - LICENSE
      - README.md
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: 'SHA256SUMS'
  algorithm: sha256

changelog:
  disable: true
```

- [ ] **Step 2: Add `dist/` to `.gitignore`**

Append to `.gitignore`:

```
# GoReleaser output
dist/
```

- [ ] **Step 3: Validate config with `goreleaser check`**

```bash
goreleaser check
```
Expected: `checks passed`.

- [ ] **Step 4: Run a snapshot release to build all 5 binaries**

```bash
goreleaser release --snapshot --clean
```
Expected: builds succeed; `dist/` contains 5 archives (4 tar.gz + 1 zip), `SHA256SUMS`, and per-target binary subdirectories.

- [ ] **Step 5: Verify build output and the host binary's `--version`**

List what GoReleaser produced:
```bash
find dist -maxdepth 2 -type f \( -name 'go-udap' -o -name 'go-udap.exe' -o -name '*.tar.gz' -o -name '*.zip' -o -name 'SHA256SUMS' \) | sort
```
Expected: 5 archives (4 `.tar.gz`, 1 `.zip`), 1 `SHA256SUMS`, plus 5 raw binaries inside per-target subdirectories.

Run the host binary (matches current OS+arch) to confirm version stamping:
```bash
HOST_BIN=$(find dist -type f -name go-udap -path "*$(go env GOOS)*$(go env GOARCH)*" | head -1)
echo "Running: $HOST_BIN"
"$HOST_BIN" --version
```
Expected output: a line like `go-udap 0.0.0-SNAPSHOT-abcd123` (GoReleaser's snapshot-mode version; exact format varies by version but always non-empty and non-`dev`). This confirms `-X go-udap/cli.Version={{.Version}}` is wired through end-to-end.

- [ ] **Step 6: Commit**

```bash
git add .goreleaser.yaml .gitignore
git commit -m "build: add GoReleaser config for cross-platform release builds

Builds 5 targets (linux/darwin/windows × amd64/arm64, minus windows-arm64),
stamps Version via -ldflags -X go-udap/cli.Version={{.Version}}, produces
tar.gz archives (zip on windows) plus SHA256SUMS. Snapshot mode (used
locally and in CI smoke checks) and tag-driven release mode share the
same config."
```

---

## Task 6: Add `.github/workflows/ci.yaml` (matrix tests)

**Files:**
- Create: `.github/workflows/ci.yaml`

- [ ] **Step 1: Create `ci.yaml`**

Write to `.github/workflows/ci.yaml`:

```yaml
name: CI

on:
  pull_request:
  push:
    branches: [main]

jobs:
  test:
    name: test (${{ matrix.os }})
    strategy:
      fail-fast: false
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - run: go vet ./...
      - run: go test -race ./...
```

- [ ] **Step 2: Validate with actionlint**

```bash
actionlint .github/workflows/ci.yaml
```
Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci.yaml
git commit -m "ci: add matrix test workflow on linux/macos/windows

Runs on every PR and on pushes to main. Race detector on (-race) since
mocksbr's integration tests exercise real loopback UDP via SpawnMock and
hit goroutines we want to keep race-free."
```

---

## Task 7: Add `.github/workflows/release.yaml` (semantic-release)

**Files:**
- Create: `.github/workflows/release.yaml`

- [ ] **Step 1: Create `release.yaml`**

Write to `.github/workflows/release.yaml`:

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

- [ ] **Step 2: Validate with actionlint**

```bash
actionlint .github/workflows/release.yaml
```
Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/release.yaml
git commit -m "ci: add semantic-release workflow

Triggered on push to main (path-filtered to source/config changes only)
and via workflow_dispatch. Uses a GitHub App token so @semantic-release/git
can commit CHANGELOG.md back to main even with branch protection enabled.
Requires SEMANTIC_RELEASE_APP_ID and SEMANTIC_RELEASE_APP_PRIVATE_KEY
secrets configured on the repo."
```

---

## Task 8: Add `.github/workflows/goreleaser.yaml` (tag-triggered build)

**Files:**
- Create: `.github/workflows/goreleaser.yaml`

- [ ] **Step 1: Create `goreleaser.yaml`**

Write to `.github/workflows/goreleaser.yaml`:

```yaml
name: GoReleaser

on:
  push:
    tags:
      # Full SemVer match across the four valid forms:
      #   vX.Y.Z              (e.g. v0.3.0)
      #   vX.Y.Z-prerelease   (e.g. v0.3.0-rc.1)
      #   vX.Y.Z+build        (e.g. v0.3.0+build.123)
      #   vX.Y.Z-prerelease+build
      # Rejects junk like "vfoo", "v0.3", "v1.0-rc1" (missing patch).
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
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 2: Validate with actionlint**

```bash
actionlint .github/workflows/goreleaser.yaml
```
Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/goreleaser.yaml
git commit -m "ci: add GoReleaser workflow triggered on v* tag pushes

Builds the cross-platform matrix and uploads archives + SHA256SUMS to
the GitHub release that semantic-release just created. Uses the default
GITHUB_TOKEN (only needs contents:write); no GitHub App needed since
this step doesn't push commits, only release assets."
```

---

## Task 9: Remove the now-redundant `build.yml`

**Files:**
- Delete: `.github/workflows/build.yml`

- [ ] **Step 1: Confirm no other references**

```bash
rg -n 'build\.yml|workflows/build' --hidden
```
Expected: only matches inside `build.yml` itself.

- [ ] **Step 2: Delete the file**

```bash
git rm .github/workflows/build.yml
```

- [ ] **Step 3: Commit**

```bash
git commit -m "ci: remove manual build.yml superseded by goreleaser.yaml

The new release flow handles cross-platform builds automatically on tag
push; one-off cross-platform builds remain available via \`task build:all\`
locally. workflow_dispatch on release.yaml covers the rare ad-hoc case
where a manual GitHub-side trigger is wanted."
```

---

## Task 10: Document setup prerequisites in `DEVELOPMENT.md`

**Files:**
- Modify: `DEVELOPMENT.md` (append a "Releasing" section)

- [ ] **Step 1: Append a "Releasing" section**

Append to the end of `DEVELOPMENT.md`:

```markdown
## Releasing

Releases are fully automated — every push to `main` that touches Go
source, `go.mod`/`go.sum`, or release configuration triggers
`.github/workflows/release.yaml`. The workflow uses semantic-release to
compute the next version from conventional-commits since the last tag,
then `.github/workflows/goreleaser.yaml` (triggered by the resulting tag
push) cross-compiles and uploads release artifacts.

Binary `--version` output is stamped from the release tag, so a binary
downloaded from a GitHub release always self-identifies as the version
of that release.

### One-time setup

Two secrets and one app must be configured on the repository before the
first release can run:

1. **GitHub App** — create a GitHub App scoped to this repo with
   permissions:
   - `Contents: Read & write`
   - `Pull requests: Read & write`
   - `Metadata: Read`

   Install the App on the `go-udap` repository and generate a private
   key.

2. **Repository secrets** — add under Settings → Secrets and variables
   → Actions:
   - `SEMANTIC_RELEASE_APP_ID` — the App's numeric ID
   - `SEMANTIC_RELEASE_APP_PRIVATE_KEY` — the full PEM contents of the
     private key

3. **Branch protection on `main`** — require CI to pass before merge,
   and add the GitHub App to the bypass list so
   `@semantic-release/git` can land CHANGELOG.md commits.

### Local version checks

`task build` stamps the binary using `git describe --tags --always
--dirty`. Output examples:

| Build state                      | `go-udap --version`         |
| -------------------------------- | --------------------------- |
| On a tagged commit `v0.3.0`      | `go-udap 0.3.0`             |
| 2 commits past v0.3.0            | `go-udap 0.3.0-2-gabc1234`  |
| Same with uncommitted changes    | `go-udap 0.3.0-2-gabc1234-dirty` |
| `go install` / no Taskfile build | `go-udap dev`               |

### Smoke-testing the release pipeline locally

```bash
# Validates .goreleaser.yaml without uploading
goreleaser check

# Builds all 5 targets locally and writes them to dist/
goreleaser release --snapshot --clean
```

### Conventional commits drive versioning

- `fix:` → patch bump (0.3.0 → 0.3.1)
- `feat:` → minor bump (0.3.0 → 0.4.0)
- `feat!:` or `BREAKING CHANGE:` footer → major bump (0.3.0 → 1.0.0)

`chore:`, `docs:`, `refactor:`, `test:`, `build:`, and `ci:` do not
trigger releases (they're committed but the path filter / no-version
check means semantic-release exits without tagging).
```

- [ ] **Step 2: Verify markdown renders OK**

```bash
# Sanity check that no nested code-block markers escaped:
grep -c '^```' DEVELOPMENT.md
```
Expected: an even number (every fence has a closer).

- [ ] **Step 3: Commit**

```bash
git add DEVELOPMENT.md
git commit -m "docs: document release pipeline and one-time setup

Adds a Releasing section to DEVELOPMENT.md covering the GitHub App
setup, branch-protection config, local version-check commands, and
which conventional-commit types trigger releases."
```

---

## Hand-off — actions that require user interaction

The implementation tasks above produce a working pipeline on the
`robin/release-automation` branch. The remaining steps in the spec's
cutover plan require user action and are not implementable as code
changes:

1. **One-time GitHub App setup** (per Task 10's documentation)
2. **Add the two repository secrets**
3. **Configure branch protection on `main`**
4. **Open PR from `robin/release-automation` to `main`**
5. **Wait for CI to pass** (validates the new `ci.yaml` works)
6. **Final dry-run of semantic-release against `main`** to predict the
   first release version
7. **Delete the existing v0.2.0 tag and GitHub release** so the first
   automated release is v0.2.0:
   ```bash
   git push origin :refs/tags/v0.2.0
   # Delete the v0.2.0 GitHub release via the UI
   ```
8. **Merge the PR**
9. **Verify post-merge:**
   - `release.yaml` ran, committed CHANGELOG.md, created tag v0.2.0
   - `goreleaser.yaml` ran on the tag and uploaded 5 archives + SHA256SUMS
   - Each downloaded binary's `--version` matches `0.2.0` exactly
