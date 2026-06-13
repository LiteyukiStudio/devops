package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LiteyukiStudio/devops/internal/id"
	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var errBuildRunNotCancelable = errors.New("build run is not cancelable")

func (h *Handlers) ListBuildProviders(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	projectID := strings.TrimSpace(ctx.Query("projectId"))
	if projectID != "" {
		if _, ok := h.findProjectForCurrentUserByID(ctx, projectID); !ok {
			return
		}
	}

	query := h.db.Order("created_at desc")
	conditions := []string{"scope = 'global'", "(scope = 'user' and owner_ref = ?)"}
	args := []any{user.ID}
	if projectID != "" {
		conditions = append(conditions, "(scope = 'project' and owner_ref = ?)")
		args = append(args, projectID)
	} else if user.Role == "platform_admin" {
		conditions = append(conditions, "scope = 'project'")
	} else {
		projectIDs := h.projectIDsForUser(user.ID)
		if len(projectIDs) > 0 {
			conditions = append(conditions, "(scope = 'project' and owner_ref in ?)")
			args = append(args, projectIDs)
		}
	}
	query = query.Where(strings.Join(conditions, " or "), args...)
	query = applySearch(ctx, query, "name", "type")

	var providers []model.BuildProvider
	if err := query.Find(&providers).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, providers)
}

func (h *Handlers) ListBuilderAgents(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	pagination := paginationFromQuery(ctx)
	var builders []model.BuilderAgent
	query := applySearch(ctx, h.db.Model(&model.BuilderAgent{}), "name", "id", "labels", "scopes", "executor", "status")
	if strings.TrimSpace(ctx.Query("includeOffline")) != "true" {
		query = query.Where("status = ?", "online")
	}
	if err := query.Order(orderByClause(pagination, map[string]string{
		"name":               "name",
		"status":             "status",
		"executor":           "executor",
		"lastHeartbeatAt":    "last_heartbeat_at",
		"currentConcurrency": "current_concurrency",
		"updatedAt":          "updated_at",
	}, "updated_at")).Find(&builders).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	if user.Role == "platform_admin" {
		total := int64(len(builders))
		ctx.JSON(http.StatusOK, paginatedResponse(paginateSlice(builders, pagination), total, pagination))
		return
	}
	projectIDs := h.projectIDsForUser(user.ID)
	visible := make([]model.BuilderAgent, 0, len(builders))
	for _, builder := range builders {
		if builderVisibleToUser(builder.Scopes, user.ID, projectIDs) {
			visible = append(visible, builder)
		}
	}
	total := int64(len(visible))
	ctx.JSON(http.StatusOK, paginatedResponse(paginateSlice(visible, pagination), total, pagination))
}

