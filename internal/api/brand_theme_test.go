package api

import "testing"

func TestBrandColorPresetOptionsMatchOfficialRadixBrandScales(t *testing.T) {
	want := []string{
		"gold", "bronze", "brown", "yellow", "amber", "orange", "tomato", "red", "ruby", "crimson",
		"pink", "plum", "purple", "violet", "iris", "indigo", "blue", "cyan", "teal", "jade",
		"green", "grass", "lime", "mint", "sky",
	}
	if len(brandColorPresetOptions) != len(want) {
		t.Fatalf("brand preset count = %d, want %d", len(brandColorPresetOptions), len(want))
	}
	for index := range want {
		if brandColorPresetOptions[index] != want[index] {
			t.Fatalf("brand preset %d = %q, want %q", index, brandColorPresetOptions[index], want[index])
		}
	}
}

func TestNormalizeBrandColorPresetFallsBackToBlue(t *testing.T) {
	if got := normalizeBrandColorPreset(" Teal "); got != "teal" {
		t.Fatalf("normalized preset = %q, want teal", got)
	}
	if got := normalizeBrandColorPreset("custom-css"); got != defaultBrandColorPreset {
		t.Fatalf("invalid preset = %q, want %q", got, defaultBrandColorPreset)
	}
}

func TestNormalizeUserBrandColorPresetAllowsFollowingPlatform(t *testing.T) {
	if got, valid := normalizeUserBrandColorPreset("  "); !valid || got != "" {
		t.Fatalf("empty user preset = %q, valid=%v; want empty and valid", got, valid)
	}
	if got, valid := normalizeUserBrandColorPreset(" Ruby "); !valid || got != "ruby" {
		t.Fatalf("official user preset = %q, valid=%v; want ruby and valid", got, valid)
	}
	if got, valid := normalizeUserBrandColorPreset("custom-css"); valid || got != "custom-css" {
		t.Fatalf("custom user preset = %q, valid=%v; want invalid", got, valid)
	}
}

func TestValidateConfigValuesRejectsUnknownBrandColorPreset(t *testing.T) {
	if _, err := validateConfigValues(map[string]any{siteBrandColorPresetKey: "custom-css"}); err == nil {
		t.Fatal("expected unknown brand color preset to be rejected")
	}
	values, err := validateConfigValues(map[string]any{siteBrandColorPresetKey: "ruby"})
	if err != nil {
		t.Fatalf("validate official brand color preset: %v", err)
	}
	if values[siteBrandColorPresetKey] != "ruby" {
		t.Fatalf("validated preset = %q, want ruby", values[siteBrandColorPresetKey])
	}
}
