[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/robinbowes/go-udap)

# Squeezebox UDAP Configuration Tool

A command-line tool for discovering and configuring Squeezebox devices on your network using the UDAP (Universal Device Access Protocol).

## Overview

This tool allows you to:
- Discover Squeezebox devices on your local network
- Configure network settings (IP address, gateway, DNS)
- Configure wireless settings (SSID, WPA/WEP keys)
- Set the Logitech Media Server (LMS) address
- Save configuration to device persistent storage
- Reset devices to apply new configuration

The tool is single-shot and command-line driven; every operation is one
invocation. There is no interactive shell.

## Installation

### Pre-built Binaries

Download the archive for your platform from the [Releases](https://github.com/robinbowes/go-udap/releases) page (`go-udap_<version>_<os>_<arch>.tar.gz`, or `.zip` on Windows). Extract the `go-udap` binary from the archive and place it on your `PATH`. `SHA256SUMS` next to each release lets you verify the download.

### Build from Source

```bash
git clone https://github.com/robinbowes/go-udap.git
cd go-udap
go build -o go-udap .
```

See [DEVELOPMENT.md](DEVELOPMENT.md) for detailed build instructions and cross-compilation.

## Usage

```
go-udap [global flags] <command> [args] [flags]
```

### Commands

| Command | Description |
|---------|-------------|
| `discover [--info]` | Discover devices; print MAC addresses (or full metadata with `--info`) |
| `info <mac>` | Show metadata for one device |
| `read <mac> [--all/-a]` | Read parameters from a device. By default skips factory-default values so the output round-trips cleanly through `set`; pass `--all`/`-a` to dump everything. |
| `get <mac> <param> [<param>...]` | Read specific parameters |
| `set <mac> [--reboot/-r] [--config FILE] [--<param> VALUE ...]` | Set parameters from any combination of `--config FILE` (or `--config -` for stdin), piped stdin, and per-param flags. The wire op writes NVRAM directly; pass `--reboot/-r` to also reboot afterward. |
| `reboot <mac>` | Reboot the device |

### Global flags

| Flag | Default | Purpose |
|---|---|---|
| `--timeout DURATION` | `5s` | Operation timeout (e.g. `5s`, `30s`, `2m`) |
| `--verbose, -v` | off | Debug logging to stderr |
| `--version` | — | Print version and exit |
| `--help, -h` | — | Print help |

Global flags are accepted before OR after the subcommand —
`go-udap -v read <mac>` and `go-udap read -v <mac>` are equivalent.

### Output

Command output is on **stdout**; logs and warnings are on **stderr**. This
keeps stdout machine-parseable.

- `discover` — one MAC per line.
- `discover --info` — multi-line metadata block per device.
- `read` — `param=value` lines, sorted by name.
- `get <mac> <param>` (single) — bare value.
- `get <mac> <p1> <p2>` (multi) — `param=value` lines in request order.

### Examples

Discover devices on the LAN:

```bash
go-udap discover
# 00:04:20:16:06:02
```

Show full metadata:

```bash
go-udap discover --info
```

Back up the device's non-default settings (output round-trips through
`set --config -`):

```bash
go-udap read 00:04:20:16:06:02 > backup.conf

# Restore later:
go-udap set 00:04:20:16:06:02 --config backup.conf --reboot
```

To inspect every parameter the device returns (including factory
defaults and any unrecognized NVRAM offsets):

```bash
go-udap read --all 00:04:20:16:06:02
```

Configure a device for DHCP on wireless with WPA2:

```bash
go-udap set 00:04:20:16:06:02 \
  --interface 0 --lan-ip-mode 1 \
  --wireless-ssid SlimNet --wireless-wpa-on 1 --wireless-wpa-mode 2 \
  --wireless-wpa-psk 'shared-secret' \
  --server-address 192.168.1.250 \
  --reboot
```

Apply a saved config file (and override one value at the CLI), then reboot:

```bash
go-udap set 00:04:20:16:06:02 --config backup.conf --hostname new-name --reboot
```

Pipe parameters from stdin (here-string or here-doc):

