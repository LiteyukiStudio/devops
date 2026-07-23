package resourceidentifier

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	ProjectMinLength     = 2
	ProjectMaxLength     = 22
	ApplicationMinLength = 2
	ApplicationMaxLength = 22
	StageMinLength       = 2
	StageMaxLength       = 12
)

var dnsLabelPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)

func Validate(value string, minLength, maxLength int) error {
	value = strings.TrimSpace(value)
	if len(value) < minLength || len(value) > maxLength {
		return fmt.Errorf("identifier length must be between %d and %d characters", minLength, maxLength)
	}
	if !dnsLabelPattern.MatchString(value) {
		return fmt.Errorf("identifier must contain only lowercase letters, numbers, and hyphens, and must start and end with a letter or number")
	}
	return nil
}

func ProjectID(identifier string) string {
	return "prj_" + strings.TrimSpace(identifier)
}

func ApplicationID(projectIdentifier, applicationIdentifier string) string {
	return "app_" + strings.TrimSpace(projectIdentifier) + "_" + strings.TrimSpace(applicationIdentifier)
}

func DeploymentTargetID(projectIdentifier, applicationIdentifier, stage string) string {
	return "dplt_" + strings.TrimSpace(projectIdentifier) + "_" + strings.TrimSpace(applicationIdentifier) + "_" + strings.TrimSpace(stage)
}

func ProjectNamespace(identifier string) string {
	return "luna-" + strings.TrimSpace(identifier)
}

func DeploymentTargetName(applicationIdentifier, stage string) string {
	return "luna-" + strings.TrimSpace(applicationIdentifier) + "-" + strings.TrimSpace(stage)
}
