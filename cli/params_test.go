package cli

import (
	"testing"

	"go-udap/udap"
)

// TestParamFlagsCoverAllParameters guards against forgetting to surface
// a newly-added udap.Parameter through the CLI. Since paramFlags() is
// derived from udap.Parameters this is structurally guaranteed today,
// but the test stays as a regression guard for any future divergence
// (e.g. if someone introduces a hand-written override list).
func TestParamFlagsCoverAllParameters(t *testing.T) {
	flags := paramFlags()
	if len(flags) != len(udap.Parameters) {
		t.Fatalf("paramFlags length %d != udap.Parameters length %d",
			len(flags), len(udap.Parameters))
	}
	byUDAP := make(map[string]paramFlag, len(flags))
	for _, f := range flags {
		byUDAP[f.udapName] = f
	}
	for _, p := range udap.Parameters {
		if _, ok := byUDAP[p.Name]; !ok {
			t.Errorf("missing flag table entry for parameter %q", p.Name)
		}
	}
}

func TestParamFlagNamesAreHyphenatedLowercase(t *testing.T) {
	for _, f := range paramFlags() {
		for _, ch := range f.flagName {
			if ch == '_' || (ch >= 'A' && ch <= 'Z') {
				t.Errorf("flag name %q must be lowercase-with-hyphens (no underscores or uppercase)", f.flagName)
				break
			}
		}
	}
}

func TestParamFlagsHaveHelpText(t *testing.T) {
	for _, f := range paramFlags() {
		if f.help == "" {
			t.Errorf("flag table entry %q has no help text", f.udapName)
		}
	}
}