func (h *Handlers) DeleteBuilderAgent(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	if user.Role != "platform_admin" {
		writeError(ctx, http.StatusForbidden, "只有平台管理员可以删除构建器注册记录")
		return
	}
	var builder model.BuilderAgent
	if err := h.db.First(&builder, "id = ?", ctx.Param("builderId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "builder not found")
		return
	}
	if err := h.db.Delete(&builder).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (h *Handlers) CreateBuildProvider(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	var input buildProviderInput
	if !bindJSON(ctx, &input) {
		return
	}
	provider, ok := h.buildProviderFromInput(ctx, user, input, "")
	if !ok {
		return
	}
	provider.ID = id.New("bldp")
	provider.CreatedBy = user.ID
	if err := h.db.Create(&provider).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusCreated, provider)
}

func (h *Handlers) UpdateBuildProvider(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	var existing model.BuildProvider
	if err := h.db.First(&existing, "id = ?", ctx.Param("providerId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "build provider not found")
		return
	}
	if !h.canManageScopedResource(ctx, user, existing.Scope, existing.OwnerRef, "无权维护该构建提供者") {
		return
	}
	var input buildProviderInput
	if !bindJSON(ctx, &input) {
		return
	}
	next, ok := h.buildProviderFromInput(ctx, user, input, existing.ID)
	if !ok {
		return
	}
	existing.Name = next.Name
	existing.Slug = next.Slug
	existing.Type = next.Type
	existing.Scope = next.Scope
	existing.OwnerRef = next.OwnerRef
	existing.Config = next.Config
	existing.Enabled = next.Enabled
	if err := h.db.Save(&existing).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, existing)
}

func (h *Handlers) DeleteBuildProvider(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	var provider model.BuildProvider
	if err := h.db.First(&provider, "id = ?", ctx.Param("providerId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "build provider not found")
		return
	}
	if !h.canManageScopedResource(ctx, user, provider.Scope, provider.OwnerRef, "无权维护该构建提供者") {
		return
	}
	if err := h.db.Delete(&provider).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (h *Handlers) ListBuildVariableSets(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	projectID := strings.TrimSpace(ctx.Query("projectId"))
	if projectID != "" {
		if _, ok := h.findProjectForCurrentUserByID(ctx, projectID); !ok {
			return
		}
	}

	query := h.db.Order("created_at desc")
	conditions := []string{"scope = 'global'", "(scope = 'user' and owner_ref = ?)"}
	args := []any{user.ID}
	if projectID != "" {
		conditions = append(conditions, "(scope = 'project' and owner_ref = ?)")
		args = append(args, projectID)
	} else if user.Role == "platform_admin" {
		conditions = append(conditions, "scope = 'project'")
	} else {
		projectIDs := h.projectIDsForUser(user.ID)
		if len(projectIDs) > 0 {
			conditions = append(conditions, "(scope = 'project' and owner_ref in ?)")
			args = append(args, projectIDs)
		}
	}
	query = query.Where(strings.Join(conditions, " or "), args...)
	query = applySearch(ctx, query, "name")

	var sets []model.BuildVariableSet
	if err := query.Find(&sets).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, buildVariableSetResponses(sets))
}

func (h *Handlers) CreateBuildVariableSet(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	var input buildVariableSetInput
	if !bindJSON(ctx, &input) {
		return
	}
	setID := id.New("bvs")
	set, ok := h.buildVariableSetFromInput(ctx, user, input, setID, nil)
	if !ok {
		return
	}
	set.CreatedBy = user.ID
	if err := h.db.Create(&set).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusCreated, buildVariableSetResponseFor(set))
}

func (h *Handlers) UpdateBuildVariableSet(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	var existing model.BuildVariableSet
	if err := h.db.First(&existing, "id = ?", ctx.Param("setId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "build variable set not found")
		return
	}
	if !h.canManageScopedResource(ctx, user, existing.Scope, existing.OwnerRef, "无权维护该变量和密钥") {
		return
	}
	var input buildVariableSetInput
	if !bindJSON(ctx, &input) {
		return
	}
	next, ok := h.buildVariableSetFromInput(ctx, user, input, existing.ID, decodeSecretRefs(existing.SecretRefs))
	if !ok {
		return
	}
	existing.Name = next.Name
	existing.Scope = next.Scope
	existing.OwnerRef = next.OwnerRef
	existing.Variables = next.Variables
	existing.SecretRefs = next.SecretRefs
	existing.Enabled = next.Enabled
	if err := h.db.Save(&existing).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, buildVariableSetResponseFor(existing))
}

func (h *Handlers) DeleteBuildVariableSet(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	var set model.BuildVariableSet
	if err := h.db.First(&set, "id = ?", ctx.Param("setId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "build variable set not found")
		return
	}
	if !h.canManageScopedResource(ctx, user, set.Scope, set.OwnerRef, "无权维护该变量和密钥") {
		return
	}
	if err := h.db.Delete(&set).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (h *Handlers) ListBuildRuns(ctx *gin.Context) {
	if _, ok := h.findProjectForCurrentUser(ctx); !ok {
		return
	}
	pagination := paginationFromQuery(ctx)
	query := h.db.Where("project_id = ?", ctx.Param("projectId"))
	if applicationID := strings.TrimSpace(ctx.Query("applicationId")); applicationID != "" {
		query = query.Where("application_id = ?", applicationID)
	}
	if targetID := strings.TrimSpace(ctx.Query("deploymentTargetId")); targetID != "" {
		query = query.Where("deployment_target_id = ?", targetID)
	}
	if status := strings.TrimSpace(ctx.Query("status")); status != "" && buildRunStatusAllowed(status) {
		query = query.Where("status = ?", status)
	}
	if triggerType := strings.TrimSpace(ctx.Query("triggerType")); triggerType != "" && buildRunTriggerAllowed(triggerType) {
		query = query.Where("trigger_type = ?", triggerType)
	}
	if branch := strings.TrimSpace(ctx.Query("sourceBranch")); branch != "" {
		query = query.Where("source_branch = ?", branch)
	}
	if actor := strings.TrimSpace(ctx.Query("createdBy")); actor != "" {
		query = query.Where("created_by = ?", actor)
	}
	query = applySearch(ctx, query, "id", "status", "trigger_type", "source_branch", "source_tag", "source_commit", "target_repository", "image_ref", "created_by")
	var runs []model.BuildRun
	if ctx.Query("page") == "" && ctx.Query("pageSize") == "" {
		if err := query.Order("created_at desc").Find(&runs).Error; err != nil {
			writeError(ctx, http.StatusInternalServerError, err.Error())
			return
		}
		ctx.JSON(http.StatusOK, runs)
		return
	}
	var total int64
	if err := query.Model(&model.BuildRun{}).Count(&total).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	orderBy := orderByClause(pagination, map[string]string{
		"createdAt":    "created_at",
		"status":       "status",
		"sourceCommit": "source_commit",
		"sourceBranch": "source_branch",
		"triggerType":  "trigger_type",
		"createdBy":    "created_by",
	}, "created_at")
	if err := query.Order(orderBy).Limit(pagination.PageSize).Offset(pagination.Offset()).Find(&runs).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, paginatedResponse(runs, total, pagination))
}

func (h *Handlers) GetBuildRun(ctx *gin.Context) {
	if _, ok := h.findProjectForCurrentUser(ctx); !ok {
		return
	}
	run, ok := h.findBuildRun(ctx)
	if !ok {
		return
	}
	ctx.JSON(http.StatusOK, run)
}

func buildRunStatusAllowed(status string) bool {
	switch status {
	case "queued", "running", "succeeded", "failed", "canceled", "lost", "timeout":
		return true
	default:
		return false
	}
}

func buildRunCancelable(status string) bool {
	return status == "queued" || status == "running"
}

func buildRunTerminal(status string) bool {
	return status == "succeeded" || status == "failed" || status == "canceled" || status == "lost" || status == "timeout"
}

func buildRunTriggerAllowed(triggerType string) bool {
	switch triggerType {
	case "manual", "webhook", "push", "tag", "api", "retry":
		return true
	default:
		return false
	}
}

func (h *Handlers) TriggerBuildRun(ctx *gin.Context) {
	user, _, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin", "developer")
	if !ok {
		return
	}
	var input buildRunInput
	if !bindJSON(ctx, &input) {
		return
	}
	run := h.buildRunFromInput(ctx.Param("projectId"), user, input)
	run.ID = id.New("bldr")
	run.Status = "queued"
	run.TriggerType = fallback(strings.TrimSpace(input.TriggerType), "manual")
	h.createQueuedBuildRun(ctx, user, run, strings.TrimSpace(input.TargetImageRef), http.StatusCreated)
}

func (h *Handlers) RetryBuildRun(ctx *gin.Context) {
	user, _, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin", "developer")
	if !ok {
		return
	}
	previous, ok := h.findBuildRun(ctx)
	if !ok {
		return
	}
	run := model.BuildRun{
		ID:                  id.New("bldr"),
		ProjectID:           previous.ProjectID,
		ApplicationID:       previous.ApplicationID,
		DeploymentTargetID:  previous.DeploymentTargetID,
		BuildProviderID:     previous.BuildProviderID,
		BuildVariableSetIDs: previous.BuildVariableSetIDs,
		Status:              "queued",
		TriggerType:         "retry",
		SourceBranch:        previous.SourceBranch,
		SourceTag:           previous.SourceTag,
		SourceCommit:        previous.SourceCommit,
		DockerfilePath:      previous.DockerfilePath,
		BuildContext:        previous.BuildContext,
		BuildDirectory:      previous.BuildDirectory,
		TargetRegistryID:    previous.TargetRegistryID,
		TargetRepository:    previous.TargetRepository,
		TargetTag:           previous.TargetTag,
		CacheConfig:         previous.CacheConfig,
		CreatedBy:           user.ID,
		TriggeredByName:     buildRunActorName(user),
		TriggeredByEmail:    strings.TrimSpace(user.Email),
		SourceAuthorName:    previous.SourceAuthorName,
		SourceAuthorEmail:   previous.SourceAuthorEmail,
	}
	h.createQueuedBuildRun(ctx, user, run, "", http.StatusCreated)
}

func (h *Handlers) CancelBuildRun(ctx *gin.Context) {
	user, _, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin", "developer")
	if !ok {
		return
	}
	run, ok := h.findBuildRun(ctx)
	if !ok {
		return
	}
	if !buildRunCancelable(run.Status) {
		writeError(ctx, http.StatusConflict, "只有排队中或运行中的构建可以终止")
		return
	}

	finishedAt := time.Now()
	var jobs []model.BuildJob
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		var lockedRun model.BuildRun
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lockedRun, "id = ? and project_id = ?", run.ID, run.ProjectID).Error; err != nil {
			return err
		}
		if !buildRunCancelable(lockedRun.Status) {
			return errBuildRunNotCancelable
		}
		if err := tx.Where("build_run_id = ? and project_id = ? and status in ?", lockedRun.ID, lockedRun.ProjectID, []string{"queued", "running"}).Find(&jobs).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.BuildJob{}).
			Where("build_run_id = ? and project_id = ? and status in ?", lockedRun.ID, lockedRun.ProjectID, []string{"queued", "running"}).
			Updates(map[string]any{
				"status":      "canceled",
				"message":     "canceled by user",
				"lease_until": nil,
				"finished_at": &finishedAt,
			}).Error; err != nil {
			return err
		}
		return tx.Model(&model.BuildRun{}).
			Where("id = ?", lockedRun.ID).
			Updates(map[string]any{
				"status":      "canceled",
				"finished_at": &finishedAt,
			}).Error
	}); err != nil {
		if errors.Is(err, errBuildRunNotCancelable) {
			writeError(ctx, http.StatusConflict, "只有排队中或运行中的构建可以终止")
			return
		}
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	h.audit(user.ID, "build_run.cancel", run.ID, true, "")
	if err := h.db.First(&run, "id = ? and project_id = ?", run.ID, run.ProjectID).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, run)
}

