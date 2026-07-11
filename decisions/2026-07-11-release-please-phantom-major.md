## Decision: Split release.yaml into a release-only scan and a PR-only scan so the next-version scan never runs before the new tag exists

Restructure the `Release` workflow's release-please usage into two
invocations:

1. **Release-only** (`skip-github-pull-request: true`) — creates the
   draft GitHub release from a merged Release PR; never opens a PR.
2. **Push `refs/tags/vX.Y.Z`** for the draft release (unchanged).
3. **PR-only** (`skip-github-release: true`, ungated) — opens/updates the
   next-version Release PR, always running last so it anchors to the
   just-pushed tag.

## Context: Recurring phantom `3.0.0` Release PRs

release-please opened a full-history `3.0.0` major-bump PR twice: #136
(after v2.4.2) and #155 (after v2.4.3). Each listed the entire commit
history and re-surfaced `BREAKING CHANGE:` footers already shipped in
v1.x/v2.x (e.g. the `--interface` → `--bind-interface` rename, release
artifact filename changes).

Mechanism: releases are created as drafts (`"draft": true`), because a
draft is needed for GoReleaser to attach assets before publish. A GitHub
draft release stores `tag_name` as metadata only — no git tag ref — so
the workflow pushes `refs/tags/vX.Y.Z` in a separate step. But the single
release-please invocation created the draft and then immediately scanned
for the next version in the same run, before the tag ref existed. With no
tag to anchor "latest release", release-please logged `No latest release
found`, fell back to `bootstrap-sha`, walked all ~360 commits, and opened
a phantom major PR.

The v2.4.3 run log is definitive:
`looking for tagName: v2.4.3` → `No latest release found for path: .,
but a previous version (2.4.3) was specified in the manifest` →
`Considering: 362 commits`.

## Alternatives considered:

- **#138 reconcile step (prior attempt):** re-ran release-please after
  the tag push, gated on `release_created`. It anchors correctly (finds
  the release, 0 commits, skips) but a *skip* does not close the phantom
  PR the first invocation already opened, so #155 survived. Read-only
  reconcile — fixes the scan, not the artifact.
- **Drop draft releases:** would let release-please push the tag itself
  so it exists at scan time, but breaks the GoReleaser flow that uploads
  assets to a draft before publishing. Larger blast radius; rejected.
- **Cleanup step that force-closes any open release-please PR after a
  skip:** treats the symptom, relies on fragile title/branch matching;
  rejected in favour of preventing the phantom being opened at all.

## Reasoning:

The race is purely ordering: the next-version scan must never run before
the tag ref exists. Splitting release creation (`skip-github-pull-request`)
from PR scanning (`skip-github-release`) and always running the PR scan
last guarantees that ordering. It cannot open a phantom because the only
step that opens PRs runs after the tag is present. Both inputs are
supported by `googleapis/release-please-action` (confirmed in the resolved
`with:` block of prior run logs).

## Trade-offs accepted:

Every push to main now runs two release-please invocations instead of one
(the release-only step is a near-no-op on non-release pushes). Minor extra
API calls / seconds per run, accepted for correctness.

## Supersedes: PR #138 ("re-run release-please after tagging"). The
reconcile step is removed; this ordering split replaces it.
