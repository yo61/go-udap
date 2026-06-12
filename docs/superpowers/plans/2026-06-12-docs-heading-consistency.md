# Site-wide Docs Heading Consistency Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Apply the canonical heading convention across every page of the docs site so they look and read as one document, remove the `<HowTo>` MDX component, and write the convention down both inside the docs site (`contributing/docs-style.mdx`) and as a one-line pointer in root `CLAUDE.md`.

**Architecture:** Five small commits on a single branch, each independently buildable. Order: write the style guide first → convert the 5 HowTo pages to plain markdown → remove the now-unused component → apply canonical structure to the tutorial / no-devices-found / install-go-udap → fix the remaining concept and reference slips. Conversion is mechanical: every fact, prerequisite, step text, command, warning, and note is preserved verbatim; only the structure changes.

**Tech Stack:** Fumadocs UI (MDX), Next.js 16, pnpm, Task. No tests required (docs-only changes); verification is `pnpm build` from `docs/site/` plus visual smoke-test via `task docs:serve`.

**Spec:** `docs/superpowers/specs/2026-06-12-docs-heading-consistency-design.md`

**Branch:** `docs/heading-consistency` (created during brainstorming, currently has the spec commit `6c1f635`).

---

## Why this plan doesn't use unit-test TDD

This is editorial content work. "Did the heading appear?" is verified by `pnpm build` succeeding and by reading the rendered page. Each task uses two verification methods in place of unit tests:

1. **`pnpm build` from `docs/site/`** — fails on MDX syntax errors, missing imports, broken internal links if `astro check`-equivalent strictness is enabled.
2. **`task docs:serve` (or `pnpm start:local`)** — serves the production build locally; engineer manually verifies the changed pages render with correct headings, sidebar TOC reflects the new structure, no console errors.

After every task that touches MDX content, both verifications must pass before the commit lands.

---

## File structure (all 15 changes)

```
docs/site/content/docs/
├── contributing/
│   ├── docs-style.mdx                                       # NEW (Task 1)
│   └── index.mdx                                            # +1 link (Task 1)
├── tutorials/
│   └── configure-your-first-squeezebox.mdx                  # rewrite (Task 4)
├── how-to/
│   ├── configure-for-dhcp.mdx                               # rewrite, drop <HowTo> (Task 2)
│   ├── configure-for-static-ip.mdx                          # rewrite, drop <HowTo> (Task 2)
│   ├── configure-wifi-wpa2.mdx                              # rewrite, drop <HowTo> (Task 2)
│   ├── discover-on-multi-nic-laptop.mdx                     # rewrite, drop <HowTo> (Task 2)
│   ├── recover-stuck-in-init-mode.mdx                       # rewrite, drop <HowTo> (Task 2)
│   ├── no-devices-found.mdx                                 # rewrite (Task 4)
│   └── install-go-udap.mdx                                  # rewrite (Task 4)
├── concepts/
│   ├── setup-vs-run-mode.mdx                                # delete one H1 line (Task 5)
│   └── squeezeplay-comparison.mdx                           # drop H2 numbering (Task 5)
└── reference/
    └── commands/
        └── set.mdx                                           # reorder two sections (Task 5)

docs/site/components/
├── how-to.tsx                                                # DELETED (Task 3)
└── mdx.tsx                                                   # remove import + entry (Task 3)

CLAUDE.md                                                     # +1 line pointer (Task 1)
```

---

## Task 1: Add the documentation style guide

**Files:**
- Create: `docs/site/content/docs/contributing/docs-style.mdx`
- Modify: `docs/site/content/docs/contributing/index.mdx`
- Modify: `CLAUDE.md`

- [ ] **Step 1: Create `docs/site/content/docs/contributing/docs-style.mdx`**

Write this exact content. The page itself follows the conventions it documents — flat H2s for parallel topics (the Explainer archetype), descriptive H2s (no anti-pattern numbering):

