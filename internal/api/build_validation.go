package api

import (
	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func (h *Handlers) validateBuildRunRequest(ctx *gin.Context, user model.User, run *model.BuildRun) bool {
	var project model.Project
	if err := h.db.First(&project, "id = ?", run.ProjectID).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, "项目空间不存在")
		return false
	}
	var app model.Application
	if err := h.db.First(&app, "id = ? and project_id = ?", run.ApplicationID, run.ProjectID).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, "应用不存在")
		return false
	}
	if !applicationCanMutate(app) {
		writeErrorCode(ctx, http.StatusConflict, "application.delete_in_progress", "应用正在删除中，不能触发构建")
		return false
	}
	config, ok := h.deploymentTargetForRun(ctx, app, run.DeploymentTargetID)
	if !ok {
		return false
	}
	run.DeploymentTargetID = config.ID
	if normalizeDeploymentSourceType(config.SourceType) != "repository" {
		writeError(ctx, http.StatusBadRequest, "镜像直部署配置不能触发构建")
		return false
	}
	if strings.TrimSpace(run.BuildVariableSetIDs) == "" {
		run.BuildVariableSetIDs = strings.TrimSpace(config.BuildVariableSetIDs)
	}
	run.DockerfilePath = fallback(strings.TrimSpace(config.DockerfilePath), "Dockerfile")
	run.BuildContext = fallback(strings.TrimSpace(config.BuildContext), ".")
	run.BuildDirectory = strings.TrimSpace(config.BuildDirectory)
	run.BuildLabels = strings.Join(normalizeBuildSelectorList(strings.Split(config.BuildLabels, ",")), ",")
	if strings.TrimSpace(config.RepositoryBindingID) != "" {
		var binding model.RepositoryBinding
		if err := h.db.First(&binding, "id = ? and project_id = ? and application_id = ?", config.RepositoryBindingID, run.ProjectID, run.ApplicationID).Error; err != nil {
			writeError(ctx, http.StatusBadRequest, "部署配置绑定的代码仓库不存在")
			return false
		}
	} else {
		writeError(ctx, http.StatusBadRequest, "部署配置未绑定代码仓库")
		return false
	}
	if strings.TrimSpace(run.TargetRegistryID) == "" {
		run.TargetRegistryID = strings.TrimSpace(config.TargetRegistryID)
	}
	if strings.TrimSpace(run.TargetRegistryID) == "" {
		writeError(ctx, http.StatusBadRequest, "目标镜像站不能为空")
		return false
	}
	var registry model.ArtifactRegistry
	if err := h.db.First(&registry, "id = ?", run.TargetRegistryID).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, "目标镜像站不存在")
		return false
	}
	if strings.TrimSpace(run.TargetRepository) == "" {
		run.TargetRepository = strings.Trim(strings.TrimSpace(config.TargetRepository), "/")
		run.TargetTag = strings.TrimSpace(config.TargetTag)
	}
	if strings.TrimSpace(run.TargetRepository) == "" {
		repository, tag := splitTargetImageRef(buildTargetImageRepository(registry, project, app))
		run.TargetRepository = repository
		run.TargetTag = tag
	}
	run.TargetRepository = strings.Trim(strings.TrimSpace(run.TargetRepository), "/")
	run.TargetTag = fallback(strings.TrimSpace(run.TargetTag), "latest")
	run.ImageRef = fallback(strings.TrimSpace(run.ImageRef), buildImageRef(registry, *run))
	if !h.usableRegistryCredentialExists(user.ID, registry) {
		writeError(ctx, http.StatusBadRequest, "目标镜像站缺少可用推送凭据")
		return false
	}
	if _, ok := h.buildVariablesForRun(ctx, user, run.ProjectID, buildVariableSetIDs(run.BuildVariableSetIDs)); !ok {
		return false
	}
	return true
}

func (h *Handlers) deploymentTargetForRun(ctx *gin.Context, app model.Application, targetID string) (model.DeploymentTarget, bool) {
	var config model.DeploymentTarget
	query := h.db.Where("project_id = ? and application_id = ? and enabled = ?", app.ProjectID, app.ID, true)
	if strings.TrimSpace(targetID) != "" {
		query = query.Where("id = ?", strings.TrimSpace(targetID))
	} else {
		query = query.Order("created_at asc")
	}
	if err := query.First(&config).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, "部署配置不存在或不可用")
		return config, false
	}
	return config, true
}

func (h *Handlers) usableRegistryCredentialExists(userID string, registry model.ArtifactRegistry) bool {
	if strings.TrimSpace(registry.CredentialRef) != "" {
		var count int64
		h.db.Model(&model.RegistryCredential{}).
			Where("registry_id = ? and scope in ?", registry.ID, []string{"push", "push-pull"}).
			Where("id = ? and (access_scope = ? or created_by = ?)", registry.CredentialRef, "registry", userID).
			Count(&count)
		if count > 0 {
			return true
		}
	}
	var count int64
	h.db.Model(&model.RegistryCredential{}).
		Where("registry_id = ? and scope in ?", registry.ID, []string{"push", "push-pull"}).
		Where("(access_scope = ? and created_by = ?) or access_scope = ?", "personal", userID, "registry").
		Count(&count)
	return count > 0
}
