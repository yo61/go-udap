package udap

import "testing"

func TestValidateParameterAcceptsKnownValid(t *testing.T) {
	if err := ValidateParameter("lan_ip_mode", "1"); err != nil {
		t.Fatalf("expected nil error for lan_ip_mode=1, got %v", err)
	}
}

func TestValidateParameterRejectsBadIP(t *testing.T) {
	if err := ValidateParameter("lan_network_address", "not.an.ip"); err == nil {
		t.Fatalf("expected error for invalid IP, got nil")
	}
}

func TestValidateParameterAcceptsUnknownParameter(t *testing.T) {
	// Matches the existing internal behavior: unknown params are not
	// rejected by validation; rejection happens at the CLI boundary.
	if err := ValidateParameter("not_a_real_param", "x"); err != nil {
		t.Fatalf("expected nil error for unknown param, got %v", err)
	}
}

func TestIsValidMAC(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"00:04:20:16:05:8f", true},
		{"AA:BB:CC:DD:EE:FF", true},
		{"aa:bb:CC:dd:EE:ff", true}, // mixed case OK
		{"", false},
		{"00:04:20:16:05:8", false},   // too short
		{"00:04:20:16:05:8ff", false}, // too long
		{"00-04-20-16-05-8f", false},  // hyphen separator
		{"00:04:20:16:058f", false},   // missing colon
		{"00:04:20:16:05:gg", false},  // non-hex
		{"00:04:20:16:05:8z", false},  // non-hex
		{":0:04:20:16:05:8f", false},  // leading colon
		{"00:04:20:16:05:8f:", false}, // trailing colon
	}
	for _, c := range cases {
		if got := isValidMAC(c.in); got != c.want {
			t.Errorf("isValidMAC(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