func (h *Handlers) DeleteBuildRun(ctx *gin.Context) {
	user, _, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin", "developer")
	if !ok {
		return
	}
	run, ok := h.findBuildRun(ctx)
	if !ok {
		return
	}
	if !buildRunTerminal(run.Status) {
		writeError(ctx, http.StatusConflict, "只有已结束的构建记录可以删除")
		return
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		var hookRuns []model.HookRun
		if err := tx.Where("build_run_id = ? and project_id = ?", run.ID, run.ProjectID).Find(&hookRuns).Error; err != nil {
			return err
		}
		hookRunIDs := make([]string, 0, len(hookRuns))
		for _, hookRun := range hookRuns {
			hookRunIDs = append(hookRunIDs, hookRun.ID)
		}
		if len(hookRunIDs) > 0 {
			if err := tx.Where("hook_run_id in ? and project_id = ?", hookRunIDs, run.ProjectID).Delete(&model.HookRunLog{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id in ? and project_id = ?", hookRunIDs, run.ProjectID).Delete(&model.HookRun{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("build_run_id = ? and project_id = ?", run.ID, run.ProjectID).Delete(&model.BuildLog{}).Error; err != nil {
			return err
		}
		if err := tx.Where("build_run_id = ? and project_id = ?", run.ID, run.ProjectID).Delete(&model.BuildJob{}).Error; err != nil {
			return err
		}
		return tx.Delete(&run).Error
	}); err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	h.audit(user.ID, "build_run.delete", run.ID, true, "")
	ctx.Status(http.StatusNoContent)
}

func (h *Handlers) createQueuedBuildRun(ctx *gin.Context, user model.User, run model.BuildRun, targetImageRef string, statusCode int) {
	if !h.validateBuildRunRequest(ctx, user, &run) {
		return
	}
	_ = targetImageRef
	job := model.BuildJob{
		ID:         id.New("bldj"),
		BuildRunID: run.ID,
		ProjectID:  run.ProjectID,
		Type:       "build",
		Status:     "queued",
	}
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&run).Error; err != nil {
			return err
		}
		return tx.Create(&job).Error
	}); err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(statusCode, run)
}

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
	if strings.TrimSpace(run.BuildProviderID) == "" {
		run.BuildProviderID = strings.TrimSpace(config.BuildProviderID)
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
	if strings.TrimSpace(run.BuildProviderID) != "" {
		var provider model.BuildProvider
		if err := h.db.First(&provider, "id = ? and enabled = ?", run.BuildProviderID, true).Error; err != nil {
			writeError(ctx, http.StatusBadRequest, "构建提供方不可用")
			return false
		}
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

func (h *Handlers) ListBuildJobs(ctx *gin.Context) {
	if _, ok := h.findProjectForCurrentUser(ctx); !ok {
		return
	}
	pagination := paginationFromQuery(ctx)
	query := h.db.Where("project_id = ?", ctx.Param("projectId"))
	if runID := strings.TrimSpace(ctx.Query("buildRunId")); runID != "" {
		query = query.Where("build_run_id = ?", runID)
	}
	var jobs []model.BuildJob
	if ctx.Query("page") == "" && ctx.Query("pageSize") == "" {
		if err := query.Order("created_at desc").Find(&jobs).Error; err != nil {
			writeError(ctx, http.StatusInternalServerError, err.Error())
			return
		}
		ctx.JSON(http.StatusOK, jobs)
		return
	}
	var total int64
	if err := query.Model(&model.BuildJob{}).Count(&total).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	orderBy := orderByClause(pagination, map[string]string{
		"createdAt": "created_at",
		"status":    "status",
		"attempts":  "attempts",
	}, "created_at")
	if err := query.Order(orderBy).Limit(pagination.PageSize).Offset(pagination.Offset()).Find(&jobs).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, paginatedResponse(jobs, total, pagination))
}

func (h *Handlers) GetBuildJob(ctx *gin.Context) {
	if _, ok := h.findProjectForCurrentUser(ctx); !ok {
		return
	}
	var job model.BuildJob
	if err := h.db.First(&job, "id = ? and project_id = ?", ctx.Param("jobId"), ctx.Param("projectId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "build job not found")
		return
	}
	ctx.JSON(http.StatusOK, job)
}

func (h *Handlers) GetBuildJobLogs(ctx *gin.Context) {
	if _, ok := h.findProjectForCurrentUser(ctx); !ok {
		return
	}
	var log model.BuildLog
	if err := h.db.First(&log, "build_job_id = ? and project_id = ?", ctx.Param("jobId"), ctx.Param("projectId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "build log not found")
		return
	}
	ctx.JSON(http.StatusOK, log)
}

func (h *Handlers) StreamBuildJobLogs(ctx *gin.Context) {
	if _, ok := h.findProjectForCurrentUser(ctx); !ok {
		return
	}
	var job model.BuildJob
	if err := h.db.First(&job, "id = ? and project_id = ?", ctx.Param("jobId"), ctx.Param("projectId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "build job not found")
		return
	}
	offset := buildLogStreamOffset(ctx)
	writer := ctx.Writer
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	writer.Header().Set("X-Accel-Buffering", "no")
	writer.WriteHeader(http.StatusOK)
	flushSSE(writer)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		nextOffset, sent, err := h.writeBuildLogStreamChunk(ctx, job, offset)
		if err != nil {
			writeSSE(writer, "error", strconv.Itoa(offset), map[string]string{"code": "build.logs.stream_error"})
			flushSSE(writer)
			return
		}
		offset = nextOffset
		if buildJobTerminal(job.Status) {
			writeSSE(writer, "done", strconv.Itoa(offset), map[string]string{"status": job.Status})
			flushSSE(writer)
			return
		}
		if sent {
			flushSSE(writer)
		}
		select {
		case <-ctx.Request.Context().Done():
			return
		case <-ticker.C:
			if err := h.db.Select("status").First(&job, "id = ? and project_id = ?", job.ID, job.ProjectID).Error; err != nil {
				return
			}
		}
	}
}

func (h *Handlers) writeBuildLogStreamChunk(ctx *gin.Context, job model.BuildJob, offset int) (int, bool, error) {
	var log model.BuildLog
	if err := h.db.First(&log, "build_job_id = ? and project_id = ?", job.ID, job.ProjectID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return offset, false, nil
		}
		return offset, false, err
	}
	content := log.Content
	if offset < 0 || offset > len(content) {
		offset = len(content)
	}
	if len(content) == offset {
		return offset, false, nil
	}
	nextOffset := len(content)
	writeSSE(ctx.Writer, "chunk", strconv.Itoa(nextOffset), map[string]any{
		"content": content[offset:],
		"offset":  nextOffset,
	})
	return nextOffset, true, nil
}

func buildLogStreamOffset(ctx *gin.Context) int {
	value := strings.TrimSpace(ctx.Query("after"))
	if value == "" {
		value = strings.TrimSpace(ctx.GetHeader("Last-Event-ID"))
	}
	offset, err := strconv.Atoi(value)
	if err != nil || offset < 0 {
		return 0
	}
	return offset
}

func buildJobTerminal(status string) bool {
	return status == "succeeded" || status == "failed" || status == "canceled" || status == "lost" || status == "timeout"
}

func writeSSE(writer http.ResponseWriter, event string, idValue string, data any) {
	payload, _ := json.Marshal(data)
	if idValue != "" {
		_, _ = fmt.Fprintf(writer, "id: %s\n", idValue)
	}
	if event != "" {
		_, _ = fmt.Fprintf(writer, "event: %s\n", event)
	}
	_, _ = fmt.Fprintf(writer, "data: %s\n\n", payload)
}

func flushSSE(writer http.ResponseWriter) {
	if flusher, ok := writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (h *Handlers) findBuildRun(ctx *gin.Context) (model.BuildRun, bool) {
	var run model.BuildRun
	if err := h.db.First(&run, "id = ? and project_id = ?", ctx.Param("runId"), ctx.Param("projectId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "build run not found")
		return run, false
	}
	return run, true
}

func (h *Handlers) buildProviderFromInput(ctx *gin.Context, user model.User, input buildProviderInput, providerID string) (model.BuildProvider, bool) {
	scope, ownerRef, ok := h.normalizeScopedOwner(ctx, user, input.Scope, input.OwnerRef, "只有平台管理员可以维护全局构建提供者")
	if !ok {
		return model.BuildProvider{}, false
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		writeError(ctx, http.StatusBadRequest, "请输入构建提供者名称")
		return model.BuildProvider{}, false
	}
	slug := dnsSafeSegment(input.Slug)
	if slug == "" {
		writeError(ctx, http.StatusBadRequest, "请输入构建提供者唯一标识")
		return model.BuildProvider{}, false
	}
	return model.BuildProvider{
		ID:       providerID,
		Slug:     slug,
		Name:     name,
		Type:     normalizeBuildProviderType(input.Type),
		Scope:    scope,
		OwnerRef: ownerRef,
		Config:   strings.TrimSpace(input.Config),
		Enabled:  input.Enabled,
	}, true
}

func (h *Handlers) buildVariableSetFromInput(ctx *gin.Context, user model.User, input buildVariableSetInput, setID string, existingSecretRefs map[string]string) (model.BuildVariableSet, bool) {
	scope, ownerRef, ok := h.normalizeScopedOwner(ctx, user, input.Scope, input.OwnerRef, "只有平台管理员可以维护全局变量和密钥")
	if !ok {
		return model.BuildVariableSet{}, false
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		writeError(ctx, http.StatusBadRequest, "请输入变量和密钥名称")
		return model.BuildVariableSet{}, false
	}
	variables, ok := normalizeBuildVariables(ctx, input.Variables)
	if !ok {
		return model.BuildVariableSet{}, false
	}
	secretRefs, ok := h.buildVariableSecretRefsFromInput(ctx, user, setID, input.Secrets, existingSecretRefs)
	if !ok {
		return model.BuildVariableSet{}, false
	}
	if len(variables) == 0 && len(secretRefs) == 0 {
		writeError(ctx, http.StatusBadRequest, "请至少配置一个构建变量或密钥")
		return model.BuildVariableSet{}, false
	}
	content, err := json.Marshal(variables)
	if err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return model.BuildVariableSet{}, false
	}
	secretContent, err := json.Marshal(secretRefs)
	if err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return model.BuildVariableSet{}, false
	}
	return model.BuildVariableSet{
		ID:         setID,
		Name:       name,
		Scope:      scope,
		OwnerRef:   ownerRef,
		Variables:  string(content),
		SecretRefs: string(secretContent),
		Enabled:    input.Enabled,
	}, true
}

func (h *Handlers) buildVariableSecretRefsFromInput(ctx *gin.Context, user model.User, setID string, input map[string]string, existing map[string]string) (map[string]string, bool) {
	output := make(map[string]string)
	for key, value := range input {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" && value == "" {
			continue
		}
		if !isBuildEnvKey(key) {
			writeError(ctx, http.StatusBadRequest, "构建密钥名只能使用字母、数字和下划线，且不能以数字开头")
			return nil, false
		}
		if value == "" {
			if existingRef := strings.TrimSpace(existing[key]); existingRef != "" {
				output[key] = existingRef
			}
			continue
		}
		if len(value) > 8192 {
			writeError(ctx, http.StatusBadRequest, "构建密钥值过长")
			return nil, false
		}
		output[key] = h.secrets.Store(value, user.ID, "build_variable_set:"+setID+":"+key)
	}
	return output, true
}

func (h *Handlers) buildRunFromInput(projectID string, user model.User, input buildRunInput) model.BuildRun {
	targetRepository, targetTag := splitTargetImageRef(input.TargetImageRef)
	if targetRepository == "" {
		targetRepository = strings.Trim(strings.TrimSpace(input.TargetRepository), "/")
		targetTag = strings.TrimSpace(input.TargetTag)
	}
	return model.BuildRun{
		ProjectID:           projectID,
		ApplicationID:       strings.TrimSpace(input.ApplicationID),
		DeploymentTargetID:  strings.TrimSpace(input.DeploymentTargetID),
		BuildProviderID:     "",
		BuildVariableSetIDs: encodeBuildVariableSetIDs(input.BuildVariableSetIDs),
		SourceBranch:        strings.TrimSpace(input.SourceBranch),
		SourceTag:           strings.TrimSpace(input.SourceTag),
		SourceCommit:        strings.TrimSpace(input.SourceCommit),
		DockerfilePath:      fallback(strings.TrimSpace(input.DockerfilePath), "Dockerfile"),
		BuildContext:        fallback(strings.TrimSpace(input.BuildContext), "."),
		BuildDirectory:      strings.TrimSpace(input.BuildDirectory),
		TargetRegistryID:    strings.TrimSpace(input.TargetRegistryID),
		TargetRepository:    targetRepository,
		TargetTag:           fallback(targetTag, "latest"),
		ImageRef:            "",
		CacheConfig:         strings.TrimSpace(input.CacheConfig),
		CreatedBy:           user.ID,
		TriggeredByName:     buildRunActorName(user),
		TriggeredByEmail:    strings.TrimSpace(user.Email),
	}
}

func buildRunActorName(user model.User) string {
	return fallback(strings.TrimSpace(user.Name), strings.TrimSpace(user.Email))
}

func splitTargetImageRef(value string) (string, string) {
	normalized := strings.Trim(strings.TrimSpace(value), "/")
	if normalized == "" {
		return "", ""
	}
	lastSlash := strings.LastIndex(normalized, "/")
	lastColon := strings.LastIndex(normalized, ":")
	if lastColon > lastSlash {
		repository := strings.Trim(strings.TrimSpace(normalized[:lastColon]), "/")
		tag := strings.TrimSpace(normalized[lastColon+1:])
		return repository, tag
	}
	return normalized, "latest"
}

func normalizeBuildProviderType(value string) string {
	return "platform"
}

func normalizeBuildVariables(ctx *gin.Context, input map[string]string) (map[string]string, bool) {
	output := make(map[string]string)
	for key, value := range input {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" && value == "" {
			continue
		}
		if !isBuildEnvKey(key) {
			writeError(ctx, http.StatusBadRequest, "构建变量名只能使用字母、数字和下划线，且不能以数字开头")
			return nil, false
		}
		if len(value) > 4096 {
			writeError(ctx, http.StatusBadRequest, "构建变量值过长")
			return nil, false
		}
		output[key] = value
	}
	return output, true
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

func encodeBuildVariableSetIDs(ids []string) string {
	normalized := normalizeStringList(ids)
	if len(normalized) == 0 {
		return ""
	}
	content, err := json.Marshal(normalized)
	if err != nil {
		return ""
	}
	return string(content)
}

func buildVariableSetIDs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(raw), &ids); err == nil {
		return normalizeStringList(ids)
	}
	return normalizeStringList(strings.Split(raw, ","))
}

