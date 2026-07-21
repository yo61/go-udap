# Homebrew cask signed-PR release flow — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Land `Casks/go-udap.rb` on the tap's protected `main` through a GitHub-signed PR instead of the now-blocked direct push, finishing the migration the formulae already completed.

**Architecture:** `go-udap` keeps *generating* the cask with GoReleaser but stops pushing it — it attaches the rendered cask to the release and fires a `repository_dispatch`. The tap *lands* it: a `bump-go-udap` workflow fetches the asset, commits it via `createCommitOnBranch` (signed), opens a `cask/go-udap-<v>` PR; `tests.yaml` lints it; a `publish-cask` workflow squash-merges on green.

**Tech Stack:** GoReleaser v2, GitHub Actions, GitHub GraphQL `createCommitOnBranch`, `gh` CLI, the `actions/create-github-app-token` App (`SEMANTIC_RELEASE_APP`).

## Global Constraints

- **Two repos.** Tap work happens in `/Users/robin/code/github.com/yo61/homebrew-tap`; `go-udap` work in `/Users/robin/code/github.com/yo61/go-udap`. Each gets its own branch and PR.
- **Merge order.** The **tap PR must merge first** — the `go-udap` release dispatch has no `bump-go-udap` workflow to target until it does. Do not cut a release until both PRs are merged.
- **No ruleset bypass, no rule weakening.** Every commit reaching tap `main` must be GitHub-signed (`createCommitOnBranch` or squash-merge); nothing pushes to `main` outside a PR.
- **Pin actions to SHA with a version comment.** Reuse the tap's existing pins verbatim: `actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd  # v6.0.2`, `actions/create-github-app-token@1b10c78c7865c340bc4f6099eb2f838309f1e8c3  # v3.1.1`. Use `persist-credentials: false` on checkout.
- **Lint gates.** Tap workflows must pass `actionlint` and `zizmor .github/workflows/`. The tap's CI runs `zizmor` (see `zizmor.yaml`); `actionlint` is a local gate — run it locally before committing. Commit messages are Conventional Commits (`commitlint` runs in both repos).
- **Cask branch prefix is `cask/`** (distinct from the formulae's `bump/`) so `publish-bottles.yaml` ignores cask PRs with no edit.
- **Cask-name hyphen hazard.** The cask is named `go-udap` (contains a hyphen). Never parse `cask/go-udap-<v>` with `${ref%%-*}` (splits on the first hyphen → wrong). Strip the literal `cask/go-udap-` prefix instead.
- **Verification is lint + dry-run + one live release.** These are infra-as-code changes with no unit tests; per-task gates are `actionlint`/`zizmor`/`goreleaser --snapshot`. The true end-to-end test is Task 7 (a real release).

---

## Task 1: Log the tap decision doc

**Repo:** homebrew-tap
**Files:**
- Create branch: `git checkout -b feat/cask-signed-release-flow`
- Create: `decisions/2026-07-21-cask-signed-release-flow.md`

- [ ] **Step 1: Create the tap branch**

```bash
cd /Users/robin/code/github.com/yo61/homebrew-tap
git checkout main && git pull --ff-only
git checkout -b feat/cask-signed-release-flow
```

- [ ] **Step 2: Write the decision doc**

Create `decisions/2026-07-21-cask-signed-release-flow.md`:

```markdown
## Decision: Land the go-udap cask via a signed bump PR, like the formulae — GoReleaser generates, the tap lands

Migrate the go-udap **cask** off GoReleaser's direct push to `main` (blocked
since the `Default Branch` ruleset's App bypass was removed on 2026-07-14) onto
the same signed-PR flow the formulae use. GoReleaser keeps rendering the cask but
attaches it to the go-udap release instead of pushing it; go-udap fires a
`repository_dispatch (bump-go-udap)`; a tap-side `bump-go-udap.yaml` fetches the
rendered cask, commits it to `Casks/go-udap.rb` via `createCommitOnBranch`
(GitHub-signed), and opens a `cask/go-udap-<v>` PR; `publish-cask.yaml`
squash-merges it once `tests.yaml` is green.

## Context

go-udap was the only artifact still on the pre-bypass-removal direct-push path.
Every go-udap release since 2026-07-14 (v2.4.4, v2.4.5) fails at the cask push
with `409 Repository rule violations found` (needs a PR; needs verified
signatures). Formulae already moved to the signed flow in
[[2026-07-13-signed-commits-via-api-drop-bypass]].

## Alternatives considered

- **Re-add the App as a ruleset bypass actor.** Rejected: reverses
  [[2026-07-13-signed-commits-via-api-drop-bypass]] deliberately.
- **Tap owns a hand-maintained cask, edited in place.** Rejected: copies the
  cask's derived manpage/completion list into the tap where it can drift from
  go-udap's actual subcommand set, needing a downstream online-audit special
  case. Generating from the build makes drift structurally impossible.
- **GoReleaser opens the PR itself.** Rejected: its branch commit is unsigned
  and would lean on squash-merge to launder the signature — the mechanism this
  tap already declined.

## Reasoning

Split by what each repo authoritatively knows: go-udap owns cask *generation*
(where the subcommand set lives), the tap owns *landing* (where main's signature
protection lives). GoReleaser never touches the tap — it only emits an artifact.
Reuses the existing signing dance via a generalized `commit-recipe-pr` action;
adds a `cask/` branch prefix so `publish-bottles.yaml` (guarded on `bump/`) is
untouched.

## Trade-offs accepted

- go-udap stays in the cask-generation business (GoReleaser), against an earlier
  "delete homebrew_casks entirely" idea — but only generation; the broken push
  is gone.
- One more merge workflow (`publish-cask.yaml`) alongside `publish-bottles.yaml`,
  kept separate rather than branching the bottle-specific one.

## Supersedes

Builds on [[2026-07-13-signed-commits-via-api-drop-bypass]] and
[[2026-07-10-pr-pull-head-sha-pin]] (the head-sha TOCTOU merge pin reused by
publish-cask). Design spec:
go-udap `docs/superpowers/specs/2026-07-21-homebrew-cask-signed-release-flow-design.md`.
```

- [ ] **Step 3: Commit**

```bash
git add decisions/2026-07-21-cask-signed-release-flow.md
git commit -m "docs(decisions): log signed-PR cask release flow"
```

---

## Task 2: Generalize `commit-formula-pr` → `commit-recipe-pr`

**Repo:** homebrew-tap
**Files:**
- Rename: `.github/actions/commit-formula-pr/` → `.github/actions/commit-recipe-pr/`
- Modify: `.github/actions/commit-recipe-pr/action.yaml` (swap the `formula` input for `file-path`)
- Modify: `.github/workflows/bump.yaml` (the `uses:` path and its `with:`)

**Interfaces:**
- Produces: composite action `./.github/actions/commit-recipe-pr` with inputs `file-path`, `branch`, `token`, `commit-headline`, `pr-title`, `pr-body`. Consumed by Task 3's `bump-go-udap.yaml` and by `bump.yaml`.

- [ ] **Step 1: Rename the action directory**

```bash
cd /Users/robin/code/github.com/yo61/homebrew-tap
git mv .github/actions/commit-formula-pr .github/actions/commit-recipe-pr
```

- [ ] **Step 2: Rewrite `action.yaml` to take a file path**

Replace `.github/actions/commit-recipe-pr/action.yaml` with:

```yaml
name: Commit recipe and open PR
description: >-
  Create the given branch at main's HEAD, commit the workspace's recipe file
  (Formula/<f>.rb or Casks/<c>.rb) as a GitHub-signed GraphQL commit, and open a
  PR. The signed commit lets the downstream publish workflow squash-merge onto
  signature-protected main with no ruleset bypass. The Contents REST API does not
  sign — only createCommitOnBranch and the web UI do.

inputs:
  file-path:
    description: Recipe file to commit, e.g. Formula/unifictl.rb or Casks/go-udap.rb
    required: true
  branch:
    description: Branch to create and open the PR from (e.g. bump/unifictl-0.4.0 or cask/go-udap-2.4.5)
    required: true
  token:
    description: App token with contents:write and pull-requests:write
    required: true
  commit-headline:
    description: Commit message headline
    required: true
  pr-title:
    description: Pull request title
    required: true
  pr-body:
    description: Pull request body
    required: true

runs:
  using: composite
  steps:
    - shell: bash
      env:
        GH_TOKEN: ${{ inputs.token }}
        FILE_PATH: ${{ inputs.file-path }}
        BRANCH: ${{ inputs.branch }}
        HEADLINE: ${{ inputs.commit-headline }}
        PR_TITLE: ${{ inputs.pr-title }}
        PR_BODY: ${{ inputs.pr-body }}
      run: |
        set -euo pipefail
        if git diff --quiet -- "$FILE_PATH"; then
          echo "::error::$FILE_PATH unchanged; nothing to commit"
          exit 1
        fi
        # createCommitOnBranch commits onto an existing branch: create it at
        # main's HEAD first (REST refs need no signing), then the mutation adds
        # the one signed commit.
        base_sha=$(gh api "repos/${GITHUB_REPOSITORY}/git/ref/heads/main" --jq .object.sha)
        gh api -X POST "repos/${GITHUB_REPOSITORY}/git/refs" \
          -f ref="refs/heads/${BRANCH}" -f sha="$base_sha" >/dev/null
        jq -nc \
          --arg repo "$GITHUB_REPOSITORY" \
          --arg branch "$BRANCH" \
          --arg oid "$base_sha" \
          --arg headline "$HEADLINE" \
          --arg path "$FILE_PATH" \
          --arg contents "$(base64 -w0 "$FILE_PATH")" \
          '{query:"mutation($input:CreateCommitOnBranchInput!){createCommitOnBranch(input:$input){commit{oid}}}",
            variables:{input:{
              branch:{repositoryNameWithOwner:$repo,branchName:$branch},
              message:{headline:$headline},
              expectedHeadOid:$oid,
              fileChanges:{additions:[{path:$path,contents:$contents}]}}}}' \
          | gh api graphql --input - >/dev/null
        gh api -X POST "repos/${GITHUB_REPOSITORY}/pulls" \
          -f title="$PR_TITLE" \
          -f head="${BRANCH}" \
          -f base=main \
          -f body="$PR_BODY" >/dev/null
```

- [ ] **Step 3: Point `bump.yaml` at the renamed action with a `file-path`**

In `.github/workflows/bump.yaml`, the final step currently reads
`uses: ./.github/actions/commit-formula-pr` with `formula: ${{ inputs.formula }}`.
Replace that step's `uses:` and `with:` head so it becomes:

```yaml
      - name: Commit and open the PR
        if: steps.gate.outputs.ready == 'true'
        uses: ./.github/actions/commit-recipe-pr
        with:
          file-path: Formula/${{ inputs.formula }}.rb
          branch: ${{ steps.gate.outputs.branch }}
          token: ${{ steps.app-token.outputs.token }}
          commit-headline: "chore(${{ inputs.formula }}): bump to ${{ steps.gate.outputs.version }}"
          pr-title: "chore(${{ inputs.formula }}): bump to ${{ steps.gate.outputs.version }}"
          pr-body: "Automated bump triggered by ${{ inputs.formula }}'s release workflow. tests.yaml builds bottles; publish-bottles.yaml adds the bottle and squash-merges once green."
```

(Only `uses:` and the `formula:` → `file-path:` line change; the other `with:` values are unchanged.)

- [ ] **Step 4: Lint**

```bash
actionlint
zizmor .github/workflows/
```
Expected: both pass (no new findings). `actionlint` validates the reusable-workflow reference; `zizmor` re-checks `bump.yaml`.

- [ ] **Step 5: Commit**

```bash
git add .github/actions/commit-recipe-pr .github/workflows/bump.yaml
git commit -m "refactor(actions): generalize commit-formula-pr to commit-recipe-pr"
```

---

## Task 3: Add `bump-go-udap.yaml`

**Repo:** homebrew-tap
**Files:**
- Create: `.github/workflows/bump-go-udap.yaml`

**Interfaces:**
- Consumes: `./.github/actions/commit-recipe-pr` (Task 2); secrets `SEMANTIC_RELEASE_APP_CLIENT_ID`, `SEMANTIC_RELEASE_APP_PRIVATE_KEY`.
- Produces: a `repository_dispatch` type `bump-go-udap` (fired by Task 6) and a `cask/go-udap-<v>` branch + PR consumed by Task 4.

- [ ] **Step 1: Create the workflow**

Create `.github/workflows/bump-go-udap.yaml`:

```yaml
name: Bump go-udap cask

# Fetches the cask GoReleaser rendered and attached to a go-udap release, then
# opens a signed cask/* PR. tests.yaml lints it; publish-cask.yaml squash-merges
# once green. Triggered by go-udap's release workflow via repository_dispatch
# (bump-go-udap), or manually with a version (re-run / backfill path).
#
# Unlike the formula bumps there is no PyPI resolution and no in-place edit: the
# rendered cask already carries the version and the four sha256s GoReleaser
# computed, so the tap just commits it wholesale.

on:
  repository_dispatch:
    types: [bump-go-udap]
  workflow_dispatch:
    inputs:
      version:
        description: "go-udap version to bump the cask to (e.g. 2.4.5)"
        required: true
        type: string

permissions: {}

jobs:
  bump:
    name: Bump go-udap cask
    runs-on: ubuntu-latest
    permissions:
      contents: read
    steps:
      - uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd  # v6.0.2
        with:
          persist-credentials: false

      - name: Resolve target version
        id: gate
        env:
          INPUT_VERSION: ${{ github.event.client_payload.version || github.event.inputs.version }}
        run: |
          set -euo pipefail
          raw="${INPUT_VERSION:-}"
          v="${raw#v}"
          if [[ ! "$v" =~ ^[0-9]+\.[0-9]+\.[0-9]+([.\-][a-zA-Z0-9._-]+)?$ ]]; then
            echo "::error::Invalid version format: $raw"
            exit 1
          fi

          # Already current? Avoid churning a no-op PR.
          current=$(sed -nE 's/^[[:space:]]*version "([^"]+)".*/\1/p' Casks/go-udap.rb | head -1)
          if [[ "$current" == "$v" ]]; then
            echo "cask already at $v; nothing to do"
            echo "ready=false" >> "$GITHUB_OUTPUT"
            exit 0
          fi

          # A bump PR for this version may already be open from an earlier run.
          if git ls-remote --exit-code --heads origin "refs/heads/cask/go-udap-$v" >/dev/null 2>&1; then
            echo "branch cask/go-udap-$v already exists; PR is already open"
            echo "ready=false" >> "$GITHUB_OUTPUT"
            exit 0
          fi

          {
            echo "version=$v"
            echo "branch=cask/go-udap-$v"
            echo "ready=true"
          } >> "$GITHUB_OUTPUT"

      - name: Fetch the rendered cask from the release
        if: steps.gate.outputs.ready == 'true'
        env:
          V: ${{ steps.gate.outputs.version }}
        run: |
          set -euo pipefail
          curl -fsSL \
            "https://github.com/yo61/go-udap/releases/download/v${V}/go-udap.rb" \
            -o Casks/go-udap.rb
          # Sanity: the fetched cask must declare the version we asked for, so a
          # stale/mismatched asset fails closed rather than opening a wrong PR.
          got=$(sed -nE 's/^[[:space:]]*version "([^"]+)".*/\1/p' Casks/go-udap.rb | head -1)
          if [[ "$got" != "$V" ]]; then
            echo "::error::fetched cask declares version '$got', expected '$V'"
            exit 1
          fi

      - name: Mint App token
        if: steps.gate.outputs.ready == 'true'
        id: app-token
        uses: actions/create-github-app-token@1b10c78c7865c340bc4f6099eb2f838309f1e8c3  # v3.1.1
        with:
          client-id: ${{ secrets.SEMANTIC_RELEASE_APP_CLIENT_ID }}
          private-key: ${{ secrets.SEMANTIC_RELEASE_APP_PRIVATE_KEY }}
          owner: ${{ github.repository_owner }}
          repositories: ${{ github.event.repository.name }}
          permission-contents: write
          permission-pull-requests: write

      - name: Commit and open the PR
        if: steps.gate.outputs.ready == 'true'
        uses: ./.github/actions/commit-recipe-pr
        with:
          file-path: Casks/go-udap.rb
          branch: ${{ steps.gate.outputs.branch }}
          token: ${{ steps.app-token.outputs.token }}
          commit-headline: "chore(go-udap): bump cask to ${{ steps.gate.outputs.version }}"
          pr-title: "chore(go-udap): bump cask to ${{ steps.gate.outputs.version }}"
          pr-body: "Automated cask bump triggered by go-udap's release. tests.yaml lints; publish-cask.yaml squash-merges once green."
```

- [ ] **Step 2: Lint**

```bash
actionlint
zizmor .github/workflows/
```
Expected: both pass. If `zizmor` flags `repository_dispatch`/`workflow_dispatch` version interpolation, confirm the regex gate is present (it is) — the pattern matches `bump-unifictl.yaml`, which already passes.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/bump-go-udap.yaml
git commit -m "feat(ci): add bump-go-udap cask workflow"
```

---

## Task 4: Add `publish-cask.yaml`

**Repo:** homebrew-tap
**Files:**
- Create: `.github/workflows/publish-cask.yaml`

**Interfaces:**
- Consumes: the `cask/go-udap-<v>` PR from Task 3; the `brew test-bot` workflow's `workflow_run` completion; secrets `SEMANTIC_RELEASE_APP_*`.
- Produces: a squash-merge onto `main` (final landing).

- [ ] **Step 1: Create the workflow**

Create `.github/workflows/publish-cask.yaml`. Note the branch parse strips the
literal `cask/go-udap-` prefix — do **not** use `${ref%%-*}` (the cask name
contains a hyphen):

```yaml
name: publish cask

# Squash-merges a cask/* bump PR onto main once brew test-bot is green. Unlike
# publish-bottles there is no bottle to add: bump-go-udap already authored the
# cask as a signed commit (createCommitOnBranch), so this only lands that commit
# via a PR merge — the squash commit is GitHub-signed, satisfying
# required_signatures and pull_request with no ruleset bypass.

on:
  workflow_run:
    # zizmor: ignore[dangerous-triggers]
    # Safe: this job only runs for same-repo cask/* branches (see the job `if:`),
    # which only the SEMANTIC_RELEASE_APP creates. Fork PRs fail the
    # head_repository check and can never reach the write-capable App token.
    workflows: ["brew test-bot"]
    types:
      - completed

permissions:
  contents: read

jobs:
  publish:
    runs-on: ubuntu-latest
    if: >-
      github.event.workflow_run.conclusion == 'success' &&
      github.event.workflow_run.event == 'pull_request' &&
      github.event.workflow_run.head_repository.full_name == github.repository &&
      startsWith(github.event.workflow_run.head_branch, 'cask/')
    permissions:
      contents: read
    steps:
      - name: Mint App token
        id: app-token
        uses: actions/create-github-app-token@1b10c78c7865c340bc4f6099eb2f838309f1e8c3  # v3.1.1
        with:
          client-id: ${{ secrets.SEMANTIC_RELEASE_APP_CLIENT_ID }}
          private-key: ${{ secrets.SEMANTIC_RELEASE_APP_PRIVATE_KEY }}
          owner: ${{ github.repository_owner }}
          repositories: ${{ github.event.repository.name }}
          permission-contents: write
          permission-pull-requests: write

      - name: Squash-merge the cask PR
        env:
          GH_TOKEN: ${{ steps.app-token.outputs.token }}
          HEAD_BRANCH: ${{ github.event.workflow_run.head_branch }}
        run: |
          set -euo pipefail

          # Branch is cask/go-udap-<version>. The cask name go-udap contains a
          # hyphen, so strip the literal prefix — never split on '-'.
          if [[ "$HEAD_BRANCH" != cask/go-udap-* ]]; then
            echo "::error::unexpected cask branch '$HEAD_BRANCH'"
            exit 1
          fi
          version="${HEAD_BRANCH#cask/go-udap-}"
          if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+([.\-][a-zA-Z0-9._-]+)?$ ]]; then
            echo "::error::unexpected version '$version' from branch '$HEAD_BRANCH'"
            exit 1
          fi

          # Pin the merge to the current branch head (head-sha TOCTOU guard): if
          # the head moved — it must not; only the release App writes it — the
          # merge fails closed and main is untouched.
          head_oid=$(gh api "repos/${GITHUB_REPOSITORY}/git/ref/heads/${HEAD_BRANCH}" --jq .object.sha)
          pr=$(gh pr list --repo "${GITHUB_REPOSITORY}" --head "${HEAD_BRANCH}" \
                 --state open --json number --jq '.[0].number // empty')
          if [[ -z "$pr" ]]; then
            echo "::error::no open cask PR for ${HEAD_BRANCH}"
            exit 1
          fi

          # Retry: GitHub computes PR mergeability asynchronously and can briefly
          # report a fresh PR as not-yet-mergeable.
          merged=""
          for attempt in 1 2 3 4 5; do
            if gh api -X PUT "repos/${GITHUB_REPOSITORY}/pulls/${pr}/merge" \
                 -f merge_method=squash \
                 -f sha="${head_oid}" \
                 -f commit_title="chore(go-udap): release cask ${version}" \
                 >/dev/null 2>/tmp/merge_err; then
              merged=1
              break
            fi
            echo "merge attempt ${attempt} not ready:"
            cat /tmp/merge_err
            sleep 5
          done
          if [[ -z "$merged" ]]; then
            echo "::error::squash-merge failed"
            cat /tmp/merge_err
            exit 1
          fi

          # The merge auto-closes the PR. Delete the branch best-effort — the
          # repo's auto-delete-head-branch may already have removed it (404).
          gh api -X DELETE \
            "repos/${GITHUB_REPOSITORY}/git/refs/heads/${HEAD_BRANCH}" \
            >/dev/null 2>&1 || echo "branch ${HEAD_BRANCH} already deleted"
