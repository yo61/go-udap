package cli

import (
	"bytes"
	"strings"
	"testing"

	"go-udap/udap"
)

func TestFormatParamMapSortsByKey(t *testing.T) {
	var buf bytes.Buffer
	err := formatParamMap(&buf, map[string]string{
		"hostname":    "foo",
		"lan_ip_mode": "1",
		"interface":   "0",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := buf.String()
	want := "hostname=foo\ninterface=0\nlan_ip_mode=1\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatGetSingleValueIsBare(t *testing.T) {
	var buf bytes.Buffer
	err := formatGetResult(&buf, []string{"lan_ip_mode"}, map[string]string{"lan_ip_mode": "1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.String() != "1\n" {
		t.Errorf("got %q, want %q", buf.String(), "1\n")
	}
}

func TestFormatGetMultipleValuesIsKeyEqValue(t *testing.T) {
	var buf bytes.Buffer
	err := formatGetResult(&buf, []string{"lan_ip_mode", "hostname"}, map[string]string{
		"lan_ip_mode": "1",
		"hostname":    "foo",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "lan_ip_mode=1\n") || !strings.Contains(got, "hostname=foo\n") {
		t.Errorf("got %q, want both param=value lines", got)
	}
}

func TestFormatDeviceInfoLines(t *testing.T) {
	var buf bytes.Buffer
	d := &udap.Device{
		MAC:      udap.MustParseMAC("aa:bb:cc:dd:ee:ff"),
		Name:     "Receiver",
		Model:    "Squeezebox Receiver",
		Firmware: "77",
		IP:       "192.168.1.50",
		State:    "wait_slimserver",
	}
	formatDeviceInfo(&buf, d)
	got := buf.String()
	for _, want := range []string{
		"aa:bb:cc:dd:ee:ff", "Receiver", "Squeezebox Receiver",
		"77", "192.168.1.50", "wait_slimserver",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("info output missing %q; got %q", want, got)
		}
	}
}

func TestFormatDeviceInfoOmitsEmptyFields(t *testing.T) {
	var buf bytes.Buffer
	d := &udap.Device{MAC: udap.MustParseMAC("aa:bb:cc:dd:ee:ff"), IP: "0.0.0.0"}
	formatDeviceInfo(&buf, d)
	got := buf.String()
	for _, unexpected := range []string{"Name:", "Model:", "Firmware:", "State:"} {
		if strings.Contains(got, unexpected) {
			t.Errorf("output should omit %q for empty value; got %q", unexpected, got)
		}
	}
}
