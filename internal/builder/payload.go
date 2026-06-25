package builder

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed executor/run.sh
var executorRunScript string

func ExecutorScript() string {
	return executorRunScript
}

func HookIDsByPhase(hooks []HookPayload) map[string][]string {
	output := make(map[string][]string, len(buildHookPhases))
	for _, phase := range buildHookPhases {
		output[phase] = []string{}
	}
	for _, hook := range hooks {
		hookID := strings.TrimSpace(hook.ID)
		phase := strings.TrimSpace(hook.Phase)
		if hookID == "" || strings.TrimSpace(hook.Script) == "" || !isBuildHookPhase(phase) {
			continue
		}
		output[phase] = append(output[phase], hookID)
	}
	return output
}

func isBuildHookPhase(phase string) bool {
	for _, item := range buildHookPhases {
		if phase == item {
			return true
		}
	}
	return false
}

func IsBuildHookPhase(phase string) bool {
	return isBuildHookPhase(phase)
}

func HookMetadataEnv(hook HookPayload) string {
	shell := strings.TrimSpace(hook.Shell)
	if shell != "bash" {
		shell = "sh"
	}
	timeoutSeconds := hook.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 300
	}
	return fmt.Sprintf("HOOK_NAME=%s\nHOOK_SHELL=%s\nHOOK_TIMEOUT_SECONDS=%d\nHOOK_FAILURE_POLICY=%s\n",
		shellQuote(hook.Name),
		shellQuote(shell),
		timeoutSeconds,
		shellQuote(hook.FailurePolicy),
	)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func BoolEnvValue(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func StringDefault(value string, defaultValue string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return defaultValue
}

func NormalizedBuildEnv(values map[string]string) map[string]string {
	output := make(map[string]string)
	for key, value := range values {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if isBuildEnvKey(key) && !strings.HasPrefix(key, "LITEYUKI_") {
			output[key] = value
		}
	}
	return output
}

func isBuildEnvKey(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for index, char := range value {
		if index == 0 {
			if char == '_' || char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' {
				continue
			}
			return false
		}
		if char == '_' || char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' || char >= '0' && char <= '9' {
			continue
		}
		return false
	}
	return true
}