```

- [ ] **Step 2: Lint**

```bash
actionlint
zizmor .github/workflows/
```
Expected: both pass. `zizmor` should accept the `dangerous-triggers` ignore + the same-repo `head_repository` guard (mirrors `publish-bottles.yaml`).

- [ ] **Step 3: Commit and open the tap PR**

```bash
git add .github/workflows/publish-cask.yaml
git commit -m "feat(ci): add publish-cask squash-merge workflow"
git push -u origin feat/cask-signed-release-flow
gh pr create --repo yo61/homebrew-tap \
  --title "feat(ci): land go-udap cask via signed bump PR" \
  --body "Finishes the direct-push→signed-PR migration for the go-udap cask. Adds commit-recipe-pr (generalized), bump-go-udap.yaml, publish-cask.yaml. Design: go-udap docs/superpowers/specs/2026-07-21-homebrew-cask-signed-release-flow-design.md"
```

**STOP — merge this tap PR before the go-udap tasks land a release.** The go-udap dispatch (Task 6) needs `bump-go-udap.yaml` present on the tap's `main`.

---

## Task 5: GoReleaser — generate the cask, stop pushing it

**Repo:** go-udap (branch `feat/cask-release-signed-pr-flow`, already created)
**Files:**
- Modify: `.goreleaser.yaml` (the `homebrew_casks` and `release` blocks)

- [ ] **Step 1: Set `skip_upload` and attach the cask to the release**

In `.goreleaser.yaml`, in the `homebrew_casks:` entry, add `skip_upload: "true"`
and remove the `token:` line (the push is gone). Keep `repository:` for now
(Step 2 verifies whether GoReleaser still requires it). The entry head becomes:

```yaml
homebrew_casks:
  - name: go-udap
    skip_upload: "true"
    repository:
      owner: yo61
      name: homebrew-tap
      branch: main
    homepage: "https://github.com/yo61/go-udap"
    description: "Squeezebox UDAP configuration tool"
    license: MIT
