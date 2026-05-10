package main

import (
	"testing"
	"time"

	"go-udap/mocksbr"
)

func TestParseDeviceFlagSlow(t *testing.T) {
	overrides, err := parseDeviceFlags([]string{"idx=1,slow=80ms"}, 1)
	if err != nil {
		t.Fatalf("parseDeviceFlags: %v", err)
	}
	if len(overrides) != 1 {
		t.Fatalf("got %d overrides, want 1", len(overrides))
	}
	if got, want := overrides[0].cfg.Slow, 80*time.Millisecond; got != want {
		t.Errorf("Slow=%v, want %v", got, want)
	}
}

func TestParseDeviceFlagUnreachable(t *testing.T) {
	overrides, err := parseDeviceFlags([]string{"idx=1,unreachable=true"}, 1)
	if err != nil {
		t.Fatalf("parseDeviceFlags: %v", err)
	}
	if !overrides[0].cfg.Unreachable {
		t.Errorf("Unreachable=false, want true")
	}
}

func TestParseDeviceFlagFailOn(t *testing.T) {
	overrides, err := parseDeviceFlags([]string{"idx=1,fail-on=get:set:reset"}, 1)
	if err != nil {
		t.Fatalf("parseDeviceFlags: %v", err)
	}
	got := overrides[0].cfg.FailOn
	want := []mocksbr.Op{mocksbr.OpGet, mocksbr.OpSet, mocksbr.OpReset}
	if len(got) != len(want) {
		t.Fatalf("FailOn=%v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("FailOn[%d]=%q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseDeviceFlagFailOnRejectsUnknownOp(t *testing.T) {
	if _, err := parseDeviceFlags([]string{"idx=1,fail-on=get:bogus"}, 1); err == nil {
		t.Fatalf("expected error for unknown op")
	}
}

func TestParseDeviceFlagMalformed(t *testing.T) {
	overrides, err := parseDeviceFlags([]string{"idx=1,malformed=oversized-count"}, 1)
	if err != nil {
		t.Fatalf("parseDeviceFlags: %v", err)
	}
	if got, want := overrides[0].cfg.Malformed, mocksbr.MalformedOversizedCount; got != want {
		t.Errorf("Malformed=%v, want %v", got, want)
	}
}

func TestParseDeviceFlagMalformedRejectsUnknownMode(t *testing.T) {
	if _, err := parseDeviceFlags([]string{"idx=1,malformed=mystery"}, 1); err == nil {
		t.Fatalf("expected error for unknown malformed mode")
	}
}
