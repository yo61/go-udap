# Homebrew Tap Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Distribute `go-udap` via Homebrew on macOS and Linux through a self-owned tap (`yo61/homebrew-tap`), with the formula auto-published by goreleaser on every tag.

**Architecture:** Two repos cooperate. `yo61/go-udap` (this repo) has its `goreleaser.yaml` workflow extended to mint a GitHub App installation token covering both repos and pass it to goreleaser, which then writes `Formula/go-udap.rb` to `yo61/homebrew-tap` on every tag. The tap repo runs `brew audit --strict --new` on every push as a safety net.

**Tech Stack:** GoReleaser v2 (`brews:` block), GitHub Actions, `actions/create-github-app-token@v3`, the existing org-level GitHub App, `Homebrew/actions/setup-homebrew`.

**Spec:** [`docs/superpowers/specs/2026-05-09-homebrew-tap-design.md`](../specs/2026-05-09-homebrew-tap-design.md)

**Working directories used by this plan:**
- This repo: `/Users/robin/code/github/yo61/go-udap`
- New tap repo (cloned alongside): `/Users/robin/code/github/yo61/homebrew-tap`

---

## Task 1: Bootstrap the tap repo

**Files (in `/Users/robin/code/github/yo61/homebrew-tap`):**
- Create: `README.md`
- Create: `LICENSE`
- Create: `.gitignore`
- Create: `.github/workflows/audit.yaml`

**Why first:** the tap repo must exist on GitHub before goreleaser can push to it, and before the App can be installed on it.

- [ ] **Step 1: Create the GitHub repo (empty, public, MIT)**

```bash
gh repo create yo61/homebrew-tap \
    --public \
    --description "Homebrew tap for yo61 tools" \
    --license MIT
```

Expected output: `✓ Created repository yo61/homebrew-tap on GitHub`

- [ ] **Step 2: Clone the new repo as a sibling of go-udap**

```bash
cd /Users/robin/code/github/yo61
git clone https://github.com/yo61/homebrew-tap.git
cd homebrew-tap
```

`gh repo create --license MIT` already populated `LICENSE` and `README.md` with default content. We'll overwrite both with project-specific content in the next steps.

- [ ] **Step 3: Write a project-specific README**

Replace `/Users/robin/code/github/yo61/homebrew-tap/README.md` with:

````markdown
# yo61 Homebrew tap