```

Then extend the existing `release:` block to attach the rendered cask:

```yaml
release:
  draft: false
  use_existing_draft: true
  extra_files:
    - glob: ./dist/homebrew/Casks/go-udap.rb
```

- [ ] **Step 2: Verify the cask still renders (snapshot dry-run)**

```bash
cd /Users/robin/code/github.com/yo61/go-udap
go-task build >/dev/null 2>&1 || true   # ensure before-hooks' deps resolve
goreleaser release --snapshot --clean --skip=publish
ls -l dist/homebrew/Casks/go-udap.rb
```
Expected: the command succeeds and `dist/homebrew/Casks/go-udap.rb` exists.
- If GoReleaser errors that `repository` is required despite `skip_upload`, that
  is the expected shape — keep the `repository:` block (already kept).
- If it errors evaluating `{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}`, confirm the
  `token:` line was removed (it should be).
- Confirm the rendered file still contains `version "..."`, four `sha256`
  lines, and the nine `manpage` stanzas.

- [ ] **Step 3: Commit**

```bash
git add .goreleaser.yaml
git commit -m "build(goreleaser): render cask without pushing; attach to release"
```

---

## Task 6: GoReleaser workflow — drop the push, dispatch the tap

**Repo:** go-udap
**Files:**
- Modify: `.github/workflows/goreleaser.yaml` (drop the tap-clone style-fix steps; add the dispatch; trim the token scope + env)

**Interfaces:**
- Consumes: `steps.tap_token.outputs.token` (App token, `contents: write` on `homebrew-tap`).
- Produces: `repository_dispatch` type `bump-go-udap` with `client_payload.version` (consumed by Task 3's tap workflow).

- [ ] **Step 1: Remove the direct-push style-fix steps**

Delete the two trailing steps from `goreleaser.yaml`: `Set up Homebrew`
(`Homebrew/actions/setup-homebrew`) and `Style-fix the generated cask in the tap`
(clones the tap and `git push`es — it would fail the same ruleset).

- [ ] **Step 2: Stop passing the tap push token to GoReleaser**

In the `goreleaser/goreleaser-action` step's `env:`, remove the
`HOMEBREW_TAP_GITHUB_TOKEN` line. Leave `GITHUB_TOKEN` (still needed to publish
the release + assets). The block becomes:

```yaml
      - uses: goreleaser/goreleaser-action@f06c13b6b1a9625abc9e6e439d9c05a8f2190e94  # v7.2.3
        with:
          distribution: goreleaser
          version: '~> v2'
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 3: Narrow the tap token to homebrew-tap and add the dispatch**

