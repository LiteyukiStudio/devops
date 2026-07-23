package api

import "strings"

const (
	siteBrandColorPresetKey   = "site.brandColorPreset"
	defaultBrandColorPreset   = "blue"
	brandThemeHTMLPlaceholder = "__LUNA_DEVOPS_BRAND_THEME__"
)

// Curated multi-color themes are followed by Radix Colors 3.0.0 single-color
// scales. The backend persists only these stable IDs.
var brandColorPresetOptions = []string{
	"aurora",
	"harbor",
	"sunset",
	"botanical",
	"meadow",
	"citrus",
	"gold",
	"bronze",
	"brown",
	"yellow",
	"amber",
	"orange",
	"tomato",
	"red",
	"ruby",
	"crimson",
	"pink",
	"plum",
	"purple",
	"violet",
	"iris",
	"indigo",
	"blue",
	"cyan",
	"teal",
	"jade",
	"green",
	"grass",
	"lime",
	"mint",
	"sky",
}

func normalizeBrandColorPreset(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if configOptionAllowed(value, brandColorPresetOptions) {
		return value
	}
	return defaultBrandColorPreset
}

func normalizeUserBrandColorPreset(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", true
	}
	return value, configOptionAllowed(value, brandColorPresetOptions)
}
