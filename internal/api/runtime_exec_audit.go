package api

import (
	"crypto/sha256"
	"fmt"
	"strings"

	kubeprovider "github.com/LiteyukiStudio/devops/internal/provider/kubernetes"
)

func runtimeExecAuditMessage(command string, container string, result kubeprovider.RuntimeExecResult) string {
	command = strings.TrimSpace(command)
	digest := sha256.Sum256([]byte(command))
	return fmt.Sprintf(
		"pod=%s container=%s exitCode=%d commandBytes=%d commandSha256=%x",
		strings.TrimSpace(result.Pod),
		strings.TrimSpace(container),
		result.ExitCode,
		len([]byte(command)),
		digest,
	)
}