Change the `tap_token` step's `repositories:` to just `homebrew-tap` (it now only
dispatches), and add a dispatch step after the `goreleaser-action` step:

In the `id: tap_token` step, set:

```yaml
          repositories: |
            homebrew-tap
```

Then append, as the final step of the job:

```yaml
      # GoReleaser no longer pushes the cask (the tap ruleset rejects direct,
      # unsigned pushes to main). Instead: tell the tap to bump. bump-go-udap.yaml
      # fetches the go-udap.rb asset attached to this release, commits it via a
      # signed createCommitOnBranch PR, and publish-cask.yaml squash-merges it.
      - name: Trigger tap cask bump
        env:
          GH_TOKEN: ${{ steps.tap_token.outputs.token }}
          VERSION: ${{ github.ref_name }}
        run: |
          set -euo pipefail
          gh api -X POST repos/yo61/homebrew-tap/dispatches \
            -f event_type=bump-go-udap \
            -f 'client_payload[version]='"${VERSION#v}"
```

- [ ] **Step 4: Lint**

```bash
actionlint
zizmor .github/workflows/
```
Expected: both pass. If `zizmor` warns the `tap_token` step is now unused before
the dispatch, confirm the dispatch step references it (it does).

- [ ] **Step 5: Commit and open the go-udap PR**