func normalizeBuildSelectorList(values []string) []string {
	seen := map[string]bool{}
	output := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		output = append(output, value)
	}
	return output
}

func builderHasLabels(rawLabels string, requiredLabels []string) bool {
	if len(requiredLabels) == 0 {
		return true
	}
	labels := map[string]bool{}
	for _, label := range normalizeBuildSelectorList(strings.Split(rawLabels, ",")) {
		labels[label] = true
	}
	for _, required := range requiredLabels {
		if !labels[required] {
			return false
		}
	}
	return true
}

func builderAllowsRun(rawScopes string, projectID string, userID string) bool {
	scopes := normalizeBuildSelectorList(strings.Split(rawScopes, ","))
	if len(scopes) == 0 {
		return true
	}
	for _, scope := range scopes {
		switch {
		case scope == "global":
			return true
		case strings.HasPrefix(scope, "project:") && strings.TrimPrefix(scope, "project:") == strings.ToLower(strings.TrimSpace(projectID)):
			return true
		case strings.HasPrefix(scope, "user:") && strings.TrimPrefix(scope, "user:") == strings.ToLower(strings.TrimSpace(userID)):
			return true
		}
	}
	return false
}

func builderVisibleToUser(rawScopes string, userID string, projectIDs []string) bool {
	scopes := normalizeBuildSelectorList(strings.Split(rawScopes, ","))
	if len(scopes) == 0 {
		return true
	}
	userID = strings.ToLower(strings.TrimSpace(userID))
	projectSet := map[string]bool{}
	for _, projectID := range projectIDs {
		projectSet[strings.ToLower(strings.TrimSpace(projectID))] = true
	}
	for _, scope := range scopes {
		switch {
		case scope == "global":
			return true
		case strings.HasPrefix(scope, "user:") && strings.TrimPrefix(scope, "user:") == userID:
			return true
		case strings.HasPrefix(scope, "project:") && projectSet[strings.TrimPrefix(scope, "project:")]:
			return true
		}
	}
	return false
}

