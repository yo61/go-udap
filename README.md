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

## Installation

### Prerequisites

- Go 1.21 or later
- [Task](https://taskfile.dev/) (optional, for build automation)

### Building from Source

```bash
# Clone the repository
git clone https://github.com/robinbowes/squeezebox-udap.git
cd squeezebox-udap

# Build using Task (recommended)
task build

# Or build directly with Go
go build -ldflags="-s -w" -trimpath -o squeezebox-udap main.go
```

### Cross-Compilation

The tool can be cross-compiled for different platforms:

```bash
# Windows
task build:windows

# Linux (amd64)
task build:linux

# Linux (arm64)
task build:linux-arm64

# All platforms
task build:all
```

## Usage

Run the tool to enter interactive mode:

```bash
./squeezebox-udap
```

### Commands

| Command | Description |
|---------|-------------|
| `discover` | Discover Squeezebox devices on the network |
| `list` | List all discovered devices |
| `info <mac>` | Show detailed information about a device |
| `read <mac>` | Read all configuration parameters from a device |
| `config <mac> get <param>` | Get a specific configuration parameter |
| `config <mac> set <param>=<value> ...` | Set one or more configuration parameters |
| `save <mac>` | Save configuration to device persistent storage |
| `reset <mac>` | Reset/reboot the device |
| `commit <mac>` | Save configuration and reset device (combined operation) |
| `help` | Show available commands |
| `quit` / `exit` | Exit the tool |

### Typical Workflow

1. **Discover devices** on your network:
   ```
   > discover
   Discovery complete. Found 1 devices.
   ```

2. **List discovered devices** to see MAC addresses:
   ```
   > list
   Discovered devices:
     00:04:20:16:06:02 - Squeezebox Device () at 0.0.0.0
   ```

3. **Configure the device** with your settings:
   ```
   > config 00:04:20:16:06:02 set interface=1 lan_ip_mode=1 server_address=192.168.1.100
   ```

4. **Save the configuration** to persistent storage:
   ```
   > save 00:04:20:16:06:02
   ```

5. **Reset the device** to apply the new configuration:
   ```
   > reset 00:04:20:16:06:02
   ```

   Or use `commit` to save and reset in one step:
   ```
   > commit 00:04:20:16:06:02
   ```

## Configuration Parameters

### Network Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `lan_ip_mode` | Integer (0-1) | 0 = DHCP, 1 = Static IP |
| `lan_network_address` | IP Address | Static IP address for the device |
| `lan_subnet_mask` | IP Address | Subnet mask (e.g., 255.255.255.0) |
| `lan_gateway` | IP Address | Default gateway |
| `primary_dns` | IP Address | Primary DNS server |
| `secondary_dns` | IP Address | Secondary DNS server |
| `hostname` | String (max 33 chars) | Device hostname |
| `bridging` | Integer (0-1) | Enable/disable bridging mode |
| `interface` | Integer (0-1) | 0 = Wired (Ethernet), 1 = Wireless |

### Server Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `server_address` | IP Address | Logitech Media Server address |
| `lms_address` | IP Address | Alternative LMS address field |
| `squeezecenter_address` | IP Address | Alias for server_address (compatibility) |
| `slimserver_address` | IP Address | Alias for server_address (compatibility) |

### Wireless Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `wireless_mode` | Integer (0-1) | 0 = Infrastructure, 1 = Ad-hoc |
| `wireless_SSID` | String (max 33 chars) | Wireless network name |
| `wireless_channel` | Integer (1-13) | Wireless channel |
| `wireless_region_id` | Integer | Wireless region identifier |

### Wireless Security - WEP

| Parameter | Type | Description |
|-----------|------|-------------|
| `wireless_wep_on` | Integer (0-1) | Enable/disable WEP |
| `wireless_keylen` | Integer (5 or 13) | WEP key length |
| `wireless_wep_key` | String | Primary WEP key |
| `wireless_wep_key_1` | String | WEP key slot 1 |
| `wireless_wep_key_2` | String | WEP key slot 2 |
| `wireless_wep_key_3` | String | WEP key slot 3 |

### Wireless Security - WPA/WPA2

| Parameter | Type | Description |
|-----------|------|-------------|
| `wireless_wpa_on` | Integer (0-1) | Enable/disable WPA |
| `wireless_wpa_mode` | Integer | WPA mode |
| `wireless_wpa_cipher` | Integer | WPA cipher type |
| `wireless_wpa_psk` | String (8-64 chars) | WPA pre-shared key |

## Examples

### Configure a device for DHCP on wireless

```
> discover
> config 00:04:20:16:06:02 set interface=1 lan_ip_mode=0 wireless_SSID=MyNetwork wireless_wpa_on=1 wireless_wpa_psk=MyPassword
> commit 00:04:20:16:06:02
```

### Configure a device with static IP on wired connection

```
> discover
> config 00:04:20:16:06:02 set interface=0 lan_ip_mode=1 lan_network_address=192.168.1.50 lan_subnet_mask=255.255.255.0 lan_gateway=192.168.1.1 primary_dns=8.8.8.8 server_address=192.168.1.100
> commit 00:04:20:16:06:02
```

### Read current device configuration

```
> discover
> read 00:04:20:16:06:02
Device Parameters (15 total):
  lan_ip_mode = 0
  interface = 1
  wireless_SSID = MyNetwork
  ...
```

## Protocol Details

- Uses UDP broadcast on port 17784 for device discovery
- Implements UDAP packet format with TLV (Type-Length-Value) encoding
- Supports both standard discovery (method 0x0001) and advanced discovery (method 0x0009)
- Compatible with Squeezebox Receiver, Squeezebox Boom, and other UDAP-enabled devices

## Troubleshooting

### No devices found

- Ensure your Squeezebox device is powered on and connected to the network
- Devices in "bootstrap mode" (unconfigured) will show IP address 0.0.0.0
- Make sure you're on the same network segment as the device
- Check that UDP port 17784 is not blocked by a firewall

### Configuration not applying

- After setting parameters, you must run `save` to persist the configuration
- After saving, run `reset` to reboot the device with new settings
- Or use `commit` to do both in one step

### Permission errors

- On Linux/macOS, you may need to run with elevated privileges to bind to UDP port 17784
- Try running with `sudo` if you encounter permission errors

## License

MIT License - see LICENSE file for details.

## Acknowledgments

- Based on the UDAP protocol implementation from [LMS-Community/squeezeplay](https://github.com/LMS-Community/squeezeplay)
- Inspired by the Perl [Net::UDAP](https://metacpan.org/pod/Net::UDAP) module