```mdx
---
title: Documentation style
description: Heading conventions and page archetypes for this site
---

Every page on this docs site fits one of three archetypes. Pick the
one your page is, follow its canonical heading shape, stop. This page
is itself written in the Explainer archetype it documents below.

## Archetypes

### Procedure

A step-by-step or option-by-option recipe. The reader follows in
order. Examples: the tutorial, every `how-to/*` page, every
`contributing/*` page.

Heading shape: a top-level group heading (`## Steps`, `## Options`,
`## Methods`, `## Setup`, or whatever names the group) followed by
numbered sub-headings of the form `### N. Title`. Never use
`### Step N — Title` or `### Option N — Title` — the leading
`N.` is the only numbering format.

If each step is a single line (typically one command), use an
ordered markdown list under `## Steps` instead of H3 sub-headings.
Substantial steps with paragraphs of explanation get H3 sub-headings;
brief one-liner steps get a numbered list.

The recipe-shaped how-to has a canonical section order:

```
(intro paragraph, optional)
(## Example config — optional, when the example is shown inline above the recipe)
## Goal
## Prerequisites
## Steps
## Verification
## Example config       (optional, when the example is shown after the recipe)
## Notes                (optional)
(trailing topical sections, optional — e.g. "## Why does this happen?")
```

The tutorial archetype uses `## Prerequisites`, `## Steps` with H3
numbered sub-headings, then any trailing tutorial sections
(`## What's next?` etc.). It does not include `## Goal` —
tutorials are framed by their title and intro, not by a Goal section.

### Explainer

A conceptual page. Parallel topics, no inherent ordering. The reader
typically arrives, reads the section they need, leaves. Examples:
every `concepts/*` page, `reference/global-flags`, this page.

Heading shape: flat unnumbered `##` headings. Multiple H2s in
sequence; each is a parallel topic. Never use `## 1. Title` /
`## Step N` / `### N.` — the numbering implies ordering the reader
does not need.

Sub-headings (`###`) are fine when a single topic decomposes
naturally; they should not be numbered.

### Reference

Looked-up data, not read top-to-bottom. The reader is verifying or
copying. Examples: every `reference/commands/*` page, the
NVRAM-parameter table, exit-codes.

Heading shape: sibling pages use the **same** H2 sequence in the
**same** order so the reader navigates by muscle memory.

For per-command pages the canonical H2 order is:

```
## What it does
## Output
## Flags
## Exit codes
## Examples
```

Pages where a section does not apply (e.g. a command with no flags)
omit the H2 entirely rather than ship an empty section.

## Universal rules

These three rules apply across every archetype.

### No H1 in body

The page title comes from frontmatter (`title:`) and is already
rendered as the page H1 by the Fumadocs layout. Body content starts
at H2. A literal `# Title` line in the body creates a double-H1.

### No "Step N —" / "Option N —" prefixes in headings

Use the Procedure archetype's group-heading + numbered-sub-heading
pattern instead. Compare:

```markdown
## Step 1 — Discover the device         ← wrong
## Step 2 — Read the parameters         ← wrong

## Steps                                 ← right
### 1. Discover the device
### 2. Read the parameters
```

### No standalone numbered H2s

```markdown
## 1. Packet framing                    ← wrong (anti-pattern)
## 2. Discovery TLVs                    ← wrong

## Packet framing                       ← right (flat parallel topics)
## Discovery TLVs

## Topics                                ← also right (group + numbered)
### 1. Packet framing
### 2. Discovery TLVs
```

If the topics are ordered, use the Procedure archetype. If they are
parallel, use the Explainer archetype. Numbered standalone H2s are
the worst of both — they imply ordering without a group heading to
frame it.

## Components

Fumadocs UI ships components — `Callout`, `Cards`, `LinkCard` — that
the docs site uses. Use them when they add value (a `Callout` for a
warning the reader must not miss; `Cards` / `LinkCard` for landing-page
navigation).

There is no `<HowTo>` component. A previous custom `<HowTo>` MDX
component was removed because it imposed a visual style (shaded boxes)
on five how-to pages that diverged from the rest of the site, and
because at least five existing how-to pages did not fit its
Goal/Steps/Verification mould. Recipe-shaped how-tos now use the
plain-markdown section order documented above.
```

- [ ] **Step 2: Add a link from `contributing/index.mdx`**

Edit `docs/site/content/docs/contributing/index.mdx`. Find the existing list of links:

```mdx
- [Building from source](./building-from-source.mdx) — prerequisites, build, test, lint
- [Cross-compilation](./cross-compilation.mdx) — targets and UPX packing
- [Release process](./release-process.mdx) — how releases are cut and published
```

Add one new line at the top of that list, so it becomes:

```mdx
- [Documentation style](./docs-style.mdx) — heading conventions and page archetypes
- [Building from source](./building-from-source.mdx) — prerequisites, build, test, lint
- [Cross-compilation](./cross-compilation.mdx) — targets and UPX packing
- [Release process](./release-process.mdx) — how releases are cut and published
```

Nothing else in `contributing/index.mdx` changes.

- [ ] **Step 3: Add a one-line pointer in root `CLAUDE.md`**

Edit `/Users/robin/code/github.com/yo61/go-udap/CLAUDE.md`. Find the section heading `## Common Commands`. Immediately *before* that section, add a new short section:

```markdown
## Documentation style

Heading conventions and page archetypes for the docs site are
documented at `docs/site/content/docs/contributing/docs-style.mdx`.
Follow that guide when editing any docs page.

```

(Blank line above and below the new section.)

- [ ] **Step 4: Build to verify nothing breaks**

```bash
cd /Users/robin/code/github.com/yo61/go-udap/docs/site
pnpm build
```

Expected: 0 errors. The new `docs-style.mdx` page builds; the link in `contributing/index.mdx` resolves.

- [ ] **Step 5: Visual smoke-test the new page**

```bash
cd /Users/robin/code/github.com/yo61/go-udap/docs/site
pnpm start:local
```

Open `http://localhost:3000/contributing/docs-style/`. Confirm: page renders, sidebar shows it in the Contributing section, no console errors. Then open `http://localhost:3000/contributing/` and confirm the new "Documentation style" link is present and clicking it goes to the new page.

Stop the server (`Ctrl+C`).

- [ ] **Step 6: Commit**

```bash
cd /Users/robin/code/github.com/yo61/go-udap
git add docs/site/content/docs/contributing/docs-style.mdx \
        docs/site/content/docs/contributing/index.mdx \
        CLAUDE.md
git commit -m "$(cat <<'EOF'
docs(site): add documentation style guide

New page documents the three archetypes (Procedure / Explainer /
Reference), the canonical heading shapes for each, the three universal
rules, and the deprecation of the <HowTo> component. The page is
itself written in the style it documents.

CLAUDE.md adds a one-line pointer so future sessions follow the guide
when editing docs pages.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Convert the five `<HowTo>` pages to plain markdown

Each page becomes plain markdown using the recipe-shaped how-to section order from the style guide. The `<HowTo>` JSX is removed; nothing else about the page (intro prose, notes, trailing sections, internal links) is changed.

**Files (all modified):**
- `docs/site/content/docs/how-to/configure-for-dhcp.mdx`
- `docs/site/content/docs/how-to/configure-for-static-ip.mdx`
- `docs/site/content/docs/how-to/configure-wifi-wpa2.mdx`
- `docs/site/content/docs/how-to/discover-on-multi-nic-laptop.mdx`
- `docs/site/content/docs/how-to/recover-stuck-in-init-mode.mdx`

- [ ] **Step 1: Replace `configure-for-dhcp.mdx` with this exact content**

```mdx
---
title: Configure a Squeezebox for DHCP
description: Wired Ethernet, DHCP-assigned IP
---

A factory-fresh Squeezebox defaults to DHCP, so the minimal config
only needs to point it at your Lyrion Music Server.

## Example config

```ini title="dhcp.conf"
# Set your music server's IP:
server_address=192.168.1.250
```

## Goal

Configure a Squeezebox to use DHCP on wired Ethernet, pointed at your Lyrion Music Server.

## Prerequisites

- Device connected by Ethernet to the same subnet as the dev machine
- Device in setup or init mode (front-button hold 3-6s if needed; new devices are already in setup mode)
- `go-udap` installed and on `PATH`
- Your Lyrion Music Server's IP address known (e.g. `192.168.1.250`)

## Steps

1. Save the example config above as `dhcp.conf` (or [download it](/examples/dhcp.conf)) and edit `server_address` to your LMS IP.
2. Discover the device: `go-udap discover`
3. Apply the config and reboot: `go-udap set <mac> --config dhcp.conf --reboot`

## Verification

After ~15 seconds, run `go-udap getip <mac>`. You should see a DHCP-assigned IP from your network's subnet.

## Notes

- Devices default to DHCP, so `lan_ip_mode=1` is omitted.
- On factory-fresh devices, `go-udap set` defaults to wired Ethernet
  when `--interface` is not passed (or wireless if `--wireless-ssid`
  is), so `interface=1` is omitted too. Requires go-udap v2.1.0+; on
  earlier versions, add `interface=1` to the config.
- To configure wireless or static IP instead, see [Configure a
  Squeezebox for Wi-Fi with WPA2](/how-to/configure-wifi-wpa2) or
  [Configure a Squeezebox for static IP](/how-to/configure-for-static-ip).
- The `--reboot` flag is necessary; network config changes don't take
  effect until the device reboots.
```

- [ ] **Step 2: Replace `configure-for-static-ip.mdx` with this exact content**

```mdx
---
title: Configure a Squeezebox for static IP
description: Wired Ethernet with a fixed IP address
---

## Goal

Configure a Squeezebox to use a fixed IP address on wired Ethernet.

## Prerequisites

- Device connected by Ethernet to the same subnet as the dev machine
- Device in setup or init mode
- `go-udap` installed and on `PATH`
- Static IP details known: target IP, subnet mask, gateway, DNS
- Your Lyrion Music Server's IP address known

## Steps

1. Download the [example config](/examples/static-ip.conf) and edit the IP / subnet / gateway / DNS / LMS values to match your network.
2. Discover the device: `go-udap discover`
3. Apply: `go-udap set <mac> --config static-ip.conf --reboot`

## Verification

After ~15 seconds, run `go-udap getip <mac>`. The IP / Subnet / Gateway should match exactly what you put in the config.

## Example config

[static-ip.conf](/examples/static-ip.conf)

## Notes

- `lan_ip_mode=0` selects static IP. The four fields
  (`lan_network_address`, `lan_subnet_mask`, `lan_gateway`,
  `primary_dns`) become authoritative.
- Test the chosen IP isn't already in use on your network first
  (otherwise the device will appear to work but cause arp conflicts).
- If the device doesn't come back after reboot, the IP you chose
  isn't reachable. Factory-reset by holding the front button for 6+
  seconds and try again with a different IP.
```

- [ ] **Step 3: Replace `configure-wifi-wpa2.mdx` with this exact content**

```mdx
---
title: Configure a Squeezebox for Wi-Fi with WPA2
description: Wireless networking with WPA2-PSK authentication
---

## Goal

Configure a Squeezebox to use Wi-Fi with WPA2-PSK.

## Prerequisites

- Device connected by Ethernet for the initial setup (Wi-Fi-first onboarding is tracked separately, see below)
- Device in setup or init mode
- `go-udap` installed and on `PATH`
- Your Wi-Fi SSID and pre-shared key known
- Your Lyrion Music Server's IP address known

## Steps

1. Download the [example config](/examples/wifi-wpa2.conf) and edit `wireless_SSID`, `wireless_wpa_psk`, and `server_address`.
2. Discover the device (still over Ethernet): `go-udap discover`
3. Apply: `go-udap set <mac> --config wifi-wpa2.conf --reboot`
4. Unplug the Ethernet cable.

## Verification

After ~30 seconds (Wi-Fi association takes longer than wired), re-run `go-udap discover`. The device should appear with its DHCP-assigned Wi-Fi IP. If it doesn't, plug the Ethernet back in to recover and check the SSID/PSK values.

## Example config

[wifi-wpa2.conf](/examples/wifi-wpa2.conf)

## Notes

- `interface=0` selects wireless. The example config sets this
  explicitly. On factory-fresh devices `go-udap set` will also infer
  `interface=0` from `wireless_SSID` if you omit it (and report the
  inference on stderr), but pinning it in the config removes any
  ambiguity.
- `wireless_wpa_mode=2` is WPA2; `wireless_wpa_mode=1` is WPA;
  `wireless_wpa_mode=0` is WPA-mixed-mode.
- WEP is also supported (`wireless_wep_on=1`, `wireless_wep_key=...`)
  but is insecure; don't use it on networks you care about.

## Wi-Fi-first onboarding (without Ethernet)

Initial onboarding of a factory-state device over Wi-Fi (using the
device's built-in ad-hoc network) is a separate procedure. See
[Onboard over Wi-Fi using ad-hoc mode](./onboard-over-wifi-ad-hoc.mdx).
```

- [ ] **Step 4: Replace `discover-on-multi-nic-laptop.mdx` with this exact content**

```mdx
---
title: Discover on a multi-NIC laptop
description: Use --bind-interface or --all-interfaces when your dev machine has multiple network interfaces
---

If your laptop has Wi-Fi + wired Ethernet both up, default
`go-udap discover` only emits broadcasts on the kernel's
default-route NIC. If your Squeezebox is on the *other* interface's
subnet, you'll see zero devices.

## Goal

Make `go-udap discover` find a Squeezebox that's reachable only via a non-default NIC.

## Prerequisites

- `go-udap` installed and on `PATH`
- Multi-NIC dev machine (Wi-Fi + Ethernet, or multiple Ethernet)
- macOS or Linux — Windows multi-NIC discovery is not yet supported (tracked as Task #29)

## Steps

1. List your usable interfaces: `go-udap interfaces`
2. Either pick one explicitly: `go-udap --bind-interface en7 discover`
3. Or fan out across all of them: `go-udap --all-interfaces discover`

## Verification

You should now see the device's MAC in the output. Run `tcpdump -i any 'udp port 17784'` in another terminal to confirm the broadcast went out the NIC you expected.

## Which one should I use?

- **`--bind-interface NAME`** — when you know exactly which NIC reaches
  the device. Faster (one socket), simpler.
- **`--all-interfaces`** — when you're unsure. Sends concurrently on
  every interface; the device replies via whichever one reaches it.

The two flags are mutually exclusive. Combining them is a usage error
(exit code 1).

## What's actually happening on the wire?

See [Multi-NIC discovery](/concepts/multi-nic-discovery) for the
underlying mechanics (`IP_BOUND_IF`, `SO_BINDTODEVICE`,
`SO_REUSEPORT`).
```

- [ ] **Step 5: Replace `recover-stuck-in-init-mode.mdx` with this exact content**

```mdx
---
title: Recover a Squeezebox stuck in init mode
description: When a device won't leave init/setup state
---

A Squeezebox reports `State: init` when it hasn't yet completed
initial configuration. If a previous `go-udap set` was interrupted or
applied bad values, the device can sit in `init` indefinitely.

## Goal

Get a Squeezebox out of init state and into connected state.

## Prerequisites

- Physical access to the device
- `go-udap` installed
- Device connected to power and on the same LAN as the dev machine

## Steps

1. Confirm the device is reachable: `go-udap discover`
2. If the device is in init state and you want to start over: factory-reset by holding the front button for 6+ seconds. The light flashes rapidly red, then the device reboots into a clean factory state.
3. Re-run the [tutorial](/tutorials/configure-your-first-squeezebox) from a clean state.
4. If the device DOESN'T appear in `discover` after factory reset, see [Troubleshoot "no devices found"](/how-to/no-devices-found).

## Verification

After the tutorial completes successfully, `go-udap info <mac>` should show `State: connected` and a real DHCP IP (not `0.0.0.0`).

## Why does this happen?

`init` is the device's default state until something — typically the
server-association handshake — confirms the device has joined a music
server. If `server_address` was set to an unreachable IP, the device
stays in init forever waiting.

Factory reset is the safe recovery: it wipes NVRAM back to defaults
and starts the setup state machine over.
```

- [ ] **Step 6: Verify the `<HowTo>` component is no longer referenced in any content file**

```bash
cd /Users/robin/code/github.com/yo61/go-udap
rg '<HowTo' docs/site/content/
```

Expected: no matches (zero output).

If any match shows up, fix the offending file before continuing — the goal of this task is that no content file references `<HowTo>` anymore.

- [ ] **Step 7: Build to verify all five pages compile**

```bash
cd /Users/robin/code/github.com/yo61/go-udap/docs/site
pnpm build
```

Expected: 0 errors. The `<HowTo>` component is still defined and still imported by `components/mdx.tsx`; it's just no longer used by any page. The build should be clean.

- [ ] **Step 8: Visual smoke-test the five rewritten pages**

```bash
pnpm start:local
```

Open each in turn at `http://localhost:3000/`:

- `/how-to/configure-for-dhcp/`
- `/how-to/configure-for-static-ip/`
- `/how-to/configure-wifi-wpa2/`
- `/how-to/discover-on-multi-nic-laptop/`
- `/how-to/recover-stuck-in-init-mode/`

For each: confirm headings render in the new shape (`## Goal`, `## Prerequisites`, `## Steps`, `## Verification`, optional `## Example config`, `## Notes`), no shaded boxes anywhere, sidebar TOC matches, no console errors.

Stop the server.

- [ ] **Step 9: Commit**

```bash
cd /Users/robin/code/github.com/yo61/go-udap
git add docs/site/content/docs/how-to/configure-for-dhcp.mdx \
        docs/site/content/docs/how-to/configure-for-static-ip.mdx \
        docs/site/content/docs/how-to/configure-wifi-wpa2.mdx \
        docs/site/content/docs/how-to/discover-on-multi-nic-laptop.mdx \
        docs/site/content/docs/how-to/recover-stuck-in-init-mode.mdx
git commit -m "$(cat <<'EOF'
docs(site): convert HowTo pages to plain markdown

The five how-to pages that used the custom <HowTo> JSX component are
now plain markdown using the canonical recipe-shaped section order:
Goal, Prerequisites, Steps, Verification, optional Example config,
Notes. Pre-prose intros and trailing topical sections (Notes, Why does
this happen?, etc.) are preserved verbatim.

The <HowTo> component itself is removed in the following commit.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Remove the now-unused `<HowTo>` component

**Files:**
- Delete: `docs/site/components/how-to.tsx`
- Modify: `docs/site/components/mdx.tsx`

- [ ] **Step 1: Delete the component source file**

```bash
cd /Users/robin/code/github.com/yo61/go-udap
trash docs/site/components/how-to.tsx
```

(Use `rm` instead of `trash` if `trash` is unavailable. The file is 76 lines and contains only the `<HowTo>` component.)

- [ ] **Step 2: Remove the import and registration from `mdx.tsx`**

Edit `docs/site/components/mdx.tsx`. Current content:

```tsx
import defaultMdxComponents from 'fumadocs-ui/mdx';
import type { MDXComponents } from 'mdx/types';
import { HowTo } from './how-to';

export function getMDXComponents(components?: MDXComponents) {
  return {
    ...defaultMdxComponents,
    HowTo,
    ...components,
  } satisfies MDXComponents;
}

export const useMDXComponents = getMDXComponents;

declare global {
  type MDXProvidedComponents = ReturnType<typeof getMDXComponents>;
}
```

New content (delete the `import { HowTo }` line and the `HowTo,` entry in the returned object):

```tsx
import defaultMdxComponents from 'fumadocs-ui/mdx';
import type { MDXComponents } from 'mdx/types';

export function getMDXComponents(components?: MDXComponents) {
  return {
    ...defaultMdxComponents,
    ...components,
  } satisfies MDXComponents;
}

export const useMDXComponents = getMDXComponents;

declare global {
  type MDXProvidedComponents = ReturnType<typeof getMDXComponents>;
}
```

- [ ] **Step 3: Build to verify nothing references the deleted component**

```bash
cd /Users/robin/code/github.com/yo61/go-udap/docs/site
pnpm build
```

Expected: 0 errors. If a stale reference exists, the build will fail with a "Cannot find module './how-to'" or similar error.

- [ ] **Step 4: Confirm `HowTo` only appears in the style guide now**

```bash
cd /Users/robin/code/github.com/yo61/go-udap
rg HowTo docs/site/
```

Expected output: exactly one match, inside `docs/site/content/docs/contributing/docs-style.mdx`, on the line that explains the deprecation. Nowhere else.

If matches appear in other files, investigate and fix before continuing.

- [ ] **Step 5: Visual smoke-test that everything still works**

```bash
pnpm start:local
```

Open `http://localhost:3000/how-to/configure-for-dhcp/` (or any other rewritten page) and confirm it still renders. The component removal should be invisible at this layer because no page uses it anymore.

Stop the server.

- [ ] **Step 6: Commit**

```bash
cd /Users/robin/code/github.com/yo61/go-udap
git add docs/site/components/how-to.tsx docs/site/components/mdx.tsx
git commit -m "$(cat <<'EOF'
docs(site): remove unused HowTo MDX component

The five how-to pages that used <HowTo> were converted to plain
markdown in the previous commit. The component source file is
deleted; the import + registration in components/mdx.tsx is removed.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

(Note: `git add` on a deleted file may require `git rm` instead, depending on how the file was deleted. If `git status` shows `deleted:` after `git add`, the staging worked. If not, run `git rm docs/site/components/how-to.tsx` to stage the deletion explicitly.)

---

## Task 4: Apply canonical procedure structure to the tutorial, no-devices-found, install-go-udap

**Files (all modified):**
- `docs/site/content/docs/tutorials/configure-your-first-squeezebox.mdx`
- `docs/site/content/docs/how-to/no-devices-found.mdx`
- `docs/site/content/docs/how-to/install-go-udap.mdx`

- [ ] **Step 1: Replace `tutorials/configure-your-first-squeezebox.mdx` with this exact content**

Changes from current: `## What you need` renamed to `## Prerequisites`; the nine `## Step N — Title` headings replaced by a single `## Steps` group followed by nine `### N. Title` sub-headings; trailing `## What's next?` preserved verbatim.

```mdx
---
title: Configure your first Squeezebox in 5 minutes
description: A complete walkthrough from factory state to a working device
---

This tutorial walks you through configuring a brand-new (or
factory-reset) Squeezebox from the command line. By the end, you'll
have a device with a hostname you chose, on DHCP, pointed at your
Lyrion Music Server.

## Prerequisites

- A Squeezebox Receiver (the UDAP protocol is shared across the
  Boom, Touch, and Radio too, but only the Receiver has been tested
  so far)
- An Ethernet cable connecting the device to the same LAN as your
  computer (Wi-Fi setup is a separate how-to; for first onboarding,
  wired is much simpler)
- `go-udap` [installed](/how-to/install-go-udap)
- A Lyrion Music Server running somewhere on your LAN (note its IP
  address)

## Steps

### 1. Put the device in setup mode

If the device is brand-new out of the box, it's already in setup mode
— skip to step 2.

If you're configuring an already-configured device, hold the front
button for 3-6 seconds until the light flashes red. (Holding for 6+
seconds triggers a full factory reset; you don't need that for
just-reconfigure.)

### 2. Discover the device

```bash
go-udap discover
```

Expected output (one MAC, your device's):

```text
00:04:20:16:06:02
```

If you see nothing, see
[Troubleshoot "no devices found"](/how-to/no-devices-found).

### 3. Look at what it currently reports

```bash
go-udap info 00:04:20:16:06:02
```

Replace the MAC with what `discover` showed you. Expected:

```text
MAC:      00:04:20:16:06:02
IP:       0.0.0.0
Name:     Squeezebox Device
Model:    Squeezebox Receiver
Firmware: 77
HW Rev:   0005
State:    init
```

The `0.0.0.0` IP and `init` state confirm the device is unconfigured.

### 4. Apply your configuration and reboot

Choose a hostname for your device (e.g. `living-room`) and find the
IP of your Lyrion Music Server (e.g. `192.168.1.250`).

```bash
go-udap set 00:04:20:16:06:02 \
  --hostname living-room \
  --server-address 192.168.1.250 \
  --reboot
```

- `--lan-ip-mode` is omitted — devices default to DHCP. (For a static
  IP instead, see [Configure a Squeezebox for static
  IP](/how-to/configure-for-static-ip).)
- `--interface` is omitted — `go-udap set` defaults to wired Ethernet
  on factory-fresh devices (go-udap v2.1.0+; earlier versions need
  `--interface 1`).
- `--server-address` points the device at your music server.
- `--reboot` reboots after writing — necessary for the new config to
  take effect.

Expected: command returns within a couple of seconds. The device
reboots and is unreachable for ~10 seconds.

### 5. Verify

Wait ~15 seconds for the device to come back up, then:

```bash
go-udap discover --info
```

Expected: the device now reports its DHCP-assigned IP, its new
hostname, and `State: connected`.

```text
MAC:      00:04:20:16:06:02
IP:       192.168.1.78
Name:     living-room
Model:    Squeezebox Receiver
Firmware: 77
HW Rev:   0005
State:    connected
IP:      192.168.1.78
Subnet:  255.255.255.0
Gateway: 192.168.1.1
```

The `IP:` / `Subnet:` / `Gateway:` lines come from a `get_ip` query
that `--info` fires per device — they confirm the device's own view
of its network state.

## What's next?

- Want to configure over Wi-Fi instead of Ethernet? See
  [Configure a Squeezebox for Wi-Fi with WPA2](/how-to/configure-wifi-wpa2).
- Want to back up the config? See
  [Back up and restore a device config](/how-to/install-go-udap)
  (uses `read` and `set --config`).
- Want to understand what's happening on the wire? See
  [How UDAP discovery works](/concepts/how-udap-discovery-works).
```

The tutorial has five steps; the rewrite renumbers them `### 1.` through `### 5.`.

- [ ] **Step 2: Replace `how-to/no-devices-found.mdx` with this exact content**

```mdx
---
title: Troubleshoot "no devices found"
description: Diagnostic steps when go-udap discover returns nothing
---

When `go-udap discover` returns nothing, work through this checklist
in order.

## Steps

### 1. Confirm the device is powered on

The most boring cause. Plug it in, wait 30 seconds for it to come
fully up, try again.

### 2. Confirm the device is on the same network segment

UDAP discovery is broadcast-based; broadcasts don't cross routers.
The device and your dev machine need to be on the same VLAN / subnet.

If the device is unconfigured (no DHCP lease), it broadcasts from
source IP `0.0.0.0` — it doesn't *have* a subnet yet, but it must be
physically on the same Layer-2 segment.

### 3. Check UDP port 17784 isn't blocked

Some host firewalls block inbound UDP on non-standard ports. Test
with tcpdump:

```bash
sudo tcpdump -i any -n -nn 'udp port 17784'
```

In another terminal, run `go-udap discover`. You should see the
outbound broadcast (length 27) and the device's reply (length 61+).

If you see the outbound but no reply, the device isn't responding
(check step 1 and step 2 again). If you see neither, your host
firewall is dropping the outbound — find and fix it.

### 4. On a multi-homed laptop, try `--bind-interface` or `--all-interfaces`

If your host has Wi-Fi + Ethernet both up, default discovery only
emits on the kernel's default-route NIC. If the device is on the
*other* NIC's subnet, you'll see zero devices.

See [How to discover on a multi-NIC laptop](./discover-on-multi-nic-laptop.mdx).
```

- [ ] **Step 3: Replace `how-to/install-go-udap.mdx` with this exact content**

```mdx
---
title: Install go-udap
description: Get the go-udap binary onto your machine
---

Three options, in order of preference for most users.

## Options

### 1. Homebrew (macOS and Linux)

```bash
brew install yo61/tap/go-udap
```

Updates flow through `brew upgrade` like any other formula. This is the
fastest path for users who already use Homebrew. Shell completions for
bash, zsh, and fish install automatically alongside the binary — see
[Install shell completions](./install-shell-completions.mdx) for the
details and manual-install path.

### 2. Pre-built binaries

Download the archive for your platform from the
[Releases](https://github.com/yo61/go-udap/releases) page:

- `go-udap_<version>_macos_arm64.tar.gz` (Apple Silicon)
- `go-udap_<version>_macos_amd64.tar.gz` (Intel Mac)
- `go-udap_<version>_linux_amd64.tar.gz`
- `go-udap_<version>_linux_arm64.tar.gz`
- `go-udap_<version>_windows_amd64.zip`

Each release also has a `SHA256SUMS` file you can verify against:

```bash
shasum -a 256 -c SHA256SUMS
```

Extract the archive; put the `go-udap` (or `go-udap.exe`) binary on
your `PATH`.

### 3. Build from source

Requires Go 1.26 or later.

```bash
git clone https://github.com/yo61/go-udap.git
cd go-udap
go build -o go-udap .
```

The repo also has a [Taskfile](https://taskfile.dev) for cross-compile
and release tasks; see the [Contributing](/contributing) section
if you're working on the tool itself.

## Verify the install

```bash
go-udap --version
```

Expected: prints the installed version.
```

- [ ] **Step 4: Build to verify all three rewrites compile**

```bash
cd /Users/robin/code/github.com/yo61/go-udap/docs/site
pnpm build
```

Expected: 0 errors.

- [ ] **Step 5: Visual smoke-test the three rewritten pages**

```bash
pnpm start:local
```

Open each:

- `http://localhost:3000/tutorials/configure-your-first-squeezebox/`
- `http://localhost:3000/how-to/no-devices-found/`
- `http://localhost:3000/how-to/install-go-udap/`

For each: confirm the new heading hierarchy renders, no `Step N —` /
`Option N —` text in headings, sidebar TOC shows the new structure
(top-level `Steps` / `Options` group + sub-items underneath), no
console errors.

Stop the server.

- [ ] **Step 6: Commit**

```bash
cd /Users/robin/code/github.com/yo61/go-udap
git add docs/site/content/docs/tutorials/configure-your-first-squeezebox.mdx \
        docs/site/content/docs/how-to/no-devices-found.mdx \
        docs/site/content/docs/how-to/install-go-udap.mdx
git commit -m "$(cat <<'EOF'
docs(site): apply canonical procedure structure to tutorial, no-devices-found, install-go-udap

The tutorial's "## What you need" is renamed to "## Prerequisites";
its five "## Step N — Title" headings become a single "## Steps"
group with "### N. Title" sub-headings.

no-devices-found's four "## Step N: Title" headings become a single
"## Steps" group with "### N. Title" sub-headings.

install-go-udap's three "## Option N — Title" headings become a
single "## Options" group with "### N. Title" sub-headings.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Fix the remaining concept and reference slips

Three small surgical edits.

**Files (all modified):**
- `docs/site/content/docs/concepts/setup-vs-run-mode.mdx`
- `docs/site/content/docs/concepts/squeezeplay-comparison.mdx`
- `docs/site/content/docs/reference/commands/set.mdx`

- [ ] **Step 1: Delete the in-body H1 from `concepts/setup-vs-run-mode.mdx`**

Edit `docs/site/content/docs/concepts/setup-vs-run-mode.mdx`. Find and delete this exact two-line block (lines 6-7 currently, an H1 followed by the blank line after it):

```mdx
# Setup mode vs. run mode

```

After the edit, the file's first lines should read:

```mdx
---
title: Setup mode vs. run mode
description: Why most UDAP operations only work when the device is in setup mode
---

A Squeezebox has two operational states relevant to go-udap:
```

The `## What was tested` heading and everything after it is unchanged.

- [ ] **Step 2: Drop the numbering prefix from `concepts/squeezeplay-comparison.mdx`**

Edit `docs/site/content/docs/concepts/squeezeplay-comparison.mdx`. Find and replace each of the ten numbered H2 lines. Replace this block of ten headings (they appear at various positions in the file as section starts; the prose between them is unchanged):

```mdx
## 1. Packet framing and protocol primitives
## 2. Discovery TLVs and decoding
## 3. `get_ip` / network-config operation
## 4. NVRAM parameter table (`configSettings`)
## 5. Broadcast and addressing
## 6. Retry and timeout strategy
## 7. Error handling
## 8. Device-type coverage (the non-SBR question)
## 9. Findings that motivated new tracker items
## 10. Squeezeplay designs intentionally NOT replicated
```

With these (numbering removed, titles unchanged):

```mdx
## Packet framing and protocol primitives
## Discovery TLVs and decoding
## `get_ip` / network-config operation
## NVRAM parameter table (`configSettings`)
## Broadcast and addressing
## Retry and timeout strategy
## Error handling
## Device-type coverage (the non-SBR question)
## Findings that motivated new tracker items
## Squeezeplay designs intentionally NOT replicated
```

Do this with ten individual Edit calls (one per heading), since `replace_all` would risk matching numbered headings inside the body (table content, prose references) that should NOT be changed.

The two H2s before the numbered block (`## Purpose`, `## Methodology`) and any references inside the body that mention numeric sections (e.g. "section 3" inside prose) are unchanged.

- [ ] **Step 3: Reorder Examples and Exit codes in `reference/commands/set.mdx`**

Edit `docs/site/content/docs/reference/commands/set.mdx`. Current structure (per the spec) puts Examples before Exit codes; reorder so Exit codes comes first, then Examples (matching the eight sibling command pages).

Find this block, which currently starts at the `## Examples` heading and continues through to the end of the file. The current order is:

```
## Examples
... (examples content)
## Exit codes
... (exit codes content)
See [Config file format](../config-file-format.mdx) for the `.conf` syntax.
```

After the edit it should read:

```
## Exit codes
... (exit codes content)
## Examples
... (examples content)
See [Config file format](../config-file-format.mdx) for the `.conf` syntax.
```

Specifically:

Find the line `## Examples` (the first one, around line 41 currently) and move the `## Examples` heading PLUS its three example blocks (DHCP-on-wireless, restore-from-backup, pipe-from-stdin) to AFTER the `## Exit codes` section (which lists the three exit codes), but BEFORE the trailing `See [Config file format]…` line.

The resulting `set.mdx`, from line 41 onwards, should look like:

```mdx
## Exit codes

- `0` — success
- `1` — invalid MAC, unknown parameter, or malformed config
- `2` — device not found, or transport error

## Examples

DHCP on wireless with WPA2:

```bash
go-udap set 00:04:20:16:06:02 \
  --interface 0 --lan-ip-mode 1 \
  --wireless-ssid SlimNet --wireless-wpa-on 1 --wireless-wpa-mode 2 \
  --wireless-wpa-psk 'shared-secret' \
  --server-address 192.168.1.250 \
  --reboot
```

Restore from a backup:

```bash
go-udap set 00:04:20:16:06:02 --config backup.conf --reboot
```

Pipe parameters from stdin:

```bash
go-udap set 00:04:20:16:06:02 <<EOF
interface=1
lan_ip_mode=0
lan_network_address=192.168.1.50
lan_subnet_mask=255.255.255.0
lan_gateway=192.168.1.1
EOF
```

See [Config file format](../config-file-format.mdx) for the `.conf` syntax.
```

(Everything above `## Exit codes` in the file — the frontmatter, the synopsis code block, `## What it does`, and `## Flags` — is unchanged.)

- [ ] **Step 4: Build to verify the three edits compile**

```bash
cd /Users/robin/code/github.com/yo61/go-udap/docs/site
pnpm build
```

Expected: 0 errors.

- [ ] **Step 5: Visual smoke-test the three changed pages**

```bash
pnpm start:local
```

Open each:

- `http://localhost:3000/concepts/setup-vs-run-mode/` — confirm no double-H1 at the top; the page title appears only once (rendered by the layout from frontmatter)
- `http://localhost:3000/concepts/squeezeplay-comparison/` — confirm the ten section H2s no longer have leading `N.` numbering; intra-page anchor links still work
- `http://localhost:3000/reference/commands/set/` — confirm section order is now What it does → Flags → Exit codes → Examples → trailing link

Stop the server.

- [ ] **Step 6: Final whole-branch verification**

From the repo root:

```bash
cd /Users/robin/code/github.com/yo61/go-udap

# 1. <HowTo> appears only in the style guide
rg HowTo docs/site/

# 2. Final build clean
cd docs/site && pnpm build && cd ../..

# 3. File scope check — every change is under docs/site/, docs/superpowers/, or CLAUDE.md
git diff --name-only main..HEAD | sort -u
```

Expected output of (1): a single match in `docs/site/content/docs/contributing/docs-style.mdx`.

Expected output of (3): the 15 files listed in the File structure section at the top of this plan, plus the spec file at `docs/superpowers/specs/2026-06-12-docs-heading-consistency-design.md` (committed at the start of the branch) and this plan at `docs/superpowers/plans/2026-06-12-docs-heading-consistency.md`. Nothing else.

- [ ] **Step 7: Commit**

```bash
cd /Users/robin/code/github.com/yo61/go-udap
git add docs/site/content/docs/concepts/setup-vs-run-mode.mdx \
        docs/site/content/docs/concepts/squeezeplay-comparison.mdx \
        docs/site/content/docs/reference/commands/set.mdx
git commit -m "$(cat <<'EOF'
docs(site): fix concept and reference heading slips

- setup-vs-run-mode: delete the in-body H1, frontmatter title is
  already rendered as the page H1 by the layout.
- squeezeplay-comparison: drop the "## N. " numbering prefix from the
  ten section headings — these are parallel comparison topics, not
  ordered steps.
- reference/commands/set: reorder Examples and Exit codes to match
  the canonical command-reference H2 order (What / Flags /
  Exit codes / Examples) used by the eight sibling command pages.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## After all five tasks land

Push the branch and open a PR:

```bash
cd /Users/robin/code/github.com/yo61/go-udap
git push -u origin docs/heading-consistency
gh pr create --title "docs(site): site-wide heading consistency + drop <HowTo> component"
```

PR body should summarise: three archetypes (Procedure / Explainer / Reference), the style guide added at `contributing/docs-style.mdx`, the five rewrites, the component removal, the three small fixes, the verification matrix run.

---

## Self-review notes (for the engineer running this plan)

If anything in the plan turns out wrong — a Fumadocs MDX shape changed, an internal link target moved, a heading already exists in a form the plan didn't anticipate — stop and report. Don't improvise.

Specifically:

- The plan assumes the eight sibling command pages all use the order `What it does` → `Output` → `Flags` → `Exit codes` → `Examples`. If a sibling has a different order, that's a pre-existing inconsistency in scope for a separate fix. Note it; do not change extra pages in this PR.
- The plan assumes the `Edit` tool can replace strings inside MDX files; some MDX has triple-backtick code fences which can be tricky to edit. If an edit fails because of fence escaping, use `Write` to replace the whole file instead.
- The plan assumes `pnpm build` from `docs/site/` is the right verification command. If the repo's build script differs by the time the plan runs, use whatever script `package.json` defines as `build`.
- Internal link conventions in this site use no trailing slash on `/how-to/foo` (the existing source files mix styles but the dominant convention is no trailing slash). The rewrites in this plan match the source pages they replace; do not change the link style unless the source page already had a trailing slash.
