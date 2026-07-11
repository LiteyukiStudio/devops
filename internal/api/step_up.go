package api

import (
	"net/http"
	"strconv"
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
	stepUpPurposeMFAManage                = "mfa_manage"
	stepUpPurposeSecuritySettingsUpdate   = "security_settings_update"

	defaultStepUpIdleTimeout     = 10 * time.Minute
	defaultStepUpAbsoluteTimeout = 60 * time.Minute
)

var allowedStepUpPurposes = map[string]struct{}{
	stepUpPurposeRuntimeExec:              {},
	stepUpPurposeRuntimeTerminal:          {},
	stepUpPurposeDataExport:               {},
	stepUpPurposeSecretUpdate:             {},
	stepUpPurposeRegistryCredentialUpdate: {},
	stepUpPurposeKubeconfigUpdate:         {},
	stepUpPurposeAuthProviderUpdate:       {},
	stepUpPurposeUserAdminUpdate:          {},
	stepUpPurposeMFAManage:                {},
	stepUpPurposeSecuritySettingsUpdate:   {},
}

func (h *Handlers) requireStepUp(ctx *gin.Context, user model.User, purpose string) bool {
	if !h.stepUpMFAEnabled() {
		return true
	}
	return h.requireMFAAssertion(ctx, user, purpose)
}

func (h *Handlers) requireMFAAssertion(ctx *gin.Context, user model.User, purpose string) bool {
	purpose = normalizeStepUpPurpose(purpose)
	if purpose == "" {
		h.audit(user.ID, "mfa.step_up_rejected", "unknown", false, "invalid purpose")
		writeErrorCode(ctx, http.StatusBadRequest, "mfa.invalid_purpose", "不支持的二次验证用途")
		return false
	}
	if requestUsesBearerToken(ctx) {
		h.audit(user.ID, "mfa.step_up_required", purpose, false, "personal access tokens cannot satisfy step-up MFA")
		writeErrorCode(ctx, http.StatusForbidden, "mfa.session_required", "二次验证仅支持浏览器会话")
		return false
	}

	session, ok := h.currentSessionFromCookie(ctx)
	if !ok || session.UserID != user.ID {
		h.audit(user.ID, "mfa.step_up_required", purpose, false, "missing browser session")
		writeMFARequired(ctx, purpose)
		return false
	}

	now := time.Now()
	h.cleanupExpiredStepUpAssertions(now)
	var assertion model.StepUpAssertion
	err := h.db.First(
		&assertion,
		"user_id = ? and session_id = ? and purpose = ? and idle_expires_at > ? and absolute_expires_at > ?",
		user.ID,
		session.ID,
		purpose,
		now,
		now,
	).Error
	if err != nil || !stepUpAssertionActive(assertion, now) {
		h.audit(user.ID, "mfa.step_up_required", purpose, false, "assertion missing or expired")
		writeMFARequired(ctx, purpose)
		return false
	}

	idleTimeout, _ := h.stepUpTimeouts()
	idleExpiresAt := refreshedStepUpIdleExpiry(now, idleTimeout, assertion.AbsoluteExpiresAt)
	result := h.db.Model(&model.StepUpAssertion{}).
		Where("id = ? and idle_expires_at > ? and absolute_expires_at > ?", assertion.ID, now, now).
		Updates(map[string]any{"last_activity_at": now, "idle_expires_at": idleExpiresAt, "updated_at": now})
	if result.Error != nil || result.RowsAffected != 1 {
		h.audit(user.ID, "mfa.step_up_required", purpose, false, "assertion refresh failed")
		writeMFARequired(ctx, purpose)
		return false
	}
	return true
}

func (h *Handlers) cleanupExpiredStepUpAssertions(now time.Time) {
	_ = h.db.Where("idle_expires_at <= ? or absolute_expires_at <= ?", now, now).Delete(&model.StepUpAssertion{}).Error
}

func (h *Handlers) stepUpMFAEnabled() bool {
	return configBool(h.configValue("security.stepUpMfa.enabled"))
}

func (h *Handlers) stepUpTimeouts() (time.Duration, time.Duration) {
	idle := configMinutes(h.configValue("security.stepUpMfa.idleTimeoutMinutes"), defaultStepUpIdleTimeout)
	absolute := configMinutes(h.configValue("security.stepUpMfa.absoluteTimeoutMinutes"), defaultStepUpAbsoluteTimeout)
	if idle > absolute {
		idle = absolute
	}
	return idle, absolute
}

func configMinutes(value string, fallback time.Duration) time.Duration {
	minutes, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || minutes <= 0 {
		return fallback
	}
	return time.Duration(minutes) * time.Minute
}

func configBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes", "on", "enabled":
		return true
	default:
		return false
	}
}

func normalizeStepUpPurpose(purpose string) string {
	purpose = strings.ToLower(strings.TrimSpace(purpose))
	if _, ok := allowedStepUpPurposes[purpose]; !ok {
		return ""
	}
	return purpose
}

func requestUsesBearerToken(ctx *gin.Context) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(ctx.GetHeader("Authorization"))), "bearer ")
}

func writeMFARequired(ctx *gin.Context, purpose string) {
	ctx.JSON(http.StatusForbidden, gin.H{
		"code":    "mfa_required",
		"error":   "需要完成敏感操作二次验证",
		"purpose": purpose,
	})
}

func stepUpAssertionActive(assertion model.StepUpAssertion, now time.Time) bool {
	return assertion.ID != "" && assertion.IdleExpiresAt.After(now) && assertion.AbsoluteExpiresAt.After(now)
}

func refreshedStepUpIdleExpiry(now time.Time, idleTimeout time.Duration, absoluteExpiresAt time.Time) time.Time {
	refreshed := now.Add(idleTimeout)
	if refreshed.After(absoluteExpiresAt) {
		return absoluteExpiresAt
	}
	return refreshed
}
