package cli

import (
	"strings"
	"testing"
)

func TestParseINIBasic(t *testing.T) {
	in := strings.NewReader("lan_ip_mode=1\nwireless_SSID=MyNet\n")
	got, err := ParseINI(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["lan_ip_mode"] != "1" {
		t.Errorf("lan_ip_mode: want 1, got %q", got["lan_ip_mode"])
	}
	if got["wireless_SSID"] != "MyNet" {
		t.Errorf("wireless_SSID: want MyNet, got %q", got["wireless_SSID"])
	}
}

func TestParseINICommentsAndBlanks(t *testing.T) {
	in := strings.NewReader("# hash comment\n; semicolon comment\n\n  lan_ip_mode = 1  \n")
	got, err := ParseINI(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["lan_ip_mode"] != "1" {
		t.Errorf("expected trimmed value 1, got %q", got["lan_ip_mode"])
	}
	if len(got) != 1 {
		t.Errorf("expected one entry, got %d", len(got))
	}
}

func TestParseINIRejectsMalformedLine(t *testing.T) {
	in := strings.NewReader("lan_ip_mode\n")
	_, err := ParseINI(in)
	if err == nil {
		t.Fatalf("expected error for line without =")
	}
}

func TestParseINIRejectsUnknownParameter(t *testing.T) {
	in := strings.NewReader("not_a_real_param=x\n")
	_, err := ParseINI(in)
	if err == nil {
		t.Fatalf("expected error for unknown parameter name")
	}
}

func TestParseINIRejectsInvalidValue(t *testing.T) {
	in := strings.NewReader("lan_network_address=not.an.ip\n")
	_, err := ParseINI(in)
	if err == nil {
		t.Fatalf("expected error for invalid IP value")
	}
}

func TestParseINIEmptyValueAllowed(t *testing.T) {
	// hostname is a string param with max length 33; empty is valid.
	in := strings.NewReader("hostname=\n")
	got, err := ParseINI(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["hostname"] != "" {
		t.Errorf("expected empty string, got %q", got["hostname"])
	}
}
