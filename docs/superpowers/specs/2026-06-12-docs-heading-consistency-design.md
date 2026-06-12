# Site-wide docs heading consistency

**Date:** 2026-06-12
**Status:** Draft — awaiting implementation plan

## Summary

Standardise the heading hierarchy, section ordering, and component
usage across every page of the Fumadocs documentation site so that
pages look and read as part of one document rather than three. The
work classifies every page into one of three archetypes
(**Procedure** / **Explainer** / **Reference**), specifies the
canonical heading shape for each, applies it across the ten existing
pages that don't already follow it, removes the `<HowTo>` MDX component
that drove the largest visible inconsistency (five pages rendered
as shaded boxes), and writes the convention down both as a
contributor-facing page in the docs site (`contributing/docs-style.mdx`)
and as a one-line pointer from the root `CLAUDE.md`.

This is an editorial sweep, not a content rewrite. Every fact,
prerequisite, step, command, and warning is preserved verbatim; only
the structure around them changes.

## Motivation

- The five `<HowTo>`-using how-to pages render as shaded boxes that
  look like a different site from the rest of the docs. They were
  visually uniform with each other (the component's original intent)
  but visually divergent from the surrounding prose, breadcrumbs, and
  every other page archetype.
- Other pages use anti-patterns the convention forbids: `## Step N — …`
  in the tutorial, `## Step N: …` in `no-devices-found`,
  `## Option N — …` in `install-go-udap`, `## 1. …` through `## 10. …`
  numbered standalone H2s in `squeezeplay-comparison`, an in-body H1
  in `setup-vs-run-mode`, and Examples/Exit-codes in the wrong order
  on `commands/set`.
- The maintainer reviewed the site and identified the inconsistency
  directly; this spec is a response, not a speculative cleanup.
- The maintainer's principle — *"if we're using a component like
  `<HowTo>`, then all how-to pages should use it; if not, might as
  well use plain markdown and audit manually"* — combined with the
  observation that at least five existing how-to pages don't naturally
  fit the `<HowTo>` Goal/Steps/Verification mould, settled the
  direction: drop the component, use plain markdown everywhere, write
  down the convention.

## References

- The `<HowTo>` MDX component: `docs/site/components/how-to.tsx`,
  introduced in PR #53 (commit `02319af`) as a "structural component"
  to standardise the M3 milestone how-to pages.
- The model page demonstrating the desired Procedure archetype:
  `docs/site/content/docs/how-to/capture-wire-traces.mdx`.
- The five pages currently using `<HowTo>`:
  `how-to/configure-for-dhcp.mdx`, `configure-for-static-ip.mdx`,
  `configure-wifi-wpa2.mdx`, `discover-on-multi-nic-laptop.mdx`,
  `recover-stuck-in-init-mode.mdx`.
- The Starlight preview branch (`docs/starlight-preview-spec`) already
  contains a plain-markdown rewrite of these five pages. The Fumadocs
  rewrite reuses the same heading shape; the two are similar but not
  byte-identical (no basePath transformation, no Starlight component
  swap).
- The recent CSS branch (#122, merged) introduced a larger heading
  scale; the new structure inherits it and is verified to render
  cleanly under it.

## Goals

- Every page on the docs site fits exactly one of three archetypes
  (Procedure / Explainer / Reference) and uses that archetype's
  canonical heading shape.
- The `<HowTo>` MDX component is deleted; no page references it.
- The convention is documented in
  `docs/site/content/docs/contributing/docs-style.mdx`, a page that
  itself follows the conventions it documents.
- `CLAUDE.md` carries a one-line pointer to the style guide so future
  Claude sessions follow it when editing any docs page.
- `pnpm build` succeeds with no errors. `task docs:serve` renders
  every changed page with no console errors and a sidebar TOC that
  matches the new structure.

## Non-goals

- **No content edits beyond heading restructure.** Prose, prerequisites,
  step text, commands, warnings, and notes are preserved verbatim.
- **No new how-to pages.** The `onboard-over-wifi-ad-hoc` placeholder
  stays a placeholder; it is not filled in here.
- **No CI lint check for heading conventions.** Manual review at PR
  time is sufficient at the current page count; if pages or
  contributors grow, a follow-up can add `markdownlint` with a custom
  rule.
- **No changes to other Fumadocs components.** `<Callout>`, `<Cards>`,
  `<LinkCard>` continue to be used where they currently are; only
  `<HowTo>` is removed.
- **No changes to `install-shell-completions.mdx` or the three
  `contributing/*` pages**, whose H2s are descriptive section names,
  not numbered anti-patterns.
- **No reorganisation of the Diataxis tree, sidebar order, or the
  page set.**

## Design

### Three archetypes

Every page on the site falls into exactly one of three archetypes.
Each archetype specifies a canonical heading shape and a small set
of conventions.

#### Procedure

Step-by-step or option-by-option recipe. The reader follows in order.

- Top-level group heading is `## Steps` (or `## Options` /
  `## Methods` / `## Setup` / `## Capture session` etc. — whatever
  describes the group).
- Numbered sub-headings are `### N. Title`. The leading `N.` is the
  numbering format; do not use `### Step N — Title` or
  `### Option N — Title` or any other prefix.
- For pages where each step is a one-liner (typically a single
  command), use an ordered markdown list under `## Steps` instead of
  H3 sub-headings. This is the recipe shape: pages with substantial
  per-step content get H3 sub-headings, pages where steps fit on one
  line get a numbered list.
- Recipe-shaped how-tos (Goal/Prerequisites/Steps/Verification/
  Example/Notes) use the canonical section order:
  optional pre-prose paragraph, optional `## Example config` (when
  the example is shown inline at the top of the page),
  `## Goal`, `## Prerequisites`, `## Steps`, `## Verification`,
  optional `## Example config` (when the example is shown at the
  bottom), optional `## Notes`, optional trailing `## …` sections
  for follow-on topics specific to the page.
- The tutorial archetype uses `## Prerequisites`, then `## Steps`
  with H3 numbered sub-headings, then any trailing sections the
  tutorial uses (`## What you've done`, `## Common variations`,
  `## Next`).

Pages of this archetype: tutorial (`configure-your-first-squeezebox`),
all 12 `how-to/*` pages (where applicable), all three
`contributing/*` pages.

#### Explainer

Conceptual page. Parallel topics, no inherent ordering. The reader
typically arrives, reads the section they need, leaves.

- Flat unnumbered `## headings`. Multiple H2s in sequence; each is a
  parallel topic.
- No `## 1. …` / `## Step N …` / `### N. …` patterns — the
  numbering implies ordering the reader does not need.
- Sub-headings (`###`) are fine when a single topic decomposes
  naturally; they should not be numbered.

Pages of this archetype: all `concepts/*` pages,
`reference/global-flags.mdx`, the index landing pages
(`docs/index.mdx`, `concepts/index.mdx`, `how-to/index.mdx`,
`tutorials/index.mdx`, `reference/index.mdx`, `contributing/index.mdx`).

#### Reference

Looked-up data, not read top-to-bottom. The reader is verifying or
copying. Section titles are sub-fields of the entity being documented;
the section order is identical across sibling pages so the reader
can navigate by muscle memory.

- Sibling pages use the **same** H2 sequence in the **same** order.
- For the per-command pages (`reference/commands/*`), the canonical
  H2 order is: `## What it does`, `## Output`, `## Flags`,
  `## Exit codes`, `## Examples`. Pages where a section does not
  apply (e.g. a command with no flags) omit the H2 entirely rather
  than ship an empty section.

Pages of this archetype: `reference/commands/*` (nine pages),
`reference/{api, config-file-format, exit-codes, nvram-parameters}.mdx`.

### Three universal rules

These apply across all archetypes and supersede archetype defaults
if they conflict:

1. **No H1 in body.** The page title comes from frontmatter and is
   already rendered as the page H1 by the Fumadocs layout. Body
   content starts at H2.
2. **No `Step N —` / `Option N —` prefixes in headings.** Use the
   archetype's group-heading + numbered-sub-heading pattern instead.
3. **No standalone numbered H2s** (`## 1. Title`, `## 2. Title`,
   etc.). If the topics are ordered, use the Procedure archetype
   (group H2 + numbered H3s). If they are parallel, use the
   Explainer archetype (flat unnumbered H2s).

### Per-page changes

The implementation rewrites or restructures the following pages.
Every other page on the site already conforms and is not touched.

**Procedure-archetype rewrites (drop `<HowTo>`, plain markdown):**

| File | Change |
|---|---|
| `how-to/configure-for-dhcp.mdx` | Preserve pre-prose intro and inline `## Example config` code block. Replace `<HowTo>` JSX with `## Goal`, `## Prerequisites`, `## Steps` (ordered list — one-liners), `## Verification`, `## Notes` (verbatim). |
| `how-to/configure-for-static-ip.mdx` | Replace `<HowTo>` JSX with `## Goal`, `## Prerequisites`, `## Steps`, `## Verification`, `## Example config` (link to `/examples/static-ip.conf`), `## Notes`. |
| `how-to/configure-wifi-wpa2.mdx` | Same six sections as static-ip; preserve trailing `## Wi-Fi-first onboarding (without Ethernet)` section verbatim. |
| `how-to/discover-on-multi-nic-laptop.mdx` | Preserve pre-prose intro. Replace `<HowTo>` JSX with five sections (no `## Example config`). Preserve trailing `## Which one should I use?` and `## What's actually happening on the wire?` verbatim. |
| `how-to/recover-stuck-in-init-mode.mdx` | Preserve pre-prose intro. Replace `<HowTo>` JSX with five sections (no `## Example config`). Preserve trailing `## Why does this happen?` verbatim. |

**Procedure-archetype rewrites (canonical structure, no component change):**

| File | Change |
|---|---|
| `tutorials/configure-your-first-squeezebox.mdx` | Rename `## What you need` → `## Prerequisites`. Replace five `## Step N — Title` headings with `## Steps` group followed by `### 1. Title` through `### 5. Title`. Preserve every trailing section. |
| `how-to/no-devices-found.mdx` | Preserve the existing intro paragraph. Replace `## Step N: Title` × 4 with `## Steps` group followed by `### 1. Title` through `### 4. Title`. |
| `how-to/install-go-udap.mdx` | Replace `## Option N — Title` × 3 with `## Options` group followed by `### 1. Homebrew (macOS and Linux)`, `### 2. Pre-built binaries`, `### 3. Build from source`. Preserve trailing `## Verify the install` section. |

**Explainer-archetype fixes:**

| File | Change |
|---|---|
| `concepts/setup-vs-run-mode.mdx` | Delete the in-body `# Setup mode vs. run mode` line. The frontmatter title already renders as the page H1. |
| `concepts/squeezeplay-comparison.mdx` | Drop the `## N.` numbering from headings 1-10; keep the descriptive titles unchanged (`## Packet framing and protocol primitives`, `## Discovery TLVs and decoding`, etc.). |

**Reference-archetype fix:**

| File | Change |
|---|---|
| `reference/commands/set.mdx` | Move the `## Examples` section to after `## Exit codes` so the canonical sequence (`What it does` → `Flags` → `Exit codes` → `Examples`) matches the eight sibling command pages. |

**Component removal:**

| File | Change |
|---|---|
| `docs/site/components/how-to.tsx` | Deleted. |
| `docs/site/components/mdx.tsx` | Remove the `import { HowTo }` line and the `HowTo` entry in the `getMDXComponents` returned object. |

**New style guide:**

| File | Change |
|---|---|
| `docs/site/content/docs/contributing/docs-style.mdx` | New page documenting the three archetypes, the three universal rules, the recipe-section order, and the convention for when to use a Callout / Cards / LinkCard. The page is itself written in the style it documents. |
| `docs/site/content/docs/contributing/index.mdx` | Add a link to the new Documentation style page. |
| `CLAUDE.md` | Add a one-line pointer: *Documentation style guide lives at `docs/site/content/docs/contributing/docs-style.mdx` — follow it when editing any docs page.* |

### Sequencing

Five commits on a single branch `docs/heading-consistency`, opened
as one PR. Each commit is independently buildable.

1. **`docs(site): add documentation style guide`** —
   `contributing/docs-style.mdx`, `contributing/index.mdx`,
   `CLAUDE.md` pointer. Lands first so subsequent commits have a
   target the reviewer can compare against.
2. **`docs(site): convert HowTo pages to plain markdown`** — the
   five `how-to/configure-*`, `how-to/discover-on-multi-nic-laptop`,
   `how-to/recover-stuck-in-init-mode` rewrites. Pages no longer
   reference `<HowTo>`; the component file still exists.
3. **`docs(site): remove unused HowTo MDX component`** —
   `components/how-to.tsx` deleted, `components/mdx.tsx` updated.
   Build passes because step 2 ensured no page still uses it.
4. **`docs(site): apply canonical procedure structure to tutorial,
   no-devices-found, install-go-udap`** — the three remaining
   Procedure-archetype rewrites.
5. **`docs(site): fix concept and reference heading slips`** —
   `setup-vs-run-mode`, `squeezeplay-comparison`, `commands/set`.

### Verification per commit

Each commit must pass before being made:

- `pnpm build` from `docs/site/` returns 0 errors.
- `task docs:serve` (or `pnpm start:local`) renders the changed
  page(s) with no console errors.

After commit 3:

- `rg '<HowTo' docs/site/content/` returns no matches.

After the final commit:

- `rg HowTo docs/site/` returns exactly one match: a one-line
  mention in `docs-style.mdx` that the previous `<HowTo>` component
  has been replaced by plain markdown. No other matches anywhere.
- Visual smoke-test in `task docs:serve` of every changed page: the
  new heading hierarchy renders correctly under the larger heading
  scale introduced by PR #122, sidebar TOC matches the new
  structure, no internal links are broken.
- `git diff --name-only main..HEAD` matches the file list above;
  nothing else has slipped in.

## Success criteria

- All 15 file changes specified above have landed on
  `docs/heading-consistency`.
- `pnpm build` from `docs/site/` succeeds with 0 errors.
- `rg '<HowTo'` and `rg "from '@/components/how-to'"` return zero
  matches anywhere under `docs/site/`.
- Visual smoke-test confirms every changed page renders the new
  heading hierarchy correctly in `task docs:serve`.
- The new `contributing/docs-style.mdx` page itself follows the
  conventions it documents (self-consistency).

## Risks and mitigations

- **Sidebar TOC noise.** Adding a `## Steps` group heading where a
  page previously had flat steps adds one TOC entry. On pages where
  this hurts more than helps, the brief-vs-substantial rule allows
  using an ordered list instead of H3 sub-headings; the
  implementation chooses per page during verification.
- **Implicit `<HowTo>` features lost.** The component's
  `exampleFile` prop auto-renders a download link. Plain markdown
  replaces this with `## Example config` containing a plain markdown
  link to the same `.conf` file. The Starlight-branch rewrite
  already proved this is acceptable.
- **Stale references to the removed component.** `rg HowTo` after
  commit 3 catches anything in the page content; the build will
  catch anything that breaks compilation. Together they are
  sufficient.
- **Factual drift during rewrite.** Conversion is mechanical:
  preserve every fact, prereq, step, command, warning, and note
  verbatim. Reviewer's diff check looks for added or removed
  semantic content, not just structural change.

## Open questions

None at design time. Decisions resolved during brainstorming:

- Archetype taxonomy: three (Procedure / Explainer / Reference).
- `<HowTo>` disposition: removed, plain markdown everywhere.
- Tutorial section name: `## Prerequisites` (not `## What you need`).
- Brief vs substantial steps: ordered list for one-liners, H3
  numbered sub-headings for substantial steps.
- Style guide location: both `contributing/docs-style.mdx` (canonical)
  and a one-line pointer in root `CLAUDE.md`.
- Scope of `install-shell-completions` and the three `contributing/*`
  pages: left unchanged; their H2s are descriptive, not anti-patterns.
