package cli

import (
	"testing"

	"go-udap/udap"
)

func TestParamFlagsCoverAllKnownParameters(t *testing.T) {
	flags := paramFlags()
	byUDAP := make(map[string]paramFlag, len(flags))
	for _, f := range flags {
		byUDAP[f.udapName] = f
	}
	for _, name := range udap.KnownParameters {
		if _, ok := byUDAP[name]; !ok {
			t.Errorf("missing flag table entry for known parameter %q", name)
		}
	}
}

func TestParamFlagsAllReferenceConfigSettings(t *testing.T) {
	for _, f := range paramFlags() {
		if _, ok := udap.ConfigSettings[f.udapName]; !ok {
			t.Errorf("flag table entry %q does not match any ConfigSettings key", f.udapName)
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
