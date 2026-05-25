# Changelog

All notable changes to this project will be documented in this file.

## [3.0.0](https://github.com/yo61/go-udap/compare/v2.0.0...v3.0.0) (2026-05-25)


### ⚠ BREAKING CHANGES

* **cli:** --interface NAME is renamed to --bind-interface NAME. Scripts and CI using the global form must be updated. The per-param --interface 0|1 (set command, NVRAM offset 52) is unaffected.
* release artifact filenames have changed. Previously each GitHub release attached raw binaries named go-udap-linux-amd64, go-udap.exe, etc. Releases now attach versioned tar.gz/zip archives containing the binary plus LICENSE and README.md:

### Features

* add automated release pipeline ([#5](https://github.com/yo61/go-udap/issues/5)) ([5792be8](https://github.com/yo61/go-udap/commit/5792be862f7d1b1fe8a263c747679b03edd680fc))
* add get_ip, hwrev/uuid surfacing, per-interface discovery ([#43](https://github.com/yo61/go-udap/issues/43)) ([3612854](https://github.com/yo61/go-udap/commit/36128542a4f1025bcdcc40e95f880f95f1d463d6))
* **cli:** add --retries N flag for opt-in send-side retransmits ([#47](https://github.com/yo61/go-udap/issues/47)) ([884af5e](https://github.com/yo61/go-udap/commit/884af5e24f8032462a4ac9b359098bafa845e8cc))
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
* **cli:** rename global --interface to --bind-interface ([7ef6add](https://github.com/yo61/go-udap/commit/7ef6addd45acc06ce945159f326b12be8d85f345))
* **cli:** show progress bar during discover, message when none found ([f5d3fd5](https://github.com/yo61/go-udap/commit/f5d3fd58a21ce4c2841e9b1318f4fd37a170b0c1))
* **cli:** show progress bar on every device-targeted command ([88cf190](https://github.com/yo61/go-udap/commit/88cf19061325771f2c26f9706226b7de284e35f3))
* **cmd/mocksbr:** add binary that runs mock SBRs on a UDP socket ([ff5c539](https://github.com/yo61/go-udap/commit/ff5c539904069ae69516b5d7784499443309960a))
* **docs:** port existing docs into Diataxis quadrants (M2) ([#51](https://github.com/yo61/go-udap/issues/51)) ([3c7bce8](https://github.com/yo61/go-udap/commit/3c7bce8505b42e4bf74b87d2bbc455f59e64866c))
* **docs:** scaffold Fumadocs site with GitHub Pages deploy (M1) ([#49](https://github.com/yo61/go-udap/issues/49)) ([4bc8837](https://github.com/yo61/go-udap/commit/4bc8837326ea95d00942c01969b935e6aa22486e))
* **mocksbr:** add Phase 3 failure injection and InjectReply ([#18](https://github.com/yo61/go-udap/issues/18)) ([85c3df5](https://github.com/yo61/go-udap/commit/85c3df544ed3ee94bd9844db5a69a6133ea591c3))
* **mocksbr:** add virtual SBR package ([4877a69](https://github.com/yo61/go-udap/commit/4877a69bb35cad3703cb626ed1458f950bb2a51e))
* release v1.0.0 with archive-based artifacts ([#7](https://github.com/yo61/go-udap/issues/7)) ([c5c69cf](https://github.com/yo61/go-udap/commit/c5c69cf93c1060341fc69ecab9635810c8e85d42))
* **release:** publish Homebrew formula to yo61/homebrew-tap on release ([#14](https://github.com/yo61/go-udap/issues/14)) ([0efd3ba](https://github.com/yo61/go-udap/commit/0efd3ba6c9de2383da46fbef436464ee64cbf473))
* **release:** switch Homebrew distribution from formula to cask ([#17](https://github.com/yo61/go-udap/issues/17)) ([b0d48d5](https://github.com/yo61/go-udap/commit/b0d48d5d3f841d70327a114f68b47f4d3fa04d99))
* **udap:** add squeezecenter_name (NVRAM offset 83) to ConfigSettings ([9469dac](https://github.com/yo61/go-udap/commit/9469dac4ae7bd43c1d70e362efe5c84e150028d0))
* **udap:** add Transport interface and UDPTransport ([6a7069a](https://github.com/yo61/go-udap/commit/6a7069ac4492a66a9a349847fe2b23bb15c6d0a6))
* **udap:** export ValidateParameter wrapper ([883aa74](https://github.com/yo61/go-udap/commit/883aa741154fa4a859a881224849fdc217d21e88))
* **udap:** implement get_uuid (0x000b) fallback when discovery omits TLV 0x0d ([#46](https://github.com/yo61/go-udap/issues/46)) ([b9fd106](https://github.com/yo61/go-udap/commit/b9fd10659bb1c6346885f512340931cbfe670050))
* **udap:** introduce MAC value object and adopt it internally ([#41](https://github.com/yo61/go-udap/issues/41)) ([645a75d](https://github.com/yo61/go-udap/commit/645a75d64affd5bec8b90a0b432fae5a70693dd6))
* **udap:** promote Device.MAC from string to MAC value object ([#42](https://github.com/yo61/go-udap/issues/42)) ([31a2825](https://github.com/yo61/go-udap/commit/31a2825a2e05cf9c9298e73012469cd77d5057c4))
* **udap:** write log output to stderr ([0844b86](https://github.com/yo61/go-udap/commit/0844b862744bcfa64bfec53052366bb604b0ccf9))


### Bug Fixes

* **cli,udap:** plumb --timeout through to read; drop hardcoded inner caps ([ce4571e](https://github.com/yo61/go-udap/commit/ce4571e3959d4276e00f5533ca417c0a23105301))
* **cli,udap:** serialize progress bar with logger via stderr wrapper ([a0390ce](https://github.com/yo61/go-udap/commit/a0390ce22485ddc0685a8061a9fa594b8f8eea61))
* **cli,udap:** validate set values at flag boundary and on packet build ([#27](https://github.com/yo61/go-udap/issues/27)) ([b0003c2](https://github.com/yo61/go-udap/commit/b0003c2096a85ea9d61e13bbb445cfe295d2dd54))
* **cli:** add --squeezecenter-name flag for the new known parameter ([726cc5c](https://github.com/yo61/go-udap/commit/726cc5c501067bc49ecb25ba0c78a1f1b2734d11))
* **cli:** clear progress bar before printing stdout output ([dcefa19](https://github.com/yo61/go-udap/commit/dcefa194ab5732213bb783f82fa4d7b7e52b2e1c))
* **cli:** drop em-dash duplication; placeholder only in --help column ([b146900](https://github.com/yo61/go-udap/commit/b146900cea97ad6e8b0c48848b7c5b7394fd952e))
* **cli:** respect -- separator in global-flag hoisting ([5e1a877](https://github.com/yo61/go-udap/commit/5e1a877ea92f857cb7d4de64eb85b59204f1c415))
* **cli:** share one context between discovery and the operation ([#26](https://github.com/yo61/go-udap/issues/26)) ([ab049e5](https://github.com/yo61/go-udap/commit/ab049e57f36919d2012bba72abe6a6543968fc12))
* **cli:** suppress get_ip warning during discover --info unless --verbose ([#45](https://github.com/yo61/go-udap/issues/45)) ([a9ff8e5](https://github.com/yo61/go-udap/commit/a9ff8e5d8e9f27f941c8eef67ba2c657319a3a14))
* **cli:** treat pflag.ErrHelp as exit-0, no "error:" line ([798f442](https://github.com/yo61/go-udap/commit/798f4428a088618016ef10ba3f3285e55cafb6b8))
* **cli:** use ANSI erase-line, not 80-space fill, to clear progress ([fb5b6de](https://github.com/yo61/go-udap/commit/fb5b6de60e20681b355da9abe4da0b1469a48c7f))
* **release:** create releases as draft, publish after assets uploaded ([#8](https://github.com/yo61/go-udap/issues/8)) ([79107dc](https://github.com/yo61/go-udap/commit/79107dcb8aef1117049769ab5f969f93e0368e10))
* **release:** tap formula path + remove duplicate-release conflict ([#15](https://github.com/yo61/go-udap/issues/15)) ([9411f45](https://github.com/yo61/go-udap/commit/9411f455b9b718fcdade57ee15f9ffe082c31bde))
* **taskfile:** drop CLI_ARGS default so dev passes -- args ([a35c807](https://github.com/yo61/go-udap/commit/a35c8077c4dd120fb75a232f1f54a48767943da3))
* **udap,cli:** correct discovery TLV types; show product name + state ([f190b3b](https://github.com/yo61/go-udap/commit/f190b3b820f92aecdcc3b46505c7d3de6ea0abf5))
* **udap:** block until discovery listener exits, no 500ms cap ([772e7ea](https://github.com/yo61/go-udap/commit/772e7eac4588c66dbf08c9b5d594d1cad1d0b124))
* **udap:** clamp parseGetDataResponse map size hint ([#22](https://github.com/yo61/go-udap/issues/22)) ([ee1ffe4](https://github.com/yo61/go-udap/commit/ee1ffe45564f55fe1e55a1bed22391fc78d1ff7c))
* **udap:** clear stale offset_NNN keys before each GetAll merge ([#32](https://github.com/yo61/go-udap/issues/32)) ([130187a](https://github.com/yo61/go-udap/commit/130187ab749267738dfd2cbd3d07fe068a0b3d02))
* **udap:** correct UDAPHeaderSize to match serialized Packet size ([dd9b47f](https://github.com/yo61/go-udap/commit/dd9b47f5b43f0e325952211e511ded8bfdf39102))
* **udap:** defer Device.Parameters mutation until SetData ack ([#38](https://github.com/yo61/go-udap/issues/38)) ([7739185](https://github.com/yo61/go-udap/commit/7739185641fee2abd988702bbef42830d2c9779b))
* **udap:** drop SourceIP=0.0.0.0 capture filter; accept any source ([d61b3cb](https://github.com/yo61/go-udap/commit/d61b3cb01ea4eb02a9eae861138628b37fe109d5))
* **udap:** error out of SetDeviceConfigWithContext when prelude read fails ([#28](https://github.com/yo61/go-udap/issues/28)) ([e550e75](https://github.com/yo61/go-udap/commit/e550e750b80c0cb3d129af5cd647506ba1b58840))
* **udap:** include squeezecenter_name in KnownParameters ([55f198e](https://github.com/yo61/go-udap/commit/55f198e6cc7fc2771b385bae43bd45998aa507a6))
* **udap:** make Client.sequence increment race-free ([#20](https://github.com/yo61/go-udap/issues/20)) ([8cb5885](https://github.com/yo61/go-udap/commit/8cb5885836355b3a0e8b4a3e2c039b61ef119c5b))
* **udap:** make Sequence field's uint16 wrap explicit and tested ([#31](https://github.com/yo61/go-udap/issues/31)) ([a52a38e](https://github.com/yo61/go-udap/commit/a52a38eed92d59f1c5659d9fec656dae8a0a17b4))
* **udap:** pin reply source against device.IP in waitForDeviceReply ([#29](https://github.com/yo61/go-udap/issues/29)) ([a56b8ac](https://github.com/yo61/go-udap/commit/a56b8ac8cbb03a5aa842c47b1cca74a8b0a06548))
* **udap:** protect Client.devices map with RWMutex ([5e6fb9f](https://github.com/yo61/go-udap/commit/5e6fb9ff0f788bf752f9ed3e34d1c6f5cd9156f9))
* **udap:** return error on malformed MAC instead of silent zero MAC ([#21](https://github.com/yo61/go-udap/issues/21)) ([aa274d1](https://github.com/yo61/go-udap/commit/aa274d17c0cf5944a957477001e547976b2cdc62))
* **udap:** skip own kernel-looped broadcasts in packet capture ([2564df2](https://github.com/yo61/go-udap/commit/2564df2bbd47e58c55a257ffd1c047d6e83b0d64))
* **udap:** surface MethodError replies from ResetDeviceWithContext ([#25](https://github.com/yo61/go-udap/issues/25)) ([6e31c4a](https://github.com/yo61/go-udap/commit/6e31c4a1bd1252eed610105e3989b73b33627fa8))
* **udap:** use offset/length wire format for GetData ([24c0dc4](https://github.com/yo61/go-udap/commit/24c0dc44055bb815657ad1c61ba26b3fef4c1c9c))
* **udap:** use SyscallConn().Control() to set socket options ([33c47e7](https://github.com/yo61/go-udap/commit/33c47e7a37d35add653edd790f0ed606dfd1820a))
* **udap:** use UDAPHeaderSize in ParsePacket; fixes empty info fields ([239b11b](https://github.com/yo61/go-udap/commit/239b11b7cd9af542ae81eccf2306e0f2f43ba307))


### Reverts

* release 1.10.0 ([2a7b51d](https://github.com/yo61/go-udap/commit/2a7b51dc0ae2d6689632ed75cf58649e671e5d2f))


### Documentation

* add CLI redesign implementation plan ([1e82152](https://github.com/yo61/go-udap/commit/1e821525dbb7b93b8b89d10d351a8531ab630f37))
* add CLI redesign spec ([e6df8fb](https://github.com/yo61/go-udap/commit/e6df8fb7ee8458c5948e8188014d43da7e6fc2f3))
* add mock SBR design spec ([f5eed57](https://github.com/yo61/go-udap/commit/f5eed5790317ddd00053c094dc40b60a9e66a03b))
* add mocksbr Phase 1 implementation plan ([72c4ca1](https://github.com/yo61/go-udap/commit/72c4ca15d314e2458e1401895ab17189c44b2a81))
* add WPA2 and scripting examples to README ([cc87efe](https://github.com/yo61/go-udap/commit/cc87efec489c560da0d402c5f76158da11e64626))
* brief summarising the May 2026 review closeout ([#36](https://github.com/yo61/go-udap/issues/36)) ([bf871fe](https://github.com/yo61/go-udap/commit/bf871feece3e2ec4e14d2660a9778ecd950946c3))
* **claude.md:** refresh architecture section for current layout ([8a2b327](https://github.com/yo61/go-udap/commit/8a2b3270ccbf6dcc9a1917955c7378a72fd24273))
* **cli:** improve --config / --timeout placeholders and help text ([cf19cf8](https://github.com/yo61/go-udap/commit/cf19cf89095fff1fb5f921bc2ed6d4051cb64340))
* **cli:** show --timeout DURATION (not "duration") in --help ([c3e8a08](https://github.com/yo61/go-udap/commit/c3e8a0891b737342a4720994152f71f175dd27ef))
* fix broken in-content links ([3bb215c](https://github.com/yo61/go-udap/commit/3bb215c8c64dcdbf9deb05be457020a90751aeed))
* fix broken in-content links ([162d19f](https://github.com/yo61/go-udap/commit/162d19f6e464e47340664e44fecce5338155c49f))
* fix incorrect values for interface and lan_ip_mode parameters ([65ea9d9](https://github.com/yo61/go-udap/commit/65ea9d9498d42ef9f0b231418590499a0ded3666))
* **homebrew:** design spec + implementation plan for Homebrew tap ([#13](https://github.com/yo61/go-udap/issues/13)) ([824fbcf](https://github.com/yo61/go-udap/commit/824fbcf1702bd82aaa96505d608d7c40abebac20))
* **mocksbr:** add Appendix A (UDAP wire format from real captures) ([4f13e6c](https://github.com/yo61/go-udap/commit/4f13e6cb3589aae15d6b25227a25fc3fe0825e8a))
* **mocksbr:** correct GetData wire format from Perl Net::UDAP capture ([bc30456](https://github.com/yo61/go-udap/commit/bc304561e4912849c5326365cb764b104cb0ea66))
* **notes:** squeezeplay UDAP cross-check note ([#44](https://github.com/yo61/go-udap/issues/44)) ([f1cad8e](https://github.com/yo61/go-udap/commit/f1cad8ec5c98bdb5751729015b43b2ae1bae986c))
* **playbook:** add real-SBR capture session playbook ([757dcf8](https://github.com/yo61/go-udap/commit/757dcf83f111c958f968ac5398c71c3fff84190c))
* **readme:** document Homebrew install via yo61/tap ([#16](https://github.com/yo61/go-udap/issues/16)) ([583470d](https://github.com/yo61/go-udap/commit/583470d87cfe0ef8ae927bc7f4cb4ca7192b79db))
* refresh README, CLAUDE.md, DEVELOPMENT.md for current branch ([dbcd277](https://github.com/yo61/go-udap/commit/dbcd2771c772460a7f63d3666f8d7ccb42cae8a2))
* rewrite README for CLI-first usage ([bf2b787](https://github.com/yo61/go-udap/commit/bf2b787d786ce16e91cd34d1c087adb0dfa28ecf))
* serve site content at root instead of /docs/ ([d59ba41](https://github.com/yo61/go-udap/commit/d59ba41a2270fe712a3cc0af5c7c8a49782d8cf6))
* serve site content at root instead of /docs/ ([8a6ec1c](https://github.com/yo61/go-udap/commit/8a6ec1c79c69eca946cb29b68df6d4ba0e759d6b))
* **site:** tutorial, how-tos, and concept pages (M3) ([#53](https://github.com/yo61/go-udap/issues/53)) ([02319af](https://github.com/yo61/go-udap/commit/02319af4ac692db3423bbee248f706f8be9caa31))
* spec + plan for go-udap documentation site (Task [#28](https://github.com/yo61/go-udap/issues/28)) ([#48](https://github.com/yo61/go-udap/issues/48)) ([52f62c6](https://github.com/yo61/go-udap/commit/52f62c685a1fe2e39bfa328468de92bcd342208f))
* **udap,cli:** per-param flag placeholders in --help ([f11ed34](https://github.com/yo61/go-udap/commit/f11ed34a2a2f839c87b95e5e8221ea853602a425))
* **udap:** clarify live-pointer contract on device accessors ([#39](https://github.com/yo61/go-udap/issues/39)) ([43404e8](https://github.com/yo61/go-udap/commit/43404e879f1ccb3597b3be99cd2217012e85185e))
* **udap:** expand wireless-region-id help with country code table ([ccd5ae3](https://github.com/yo61/go-udap/commit/ccd5ae35fdee430d8e35d63a0c5d226261bdd0f1))
* **udap:** expand WPA cipher/mode help with valid values ([669d340](https://github.com/yo61/go-udap/commit/669d340de0c7981ce86af1436f693e81fda02420))
* update CLAUDE.md and DEVELOPMENT.md for CLI-first model ([8a12bd1](https://github.com/yo61/go-udap/commit/8a12bd15cb4f3c6fec67acd2d616b15717e9e534))


### Code Refactoring

* **cli,udap:** code-quality cleanups from review ([cb31149](https://github.com/yo61/go-udap/commit/cb311494bbba75d3d395069b5d9a628717147a1e))
* **cli,udap:** replace save/commit subcommands with set --reboot, rename reset → reboot ([19db596](https://github.com/yo61/go-udap/commit/19db59663aa1428f431672637962fa2ccf556666))
* drop regexp dependency for ~250KB binary size win ([a907320](https://github.com/yo61/go-udap/commit/a90732074348f59e71022e8ef1e519cc9b3d9270))
* replace interactive shell with CLI dispatcher ([6cf5062](https://github.com/yo61/go-udap/commit/6cf50625a72a892aeb3b72620325b3ace5f607ab))
* **udap,cli:** single source of truth for parameter table ([24ef7fe](https://github.com/yo61/go-udap/commit/24ef7fe3ab1685bb2ce4f439c9872165c35ce48b))
* **udap:** add port-configurable client constructor for tests ([33b8212](https://github.com/yo61/go-udap/commit/33b82125c4ff522a11e695b752f01281344c385e))
* **udap:** correct UCP method names; drop misleading aliases ([7abe200](https://github.com/yo61/go-udap/commit/7abe20056c1a81db57d6ab25099251fac5a4fb64))
* **udap:** drop dead listenForResponses + tighten ParsePacket ([4ba86ce](https://github.com/yo61/go-udap/commit/4ba86ce6475018fd077d7281528be52290242c33))
* **udap:** drop dead parseConfigResponse from discovery.go ([214d9f1](https://github.com/yo61/go-udap/commit/214d9f1df6a765e469a68f98ca2b1c106649441e))
* **udap:** drop legacy non-context entrypoints, collapse discovery chain ([2b57355](https://github.com/yo61/go-udap/commit/2b57355a32dbfda50cc5bb06e8307b66902078a6))
* **udap:** drop unreachable bind-fresh-socket path in capture ([a58de6a](https://github.com/yo61/go-udap/commit/a58de6a550088177837fe7cdb391451e92e87a53))
* **udap:** move wire-encoding into Parameter.Encode ([#40](https://github.com/yo61/go-udap/issues/40)) ([c231e8f](https://github.com/yo61/go-udap/commit/c231e8fb0a514de2e9afbbcbd423f46cb48d792d))
* **udap:** switch Client onto Transport, drop capture helpers ([5d0eba1](https://github.com/yo61/go-udap/commit/5d0eba1c0ef42fe0e88b8d10bc9ef7e2dbb69484))

## [2.0.0](https://github.com/yo61/go-udap/compare/v1.10.1...v2.0.0) (2026-05-25)


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