```bash
git add .github/workflows/goreleaser.yaml
git commit -m "ci(release): dispatch tap cask bump instead of pushing"
git push -u origin feat/cask-release-signed-pr-flow
gh pr create --repo yo61/go-udap \
  --title "ci(release): land homebrew cask via signed tap PR" \
  --body "GoReleaser stops pushing the cask (blocked by the tap ruleset) and instead attaches it to the release + dispatches the tap's bump-go-udap workflow. Requires the homebrew-tap PR to be merged first. Spec: docs/superpowers/specs/2026-07-21-homebrew-cask-signed-release-flow-design.md"
```

---

## Task 7: End-to-end verification and backfill

**Repos:** both. Human-driven; not automatable in this plan. Do only after **both** PRs merge (tap first).

- [ ] **Step 1: Confirm dispatch-token permission (resolves the open risk)**

The `POST /repos/.../dispatches` call needs the App token to satisfy the
endpoint's permission. The `tap_token` requests `contents: write` on
`homebrew-tap`. If the first release's dispatch step 403s, widen minimally
(add the metadata/administration scope the error names) and note it in the
decision doc.

- [ ] **Step 2: Cut the next go-udap release**

Merge a `fix:`/`feat:` commit to `go-udap` `main` so release-please opens a
Release PR; merge it. The tag push triggers `goreleaser.yaml`.

