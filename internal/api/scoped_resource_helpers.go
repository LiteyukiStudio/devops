package api

import (
	"net/http"
	"strings"

	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) normalizeScopedOwner(ctx *gin.Context, user model.User, rawScope, rawOwnerRef, globalError string) (string, string, bool) {
	scope := normalizeOwnerScope(rawScope)
	ownerRef := strings.TrimSpace(rawOwnerRef)
	switch scope {
	case "global":
		if user.Role != "platform_admin" {
			writeError(ctx, http.StatusForbidden, globalError)
			return "", "", false
		}
		ownerRef = ""
	case "project":
		if ownerRef == "" {
			writeError(ctx, http.StatusBadRequest, "请选择项目空间")
			return "", "", false
		}
		if _, ok := h.findProjectForCurrentUserWithRolesByID(ctx, ownerRef, "owner", "admin"); !ok {
			return "", "", false
		}
	case "user":
		ownerRef = user.ID
	}
	return scope, ownerRef, true
}

func (h *Handlers) canManageScopedResource(ctx *gin.Context, user model.User, scope, ownerRef, errorMessage string) bool {
	switch normalizeOwnerScope(scope) {
	case "global":
		if user.Role == "platform_admin" {
			return true
		}
	case "user":
		if ownerRef == user.ID {
			return true
		}
	case "project":
		if user.Role == "platform_admin" {
			return true
		}
		if _, ok := h.findProjectForCurrentUserWithRolesByID(ctx, ownerRef, "owner", "admin"); ok {
			return true
		}
	}
	writeError(ctx, http.StatusForbidden, errorMessage)
	return false
}

func (h *Handlers) canInspectScopedResourceConfig(user model.User, scope, ownerRef string) bool {
	switch normalizeOwnerScope(scope) {
	case "global":
		return user.Role == "platform_admin"
	case "user":
		return ownerRef == user.ID
	case "project":
		if user.Role == "platform_admin" {
			return true
		}
		var member model.ProjectMember
		err := h.db.First(&member, "project_id = ? and user_id = ? and role in ?", ownerRef, user.ID, []string{"owner", "admin"}).Error
		return err == nil
	default:
		return false
	}
}

func normalizeOwnerScope(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "project", "user":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "global"
	}
}
