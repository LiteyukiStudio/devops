package kubernetes

import (
	"regexp"
	"strings"
)

var dnsLabelInvalidPattern = regexp.MustCompile(`[^a-z0-9-]+`)

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func dnsLabel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = dnsLabelInvalidPattern.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "liteyuki-resource"
	}
	if len(value) > 63 {
		value = value[:63]
	}
	return strings.Trim(value, "-")
}