- [ ] **Step 3: Verify the release side**

```bash
gh release view v<new> --repo yo61/go-udap --json assets --jq '.assets[].name' | grep '^go-udap.rb$'
gh run list --repo yo61/go-udap --workflow goreleaser.yaml --limit 1
```
Expected: the `go-udap.rb` asset is present; the GoReleaser run is **success**
(no cask-push failure).

- [ ] **Step 4: Verify the tap side**

```bash
gh run list --repo yo61/homebrew-tap --workflow bump-go-udap.yaml --limit 1
gh pr list --repo yo61/homebrew-tap --state all --head "cask/go-udap-<new>" --json number,state,mergeCommit
```
Expected: `bump-go-udap` ran; a `cask/go-udap-<new>` PR opened with a **verified**
commit (no "commits must have verified signatures" banner); `tests.yaml` green;
`publish-cask` squash-merged it; `Casks/go-udap.rb` on `main` at the new version.

```bash
gh api repos/yo61/homebrew-tap/commits/main --jq '.commit.verification.verified'
```
Expected: `true`.

- [ ] **Step 5: Verify from the consumer side**

```bash
brew update && brew info yo61/tap/go-udap | head -3
```
Expected: the new version.

- [ ] **Step 6: Backfill the stuck version (2.4.5)**