func (h *Handlers) buildVariablesForRun(ctx *gin.Context, user model.User, projectID string, setIDs []string) (map[string]string, bool) {
	variables, err := h.buildVariablesForRunByIDs(h.db, user, projectID, setIDs)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return nil, false
	}
	return variables, true
}

func (h *Handlers) buildVariablesForRunByIDs(db *gorm.DB, user model.User, projectID string, setIDs []string) (map[string]string, error) {
	output := make(map[string]string)
	seen := make(map[string]bool)
	var defaultSets []model.BuildVariableSet
	if err := db.Where("scope = ? and owner_ref = ? and enabled = ?", "project", projectID, true).Order("created_at asc").Find(&defaultSets).Error; err != nil {
		return nil, err
	}
	for _, set := range defaultSets {
		if !buildVariableSetAccessible(user, projectID, set) {
			continue
		}
		applyBuildVariableSetValues(output, set, h.secrets.Resolve)
		seen[set.ID] = true
	}
	for _, setID := range normalizeStringList(setIDs) {
		if seen[setID] {
			continue
		}
		seen[setID] = true
		var set model.BuildVariableSet
		if err := db.First(&set, "id = ? and enabled = ?", setID, true).Error; err != nil {
			return nil, errors.New("变量和密钥不可用")
		}
		if !buildVariableSetAccessible(user, projectID, set) {
			return nil, errors.New("无权使用该变量和密钥")
		}
		applyBuildVariableSetValues(output, set, h.secrets.Resolve)
	}
	return output, nil
}

