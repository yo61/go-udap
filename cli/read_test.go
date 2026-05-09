package cli

import "testing"

// TestFilterReadOutputDropsFactoryDefaults uses the exact 26-param
// dump captured from a freshly-reset Squeezebox Receiver (every value
// is the factory default). All entries should be filtered.
func TestFilterReadOutputDropsFactoryDefaults(t *testing.T) {
	in := map[string]string{
		"bridging":            "0",
		"hostname":            "",
		"interface":           "128",
		"lan_gateway":         "0.0.0.0",
		"lan_ip_mode":         "1",
		"lan_network_address": "0.0.0.0",
		"lan_subnet_mask":     "255.255.255.0",
		"lms_address":         "0.0.0.0",
		"primary_dns":         "0.0.0.0",
		"secondary_dns":       "0.0.0.0",
		"server_address":      "0.0.0.0",
		"squeezecenter_name":  "",
		"wireless_SSID":       "",
		"wireless_channel":    "6",
		"wireless_keylen":     "0",
		"wireless_mode":       "0",
		"wireless_region_id":  "4",
		"wireless_wep_key":    "",
		"wireless_wep_key_1":  "",
		"wireless_wep_key_2":  "",
		"wireless_wep_key_3":  "",
		"wireless_wep_on":     "0",
		"wireless_wpa_cipher": "3",
		"wireless_wpa_mode":   "1",
		"wireless_wpa_on":     "0",
		"wireless_wpa_psk":    "",
	}
	out := filterReadOutput(in)
	if len(out) != 0 {
		t.Errorf("freshly-reset device should produce empty filtered output, got %d entries: %v", len(out), out)
	}
}

// TestFilterReadOutputKeepsNonDefaults verifies that values diverging
// from the factory default survive the filter.
func TestFilterReadOutputKeepsNonDefaults(t *testing.T) {
	in := map[string]string{
		"hostname":            "my-receiver", // changed from ""
		"lan_ip_mode":         "1",           // factory default — drop
		"lan_network_address": "192.168.1.50",
		"interface":           "1", // changed from 128
		"squeezecenter_name":  "",  // factory default — drop
	}
	out := filterReadOutput(in)
	for _, want := range []string{"hostname", "lan_network_address", "interface"} {
		if _, ok := out[want]; !ok {
			t.Errorf("non-default param %q dropped", want)
		}
	}
	for _, gone := range []string{"lan_ip_mode", "squeezecenter_name"} {
		if _, present := out[gone]; present {
			t.Errorf("factory default %q should have been filtered", gone)
		}
	}
	if got, want := len(out), 3; got != want {
		t.Errorf("len(out): got %d, want %d", got, want)
	}
}

// TestFilterReadOutputDropsUnknownOffsets confirms the offset_NNN
// synthetic keys are still filtered (regression guard for the prior
// filterUnknownOffsets behavior).
func TestFilterReadOutputDropsUnknownOffsets(t *testing.T) {
	in := map[string]string{
		"hostname":   "my-receiver",
		"offset_999": "deadbeef",
		"offset_42":  "00",
	}
	out := filterReadOutput(in)
	if _, ok := out["hostname"]; !ok {
		t.Error("hostname dropped")
	}
	for _, gone := range []string{"offset_999", "offset_42"} {
		if _, present := out[gone]; present {
			t.Errorf("synthetic key %q should have been filtered", gone)
		}
	}
}