If the tap is still behind (it was at 2.4.3), land the current release without
waiting for another:

```bash
gh workflow run bump-go-udap.yaml --repo yo61/homebrew-tap -f version=2.4.5
```
Expected: opens `cask/go-udap-2.4.5`, lints, squash-merges — same as Step 4.
(Requires the `go-udap.rb` asset on the v2.4.5 release; if absent — v2.4.5
predates this flow — re-run GoReleaser for that tag or bump forward to the next
release instead.)

---

## Self-Review

**Spec coverage:**
- Direct-push broken / migration framing → Task 1 decision doc, Tasks 5–6.
- GoReleaser generates, no push, attach asset → Task 5.
- Dispatch `{version}` → Task 6.
- Tap fetches rendered cask, signed commit, PR → Task 3 + Task 2 action.
- `tests.yaml` gate (already covers `Casks/**`) → relied on, no change needed (noted in Task 4).
- `cask/` prefix keeps `publish-bottles.yaml` untouched → Global Constraints + Task 4.
- Squash-merge on green → Task 4.
- `commit-formula-pr` → `commit-recipe-pr` generalization → Task 2.
- Decision doc in tap → Task 1.
- Open risks (dispatch permission, `skip_upload` vs `repository`, asset timing) → Task 5 Step 2, Task 7 Steps 1 & 3.

**Placeholder scan:** No TBD/TODO; every workflow/action file is given in full; every command has an expected result.

**Type/name consistency:** action `commit-recipe-pr` inputs (`file-path`, `branch`, `token`, `commit-headline`, `pr-title`, `pr-body`) match between Task 2 (definition) and Task 3 (use) and `bump.yaml` (use). Branch name `cask/go-udap-<v>` is constructed in Task 3 and parsed by literal-prefix strip in Task 4 (consistent; the hyphen hazard is handled). Dispatch `event_type=bump-go-udap` matches Task 3's `repository_dispatch.types`. `client_payload.version` (Task 6) matches Task 3's `github.event.client_payload.version`.