func applyBuildVariableSetValues(output map[string]string, set model.BuildVariableSet, resolveSecret func(string) string) {
	var values map[string]string
	if err := json.Unmarshal([]byte(fallback(set.Variables, "{}")), &values); err == nil {
		for key, value := range values {
			if isBuildEnvKey(key) {
				output[key] = value
			}
		}
	}
	for key, ref := range decodeSecretRefs(set.SecretRefs) {
		if !isBuildEnvKey(key) {
			continue
		}
		if secretValue := resolveSecret(ref); secretValue != "" {
			output[key] = secretValue
		}
	}
}

func decodeSecretRefs(raw string) map[string]string {
	refs := map[string]string{}
	if err := json.Unmarshal([]byte(fallback(raw, "{}")), &refs); err != nil {
		return map[string]string{}
	}
	return refs
}

func buildVariableSetAccessible(user model.User, projectID string, set model.BuildVariableSet) bool {
	switch set.Scope {
	case "global":
		return true
	case "user":
		return set.OwnerRef == user.ID
	case "project":
		return set.OwnerRef == projectID || user.Role == "platform_admin"
	default:
		return false
	}
}

type buildProviderInput struct {
	Name     string `json:"name" binding:"required"`
	Slug     string `json:"slug" binding:"required"`
	Type     string `json:"type"`
	Scope    string `json:"scope"`
	OwnerRef string `json:"ownerRef"`
	Config   string `json:"config"`
	Enabled  bool   `json:"enabled"`
}