Homebrew formulas for tools published by [yo61](https://github.com/yo61).

## Install

```bash
brew install yo61/tap/<formula-name>
```

For example:

```bash
brew install yo61/tap/go-udap
```

## Available formulas

| Formula | Description |
| --- | --- |
| [`go-udap`](https://github.com/yo61/go-udap) | Squeezebox UDAP configuration tool |

## How formulas land here

Formulas in this tap are generated and pushed automatically by [GoReleaser](https://goreleaser.com/) from the upstream project's release workflow. Don't hand-edit files under `Formula/` — your changes will be overwritten on the next upstream release.

Every push and PR runs `brew audit --strict --new` and `brew test` against each formula via `.github/workflows/audit.yaml`.

## License

MIT — see [`LICENSE`](LICENSE).
````

- [ ] **Step 4: Add a minimal .gitignore**

Create `/Users/robin/code/github/yo61/homebrew-tap/.gitignore`:

```
.DS_Store
*.swp
```

- [ ] **Step 5: Add the audit workflow**

Create `/Users/robin/code/github/yo61/homebrew-tap/.github/workflows/audit.yaml`:

```yaml
name: Audit

on:
  pull_request:
  push:
    branches: [main]

permissions:
  contents: read

jobs:
  audit:
    name: brew audit + test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6

      - name: Set up Homebrew
        id: setup
        uses: Homebrew/actions/setup-homebrew@master

      - name: Find formulas
        id: formulas
        run: |
          set -euo pipefail
          formulas="$(find Formula -name '*.rb' -print 2>/dev/null || true)"
          if [ -z "$formulas" ]; then
            echo "No formulas in Formula/ yet — skipping audit."
            echo "skip=true" >> "$GITHUB_OUTPUT"
          else
            echo "skip=false" >> "$GITHUB_OUTPUT"
            {
              echo "formulas<<EOF"
              echo "$formulas"
              echo "EOF"
            } >> "$GITHUB_OUTPUT"
          fi

      - name: brew audit --strict --new
        if: steps.formulas.outputs.skip != 'true'
        run: |
          set -euo pipefail
          while IFS= read -r f; do
            echo "::group::audit $f"
            brew audit --strict --new "$f"
            echo "::endgroup::"
          done <<< "${{ steps.formulas.outputs.formulas }}"

      - name: brew install + brew test
        if: steps.formulas.outputs.skip != 'true'
        run: |
          set -euo pipefail
          while IFS= read -r f; do
            name="$(basename "$f" .rb)"
            echo "::group::install + test $name"
            brew install --verbose "$f"
            brew test "$name"
            echo "::endgroup::"
          done <<< "${{ steps.formulas.outputs.formulas }}"
```

The `skip=true` branch matters because the very first push (before goreleaser ever runs) will have no formulas yet, and we don't want CI red on day one.

- [ ] **Step 6: Verify the workflow file is syntactically valid YAML**

Run from `/Users/robin/code/github/yo61/homebrew-tap`:

```bash
python3 -c 'import yaml,sys; yaml.safe_load(open(".github/workflows/audit.yaml"))' && echo OK
```

Expected: `OK`. (No need to install actionlint just for this; the runtime will catch action-input issues on first run.)

- [ ] **Step 7: Commit and push**

```bash
cd /Users/robin/code/github/yo61/homebrew-tap
git add README.md LICENSE .gitignore .github/workflows/audit.yaml
git commit -m "chore: bootstrap tap with README, LICENSE, and audit workflow"
git push origin main
```

(If `LICENSE` was already committed by `gh repo create`, that's fine — `git add` is idempotent.)

- [ ] **Step 8: Confirm the audit workflow ran and passed (no formulas yet)**

```bash
sleep 5  # let GH register the run
gh run list --repo yo61/homebrew-tap --workflow=audit.yaml --limit 1
```

Expected: `completed	success	chore: bootstrap tap...	Audit	main	push`

If it shows `failure`, view logs: `gh run view <RUN_ID> --repo yo61/homebrew-tap --log-failed | head -40`

---

## Task 2: Extend the GitHub App installation to cover the tap

**Files:** none — this is a GitHub UI action.

**Why now:** the App must have access to `yo61/homebrew-tap` before the goreleaser workflow tries to mint a token for it. Doing this before the workflow change avoids a failed release.

- [ ] **Step 1: Open the App's "Repository access" settings**

In the browser, go to the org's installed-apps settings:

```
https://github.com/organizations/yo61/settings/installations
```

Find the App that backs `SEMANTIC_RELEASE_APP_CLIENT_ID` and click **Configure**.

- [ ] **Step 2: Grant access to the tap repo**

Under **Repository access**:

- If "Only select repositories" is selected: click **Select repositories** and add `homebrew-tap` (it will already include `go-udap`).
- If "All repositories" is selected: nothing to do; the App already covers everything in the org.

Click **Save**.

- [ ] **Step 3: Verify the App's permissions include Contents: Read & write**

Still on the App's installation page, scroll to **Permissions**. Confirm `Contents: Read & write` is listed. (Already true since the App pushes CHANGELOG.md commits; this is a sanity check.) If not, request the new permission and accept on the install side.

- [ ] **Step 4: Smoke-test the token from a workflow_dispatch**

This step verifies the App can issue a token covering both repos before we wire it into the real workflow. Create a one-off workflow on a throwaway branch:

```bash
cd /Users/robin/code/github/yo61/go-udap
git checkout -b chore/test-tap-token
```

Create `.github/workflows/_smoke-tap-token.yaml`:

```yaml
name: _smoke-tap-token

on:
  workflow_dispatch:

permissions:
  contents: read

jobs:
  smoke:
    runs-on: ubuntu-latest
    steps:
      - id: token
        uses: actions/create-github-app-token@v3
        with:
          client-id: ${{ secrets.SEMANTIC_RELEASE_APP_CLIENT_ID }}
          private-key: ${{ secrets.SEMANTIC_RELEASE_APP_PRIVATE_KEY }}
          owner: ${{ github.repository_owner }}
          repositories: |
            go-udap
            homebrew-tap

      - name: Read tap repo with the token
        env:
          GH_TOKEN: ${{ steps.token.outputs.token }}
        run: |
          gh api repos/yo61/homebrew-tap --jq '.full_name'
```

- [ ] **Step 5: Push and dispatch**

```bash
git add .github/workflows/_smoke-tap-token.yaml
git commit -m "chore: temporary smoke test for tap token"
git push -u origin chore/test-tap-token
gh workflow run _smoke-tap-token.yaml --ref chore/test-tap-token
sleep 5
gh run list --workflow=_smoke-tap-token.yaml --limit 1
```

Expected: a `completed	success` run, with logs showing `yo61/homebrew-tap` printed.

- [ ] **Step 6: Tear the smoke workflow down**

```bash
git rm .github/workflows/_smoke-tap-token.yaml
git commit -m "chore: remove smoke test workflow"
git push
git checkout main
git push origin --delete chore/test-tap-token
git branch -D chore/test-tap-token
```

The smoke branch is intentionally not merged — it's deleted to keep `main` clean.

---

## Task 3: Add the `brews:` block to `.goreleaser.yaml`

**Files:**
- Modify: `/Users/robin/code/github/yo61/go-udap/.goreleaser.yaml`

**Why now:** the GoReleaser config drives formula generation. We add the block on a feature branch (no tag yet), so nothing pushes to the tap until we explicitly cut a release.

- [ ] **Step 1: Branch from main**

```bash
cd /Users/robin/code/github/yo61/go-udap
git checkout main && git pull
git checkout -b feat/homebrew-tap-formula
```

- [ ] **Step 2: Add the `brews:` block**

Append to `/Users/robin/code/github/yo61/go-udap/.goreleaser.yaml`, immediately after the `release:` block (and before the `changelog:` block, to keep related sections grouped):

```yaml
brews:
  - name: go-udap
    repository:
      owner: yo61
      name: homebrew-tap
      branch: main
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
    homepage: "https://github.com/yo61/go-udap"
    description: "Squeezebox UDAP configuration tool"
    license: MIT
    test: |
      assert_match "go-udap", shell_output("#{bin}/go-udap --version")
      assert_match "Usage:", shell_output("#{bin}/go-udap --help")
      assert_match "unknown command", shell_output("#{bin}/go-udap floop 2>&1", 1)
    install: |
      bin.install "go-udap"
```

- [ ] **Step 3: Validate the goreleaser config locally**

```bash
goreleaser check
```

Expected: `• config is valid` (and a warning about no Git tag, which is fine — we're not releasing yet).

If `goreleaser` isn't installed: `brew install goreleaser` first.

- [ ] **Step 4: Dry-run the formula generation**

```bash
goreleaser release --snapshot --skip publish --clean
```

This builds binaries and writes the formula to `dist/homebrew/Formula/go-udap.rb` without pushing anywhere. Expected: command exits 0, and `dist/homebrew/Formula/go-udap.rb` exists.

- [ ] **Step 5: Sanity-check the generated formula**

```bash
cat dist/homebrew/Formula/go-udap.rb
```

Expected: a Ruby formula with:
- `class GoUdap < Formula`
- `desc "Squeezebox UDAP configuration tool"`
- `homepage "https://github.com/yo61/go-udap"`
- `license "MIT"`
- `url`/`sha256` blocks for each `darwin`/`linux` × `amd64`/`arm64` combination
- `def install` containing `bin.install "go-udap"`
- `test do` with the three `assert_match` calls

If anything is off, edit the `brews:` block and rerun Step 4.

- [ ] **Step 6: Run brew audit on the dry-run formula (optional, requires brew locally)**

```bash
brew audit --strict --new dist/homebrew/Formula/go-udap.rb
```

Expected: no output (audit passed). If audit complains about `head` URL or `desc` length etc., fix the `brews:` block. If `brew` isn't installed locally, skip — the tap-repo's `audit.yaml` will run the same check on the next release.

- [ ] **Step 7: Commit**

```bash
git add .goreleaser.yaml
git commit -m "feat(release): generate Homebrew formula via goreleaser"
```

(Don't push yet — Task 4 belongs on the same branch.)

---

## Task 4: Wire the App token into `goreleaser.yaml` workflow

**Files:**
- Modify: `/Users/robin/code/github/yo61/go-udap/.github/workflows/goreleaser.yaml`

- [ ] **Step 1: Add the App-token step and env var**

Edit `/Users/robin/code/github/yo61/go-udap/.github/workflows/goreleaser.yaml`. Make three changes:

**1a. Add a `tap_token` step before `actions/checkout@v6`** so it's the first step in the job:

```yaml
      - id: tap_token
        uses: actions/create-github-app-token@v3
        with:
          client-id: ${{ secrets.SEMANTIC_RELEASE_APP_CLIENT_ID }}
          private-key: ${{ secrets.SEMANTIC_RELEASE_APP_PRIVATE_KEY }}
          owner: ${{ github.repository_owner }}
          repositories: |
            go-udap
            homebrew-tap
```

**1b. Add `HOMEBREW_TAP_GITHUB_TOKEN` to the goreleaser step's `env`:**

Find this block:

```yaml
      - uses: goreleaser/goreleaser-action@v7
        with:
          distribution: goreleaser
          version: '~> v2'
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

Change it to:

```yaml
      - uses: goreleaser/goreleaser-action@v7
        with:
          distribution: goreleaser
          version: '~> v2'
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_GITHUB_TOKEN: ${{ steps.tap_token.outputs.token }}
```

**1c. The existing top-level `permissions: contents: write` is unchanged** — it's needed by the existing `gh release edit --draft=false` step, which keeps using the default `GITHUB_TOKEN`.

- [ ] **Step 2: Confirm the resulting file**

After your edits, the workflow should look like:

```yaml
name: GoReleaser

on:
  push:
    tags:
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
      - id: tap_token
        uses: actions/create-github-app-token@v3
        with:
          client-id: ${{ secrets.SEMANTIC_RELEASE_APP_CLIENT_ID }}
          private-key: ${{ secrets.SEMANTIC_RELEASE_APP_PRIVATE_KEY }}
          owner: ${{ github.repository_owner }}
          repositories: |
            go-udap
            homebrew-tap

      - uses: actions/checkout@v6
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v6
        with:
          go-version-file: go.mod

      - uses: goreleaser/goreleaser-action@v7
        with:
          distribution: goreleaser
          version: '~> v2'
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_GITHUB_TOKEN: ${{ steps.tap_token.outputs.token }}

      - name: Publish release (un-draft)
        run: gh release edit "${GITHUB_REF#refs/tags/}" --draft=false
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 3: Validate the YAML**

```bash
python3 -c 'import yaml; yaml.safe_load(open(".github/workflows/goreleaser.yaml"))' && echo OK
```

Expected: `OK`.

- [ ] **Step 4: Commit and push the branch**

```bash
git add .github/workflows/goreleaser.yaml
git commit -m "feat(release): mint App token covering tap repo for goreleaser"
git push -u origin feat/homebrew-tap-formula
```

- [ ] **Step 5: Open a PR**

```bash
gh pr create --title "feat(release): publish Homebrew formula to yo61/homebrew-tap on release" --body "$(cat <<'EOF'
## Summary

- Add a \`brews:\` block to \`.goreleaser.yaml\` that generates \`Formula/go-udap.rb\` and pushes it to \`yo61/homebrew-tap\` on every tag.
- Mint a GitHub App installation token covering both \`go-udap\` and \`homebrew-tap\` and pass it to goreleaser as \`HOMEBREW_TAP_GITHUB_TOKEN\`.
- The default \`GITHUB_TOKEN\` continues to drive the GitHub release upload and \`gh release edit --draft=false\`.

Spec: \`docs/superpowers/specs/2026-05-09-homebrew-tap-design.md\`
Plan: \`docs/superpowers/plans/2026-05-09-homebrew-tap.md\`

## Test plan
- [ ] Once merged, cut a \`vX.Y.Z-rc.1\` tag (Task 5 of the plan) and verify:
  - GoReleaser run completes
  - \`yo61/homebrew-tap\` gains a new commit creating \`Formula/go-udap.rb\`
  - The tap's \`audit.yaml\` run passes (\`brew audit --strict --new\` + \`brew install\` + \`brew test\`)
  - \`brew install yo61/tap/go-udap\` works locally on macOS

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

- [ ] **Step 6: Wait for the PR to be approved and merged**

The user reviews and merges. No automated check here — the PR doesn't trigger a goreleaser run because that workflow is gated on tag pushes, not branches/PRs.

---

## Task 5: End-to-end smoke test with a release-candidate tag

**Files:** none modified — this task creates a Git tag and observes the resulting workflow runs.

**Why an RC tag:** the next "real" tag (`v1.0.2` or whatever) will publish the formula publicly. An RC like `v1.0.2-rc.1` lets us verify the end-to-end flow without committing a public formula version we can't take back. semantic-release won't auto-cut it; we create it by hand.

- [ ] **Step 1: Pick the next RC tag**

From the `go-udap` repo:

```bash
cd /Users/robin/code/github/yo61/go-udap
git checkout main && git pull
git tag --list 'v*' --sort=-v:refname | head -3
```

Expected: latest tag at top (e.g., `v1.0.1`). Pick the next patch version with an `-rc.1` suffix, e.g., `v1.0.2-rc.1`.

- [ ] **Step 2: Create and push the tag**

```bash
NEXT_TAG=v1.0.2-rc.1   # adjust based on Step 1
git tag -a "$NEXT_TAG" -m "Release candidate for Homebrew tap end-to-end test"
git push origin "$NEXT_TAG"
```

- [ ] **Step 3: Watch the goreleaser run**

```bash
sleep 5
gh run list --workflow=goreleaser.yaml --limit 1
RUN_ID=$(gh run list --workflow=goreleaser.yaml --limit 1 --json databaseId --jq '.[0].databaseId')
gh run watch "$RUN_ID" --exit-status
```

Expected: workflow succeeds. If it fails on `tap_token`, the App is not installed on `homebrew-tap` (revisit Task 2).

- [ ] **Step 4: Verify the formula was pushed to the tap**

```bash
gh api repos/yo61/homebrew-tap/contents/Formula/go-udap.rb --jq '.name, .size, .sha'
```

Expected: name `go-udap.rb`, non-zero size, a SHA.

- [ ] **Step 5: Watch the tap's audit workflow**

```bash
sleep 5
gh run list --repo yo61/homebrew-tap --workflow=audit.yaml --limit 1
TAP_RUN_ID=$(gh run list --repo yo61/homebrew-tap --workflow=audit.yaml --limit 1 --json databaseId --jq '.[0].databaseId')
gh run watch "$TAP_RUN_ID" --repo yo61/homebrew-tap --exit-status
```

Expected: audit + install + test all pass.

If `brew audit --strict --new` fails: read the run logs (`gh run view "$TAP_RUN_ID" --repo yo61/homebrew-tap --log-failed`), fix the `brews:` block in `.goreleaser.yaml` on a follow-up branch, merge, then re-cut a fresh `-rc.2` tag.

- [ ] **Step 6: Verify install works locally on macOS**

(Skip if not on macOS; the audit workflow already covered Linux.)

```bash
brew install yo61/tap/go-udap
go-udap --version
go-udap --help
brew uninstall yo61/tap/go-udap
brew untap yo61/tap   # optional cleanup
```

Expected: install completes, `--version` prints `go-udap v1.0.2-rc.1` (or whichever RC), `--help` prints usage.

- [ ] **Step 7: Decide whether the RC's GitHub release should be deleted**

The RC produced a published GitHub release. Two options:

- **Keep it**: the RC is an honest artifact and signals to users that pre-releases happen. Conventionally fine for projects using semantic-release.
- **Delete it**: cleaner if you don't want RC tags floating in the release list.

To delete (your call):

```bash
gh release delete "$NEXT_TAG" --yes
git push origin --delete "$NEXT_TAG"
git tag -d "$NEXT_TAG"
```

The formula in the tap will then point at a non-existent tarball — that's fine because the next real release will overwrite the formula. Don't try to install from this formula after deleting the release; just wait for the next tag.

---

## Task 6: Update `README.md` install instructions

**Files:**
- Modify: `/Users/robin/code/github/yo61/go-udap/README.md`

- [ ] **Step 1: Branch and read current install section**

```bash
cd /Users/robin/code/github/yo61/go-udap
git checkout main && git pull
git checkout -b docs/homebrew-install-instructions
```

The current README opens with the DeepWiki badge and an Installation section starting with "Pre-built Binaries". We add Homebrew above that — it's the lowest-friction option.

- [ ] **Step 2: Insert the Homebrew section**

Open `README.md` and find the line:

```
### Pre-built Binaries
```

Immediately above that line, insert:

```markdown
### Homebrew (macOS and Linux)

```bash
brew install yo61/tap/go-udap
```

The formula lives at [yo61/homebrew-tap](https://github.com/yo61/homebrew-tap) and is auto-published on every release.

```

(Note: literal triple-backtick code fence — leave the trailing blank line so the existing `### Pre-built Binaries` heading separates cleanly.)

- [ ] **Step 3: Commit, push, open PR**

```bash
git add README.md
git commit -m "docs(readme): document Homebrew install via yo61/tap"
git push -u origin docs/homebrew-install-instructions
gh pr create --title "docs(readme): document Homebrew install via yo61/tap" --body "$(cat <<'EOF'
## Summary

Add a Homebrew install section to the README, above the existing pre-built binaries instructions.

\`brew install yo61/tap/go-udap\` is now the lowest-friction install option on macOS and Linux.

## Test plan
- [ ] Render the README on GitHub and confirm the new section sits above "Pre-built Binaries" with correct markdown
- [ ] Copy the \`brew install\` command and verify it works on a clean macOS machine

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

- [ ] **Step 4: Wait for review and merge**

---

## Task 7: Final verification and cleanup

**Files:** none modified — this is a closing-the-loop check.

- [ ] **Step 1: Confirm tap repo state is clean**

```bash
gh api repos/yo61/homebrew-tap/contents/Formula --jq '.[].name'
```

Expected: `go-udap.rb`.

- [ ] **Step 2: Confirm last audit run on the tap is green**

```bash
gh run list --repo yo61/homebrew-tap --workflow=audit.yaml --limit 1
```

Expected: most recent run has `success` conclusion.

- [ ] **Step 3: Confirm last release run on go-udap is green**

```bash
gh run list --workflow=goreleaser.yaml --limit 1
```

Expected: success.

- [ ] **Step 4: Verify the formula resolves end-to-end (macOS, fresh shell)**

```bash
brew untap yo61/tap 2>/dev/null || true
brew install yo61/tap/go-udap
go-udap --version
go-udap --help | head -5
brew uninstall yo61/tap/go-udap
```

Expected: install succeeds, `--version` and `--help` produce expected output.

- [ ] **Step 5: Check code-scanning alerts (no regressions)**

```bash
gh api repos/yo61/go-udap/code-scanning/alerts --jq '.[] | select(.state == "open")'
gh api repos/yo61/homebrew-tap/code-scanning/alerts --jq '.[] | select(.state == "open")' 2>&1 | head -5
```

Expected: no open alerts on either repo. (The tap repo may not have code-scanning enabled yet; a 404 from the second command is acceptable.)

- [ ] **Step 6: Move the spec and plan to "completed"**

No file move required — they live under `docs/superpowers/specs/` and `docs/superpowers/plans/` respectively, both of which are append-only. The work is done when the implementation PRs (Tasks 4 and 6) and the tag from Task 5 are all on `main`.
