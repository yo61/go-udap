package output

import (
	"testing"
)

func TestFormatSetAcceptsEachFormatName(t *testing.T) {
	for _, name := range []string{"text", "json", "csv"} {
		var f Format
		if err := f.Set(name); err != nil {
			t.Fatalf("Set(%q): %v", name, err)
		}
		if f.String() != name {
			t.Errorf("after Set(%q), String() = %q", name, f.String())
		}
	}
}

func TestFormatDefaultsToText(t *testing.T) {
	var f Format
	if f.String() != "text" {
		t.Errorf("zero Format = %q, want text", f.String())
	}
}

func TestFormatSetRejectsOtherValues(t *testing.T) {
	for _, value := range []string{"JSON", "Text", "yaml", ""} {
		var f Format
		err := f.Set(value)
		if err == nil {
			t.Fatalf("Set(%q) accepted", value)
		}
		if err.Error() != "want text, json, csv" {
			t.Errorf("Set(%q) error = %q, want %q", value, err, "want text, json, csv")
		}
	}
}
