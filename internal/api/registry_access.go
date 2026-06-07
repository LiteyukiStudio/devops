package api

import (
	"net/http"
	"strings"

	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/LiteyukiStudio/devops/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) registryForCurrentUser(ctx *gin.Context) (model.User, model.ArtifactRegistry, bool) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return model.User{}, model.ArtifactRegistry{}, false
	}
	registry, ok := h.findAccessibleRegistry(ctx, user, ctx.Param("registryId"))
	return user, registry, ok
}

func (h *Handlers) findAccessibleRegistry(ctx *gin.Context, user model.User, registryID string) (model.ArtifactRegistry, bool) {
	var registry model.ArtifactRegistry
	if err := h.db.First(&registry, "id = ?", strings.TrimSpace(registryID)).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "artifact registry not found")
		return registry, false
	}
	if !h.canUseRegistry(ctx, user, registry) {
		return registry, false
	}
	return registry, true
}

func (h *Handlers) canUseRegistry(ctx *gin.Context, user model.User, registry model.ArtifactRegistry) bool {
	if service.CanUseRegistry(user, registry, h.projects.UserHasProject) {
		return true
	}
	writeError(ctx, http.StatusForbidden, "无权访问该镜像站")
	return false
}

func (h *Handlers) canManageRegistry(ctx *gin.Context, user model.User, registry model.ArtifactRegistry) bool {
	if service.CanManageRegistry(user, registry, func(projectID string, roles ...string) bool {
		_, ok := h.findProjectForCurrentUserWithRolesByID(ctx, projectID, roles...)
		return ok
	}) {
		return true
	}
	writeError(ctx, http.StatusForbidden, "无权维护该镜像站")
	return false
}

func (h *Handlers) canManageRegistryCredentials(ctx *gin.Context, user model.User, registry model.ArtifactRegistry) bool {
	if registry.Scope == "global" {
		return true
	}
	return h.canManageRegistry(ctx, user, registry)
}

func (h *Handlers) canManageRegistryCredential(ctx *gin.Context, user model.User, registry model.ArtifactRegistry, credential model.RegistryCredential) bool {
	if service.CanManageRegistryCredential(user, registry, credential, func(projectID string, roles ...string) bool {
		_, ok := h.findProjectForCurrentUserWithRolesByID(ctx, projectID, roles...)
		return ok
	}) {
		return true
	}
	if credential.AccessScope == "personal" {
		writeError(ctx, http.StatusForbidden, "无权维护该个人凭据")
		return false
	}
	writeError(ctx, http.StatusForbidden, "无权维护该镜像站")
	return false
}

func (h *Handlers) defaultRegistryFor(userID, projectID string) (model.ArtifactRegistry, bool) {
	candidates := []struct {
		scope string
		owner string
	}{
		{scope: "project", owner: projectID},
		{scope: "user", owner: userID},
		{scope: "global", owner: ""},
	}
	for _, candidate := range candidates {
		var registry model.ArtifactRegistry
		err := h.db.First(&registry, "scope = ? and owner_ref = ? and is_default = ?", candidate.scope, candidate.owner, true).Error
		if err == nil {
			return registry, true
		}
	}
	return model.ArtifactRegistry{}, false
}
