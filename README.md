# go-udap

[![CI](https://github.com/yo61/go-udap/actions/workflows/ci.yaml/badge.svg)](https://github.com/yo61/go-udap/actions/workflows/ci.yaml)
[![Docs](https://github.com/yo61/go-udap/actions/workflows/docs.yaml/badge.svg)](https://yo61.github.io/go-udap/)
[![Release](https://img.shields.io/github/v/release/yo61/go-udap)](https://github.com/yo61/go-udap/releases)
[![Go Reference](https://pkg.go.dev/badge/go-udap.svg)](https://pkg.go.dev/go-udap)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/yo61/go-udap)

A small command-line tool that discovers and configures
[Squeezebox](https://wiki.lyrion.org/) devices over UDAP — the
on-the-wire protocol Logitech's original setup app spoke. One static
binary, no Lyrion Music Server required for setup, no Perl runtime, no
GUI.

> [!NOTE]
> UDAP only talks to Squeezeboxes that are in *setup mode* (the light
> flashes red). Brand-new devices arrive in setup mode; existing devices
> can be put back in by holding the front button for 3–6 seconds.

## Features

- **Discover** every Squeezebox on the LAN via UDP broadcast on port 17784
- **Read & write all 26 NVRAM parameters** — network, wireless, server pointer, hostname, and more
- **Query live network config** (`getip`) — actual IP / subnet / gateway, distinct from passive discovery
- **Multi-NIC aware** — pick one interface (`--bind-interface`) or fan out across all of them (`--all-interfaces`)
- **Shell completions** for bash, zsh, and fish, including MAC-address completion via short-timeout discovery
- **Single static binary**, ~2.8 MB, no runtime dependencies — macOS / Linux / Windows on amd64 and arm64
- **Round-trippable config** — `read | set` cycles cleanly through file or stdin

## Install

```bash
# Homebrew (macOS / Linux) — installs the binary and shell completions
brew install yo61/tap/go-udap

# Or download a release binary directly:
# https://github.com/yo61/go-udap/releases
```

Full install options (binary download, `go install`, building from
source) live in the
[installation guide](https://yo61.github.io/go-udap/docs/how-to/install-go-udap).

## Quick start

Find devices on the LAN:

```bash
$ go-udap discover
00:04:20:16:06:02
```

Inspect one device:

```bash
$ go-udap info 00:04:20:16:06:02
MAC:      00:04:20:16:06:02
IP:       0.0.0.0
Name:     Squeezebox Device
Model:    Squeezebox Receiver
Firmware: 77
HW Rev:   0005
State:    init
```

Give it a hostname, point it at your server, reboot:

```bash
go-udap set 00:04:20:16:06:02 \
  --hostname living-room \
  --server-address 192.168.1.250 \
  --reboot
```

The
[5-minute tutorial](https://yo61.github.io/go-udap/docs/tutorials/configure-your-first-squeezebox)
walks through a full first-time setup, including DHCP vs. static IP and
recovering a stuck device.

## Documentation

Full documentation lives at
**[yo61.github.io/go-udap](https://yo61.github.io/go-udap/)**:

| Section | What lives there |
| --- | --- |
| [Tutorial](https://yo61.github.io/go-udap/docs/tutorials/configure-your-first-squeezebox) | Start here — configure a brand-new Squeezebox end-to-end |
| [How-to guides](https://yo61.github.io/go-udap/docs/how-to) | DHCP, static IP, Wi-Fi WPA2, multi-NIC discovery, recovery |
| [Reference](https://yo61.github.io/go-udap/docs/reference/commands) | Every subcommand, flag, NVRAM parameter, exit code |
| [Concepts](https://yo61.github.io/go-udap/docs/concepts) | How UDAP discovery works on the wire, why 26 parameters |
| [Go API](https://pkg.go.dev/go-udap) | `udap` package — embed UDAP in your own Go programs |

## Shell completions

Homebrew installs bash / zsh / fish completions automatically and
keeps them in sync on `brew upgrade`. For other installs:

```bash
go-udap completion bash > ~/.local/share/bash-completion/completions/go-udap
go-udap completion zsh  > ~/.zsh/completions/_go-udap
go-udap completion fish > ~/.config/fish/completions/go-udap.fish
```

Completions cover subcommands, flag names, the 26 NVRAM parameters,
discovered MAC addresses, and `--bind-interface` values.

## Protocol provenance

go-udap mirrors the Perl
[Net::UDAP](https://metacpan.org/dist/Net-UDAP) reference
implementation. Where the two diverge, Net::UDAP is the source of
truth. See
[Concepts: What is go-udap?](https://yo61.github.io/go-udap/docs/concepts/what-is-go-udap)
for the full lineage and the rationale for the rewrite.
