package cli

import "testing"

func TestNormalizeMAC(t *testing.T) {
	cases := []struct {
		in, want string
		ok       bool
	}{
		{"AA:BB:CC:DD:EE:FF", "aa:bb:cc:dd:ee:ff", true},
		{"aa:bb:cc:dd:ee:ff", "aa:bb:cc:dd:ee:ff", true},
		{"aa-bb-cc-dd-ee-ff", "aa:bb:cc:dd:ee:ff", true},
		{"aabbccddeeff", "aa:bb:cc:dd:ee:ff", true},
		{"not a mac", "", false},
		{"aa:bb:cc:dd:ee", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, err := normalizeMAC(c.in)
		if (err == nil) != c.ok {
			t.Errorf("normalizeMAC(%q): ok=%v, err=%v", c.in, c.ok, err)
			continue
		}
		if got != c.want {
			t.Errorf("normalizeMAC(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