type buildVariableSetInput struct {
	Name      string            `json:"name" binding:"required"`
	Scope     string            `json:"scope"`
	OwnerRef  string            `json:"ownerRef"`
	Variables map[string]string `json:"variables"`
	Secrets   map[string]string `json:"secrets"`
	Enabled   bool              `json:"enabled"`
}

type buildVariableSetResponse struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Scope     string          `json:"scope"`
	OwnerRef  string          `json:"ownerRef"`
	Variables string          `json:"variables"`
	Secrets   map[string]bool `json:"secrets"`
	Enabled   bool            `json:"enabled"`
	CreatedBy string          `json:"createdBy"`
	CreatedAt time.Time       `json:"createdAt"`
}

func buildVariableSetResponses(sets []model.BuildVariableSet) []buildVariableSetResponse {
	output := make([]buildVariableSetResponse, 0, len(sets))
	for _, set := range sets {
		output = append(output, buildVariableSetResponseFor(set))
	}
	return output
}

func buildVariableSetResponseFor(set model.BuildVariableSet) buildVariableSetResponse {
	secrets := map[string]bool{}
	for key, ref := range decodeSecretRefs(set.SecretRefs) {
		if isBuildEnvKey(key) && strings.TrimSpace(ref) != "" {
			secrets[key] = true
		}
	}
	return buildVariableSetResponse{
		ID:        set.ID,
		Name:      set.Name,
		Scope:     set.Scope,
		OwnerRef:  set.OwnerRef,
		Variables: set.Variables,
		Secrets:   secrets,
		Enabled:   set.Enabled,
		CreatedBy: set.CreatedBy,
		CreatedAt: set.CreatedAt,
	}
}

type buildRunInput struct {
	ApplicationID       string   `json:"applicationId"`
	DeploymentTargetID  string   `json:"deploymentTargetId"`
	BuildProviderID     string   `json:"buildProviderId"`
	BuildVariableSetIDs []string `json:"buildVariableSetIds"`
	TriggerType         string   `json:"triggerType"`
	SourceBranch        string   `json:"sourceBranch"`
	SourceTag           string   `json:"sourceTag"`
	SourceCommit        string   `json:"sourceCommit"`
	DockerfilePath      string   `json:"dockerfilePath"`
	BuildContext        string   `json:"buildContext"`
	BuildDirectory      string   `json:"buildDirectory"`
	TargetRegistryID    string   `json:"targetRegistryId"`
	TargetImageRef      string   `json:"targetImageRef"`
	TargetRepository    string   `json:"targetRepository"`
	TargetTag           string   `json:"targetTag"`
	ImageRef            string   `json:"imageRef"`
	CacheConfig         string   `json:"cacheConfig"`
}
