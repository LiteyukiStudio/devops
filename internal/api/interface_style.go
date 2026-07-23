package api

import "strings"

const siteMinimalModeDefaultKey = "site.minimalModeDefault"

var userInterfaceStyleOptions = []string{"minimal", "themed"}

func normalizeUserInterfaceStyle(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", true
	}
	return value, configOptionAllowed(value, userInterfaceStyleOptions)
}
