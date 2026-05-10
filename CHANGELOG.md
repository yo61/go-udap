# Changelog

All notable changes to this project will be documented in this file.

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
