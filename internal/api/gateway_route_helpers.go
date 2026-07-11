package api

import (
	"regexp"
	"strings"

	"github.com/LiteyukiStudio/devops/internal/model"
)

var gatewayHostSegmentPattern = regexp.MustCompile(`[^a-z0-9-]+`)

func gatewayHostSegment(value string) string {
	segment := strings.Trim(strings.ToLower(strings.TrimSpace(value)), "-")
	segment = gatewayHostSegmentPattern.ReplaceAllString(segment, "-")
	segment = strings.Join(strings.FieldsFunc(segment, func(char rune) bool { return char == '-' }), "-")
	return strings.Trim(segment, "-")
}

func apiProjectNamespace(projectID string) string {
	return apiIDResourceName("ns", projectID)
}

func apiApplicationResourceName(target model.DeploymentTarget) string {
	return apiIDResourceName("dplt", target.ID)
}

func apiIDResourceName(prefix string, value string) string {
	suffix := apiShortID(value)
	if suffix == "" {
		return gatewayHostSegment(prefix)
	}
	return gatewayHostSegment(prefix + "-" + suffix)
}

func apiShortID(value string) string {
	value = strings.TrimSpace(value)
	if index := strings.Index(value, "_"); index >= 0 {
		value = value[index+1:]
	}
	value = gatewayHostSegment(value)
	if len(value) > 10 {
		return value[:10]
	}
	return value
}

func (h *Handlers) configValue(key string) string {
	values := h.configs.get([]string{key})
	return values[key]
}