```bash
go-udap set 00:04:20:16:06:02 <<< "lan_ip_mode=1
wireless_SSID=foo"

go-udap set 00:04:20:16:06:02 <<EOF
interface=1
lan_ip_mode=0
lan_network_address=192.168.1.50
lan_subnet_mask=255.255.255.0
lan_gateway=192.168.1.1
EOF
```

Get a single value for use in a script:

```bash
ip=$(go-udap get 00:04:20:16:06:02 lan_network_address)
```

### Config file format

INI-style: one `key=value` per line; `#` and `;` start comments; blank
lines ignored. Keys must be canonical UDAP parameter names (e.g.
`wireless_SSID`, not `wireless-ssid`). The format matches `read` output, so
round-tripping works without conversion.

```ini
# Network
interface=1
lan_ip_mode=1

# Wireless
wireless_SSID=MyNet
wireless_wpa_on=1
wireless_wpa_psk=secret
```

## Configuration Parameters

### Network

| Parameter | Type | Description |
|-----------|------|-------------|
| `lan_ip_mode` | Integer (0-1) | 0 = Static IP, 1 = DHCP |
| `lan_network_address` | IPv4 | Static IP address |
| `lan_subnet_mask` | IPv4 | Subnet mask |
| `lan_gateway` | IPv4 | Default gateway |
| `primary_dns` | IPv4 | Primary DNS |
| `secondary_dns` | IPv4 | Secondary DNS |
| `hostname` | String (max 33) | Device hostname |
| `bridging` | Integer (0-1) | Enable/disable bridging |
| `interface` | Integer (0-1) | 0 = Wireless, 1 = Wired |

### Server

| Parameter | Type | Description |
|-----------|------|-------------|
| `server_address` | IPv4 | LMS address |
| `lms_address` | IPv4 | Alternative LMS address |
| `squeezecenter_address` | IPv4 | Alias for `server_address` |
| `slimserver_address` | IPv4 | Alias for `server_address` |

### Wireless

| Parameter | Type | Description |
|-----------|------|-------------|
| `wireless_mode` | Integer (0-1) | 0 = Infrastructure, 1 = Ad-hoc |
| `wireless_SSID` | String (1-32) | Network name |
| `wireless_channel` | Integer (1-13) | Channel |
| `wireless_region_id` | Integer | Region |

### Wireless security — WEP

| Parameter | Type | Description |
|-----------|------|-------------|
| `wireless_wep_on` | Integer (0-1) | Enable/disable WEP |
| `wireless_keylen` | Integer (5/13) | WEP key length |
| `wireless_wep_key` | String | Primary WEP key |
| `wireless_wep_key_1`..`_3` | String | WEP key slots 1-3 |

### Wireless security — WPA/WPA2

| Parameter | Type | Description |
|-----------|------|-------------|
| `wireless_wpa_on` | Integer (0-1) | Enable/disable WPA |
| `wireless_wpa_mode` | Integer | WPA mode |
| `wireless_wpa_cipher` | Integer | WPA cipher |
| `wireless_wpa_psk` | String (8-63) | WPA pre-shared key |

### Factory reset

Factory reset is **not** exposed by the protocol. Perform it on the device
itself: hold the front button for ~6 seconds until it blinks fast red.
See https://wiki.lyrion.org/index.php/SBRFrontButtonAndLED.

## Troubleshooting

### No devices found

- Ensure the device is powered on and on the same network segment.
- Devices in bootstrap mode (unconfigured) report IP `0.0.0.0` and are still
  reachable via broadcast.
- Make sure UDP port 17784 is not blocked by a firewall.

### Configuration not applying

- `set` writes to NVRAM directly on the wire (single UCP_METHOD_SET_DATA op
  per the Net::UDAP reference). Pass `--reboot/-r` to also reboot, or run
  `reboot <mac>` separately, since some changes only take effect after reboot.

### Permission errors

- Binding to UDP 17784 typically does not require root, but on some platforms
  you may need elevated privileges if the port is otherwise restricted.

## License

MIT License — see [LICENSE](LICENSE) for details.

You must retain the copyright notice and license in any copies or substantial
portions of the software.
