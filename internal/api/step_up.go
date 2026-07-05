package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/gin-gonic/gin"
)

const (
	stepUpPurposeRuntimeExec              = "runtime_exec"
	stepUpPurposeRuntimeTerminal          = "runtime_terminal"
	stepUpPurposeDataExport               = "data_export"
	stepUpPurposeSecretUpdate             = "secret_update"
	stepUpPurposeRegistryCredentialUpdate = "registry_credential_update"
	stepUpPurposeKubeconfigUpdate         = "kubeconfig_update"
	stepUpPurposeAuthProviderUpdate       = "auth_provider_update"
	stepUpPurposeUserAdminUpdate          = "user_admin_update"
	stepUpAssertionTTL                    = 10 * time.Minute
)

func (h *Handlers) requireStepUp(ctx *gin.Context, user model.User, purpose string) bool {
	purpose = strings.TrimSpace(purpose)
	if purpose == "" || !h.stepUpMFAEnabled() {
		return true
	}
	session, ok := h.currentSessionFromCookie(ctx)
	if !ok || session.UserID != user.ID {
		h.audit(user.ID, "mfa.step_up_required", purpose, false, "missing session assertion")
		writeErrorCode(ctx, http.StatusForbidden, "mfa_required", "需要完成敏感操作二次验证")
		return false
	}
	_ = h.db.Where("expires_at <= ?", time.Now()).Delete(&model.StepUpAssertion{}).Error
	var assertion model.StepUpAssertion
	err := h.db.First(&assertion, "user_id = ? and session_id = ? and purpose = ? and expires_at > ?", user.ID, session.ID, purpose, time.Now()).Error
	if err == nil {
		return true
	}
	h.audit(user.ID, "mfa.step_up_required", purpose, false, "assertion missing or expired")
	writeErrorCode(ctx, http.StatusForbidden, "mfa_required", "需要完成敏感操作二次验证")
	return false
}

func (h *Handlers) stepUpMFAEnabled() bool {
	return configBool(h.configValue("security.stepUpMfa.enabled"))
}

func configBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes", "on", "enabled":
		return true
	default:
		return false
	}
}
