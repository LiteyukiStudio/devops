package api

import (
	"regexp"
	"strings"

	"github.com/LiteyukiStudio/devops/internal/model"
)

var apiDNSLabelInvalidPattern = regexp.MustCompile(`[^a-z0-9-]+`)

func runtimeProjectNamespace(project model.Project) string {
	return runtimeIDResourceName("ns", project.ID)
}

func deploymentTargetResourceName(target model.DeploymentTarget) string {
	return runtimeIDResourceName("dplt", target.ID)
}

func shortResourceID(value string) string {
	return runtimeShortID(value)
}

func runtimeIDResourceName(prefix string, value string) string {
	suffix := runtimeShortID(value)
	if suffix == "" {
		return runtimeDNSLabel(prefix)
	}
	return runtimeDNSLabel(prefix + "-" + suffix)
}

func runtimeShortID(value string) string {
	value = strings.TrimSpace(value)
	if index := strings.Index(value, "_"); index >= 0 {
		value = value[index+1:]
	}
	value = runtimeDNSLabel(value)
	if len(value) > 10 {
		return value[:10]
	}
	return value
}

func runtimeDNSLabel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = apiDNSLabelInvalidPattern.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "app"
	}
	if len(value) > 63 {
		value = strings.Trim(value[:63], "-")
	}
	if value == "" {
		return "app"
	}
	return value
}
