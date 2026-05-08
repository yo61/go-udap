package cli

// paramFlag maps a CLI flag name to its canonical UDAP parameter name and help text.
//   - udapName is used in protocol messages and INI files (e.g. "wireless_SSID").
//   - flagName is the CLI form, lowercase-with-hyphens (e.g. "wireless-ssid").
type paramFlag struct {
	udapName string
	flagName string
	help     string
}

// paramFlags returns the full table of CLI flags for known UDAP parameters.
// The table must stay in sync with udap.KnownParameters; cli/params_test.go
// asserts coverage in both directions.
func paramFlags() []paramFlag {
	return []paramFlag{
		{"lan_ip_mode", "lan-ip-mode", "0=static, 1=DHCP"},
		{"lan_network_address", "lan-network-address", "Static IPv4 address (e.g. 192.168.1.50)"},
		{"lan_subnet_mask", "lan-subnet-mask", "Subnet mask (e.g. 255.255.255.0)"},
		{"lan_gateway", "lan-gateway", "Default gateway IPv4 address"},
		{"hostname", "hostname", "Device hostname (max 33 chars)"},
		{"bridging", "bridging", "0=disabled, 1=enabled"},
		{"interface", "interface", "0=wireless, 1=wired (Ethernet)"},
		{"primary_dns", "primary-dns", "Primary DNS server IPv4 address"},
		{"secondary_dns", "secondary-dns", "Secondary DNS server IPv4 address"},
		{"server_address", "server-address", "Logitech Media Server IPv4 address"},
		{"lms_address", "lms-address", "Alternative LMS server IPv4 address"},
		{"squeezecenter_name", "squeezecenter-name", "Squeezecenter / LMS server name (max 33 chars)"},
		{"wireless_mode", "wireless-mode", "0=infrastructure, 1=ad-hoc"},
		{"wireless_SSID", "wireless-ssid", "Wireless SSID (1-32 chars)"},
		{"wireless_channel", "wireless-channel", "Wireless channel (1-13)"},
		{"wireless_region_id", "wireless-region-id", "Wireless region identifier"},
		{"wireless_keylen", "wireless-keylen", "WEP key length: 5 or 13"},
		{"wireless_wep_key", "wireless-wep-key", "Primary WEP key"},
		{"wireless_wep_key_1", "wireless-wep-key-1", "WEP key slot 1"},
		{"wireless_wep_key_2", "wireless-wep-key-2", "WEP key slot 2"},
		{"wireless_wep_key_3", "wireless-wep-key-3", "WEP key slot 3"},
		{"wireless_wep_on", "wireless-wep-on", "0=disabled, 1=enabled"},
		{"wireless_wpa_cipher", "wireless-wpa-cipher", "WPA cipher type"},
		{"wireless_wpa_mode", "wireless-wpa-mode", "WPA mode"},
		{"wireless_wpa_on", "wireless-wpa-on", "0=disabled, 1=enabled"},
		{"wireless_wpa_psk", "wireless-wpa-psk", "WPA pre-shared key (8-63 chars)"},
	}
}
