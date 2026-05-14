# go-udap

[![CI](https://github.com/yo61/go-udap/actions/workflows/ci.yaml/badge.svg)](https://github.com/yo61/go-udap/actions/workflows/ci.yaml)
[![Docs](https://github.com/yo61/go-udap/actions/workflows/docs.yaml/badge.svg)](https://yo61.github.io/go-udap/)
[![Release](https://img.shields.io/github/v/release/yo61/go-udap)](https://github.com/yo61/go-udap/releases)

A command-line tool for discovering and configuring Squeezebox devices
on your network using the UDAP (Universal Device Access Protocol).

## 📖 Documentation

**Full documentation lives at [yo61.github.io/go-udap](https://yo61.github.io/go-udap/)**.

Quick links:
- [Tutorial: configure your first Squeezebox](https://yo61.github.io/go-udap/docs/tutorials/configure-your-first-squeezebox)
- [How-to guides](https://yo61.github.io/go-udap/docs/how-to)
- [Command reference](https://yo61.github.io/go-udap/docs/reference/commands)
- [Concepts](https://yo61.github.io/go-udap/docs/concepts)
- [API reference (pkg.go.dev)](https://pkg.go.dev/go-udap)

## Install

```bash
# Homebrew
brew install yo61/tap/go-udap

# Or download from the Releases page:
# https://github.com/yo61/go-udap/releases
```

See [installation guide](https://yo61.github.io/go-udap/docs/how-to/install-go-udap)
for all options.

## Quick start

```bash
go-udap discover                          # find devices on the LAN
go-udap info 00:04:20:16:06:02            # show one device's metadata
go-udap getip 00:04:20:16:06:02           # check its IP / subnet / gateway
go-udap set 00:04:20:16:06:02 \
  --hostname living-room --reboot         # rename and reboot
```

The [tutorial](https://yo61.github.io/go-udap/docs/tutorials/configure-your-first-squeezebox)
walks through a complete first-time setup.

## Contributing

See [contributing docs](https://yo61.github.io/go-udap/docs/contributing).

## License

MIT — see [LICENSE](LICENSE).
