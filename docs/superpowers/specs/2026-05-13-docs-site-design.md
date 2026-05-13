# go-udap documentation site

**Date:** 2026-05-13
**Status:** Draft — awaiting implementation plan
**Closes (tracker):** Task #28

## Summary

Replace the current scattered, plain-Markdown docs (`README.md`,
`CLAUDE.md`, `DEVELOPMENT.md`, `docs/notes/`, `docs/playbooks/`) with a
modern, navigable, searchable documentation site published to GitHub
Pages. The site applies the [Diataxis](https://diataxis.fr) framework
to separate four distinct documentation modes (Tutorial / How-to
guides / Reference / Explanation) and is built with
[Fumadocs](https://fumadocs.vercel.app/). Versioning is supported from
day one: `https://yo61.github.io/go-udap/` always reflects the current
`main` HEAD; archived snapshots live at
`https://yo61.github.io/go-udap/v<X.Y.Z>/`.

The site is built iteratively — existing docs are re-classified by
Diataxis quadrant and improved in place rather than starting from an
empty navigation tree. Two genuine content gaps (a hand-held Tutorial
and a set of task-templated How-to guides) are filled net-new during
implementation.

## Motivation

- **User-facing docs are scattered**. A reader who lands on the repo
  has no single entry point that surfaces installation, common tasks,
  the CLI surface, and the protocol notes in a coherent flow.
- **Task-based docs don't exist**. Every existing "how do I configure
  a Squeezebox" answer is either embedded in `README.md`'s flat
  Examples list or inferred from the per-subcommand `--help` output.
  A new user has to assemble the procedure themselves.
- **No version-aware docs**. A user running v1.6.0 has no way to read
  the v1.6.0 docs — the README they see is always whatever's on `main`.
- **The current README has Diataxis-violating mixed sections**.
  `Examples` blends tutorial-shaped and how-to-shaped content; the
  `Troubleshooting` section blends actionable recipes with conceptual
  explanation. Users seeking different kinds of help find them
  interleaved on the same page.

## References

- [Diataxis framework](https://diataxis.fr) — Daniele Procida's
  four-quadrant docs taxonomy (Tutorial / How-to / Reference /
  Explanation), the organising principle for this site.
- [Fumadocs](https://fumadocs.vercel.app/) — Next.js-based docs
  framework chosen for modern aesthetic + MDX-driven component reuse.
- `documentation` skill (locally installed from
  smithery.ai/skills/wodsmith/documentation) — Diataxis-aware Claude
  Code skill that will be loaded during content authoring to enforce
  per-quadrant principles.
- Task #28 (tracker) — the parent feature.
- Task #34 (tracker) — ad-hoc Wi-Fi onboarding investigation;
  surfaces as a "coming soon" stub page in this site.
- Task #35 (tracker) — release-please switch; affects snapshot
  cadence but not the site's design.

## Design lens

**Diataxis bottom-up, not top-down.** Per the loaded `documentation`
skill: "Do NOT create empty structures. Start small. Pick any existing
documentation. Apply compass to determine what type it should be.
Improve it. Repeat. Let structure emerge from the inside out." The
implementation classifies and improves existing pages first; site
navigation falls out of the classified set rather than being designed
in advance.

**Task-first ordering.** The user has explicitly asked for the front
door to lead with "how do I..." content. Tutorials and How-to guides
sit higher in the visual hierarchy than Reference and Explanation. A
new reader looking for "configure my first Squeezebox" finds it
without having to know what subcommand to invoke.

**Modern aesthetic.** The visual mood is "tech-startup-2025": sharp
typography, generous whitespace, dark-mode-first, distinctive enough
to not look like every other Sphinx-derived docs site. Fumadocs gives
this out of the box.

## Architecture

```
yo61/go-udap repo
└── docs/
    └── site/                              # Fumadocs project
        ├── package.json
        ├── next.config.mjs
        ├── tailwind.config.ts             # if Fumadocs requires
        ├── source.config.ts               # Fumadocs source config
        ├── content/
        │   └── docs/
        │       ├── tutorials/             # Diataxis quadrant 1
        │       ├── how-to/                # Diataxis quadrant 2
        │       ├── reference/             # Diataxis quadrant 3
        │       ├── concepts/              # Diataxis quadrant 4
        │       ├── contributing/          # Internal navigation
        │       ├── examples/              # Downloadable .conf files
        │       └── v1.6.0/                # Archived per-release snapshots
        │           └── (content tree)     #   appear here as releases ship
        ├── components/
        │   └── how-to.tsx                 # MDX <HowTo> component
        └── public/
            └── (static assets, OG images)
```

Hosting: GitHub Pages, deployed from the `gh-pages` branch via
`.github/workflows/docs.yml`.

URL contract:
- `https://yo61.github.io/go-udap/` — main HEAD (the working "latest")
- `https://yo61.github.io/go-udap/v<X.Y.Z>/` — released-version snapshot
- `https://yo61.github.io/go-udap/v<X.Y.Z>-rc.N/` — pre-release snapshot
  (when release-please pre-release support lands per Task #35)

The version selector in the site header lists "Latest" + each released
version. Pre-releases live in an expandable "Pre-releases" sub-group to
avoid cluttering the main list.

## Components

### Per-quadrant page templates

Each Diataxis quadrant has shape rules. Pages are reviewed against
their quadrant's reference (loaded by the `documentation` skill on
demand) before being merged.

| Quadrant | Page shape | Example |
|---|---|---|
| **Tutorial** | One curated linear walkthrough. Single happy path. Hand-holds. Includes "verify it worked" cues at every step. | `tutorials/configure-your-first-squeezebox.md` |
| **How-to guide** | Strict template: `<HowTo goal prerequisites steps verification exampleFile>`. One user goal per page. Multiple acceptable routes are out of scope; if a goal forks, it's two how-tos. | `how-to/configure-for-dhcp.md` |
| **Reference** | Lookup-shaped. Tables, lists, type signatures, exit codes. No prose narrative beyond a one-line page summary. | `reference/commands/discover.md`, `reference/nvram-parameters.md` |
| **Explanation** | Long-form prose. Answers "why does it work this way?". Cross-links to relevant reference and how-to pages but doesn't try to be either. | `concepts/how-udap-discovery-works.md` |

### The `<HowTo>` MDX component

How-to pages use a single MDX component for structural consistency:

```mdx
<HowTo
  goal="Configure a Squeezebox for DHCP"
  prerequisites={[
    "Device connected by Ethernet to the same subnet as the dev machine",
    "Device in init or setup mode (front button held 3-6s, light flashing red)",
    "go-udap installed and on PATH"
  ]}
  steps={[
    "Discover the device's MAC: `go-udap discover`",
    "Apply the config: `go-udap set <mac> --config dhcp.conf --reboot`"
  ]}
  verification="Run `go-udap getip <mac>` — should return a leased IP from your DHCP server"
  exampleFile="/examples/dhcp.conf"
/>
```

Renders as four labelled sections with consistent typography and a
downloadable-file CTA. Built once, reused across every how-to page.

### Reference pages — API surface

The udap library's API (`Client`, `Device`, `NetworkConfig`,
`NetInterface`, `Transport`, etc.) does NOT live in this site as
hand-written reference. Instead, the Reference section's API page is a
short index page that points readers at `pkg.go.dev/go-udap` for the
auto-generated, always-current Go API reference. Rationale: pkg.go.dev
already does this well for free and updates on every release tag; we
shouldn't duplicate it.

CLI subcommands (one page per subcommand) and NVRAM parameters (the
26-entry table) ARE hand-written reference pages — they're not
auto-derivable.

### Versioning snapshot mechanism

On every release tag (`v<X.Y.Z>` or `v<X.Y.Z>-rc.N`), the Release
workflow:

1. Builds the docs from `docs/site/content/docs/` (the "latest" tree).
2. Copies that tree into `docs/site/content/docs/v<X.Y.Z>/`.
3. Commits the copy as part of the release PR.
4. Triggers the Pages deploy.

Result: `/v<X.Y.Z>/` URLs serve frozen content corresponding to that
release. "Latest" continues to track `main` HEAD.

The site chrome (theme, layout, components, version selector) is
single-source-of-truth at `docs/site/` top level; chrome updates
propagate to every version's pages. Content is frozen per version.

Storage cost: each snapshot duplicates the content tree. For
documentation-sized text (kilobytes per page), this is negligible. The
alternative — per-page versioning or git-tag-based rebuilds — adds
significant CI complexity for what should be a write-once-then-frozen
artifact.

## Content architecture

### Inventory + Diataxis classification of existing docs

| Source | Target quadrant | New location |
|---|---|---|
| `README.md` § Overview | Explanation | `concepts/what-is-go-udap.md` |
| `README.md` § Installation | How-to | `how-to/install-go-udap.md` |
| `README.md` § Usage / Commands table | Reference | Index of `reference/commands/`; each subcommand gets its own page |
| `README.md` § Output | Reference | Folded into each subcommand's reference page |
| `README.md` § Examples (mixed) | **Splits**: most become how-tos; one becomes the seed for the tutorial | See "Net-new content" below |
| `README.md` § Config file format | Reference | `reference/config-file-format.md` |
| `README.md` § Configuration Parameters | Reference | `reference/nvram-parameters.md` |
| `README.md` § Troubleshooting (mixed) | **Splits**: actionable items become how-tos; "why does it work this way" become explanation pages | See "Net-new content" below |
| `DEVELOPMENT.md` | Contributor docs | `contributing/index.md` and sub-pages |
| `docs/notes/squeezeplay-comparison.md` | Explanation | `concepts/squeezeplay-comparison.md` |
| `docs/playbooks/sbr-capture.md` | How-to (operational) | `how-to/capture-wire-traces.md` |
| `CLAUDE.md` | (NOT in public site) | Stays in repo root as agent instructions |

### Net-new content

**The tutorial** — `tutorials/configure-your-first-squeezebox.md`. ~500-800 words.
Single linear walkthrough assuming wired Ethernet, same subnet,
factory-state device. Hand-holds through:
`discover` → `info` → `set --hostname <name>` →
`set --config first-setup.conf --reboot` → re-discover to verify.
Includes expected-output screenshots at each step.

**How-to guides surfaced by Phase 6-7 / Phase 2-3 work**:

- `how-to/configure-for-dhcp.md` + `examples/dhcp.conf`
- `how-to/configure-for-static-ip.md` + `examples/static-ip.conf`
- `how-to/configure-wifi-wpa2.md` + `examples/wifi-wpa2.conf`
- `how-to/discover-on-multi-nic-laptop.md` — explains `--interface` / `--all-interfaces`
- `how-to/recover-stuck-in-init-mode.md` — operational recovery recipe
- `how-to/onboard-over-wifi-ad-hoc.md` — stub linking to Task #34

**Explanation pages surfaced from `Troubleshooting` split**:

- `concepts/how-udap-discovery-works.md` — explains limited
  broadcast, why unconfigured devices reply from `0.0.0.0`,
  the "TLV 0x0d may be absent on older firmware" gap and the
  `get_uuid` fallback (commit `b9fd106`).
- `concepts/multi-nic-discovery.md` — explains the
  `IP_BOUND_IF` / `SO_BINDTODEVICE` / `SO_REUSEPORT` plumbing
  surfaced by Phase 6/7 work. References the spike-result
  notes in `docs/superpowers/plans/2026-05-13-getip-hwrev-uuid-iface.md`.
- `concepts/the-26-nvram-parameters.md` — explains why the
  parameter table is what it is, with Net::UDAP citation.

### After-the-fact disposition of original files

- `README.md` becomes a thin landing page: project tagline, install
  one-liner, pointer to the published site, license. ~30-50 lines max.
- `DEVELOPMENT.md` is deleted from the repo root once its content lives
  in `contributing/`.
- `docs/notes/squeezeplay-comparison.md` and
  `docs/playbooks/sbr-capture.md` are deleted from their current
  locations once their content lives in the site.

## Implementation phasing

Four milestones, each a separately mergeable PR.

| M | Goal | Acceptance |
|---|---|---|
| **M1: Skeleton deploys** | Fumadocs scaffolded in `docs/site/`. Theme, header/nav shell, version selector, GitHub Actions deploy workflow all working. Single placeholder page lands at `https://yo61.github.io/go-udap/` | Site URL serves something. CI deploys on push to `main`. Version selector renders even though only "Latest" exists. |
| **M2: Existing content classified and live** | Phase A complete. All inventoried existing docs re-classified and ported into the Diataxis quadrant structure. The two mixed-quadrant offenders (README §Examples, §Troubleshooting) properly split. | Every existing piece of public docs now lives in exactly one quadrant. `README.md` is reduced to a thin landing page. `DEVELOPMENT.md`, `docs/notes/squeezeplay-comparison.md`, `docs/playbooks/sbr-capture.md` deleted from their old locations. |
| **M3: Net-new content** | Phase B complete. Tutorial written. Six how-to pages written (one is a "coming soon" stub for ad-hoc Wi-Fi pending Task #34). MDX `<HowTo>` component built and applied to every how-to page. Three explanation pages written. | A new reader can land cold, follow the tutorial, and successfully configure a Squeezebox. Every how-to page renders with the `<HowTo>` component's consistent structure. |
| **M4: Polish + versioning snapshot mechanism** | Custom landing page with impact-mode hero. Versioning snapshot wired into the Release workflow. First snapshot (current `main`) tagged as `/v<current-version>/`. | Landing page lands. Releasing the next version (via semantic-release or release-please per Task #35) auto-snapshots its content tree. |

M1 ships a working empty site; subsequent milestones each add
independent value. No hard dependencies between PRs other than
ordering; M2 expects M1's scaffolding to exist, M3 expects M2's
classified content to exist, M4 expects M3's content to be stable
enough to snapshot.

## Out of scope

- **Auto-generated API reference** for the `udap` and `mocksbr`
  packages. Link to `pkg.go.dev/go-udap` instead. Auto-generation
  would need gomarkdoc tooling, regen on every release, and would
  duplicate what pkg.go.dev already does well for free.
- **Custom search backend** (Algolia, Meilisearch, etc.). Fumadocs's
  built-in client-side search is sufficient for a ~30-page site.
- **Interactive widgets** (live UDAP command runner, etc.). Out of
  scope and complicated by the fact that UDAP operations need real
  hardware on the local LAN; no useful interactive demo is possible.
- **Internationalisation**. Single-language (English) only.
- **Per-release branch-based docs**. The folder-snapshot approach is
  simpler and sufficient.
- **Site analytics / telemetry**. Not adding any tracking.
- **The ad-hoc Wi-Fi onboarding how-to**. Tracked as Task #34; this
  spec ships a "coming soon" stub page that will be filled by that
  task's outcome.
- **The release-please switch**. Tracked as Task #35; this spec
  designs the snapshot trigger to be release-tag-based so it survives
  the eventual tooling switch without change.

## Open questions / risks

- **Fumadocs API churn**. Fumadocs is younger than alternatives like
  Astro Starlight; its APIs evolve faster. Mitigation: pin the Fumadocs
  version in `package.json` and update deliberately; the site is small
  enough that breaking changes are tractable to absorb.
- **Node toolchain in CI** — adds a new dependency to the build
  matrix. The repo already has Go-based CI; we're adding Node-based
  CI alongside. Build times stay parallelisable. Risk is low; mature
  GitHub Actions exist for `actions/setup-node` and Fumadocs is a
  standard Next.js project.
- **Disk-space accumulation of versioned snapshots**. After a year of
  releases the `docs/site/content/docs/v*` tree could hold dozens of
  near-duplicate trees. Mitigation: documented in the spec, monitored
  during M4. A future task can introduce git-tag-based snapshot
  retrieval if storage becomes a real concern (it won't).
- **The "Latest" docs may show pre-release work**. The site's
  unprefixed URL tracks `main` HEAD, which can include unreleased
  changes. Acceptable: users browsing latest are by definition looking
  for the leading edge; users wanting a stable release pin should use
  the version selector or a `/v<X.Y.Z>/` URL.
- **`<HowTo>` template rigidity**. A strict component may not fit every
  how-to. Mitigation: pages that don't fit fall back to plain Markdown
  and an open question is logged for whether the template should grow
  or shrink.

## Dependencies

- **Task #34** (ad-hoc Wi-Fi onboarding) — soft dependency. The
  "coming soon" stub page in M3 references that task; the page
  becomes a real how-to when Task #34 completes. No blocker.
- **Task #35** (release-please switch) — soft dependency. The
  versioning-snapshot mechanism is designed to be release-tag-based,
  agnostic to whether the tag came from semantic-release (current) or
  release-please (future). No blocker.
- **Hardware verification (Task #31)** — soft dependency. The
  Tutorial assumes a real Squeezebox is available to the user; if
  Task #31 surfaces non-SBR-specific procedures, additional how-to
  pages may be added later. Not blocking M1-M4.

## Acceptance summary

The docs site is complete when:

1. `https://yo61.github.io/go-udap/` serves a modern, navigable docs
   site reflecting `main` HEAD.
2. Every Diataxis quadrant (Tutorial / How-to / Reference /
   Explanation) is populated; no empty navigation entries.
3. Every existing piece of public-facing docs has been moved into the
   site and removed from its original location, with `README.md`
   reduced to a thin landing page.
4. The version selector shows "Latest" + the currently-tagged
   release; releasing a new version triggers a new snapshot via the
   Release workflow without manual intervention.
5. The site survives `release-please` swap (Task #35) without code
   changes to the docs deploy pipeline.
