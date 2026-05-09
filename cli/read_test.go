package cli

import "testing"

func TestFilterUnknownOffsetsDropsSyntheticKeys(t *testing.T) {
	in := map[string]string{
		"hostname":      "myhost",
		"offset_999":    "deadbeef",
		"lan_ip_mode":   "1",
		"offset_42":     "00",
		"wireless_SSID": "MyNet",
	}
	out := filterUnknownOffsets(in)

	for _, want := range []string{"hostname", "lan_ip_mode", "wireless_SSID"} {
		if _, ok := out[want]; !ok {
			t.Errorf("known param %q dropped", want)
		}
	}
	for _, gone := range []string{"offset_999", "offset_42"} {
		if _, present := out[gone]; present {
			t.Errorf("synthetic key %q should have been filtered", gone)
		}
	}
	if got, want := len(out), 3; got != want {
		t.Errorf("len(out): got %d, want %d", got, want)
	}
}

func TestFilterUnknownOffsetsEmptyMap(t *testing.T) {
	out := filterUnknownOffsets(map[string]string{})
	if len(out) != 0 {
		t.Errorf("empty in → empty out, got %v", out)
	}
}
