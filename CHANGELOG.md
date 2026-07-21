# Changelog

All notable changes to this project will be documented in this file.

## [2.4.6](https://github.com/yo61/go-udap/compare/v2.4.5...v2.4.6) (2026-07-21)


### Bug Fixes

* **cli:** plain-English timeout errors for device ops ([#168](https://github.com/yo61/go-udap/issues/168)) ([7fac6cc](https://github.com/yo61/go-udap/commit/7fac6cc21511675719a5ba977089bf093fa7cace)), closes [#110](https://github.com/yo61/go-udap/issues/110)


### Documentation

* record CLI timeout error-layer decision and quality criteria ([#170](https://github.com/yo61/go-udap/issues/170)) ([502ebbf](https://github.com/yo61/go-udap/commit/502ebbfa6e11f237bf3cce612ca9f2579622eca1))

## [2.4.5](https://github.com/yo61/go-udap/compare/v2.4.4...v2.4.5) (2026-07-21)


### Bug Fixes

* **security:** bump brace-expansion to 1.1.16 (GHSA-3jxr-9vmj-r5cp) ([#165](https://github.com/yo61/go-udap/issues/165)) ([57d3464](https://github.com/yo61/go-udap/commit/57d346425fa658f290e106cc2ab0da621a978ad3))

## [2.4.4](https://github.com/yo61/go-udap/compare/v2.4.3...v2.4.4) (2026-07-20)


### Bug Fixes

* **docs:** pin typescript to 6.x and gate docs build on PRs ([#161](https://github.com/yo61/go-udap/issues/161)) ([0327986](https://github.com/yo61/go-udap/commit/03279869df1821ab1ed8830c635617e084a1ac9f))

## [2.4.3](https://github.com/yo61/go-udap/compare/v2.4.2...v2.4.3) (2026-07-10)


### Bug Fixes

* emit a brew-style-clean go-udap cask ([#152](https://github.com/yo61/go-udap/issues/152)) ([c705697](https://github.com/yo61/go-udap/commit/c705697624246bf139af1dd8283e9d8440aceafd))

## [2.4.2](https://github.com/yo61/go-udap/compare/v2.4.1...v2.4.2) (2026-06-14)


### Documentation

* correct update:docs description — pnpm update writes package.json too ([#133](https://github.com/yo61/go-udap/issues/133)) ([ccf392d](https://github.com/yo61/go-udap/commit/ccf392d72c6df414184371eebfc66c729c362120))
* **site:** borrow Starlight visual elements + local-viewing scripts ([#122](https://github.com/yo61/go-udap/issues/122)) ([214d81d](https://github.com/yo61/go-udap/commit/214d81d6ee02c571ab53c94ed44209d36bb5d5d9))
* **site:** site-wide heading consistency + drop &lt;HowTo&gt; component ([#126](https://github.com/yo61/go-udap/issues/126)) ([73d2791](https://github.com/yo61/go-udap/commit/73d27916cb959edbf91acdcf584325d504103d6f))

## [2.4.1](https://github.com/yo61/go-udap/compare/v2.4.0...v2.4.1) (2026-06-01)


### Bug Fixes

* **deps:** patch postcss XSS (CVE-2026-41305) via pnpm override ([#115](https://github.com/yo61/go-udap/issues/115)) ([fa8a8f7](https://github.com/yo61/go-udap/commit/fa8a8f71d9d3d12bb7d992a508b21ce34954043b))


### Documentation

* add concepts page on setup mode vs run mode ([#112](https://github.com/yo61/go-udap/issues/112)) ([c111544](https://github.com/yo61/go-udap/commit/c11154490dc10e19a01066e168ea20305d7c4730))
* add release announcement archive for v2.3.0 and v2.4.0 ([#106](https://github.com/yo61/go-udap/issues/106)) ([9d42048](https://github.com/yo61/go-udap/commit/9d42048c57a091dc14f8ae66798ca8a8edc529b3))
* list goimports as a pre-commit prerequisite ([#108](https://github.com/yo61/go-udap/issues/108)) ([8d0405f](https://github.com/yo61/go-udap/commit/8d0405f858324f17eac82f505c5b4b2ffbc6677e))

## [2.4.0](https://github.com/yo61/go-udap/compare/v2.3.0...v2.4.0) (2026-05-27)


### Features

* ship man pages and refresh README ([#104](https://github.com/yo61/go-udap/issues/104)) ([c76994c](https://github.com/yo61/go-udap/commit/c76994cac7a7ad47838da10060d5d6f732f70cbd))


### Documentation

* mark sbom + scanning spec and plan as implemented ([#103](https://github.com/yo61/go-udap/issues/103)) ([8b1c18c](https://github.com/yo61/go-udap/commit/8b1c18ca0356b79cee931134494022c9176260cb))

## [2.3.0](https://github.com/yo61/go-udap/compare/v2.2.1...v2.3.0) (2026-05-27)


### Features

* emit per-archive SBOMs (spdx + cyclonedx) ([#98](https://github.com/yo61/go-udap/issues/98)) ([4cf5853](https://github.com/yo61/go-udap/commit/4cf585395340ad22baa5c63f492b3ff28f98d8f3))


### Documentation

* SBOM + vulnerability scanning design and implementation plan ([#94](https://github.com/yo61/go-udap/issues/94)) ([d61ae28](https://github.com/yo61/go-udap/commit/d61ae28042d88b6d60612a10e7f1efe07ae836e8))

## [2.2.1](https://github.com/yo61/go-udap/compare/v2.2.0...v2.2.1) (2026-05-26)


### Bug Fixes

* **docs-site:** basePath in og:image meta tags ([#91](https://github.com/yo61/go-udap/issues/91)) ([1654703](https://github.com/yo61/go-udap/commit/165470395f94c6633aea51b021e359400da517fb))
* **docs-site:** pass basePath to static search client fetch URL ([#90](https://github.com/yo61/go-udap/issues/90)) ([41fce13](https://github.com/yo61/go-udap/commit/41fce13d56fdf6b9e3d8aa8e96e2396d037af785))


### Documentation

* **site:** add Install shell completions how-to ([#89](https://github.com/yo61/go-udap/issues/89)) ([6e2de30](https://github.com/yo61/go-udap/commit/6e2de30dde81bebf797e4e5a5e694f79baf89b0f))
* **site:** sync timeout default 5s -&gt; 2s ([#88](https://github.com/yo61/go-udap/issues/88)) ([3972460](https://github.com/yo61/go-udap/commit/39724600d7413fb0318f41b2d9ac3983cba64693))

## [2.2.0](https://github.com/yo61/go-udap/compare/v2.1.0...v2.2.0) (2026-05-26)


### Features

* shell completions (bash/zsh/fish) via Homebrew Cask ([#85](https://github.com/yo61/go-udap/issues/85)) ([70b3a2c](https://github.com/yo61/go-udap/commit/70b3a2cfaac8d0e98e15586740b5b992b8b91c96))


### Bug Fixes

* **cli:** drop trailer hint from interface-default notice ([#86](https://github.com/yo61/go-udap/issues/86)) ([fdd563e](https://github.com/yo61/go-udap/commit/fdd563ef8d545e2f1197864e42b3d85e94b72586))


### Documentation

* clean up landing, tutorial, DHCP how-to, double titles, broken links ([#80](https://github.com/yo61/go-udap/issues/80)) ([80db218](https://github.com/yo61/go-udap/commit/80db218021a3a6011d822eda50b3d45b31299b9d))
* **plan:** shell completions implementation plan (PR 2) ([#84](https://github.com/yo61/go-udap/issues/84)) ([a892225](https://github.com/yo61/go-udap/commit/a892225842f97e7f2c7ba136eb58e02dec977168))
* shell completions design + cobra refactor plan ([#82](https://github.com/yo61/go-udap/issues/82)) ([16d6f5d](https://github.com/yo61/go-udap/commit/16d6f5d3f00ede4490d1c2991574027f7254250a))


### Code Refactoring

* **cli:** replace hand-rolled dispatch with spf13/cobra ([#83](https://github.com/yo61/go-udap/issues/83)) ([0472fc8](https://github.com/yo61/go-udap/commit/0472fc8d3d4c66ecef5f946b90f41f646a20d1f3))

## [2.1.0](https://github.com/yo61/go-udap/compare/v2.0.1...v2.1.0) (2026-05-25)


### Features

* **cli:** default interface on factory-fresh devices in set ([2a72528](https://github.com/yo61/go-udap/commit/2a72528533b6b318eae702ced82710e21b7e593f))

## [2.0.1](https://github.com/yo61/go-udap/compare/v1.10.1...v2.0.1) (2026-05-25)

> **Note:** 2.0.0 was never published. GoReleaser created a duplicate
> draft release alongside release-please's draft, then the workflow's
> un-draft step published the empty release-please draft instead of
> GoReleaser's assets-laden one. Repository immutability then locked
> the `v2.0.0` tag name against republishing. See issues
> [#70](https://github.com/yo61/go-udap/issues/70) and
> [#71](https://github.com/yo61/go-udap/issues/71) for the underlying
> release-pipeline bugs; the same changes ship here as 2.0.1.


### ⚠ BREAKING CHANGES

* **cli:** --interface NAME is renamed to --bind-interface NAME. Scripts and CI using the global form must be updated. The per-param --interface 0|1 (set command, NVRAM offset 52) is unaffected.

### Features

* **cli:** rename global --interface to --bind-interface ([7ef6add](https://github.com/yo61/go-udap/commit/7ef6addd45acc06ce945159f326b12be8d85f345))


### Documentation

* fix broken in-content links ([162d19f](https://github.com/yo61/go-udap/commit/162d19f6e464e47340664e44fecce5338155c49f))
* serve site content at root instead of /docs/ ([8a6ec1c](https://github.com/yo61/go-udap/commit/8a6ec1c79c69eca946cb29b68df6d4ba0e759d6b))

## [1.10.1](https://github.com/yo61/go-udap/compare/v1.9.0...v1.10.1) (2026-05-22)

> **Note:** 1.10.0 was never published. The GoReleaser run for that tag failed
> to upload assets with `422 Cannot upload assets to an immutable release`, the
> draft release and tag were withdrawn, and the same changes ship here as
> 1.10.1.


### Features

* **docs:** port existing docs into Diataxis quadrants (M2) ([#51](https://github.com/yo61/go-udap/issues/51)) ([3c7bce8](https://github.com/yo61/go-udap/commit/3c7bce8505b42e4bf74b87d2bbc455f59e64866c))


### Documentation

* **site:** tutorial, how-tos, and concept pages (M3) ([#53](https://github.com/yo61/go-udap/issues/53)) ([02319af](https://github.com/yo61/go-udap/commit/02319af4ac692db3423bbee248f706f8be9caa31))

## [1.9.0](https://github.com/yo61/go-udap/compare/v1.8.0...v1.9.0) (2026-05-13)

### Features

* **docs:** scaffold Fumadocs site with GitHub Pages deploy (M1) ([#49](https://github.com/yo61/go-udap/issues/49)) ([4bc8837](https://github.com/yo61/go-udap/commit/4bc8837326ea95d00942c01969b935e6aa22486e))

## [1.8.0](https://github.com/yo61/go-udap/compare/v1.7.0...v1.8.0) (2026-05-13)

### Features

* **cli:** add --retries N flag for opt-in send-side retransmits ([#47](https://github.com/yo61/go-udap/issues/47)) ([884af5e](https://github.com/yo61/go-udap/commit/884af5e24f8032462a4ac9b359098bafa845e8cc))

## [1.7.0](https://github.com/yo61/go-udap/compare/v1.6.1...v1.7.0) (2026-05-13)

### Features

* **udap:** implement get_uuid (0x000b) fallback when discovery omits TLV 0x0d ([#46](https://github.com/yo61/go-udap/issues/46)) ([b9fd106](https://github.com/yo61/go-udap/commit/b9fd10659bb1c6346885f512340931cbfe670050))

## [1.6.1](https://github.com/yo61/go-udap/compare/v1.6.0...v1.6.1) (2026-05-13)

### Bug Fixes

* **cli:** suppress get_ip warning during discover --info unless --verbose ([#45](https://github.com/yo61/go-udap/issues/45)) ([a9ff8e5](https://github.com/yo61/go-udap/commit/a9ff8e5d8e9f27f941c8eef67ba2c657319a3a14)), closes [#33](https://github.com/yo61/go-udap/issues/33)

## [1.6.0](https://github.com/yo61/go-udap/compare/v1.5.0...v1.6.0) (2026-05-13)

### Features

* add get_ip, hwrev/uuid surfacing, per-interface discovery ([#43](https://github.com/yo61/go-udap/issues/43)) ([3612854](https://github.com/yo61/go-udap/commit/36128542a4f1025bcdcc40e95f880f95f1d463d6)), closes [#29](https://github.com/yo61/go-udap/issues/29)

## [1.5.0](https://github.com/yo61/go-udap/compare/v1.4.0...v1.5.0) (2026-05-11)

### Features

* **udap:** promote Device.MAC from string to MAC value object ([#42](https://github.com/yo61/go-udap/issues/42)) ([31a2825](https://github.com/yo61/go-udap/commit/31a2825a2e05cf9c9298e73012469cd77d5057c4)), closes [#41](https://github.com/yo61/go-udap/issues/41)

## [1.4.0](https://github.com/yo61/go-udap/compare/v1.3.10...v1.4.0) (2026-05-11)

### Features

* **udap:** introduce MAC value object and adopt it internally ([#41](https://github.com/yo61/go-udap/issues/41)) ([645a75d](https://github.com/yo61/go-udap/commit/645a75d64affd5bec8b90a0b432fae5a70693dd6))

## [1.3.10](https://github.com/yo61/go-udap/compare/v1.3.9...v1.3.10) (2026-05-11)

### Bug Fixes

* **udap:** defer Device.Parameters mutation until SetData ack ([#38](https://github.com/yo61/go-udap/issues/38)) ([7739185](https://github.com/yo61/go-udap/commit/7739185641fee2abd988702bbef42830d2c9779b))

## [1.3.9](https://github.com/yo61/go-udap/compare/v1.3.8...v1.3.9) (2026-05-10)

### Bug Fixes

* **udap:** clear stale offset_NNN keys before each GetAll merge ([#32](https://github.com/yo61/go-udap/issues/32)) ([130187a](https://github.com/yo61/go-udap/commit/130187ab749267738dfd2cbd3d07fe068a0b3d02))
* **udap:** make Sequence field's uint16 wrap explicit and tested ([#31](https://github.com/yo61/go-udap/issues/31)) ([a52a38e](https://github.com/yo61/go-udap/commit/a52a38eed92d59f1c5659d9fec656dae8a0a17b4))

## [1.3.8](https://github.com/yo61/go-udap/compare/v1.3.7...v1.3.8) (2026-05-10)

### Bug Fixes

* **udap:** pin reply source against device.IP in waitForDeviceReply ([#29](https://github.com/yo61/go-udap/issues/29)) ([a56b8ac](https://github.com/yo61/go-udap/commit/a56b8ac8cbb03a5aa842c47b1cca74a8b0a06548))

## [1.3.7](https://github.com/yo61/go-udap/compare/v1.3.6...v1.3.7) (2026-05-10)

### Bug Fixes

* **udap:** error out of SetDeviceConfigWithContext when prelude read fails ([#28](https://github.com/yo61/go-udap/issues/28)) ([e550e75](https://github.com/yo61/go-udap/commit/e550e750b80c0cb3d129af5cd647506ba1b58840))

## [1.3.6](https://github.com/yo61/go-udap/compare/v1.3.5...v1.3.6) (2026-05-10)

### Bug Fixes

* **cli,udap:** validate set values at flag boundary and on packet build ([#27](https://github.com/yo61/go-udap/issues/27)) ([b0003c2](https://github.com/yo61/go-udap/commit/b0003c2096a85ea9d61e13bbb445cfe295d2dd54)), closes [#2](https://github.com/yo61/go-udap/issues/2) [#2](https://github.com/yo61/go-udap/issues/2) [#3](https://github.com/yo61/go-udap/issues/3)

## [1.3.5](https://github.com/yo61/go-udap/compare/v1.3.4...v1.3.5) (2026-05-10)

### Bug Fixes

* **cli:** share one context between discovery and the operation ([#26](https://github.com/yo61/go-udap/issues/26)) ([ab049e5](https://github.com/yo61/go-udap/commit/ab049e57f36919d2012bba72abe6a6543968fc12))

## [1.3.4](https://github.com/yo61/go-udap/compare/v1.3.3...v1.3.4) (2026-05-10)

### Bug Fixes

* **udap:** clamp parseGetDataResponse map size hint ([#22](https://github.com/yo61/go-udap/issues/22)) ([ee1ffe4](https://github.com/yo61/go-udap/commit/ee1ffe45564f55fe1e55a1bed22391fc78d1ff7c))

## [1.3.3](https://github.com/yo61/go-udap/compare/v1.3.2...v1.3.3) (2026-05-10)

### Bug Fixes

* **udap:** surface MethodError replies from ResetDeviceWithContext ([#25](https://github.com/yo61/go-udap/issues/25)) ([6e31c4a](https://github.com/yo61/go-udap/commit/6e31c4a1bd1252eed610105e3989b73b33627fa8))

## [1.3.2](https://github.com/yo61/go-udap/compare/v1.3.1...v1.3.2) (2026-05-10)

### Bug Fixes

* **udap:** return error on malformed MAC instead of silent zero MAC ([#21](https://github.com/yo61/go-udap/issues/21)) ([aa274d1](https://github.com/yo61/go-udap/commit/aa274d17c0cf5944a957477001e547976b2cdc62))

## [1.3.1](https://github.com/yo61/go-udap/compare/v1.3.0...v1.3.1) (2026-05-10)

### Bug Fixes

* **udap:** make Client.sequence increment race-free ([#20](https://github.com/yo61/go-udap/issues/20)) ([8cb5885](https://github.com/yo61/go-udap/commit/8cb5885836355b3a0e8b4a3e2c039b61ef119c5b))

## [1.3.0](https://github.com/yo61/go-udap/compare/v1.2.0...v1.3.0) (2026-05-10)

### Features

* **mocksbr:** add Phase 3 failure injection and InjectReply ([#18](https://github.com/yo61/go-udap/issues/18)) ([85c3df5](https://github.com/yo61/go-udap/commit/85c3df544ed3ee94bd9844db5a69a6133ea591c3))

## [1.2.0](https://github.com/yo61/go-udap/compare/v1.1.1...v1.2.0) (2026-05-10)

### Features

* **release:** switch Homebrew distribution from formula to cask ([#17](https://github.com/yo61/go-udap/issues/17)) ([b0d48d5](https://github.com/yo61/go-udap/commit/b0d48d5d3f841d70327a114f68b47f4d3fa04d99)), closes [yo61/homebrew-tap#2](https://github.com/yo61/homebrew-tap/issues/2)

## [1.1.1](https://github.com/yo61/go-udap/compare/v1.1.0...v1.1.1) (2026-05-09)

### Bug Fixes

* **release:** tap formula path + remove duplicate-release conflict ([#15](https://github.com/yo61/go-udap/issues/15)) ([9411f45](https://github.com/yo61/go-udap/commit/9411f455b9b718fcdade57ee15f9ffe082c31bde)), closes [#6](https://github.com/yo61/go-udap/issues/6)

## [1.1.0](https://github.com/yo61/go-udap/compare/v1.0.1...v1.1.0) (2026-05-09)

### Features

* **release:** publish Homebrew formula to yo61/homebrew-tap on release ([#14](https://github.com/yo61/go-udap/issues/14)) ([0efd3ba](https://github.com/yo61/go-udap/commit/0efd3ba6c9de2383da46fbef436464ee64cbf473))

## [1.0.0](https://github.com/yo61/go-udap/compare/v0.1.0...v1.0.0) (2026-05-09)

### ⚠ BREAKING CHANGES

* release artifact filenames have changed. Previously
each GitHub release attached raw binaries named go-udap-linux-amd64,
go-udap.exe, etc. Releases now attach versioned tar.gz/zip archives
containing the binary plus LICENSE and README.md:

  go-udap_1.0.0_linux_x86_64.tar.gz
  go-udap_1.0.0_linux_arm64.tar.gz
  go-udap_1.0.0_macos_x86_64.tar.gz
  go-udap_1.0.0_macos_arm64.tar.gz
  go-udap_1.0.0_windows_x86_64.zip
  SHA256SUMS

Anyone with a download script that hardcodes the old filenames will
need to update. Inside each archive the binary is named go-udap (or
go-udap.exe on Windows).

### Features

* add automated release pipeline ([#5](https://github.com/yo61/go-udap/issues/5)) ([5792be8](https://github.com/yo61/go-udap/commit/5792be862f7d1b1fe8a263c747679b03edd680fc))
* **cli:** add dispatcher, global flags, exit codes ([5d33e4e](https://github.com/yo61/go-udap/commit/5d33e4eaec02c40310ae526d153c09ea3d22dc91))
* **cli:** add flag table for known UDAP parameters ([be75981](https://github.com/yo61/go-udap/commit/be7598188174da2df2aa604897bf2337d1774587))
* **cli:** add INI parser for set source files ([676c229](https://github.com/yo61/go-udap/commit/676c22950c47c224e8870db24102bafe75aa783c))
* **cli:** add MAC normalization and device-find helper ([cbb4ede](https://github.com/yo61/go-udap/commit/cbb4eded3dfa686f4d840bcdf865d951e7ab91d2))
* **cli:** add output formatting helpers ([448133d](https://github.com/yo61/go-udap/commit/448133deedccddfe8fe07a9ffa4560cdb9b30d4c))
* **cli:** allow global flags before the subcommand ([6fd5bbc](https://github.com/yo61/go-udap/commit/6fd5bbc9b1a79ed27927d3b381d4adc5d965eac8))
* **cli:** early-return discoverAndFind once target MAC responds ([73fb9d5](https://github.com/yo61/go-udap/commit/73fb9d5d3cdb737dc4a701b8d8637da85e7eba68))
* **cli:** filter factory defaults from read by default; add -a/--all ([0784ced](https://github.com/yo61/go-udap/commit/0784ced1b30cde0e1f2afe27da1a98ea63edc27d))
* **cli:** hide unknown-offset entries from read by default; --include-unknown opt-in ([b15741d](https://github.com/yo61/go-udap/commit/b15741df846a59f6528b2ca711fd9029b9b619e1))
* **cli:** implement discover subcommand ([a1a51ee](https://github.com/yo61/go-udap/commit/a1a51ee8afa084fa301e610a648f33e9151815d6))
* **cli:** implement get subcommand ([ae2fbd3](https://github.com/yo61/go-udap/commit/ae2fbd3bd403fac867ffb3e837a9fd3064e52b24))
* **cli:** implement info subcommand ([6be60be](https://github.com/yo61/go-udap/commit/6be60be8e49d28f084ac9023d374b602bf0b2a9c))
* **cli:** implement read subcommand ([495a3a8](https://github.com/yo61/go-udap/commit/495a3a867ff0ba43bb4461d081e2bd652369ae18))
* **cli:** implement save, reset, and commit subcommands ([c27b6cf](https://github.com/yo61/go-udap/commit/c27b6cf265ea7630a2a9bd16dc883d04d1f2d8e2))
* **cli:** implement set subcommand with layered sources ([93a07c4](https://github.com/yo61/go-udap/commit/93a07c46a9fb8b025f2f569e3636f931468efbca))
* **cli:** layer config file, stdin, and flags for set ([bbba0b8](https://github.com/yo61/go-udap/commit/bbba0b8269e0b7fac7099d76c2de899c2885aadd))
* **cli:** show progress bar during discover, message when none found ([f5d3fd5](https://github.com/yo61/go-udap/commit/f5d3fd58a21ce4c2841e9b1318f4fd37a170b0c1))
* **cli:** show progress bar on every device-targeted command ([88cf190](https://github.com/yo61/go-udap/commit/88cf19061325771f2c26f9706226b7de284e35f3))
* **cmd/mocksbr:** add binary that runs mock SBRs on a UDP socket ([ff5c539](https://github.com/yo61/go-udap/commit/ff5c539904069ae69516b5d7784499443309960a))
* **mocksbr:** add virtual SBR package ([4877a69](https://github.com/yo61/go-udap/commit/4877a69bb35cad3703cb626ed1458f950bb2a51e))
* release v1.0.0 with archive-based artifacts ([#7](https://github.com/yo61/go-udap/issues/7)) ([c5c69cf](https://github.com/yo61/go-udap/commit/c5c69cf93c1060341fc69ecab9635810c8e85d42))
* **udap:** add squeezecenter_name (NVRAM offset 83) to ConfigSettings ([9469dac](https://github.com/yo61/go-udap/commit/9469dac4ae7bd43c1d70e362efe5c84e150028d0))
* **udap:** add Transport interface and UDPTransport ([6a7069a](https://github.com/yo61/go-udap/commit/6a7069ac4492a66a9a349847fe2b23bb15c6d0a6))
* **udap:** export ValidateParameter wrapper ([883aa74](https://github.com/yo61/go-udap/commit/883aa741154fa4a859a881224849fdc217d21e88))
* **udap:** write log output to stderr ([0844b86](https://github.com/yo61/go-udap/commit/0844b862744bcfa64bfec53052366bb604b0ccf9))

### Bug Fixes

* **cli,udap:** plumb --timeout through to read; drop hardcoded inner caps ([ce4571e](https://github.com/yo61/go-udap/commit/ce4571e3959d4276e00f5533ca417c0a23105301))
* **cli,udap:** serialize progress bar with logger via stderr wrapper ([a0390ce](https://github.com/yo61/go-udap/commit/a0390ce22485ddc0685a8061a9fa594b8f8eea61))
* **cli:** add --squeezecenter-name flag for the new known parameter ([726cc5c](https://github.com/yo61/go-udap/commit/726cc5c501067bc49ecb25ba0c78a1f1b2734d11))
* **cli:** clear progress bar before printing stdout output ([dcefa19](https://github.com/yo61/go-udap/commit/dcefa194ab5732213bb783f82fa4d7b7e52b2e1c))
* **cli:** drop em-dash duplication; placeholder only in --help column ([b146900](https://github.com/yo61/go-udap/commit/b146900cea97ad6e8b0c48848b7c5b7394fd952e))
* **cli:** respect -- separator in global-flag hoisting ([5e1a877](https://github.com/yo61/go-udap/commit/5e1a877ea92f857cb7d4de64eb85b59204f1c415))
* **cli:** treat pflag.ErrHelp as exit-0, no "error:" line ([798f442](https://github.com/yo61/go-udap/commit/798f4428a088618016ef10ba3f3285e55cafb6b8))
* **cli:** use ANSI erase-line, not 80-space fill, to clear progress ([fb5b6de](https://github.com/yo61/go-udap/commit/fb5b6de60e20681b355da9abe4da0b1469a48c7f))
* **release:** create releases as draft, publish after assets uploaded ([#8](https://github.com/yo61/go-udap/issues/8)) ([79107dc](https://github.com/yo61/go-udap/commit/79107dcb8aef1117049769ab5f969f93e0368e10))
* **taskfile:** drop CLI_ARGS default so dev passes -- args ([a35c807](https://github.com/yo61/go-udap/commit/a35c8077c4dd120fb75a232f1f54a48767943da3))
* **udap,cli:** correct discovery TLV types; show product name + state ([f190b3b](https://github.com/yo61/go-udap/commit/f190b3b820f92aecdcc3b46505c7d3de6ea0abf5))
* **udap:** block until discovery listener exits, no 500ms cap ([772e7ea](https://github.com/yo61/go-udap/commit/772e7eac4588c66dbf08c9b5d594d1cad1d0b124))
* **udap:** correct UDAPHeaderSize to match serialized Packet size ([dd9b47f](https://github.com/yo61/go-udap/commit/dd9b47f5b43f0e325952211e511ded8bfdf39102))
* **udap:** drop SourceIP=0.0.0.0 capture filter; accept any source ([d61b3cb](https://github.com/yo61/go-udap/commit/d61b3cb01ea4eb02a9eae861138628b37fe109d5))
* **udap:** include squeezecenter_name in KnownParameters ([55f198e](https://github.com/yo61/go-udap/commit/55f198e6cc7fc2771b385bae43bd45998aa507a6))
* **udap:** protect Client.devices map with RWMutex ([5e6fb9f](https://github.com/yo61/go-udap/commit/5e6fb9ff0f788bf752f9ed3e34d1c6f5cd9156f9))
* **udap:** skip own kernel-looped broadcasts in packet capture ([2564df2](https://github.com/yo61/go-udap/commit/2564df2bbd47e58c55a257ffd1c047d6e83b0d64))
* **udap:** use offset/length wire format for GetData ([24c0dc4](https://github.com/yo61/go-udap/commit/24c0dc44055bb815657ad1c61ba26b3fef4c1c9c))
* **udap:** use SyscallConn().Control() to set socket options ([33c47e7](https://github.com/yo61/go-udap/commit/33c47e7a37d35add653edd790f0ed606dfd1820a))
* **udap:** use UDAPHeaderSize in ParsePacket; fixes empty info fields ([239b11b](https://github.com/yo61/go-udap/commit/239b11b7cd9af542ae81eccf2306e0f2f43ba307))
