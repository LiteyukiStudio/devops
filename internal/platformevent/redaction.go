package platformevent

import "regexp"

const redactedValue = "[REDACTED]"

var sensitiveTextPatterns = []struct {
	regex       *regexp.Regexp
	replacement string
}{
	{regexp.MustCompile(`(?i)(authorization["']?\s*[:=]\s*["']?(?:bearer|basic)\s+)[^\s"',}]+`), "${1}" + redactedValue},
	{regexp.MustCompile(`(?i)(x-access-token:)[^@\s"']+(@)`), "${1}" + redactedValue + "${2}"},
	{regexp.MustCompile(`(?i)((?:password|token|secret|access_token|refresh_token)["']?\s*[:=]\s*["']?)[^\s&"',}]+`), "${1}" + redactedValue},
}

func redactSensitiveText(value string) string {
	result := value
	for _, pattern := range sensitiveTextPatterns {
		result = pattern.regex.ReplaceAllString(result, pattern.replacement)
	}
	return result
}
