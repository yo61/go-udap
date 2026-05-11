package udap

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
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
// FactoryDefault is the value the device reports for this parameter
// after a hardware reset. Used by `read` to filter out factory-default
// values (which aren't interesting for backup, and some — like
// wireless_keylen=0 or interface=128 — aren't even valid input to
// `set`). Captured from a real Squeezebox Receiver immediately after
// reset; may differ slightly across hardware variants and regional
// SKUs, but it's the best baseline we have.
//
// To add a new parameter: append one entry to Parameters below. The CLI
// flag, `read` coverage, and offset-reverse-lookup are all derived.
type Parameter struct {
	Name           string
	Offset         uint16
	Length         uint16
	Placeholder    string
	Help           string
	FactoryDefault string
}

// FlagName returns the CLI flag form of the parameter name: lowercased
// with underscores converted to hyphens (e.g. "wireless_SSID" →
// "wireless-ssid").
func (p Parameter) FlagName() string {
	return strings.ReplaceAll(strings.ToLower(p.Name), "_", "-")
}

// Encode produces the wire-format bytes for value, always returning a
// slice of exactly p.Length bytes on success. The encoding switches on
// p.Length:
//
//   - 1: parsed as uint8 (decimal).
//   - 2: parsed as uint16 (decimal), written big-endian.
//   - 4: parsed as an IPv4 address (rejects IPv6, malformed octets).
//   - other: treated as a UTF-8 string; zero-padded if shorter, silently
//     truncated if longer. The truncation branch is unreachable from the
//     CLI (validateParameter rejects over-length strings upstream); it is
//     preserved for library callers that bypass CLI validation, matching
//     the historical CreateSetDataPacket behavior.
//
// Encode is the wire-side counterpart to validateParameter: this method
// owns the bytes-on-the-wire contract; validateParameter owns
// user-input validation with friendlier messages and per-parameter rules.
func (p Parameter) Encode(value string) ([]byte, error) {
	switch p.Length {
	case 1:
		n, err := strconv.ParseUint(value, 10, 8)
		if err != nil {
			return nil, fmt.Errorf("%q is not a valid uint8: %w", value, err)
		}
		return []byte{byte(n)}, nil
	case 2:
		n, err := strconv.ParseUint(value, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("%q is not a valid uint16: %w", value, err)
		}
		out := make([]byte, 2)
		binary.BigEndian.PutUint16(out, uint16(n))
		return out, nil
	case 4:
		ip := net.ParseIP(value)
		if ip == nil {
			return nil, fmt.Errorf("cannot parse %q as IPv4 address", value)
		}
		ip4 := ip.To4()
		if ip4 == nil {
			return nil, fmt.Errorf("%q is not an IPv4 address", value)
		}
		out := make([]byte, 4)
		copy(out, ip4)
		return out, nil
	default:
		out := make([]byte, p.Length)
		copy(out, value)
		return out, nil
	}
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
	{"lan_ip_mode", 4, 1, "0|1", "0=static, 1=DHCP", "1"},
	{"lan_network_address", 5, 4, "IP", "Static IPv4 address (e.g. 192.168.1.50)", "0.0.0.0"},
	{"lan_subnet_mask", 9, 4, "MASK", "Subnet mask (e.g. 255.255.255.0)", "255.255.255.0"},
	{"lan_gateway", 13, 4, "IP", "Default gateway IPv4 address", "0.0.0.0"},
	{"hostname", 17, 33, "NAME", "Device hostname (max 33 chars)", ""},
	{"bridging", 50, 1, "0|1", "0=disabled, 1=enabled", "0"},
	{"interface", 52, 1, "0|1", "0=wireless, 1=wired (Ethernet)", "128"},
	{"primary_dns", 59, 4, "IP", "Primary DNS server IPv4 address", "0.0.0.0"},
	{"secondary_dns", 67, 4, "IP", "Secondary DNS server IPv4 address", "0.0.0.0"},
	{"server_address", 71, 4, "IP", "Logitech Media Server IPv4 address", "0.0.0.0"},
	{"lms_address", 79, 4, "IP", "Alternative LMS server IPv4 address", "0.0.0.0"},
	{"squeezecenter_name", 83, 33, "NAME", "Squeezecenter / LMS server name (max 33 chars)", ""},
	{"wireless_mode", 173, 1, "0|1", "0=infrastructure, 1=ad-hoc", "0"},
	{"wireless_SSID", 183, 33, "SSID", "Wireless SSID (1-32 chars)", ""},
	{"wireless_channel", 216, 1, "N", "Wireless channel (1-13)", "6"},
	{"wireless_region_id", 218, 1, "ID", "Wireless region identifier (4=US, 6=CA, 7=AU, 13=FR, 14=EU, 16=JP, 21=TW, 23=CH)", "4"},
	{"wireless_keylen", 220, 1, "5|13", "WEP key length", "0"},
	{"wireless_wep_key", 222, 13, "HEX", "Primary WEP key", ""},
	{"wireless_wep_key_1", 235, 13, "HEX", "WEP key slot 1", ""},
	{"wireless_wep_key_2", 248, 13, "HEX", "WEP key slot 2", ""},
	{"wireless_wep_key_3", 261, 13, "HEX", "WEP key slot 3", ""},
	{"wireless_wep_on", 274, 1, "0|1", "0=disabled, 1=enabled", "0"},
	{"wireless_wpa_cipher", 275, 1, "1|2|3", "1=TKIP, 2=AES (CCMP), 3=TKIP+AES", "3"},
	{"wireless_wpa_mode", 276, 1, "1|2", "1=WPA, 2=WPA2", "1"},
	{"wireless_wpa_on", 277, 1, "0|1", "0=disabled, 1=enabled", "0"},
	{"wireless_wpa_psk", 278, 64, "PSK", "WPA pre-shared key (8-63 chars)", ""},
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
// in their declared order. Used by GetAllDeviceConfigWithContext to
// request every known param.
func ParameterNames() []string {
	out := make([]string, len(Parameters))
	for i := range Parameters {
		out[i] = Parameters[i].Name
	}
	return out
}
