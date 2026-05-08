package udap

import (
	"fmt"
	"strings"
)

// Parameter is the single source of truth for one UDAP NVRAM-resident
// parameter. The Name is the canonical wire name used in protocol
// messages, INI files, and `get`/`set` output. The Help is end-user
// documentation surfaced by the CLI's `--help` text and (potentially)
// generated docs. The Placeholder is the value-form shown after the
// flag name in --help (e.g. "IP", "0|1", "NAME"); it overrides pflag's
// default "string" placeholder so users can see at a glance what kind
// of value each flag expects. Empty Placeholder falls back to "string".
//
// To add a new parameter: append one entry to Parameters below. The CLI
// flag, `read` coverage, and offset-reverse-lookup are all derived.
type Parameter struct {
	Name        string
	Offset      uint16
	Length      uint16
	Placeholder string
	Help        string
}

// FlagName returns the CLI flag form of the parameter name: lowercased
// with underscores converted to hyphens (e.g. "wireless_SSID" →
// "wireless-ssid").
func (p Parameter) FlagName() string {
	return strings.ReplaceAll(strings.ToLower(p.Name), "_", "-")
}

// Validate sanity-checks the Parameter's NVRAM offset and length.
func (p Parameter) Validate() error {
	if p.Offset > MaxNVRAMOffset {
		return fmt.Errorf("offset too large: %d", p.Offset)
	}
	if p.Length == 0 {
		return fmt.Errorf("length cannot be zero")
	}
	if p.Length > MaxConfigLength {
		return fmt.Errorf("length too large: %d (max %d)", p.Length, MaxConfigLength)
	}
	return nil
}

// Parameters is the canonical, ordered list of all known UDAP NVRAM
// parameters. Order is intentional and stable — `read` emits in this
// order (after sort), and SetData/GetData wire packets sort by Offset
// independently. Don't reorder for cosmetic reasons.
//
// NVRAM offsets and lengths come from the LMS-Community/squeezeplay
// Lua reference implementation; cross-referenced against the
// Net::UDAP Perl shell session in perl_shell_session.txt.
var Parameters = []Parameter{
	{"lan_ip_mode", 4, 1, "0|1", "0=static, 1=DHCP"},
	{"lan_network_address", 5, 4, "IP", "Static IPv4 address (e.g. 192.168.1.50)"},
	{"lan_subnet_mask", 9, 4, "MASK", "Subnet mask (e.g. 255.255.255.0)"},
	{"lan_gateway", 13, 4, "IP", "Default gateway IPv4 address"},
	{"hostname", 17, 33, "NAME", "Device hostname (max 33 chars)"},
	{"bridging", 50, 1, "0|1", "0=disabled, 1=enabled"},
	{"interface", 52, 1, "0|1", "0=wireless, 1=wired (Ethernet)"},
	{"primary_dns", 59, 4, "IP", "Primary DNS server IPv4 address"},
	{"secondary_dns", 67, 4, "IP", "Secondary DNS server IPv4 address"},
	{"server_address", 71, 4, "IP", "Logitech Media Server IPv4 address"},
	{"lms_address", 79, 4, "IP", "Alternative LMS server IPv4 address"},
	{"squeezecenter_name", 83, 33, "NAME", "Squeezecenter / LMS server name (max 33 chars)"},
	{"wireless_mode", 173, 1, "0|1", "0=infrastructure, 1=ad-hoc"},
	{"wireless_SSID", 183, 33, "SSID", "Wireless SSID (1-32 chars)"},
	{"wireless_channel", 216, 1, "N", "Wireless channel (1-13)"},
	{"wireless_region_id", 218, 1, "ID", "Wireless region identifier (4=US, 6=CA, 7=AU, 13=FR, 14=EU, 16=JP, 21=TW, 23=CH)"},
	{"wireless_keylen", 220, 1, "5|13", "WEP key length"},
	{"wireless_wep_key", 222, 13, "HEX", "Primary WEP key"},
	{"wireless_wep_key_1", 235, 13, "HEX", "WEP key slot 1"},
	{"wireless_wep_key_2", 248, 13, "HEX", "WEP key slot 2"},
	{"wireless_wep_key_3", 261, 13, "HEX", "WEP key slot 3"},
	{"wireless_wep_on", 274, 1, "0|1", "0=disabled, 1=enabled"},
	{"wireless_wpa_cipher", 275, 1, "1|2|3", "1=TKIP, 2=AES (CCMP), 3=TKIP+AES"},
	{"wireless_wpa_mode", 276, 1, "1|2", "1=WPA, 2=WPA2"},
	{"wireless_wpa_on", 277, 1, "0|1", "0=disabled, 1=enabled"},
	{"wireless_wpa_psk", 278, 64, "PSK", "WPA pre-shared key (8-63 chars)"},
}

// parameterAliases lets the CLI accept legacy or third-party-tool names
// that historically referred to the same NVRAM byte. These don't appear
// in Parameters (no separate `read` slot, no separate CLI flag) but
// ParameterByName will resolve them to the canonical entry.
var parameterAliases = map[string]string{
	"slimserver_address":    "server_address",
	"squeezecenter_address": "server_address",
}

// parameterIndex caches Name → *Parameter for O(1) lookup.
var parameterIndex = func() map[string]*Parameter {
	out := make(map[string]*Parameter, len(Parameters))
	for i := range Parameters {
		out[Parameters[i].Name] = &Parameters[i]
	}
	return out
}()

// ParameterByName returns the Parameter for the given canonical UDAP
// name, transparently resolving registered aliases. Returns false if
// the name is unknown.
func ParameterByName(name string) (Parameter, bool) {
	if p, ok := parameterIndex[name]; ok {
		return *p, true
	}
	if canonical, ok := parameterAliases[name]; ok {
		if p, ok := parameterIndex[canonical]; ok {
			return *p, true
		}
	}
	return Parameter{}, false
}

// ParameterNames returns a fresh slice of all canonical parameter names
// in their declared order. Used by GetAllDeviceConfig to request every
// known param.
func ParameterNames() []string {
	out := make([]string, len(Parameters))
	for i := range Parameters {
		out[i] = Parameters[i].Name
	}
	return out
}
