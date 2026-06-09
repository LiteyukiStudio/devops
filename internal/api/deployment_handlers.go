package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/LiteyukiStudio/devops/internal/id"
	"github.com/LiteyukiStudio/devops/internal/model"
	kubeprovider "github.com/LiteyukiStudio/devops/internal/provider/kubernetes"
	"github.com/LiteyukiStudio/devops/internal/tasks"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func (h *Handlers) ListRuntimeClusters(ctx *gin.Context) {
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

	var clusters []model.RuntimeCluster
	query := h.db.Order("is_default desc, created_at desc")
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
	if err := applySearch(ctx, query, "name", "endpoint").Find(&clusters).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	for index := range clusters {
		clusters[index] = h.runtimeClusterResponseForUser(user, clusters[index])
	}
	ctx.JSON(http.StatusOK, clusters)
}

func (h *Handlers) CreateRuntimeCluster(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	var input runtimeClusterInput
	if !bindJSON(ctx, &input) {
		return
	}
	clusterID := id.New("clu")
	cluster, ok := h.runtimeClusterFromInput(ctx, user, input, clusterID)
	if !ok {
		return
	}
	if err := h.saveRuntimeClusterWithDefault(cluster); err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusCreated, h.runtimeClusterResponseForUser(user, cluster))
}

func (h *Handlers) UpdateRuntimeCluster(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	var existing model.RuntimeCluster
	if err := h.db.First(&existing, "id = ?", ctx.Param("clusterId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "runtime cluster not found")
		return
	}
	if !h.canManageScopedResource(ctx, user, existing.Scope, existing.OwnerRef, "无权维护该运行集群") {
		return
	}
	var input runtimeClusterInput
	if !bindJSON(ctx, &input) {
		return
	}
	if strings.TrimSpace(input.Kubeconfig) != "" && !h.canInspectRuntimeClusterKubeconfig(user, existing) {
		writeError(ctx, http.StatusForbidden, "只有创建者或平台管理员可以编辑 kubeconfig")
		return
	}
	next, ok := h.runtimeClusterFromInput(ctx, user, input, existing.ID)
	if !ok {
		return
	}
	existing.Name = next.Name
	existing.Type = next.Type
	existing.Endpoint = next.Endpoint
	existing.Scope = next.Scope
	existing.OwnerRef = next.OwnerRef
	if next.KubeconfigRef != "" {
		existing.KubeconfigRef = next.KubeconfigRef
	}
	existing.IsDefault = next.IsDefault
	existing.Status = next.Status
	if err := h.saveRuntimeClusterWithDefault(existing); err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, h.runtimeClusterResponseForUser(user, existing))
}

func (h *Handlers) DeleteRuntimeCluster(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	var cluster model.RuntimeCluster
	if err := h.db.First(&cluster, "id = ?", ctx.Param("clusterId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "runtime cluster not found")
		return
	}
	if !h.canManageScopedResource(ctx, user, cluster.Scope, cluster.OwnerRef, "无权维护该运行集群") {
		return
	}
	if err := h.db.Delete(&cluster).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (h *Handlers) TestRuntimeCluster(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	var cluster model.RuntimeCluster
	if err := h.db.First(&cluster, "id = ?", ctx.Param("clusterId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "runtime cluster not found")
		return
	}
	if !h.canManageScopedResource(ctx, user, cluster.Scope, cluster.OwnerRef, "无权测试该运行集群") {
		return
	}
	now := time.Now()
	cluster.LastCheckedAt = &now
	if cluster.KubeconfigRef == "" {
		cluster.Status = "missing-credential"
		if err := h.db.Save(&cluster).Error; err != nil {
			writeError(ctx, http.StatusInternalServerError, err.Error())
			return
		}
		writeError(ctx, http.StatusBadRequest, "运行集群缺少 kubeconfig，无法测试连接")
		return
	}
	kubeconfig := h.secrets.Resolve(cluster.KubeconfigRef)
	if strings.TrimSpace(kubeconfig) == "" {
		cluster.Status = "missing-credential"
		if err := h.db.Save(&cluster).Error; err != nil {
			writeError(ctx, http.StatusInternalServerError, err.Error())
			return
		}
		writeError(ctx, http.StatusBadRequest, "运行集群缺少 kubeconfig，无法测试连接")
		return
	}
	client, err := kubeprovider.NewClientFromKubeconfig(kubeconfig)
	if err != nil {
		cluster.Status = "unhealthy"
		if saveErr := h.db.Save(&cluster).Error; saveErr != nil {
			writeError(ctx, http.StatusInternalServerError, saveErr.Error())
			return
		}
		writeError(ctx, http.StatusBadRequest, "运行集群 kubeconfig 无效")
		return
	}
	pingCtx, cancel := context.WithTimeout(ctx.Request.Context(), 8*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx); err != nil {
		cluster.Status = "unhealthy"
		if saveErr := h.db.Save(&cluster).Error; saveErr != nil {
			writeError(ctx, http.StatusInternalServerError, saveErr.Error())
			return
		}
		writeError(ctx, http.StatusBadGateway, "运行集群连接测试失败，请检查 kubeconfig、集群地址和网络连通性")
		return
	}
	cluster.Status = "ready"
	if err := h.db.Save(&cluster).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, h.runtimeClusterResponseForUser(user, cluster))
}

func (h *Handlers) ListRuntimeClusterResources(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	var cluster model.RuntimeCluster
	if err := h.db.First(&cluster, "id = ?", ctx.Param("clusterId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "runtime cluster not found")
		return
	}
	if !h.canManageScopedResource(ctx, user, cluster.Scope, cluster.OwnerRef, "无权查看该集群资源") {
		return
	}
	kubeconfig := h.secrets.Resolve(cluster.KubeconfigRef)
	if strings.TrimSpace(kubeconfig) == "" {
		writeError(ctx, http.StatusBadRequest, "运行集群缺少 kubeconfig，无法读取资源")
		return
	}
	client, err := kubeprovider.NewClientFromKubeconfig(kubeconfig)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, "运行集群 kubeconfig 无效")
		return
	}
	options := kubeprovider.ResourceListOptions{
		Kind:          strings.TrimSpace(ctx.Query("kind")),
		Namespace:     strings.TrimSpace(ctx.Query("namespace")),
		ProjectID:     strings.TrimSpace(ctx.Query("projectId")),
		ApplicationID: strings.TrimSpace(ctx.Query("applicationId")),
		EnvironmentID: strings.TrimSpace(ctx.Query("environmentId")),
	}
	if options.ProjectID != "" && !h.canInspectClusterResourceProject(ctx, user, options.ProjectID) {
		return
	}
	requestCtx, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()
	items, err := client.ListManagedResources(requestCtx, options)
	if err != nil {
		writeError(ctx, http.StatusBadGateway, "集群资源读取失败，请检查集群连接和权限")
		return
	}
	items = h.filterClusterResourceSnapshots(ctx, user, items)
	ctx.JSON(http.StatusOK, items)
}

func (h *Handlers) ListEnvironments(ctx *gin.Context) {
	if _, ok := h.findProjectForCurrentUser(ctx); !ok {
		return
	}
	var environments []model.Environment
	query := h.db.Where("project_id = ?", ctx.Param("projectId")).Order("created_at desc")
	query = applySearch(ctx, query, "name", "slug", "stage", "namespace")
	if err := query.Find(&environments).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, environments)
}

func (h *Handlers) CreateEnvironment(ctx *gin.Context) {
	user, _, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin", "developer")
	if !ok {
		return
	}
	var input environmentInput
	if !bindJSON(ctx, &input) {
		return
	}
	if !validateEnvironmentSlug(ctx, input.Slug) {
		return
	}
	environment := environmentFromInput(ctx.Param("projectId"), user.ID, input, "")
	environment.ID = id.New("env")
	if err := h.db.Create(&environment).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusCreated, environment)
}

func (h *Handlers) UpdateEnvironment(ctx *gin.Context) {
	if _, ok := h.findProjectForCurrentUserWithRoles(ctx, "owner", "admin", "developer"); !ok {
		return
	}
	environment, ok := h.findEnvironment(ctx)
	if !ok {
		return
	}
	var input environmentInput
	if !bindJSON(ctx, &input) {
		return
	}
	if !validateEnvironmentSlug(ctx, input.Slug) {
		return
	}
	next := environmentFromInput(ctx.Param("projectId"), environment.CreatedBy, input, environment.ID)
	environment.Name = next.Name
	environment.Slug = next.Slug
	environment.Stage = next.Stage
	environment.ClusterID = next.ClusterID
	environment.Namespace = next.Namespace
	environment.Replicas = next.Replicas
	environment.CPURequest = next.CPURequest
	environment.MemoryRequest = next.MemoryRequest
	environment.EnvVars = next.EnvVars
	environment.ConfigRefs = next.ConfigRefs
	environment.SecretRefs = next.SecretRefs
	if err := h.db.Save(&environment).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, environment)
}

func (h *Handlers) DeleteEnvironment(ctx *gin.Context) {
	if _, ok := h.findProjectForCurrentUserWithRoles(ctx, "owner", "admin"); !ok {
		return
	}
	environment, ok := h.findEnvironment(ctx)
	if !ok {
		return
	}
	if err := h.db.Delete(&environment).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (h *Handlers) ListReleases(ctx *gin.Context) {
	if _, ok := h.findProjectForCurrentUser(ctx); !ok {
		return
	}
	query := h.db.Where("project_id = ?", ctx.Param("projectId")).Order("created_at desc")
	if environmentID := strings.TrimSpace(ctx.Query("environmentId")); environmentID != "" {
		query = query.Where("environment_id = ?", environmentID)
	}
	if moduleID := strings.TrimSpace(ctx.Query("moduleId")); moduleID != "" {
		query = query.Where("module_id = ?", moduleID)
	}
	var releases []model.Release
	if err := query.Find(&releases).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, releases)
}

func (h *Handlers) ListDeploymentTargets(ctx *gin.Context) {
	if _, _, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin", "developer", "viewer"); !ok {
		return
	}
	app, ok := h.findApplication(ctx)
	if !ok {
		return
	}
	var targets []model.DeploymentTarget
	if err := h.db.Where("project_id = ? and application_id = ?", app.ProjectID, app.ID).Order("created_at asc").Find(&targets).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, targets)
}

func (h *Handlers) CreateDeploymentTarget(ctx *gin.Context) {
	user, _, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin", "developer")
	if !ok {
		return
	}
	app, ok := h.findApplication(ctx)
	if !ok {
		return
	}
	var input deploymentTargetInput
	if !bindJSON(ctx, &input) {
		return
	}
	input.Enabled = true
	target, ok := h.deploymentTargetFromInput(ctx, user, app, input, id.New("dplt"))
	if !ok {
		return
	}
	if err := h.db.Create(&target).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusCreated, target)
}

func (h *Handlers) UpdateDeploymentTarget(ctx *gin.Context) {
	user, _, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin", "developer")
	if !ok {
		return
	}
	app, ok := h.findApplication(ctx)
	if !ok {
		return
	}
	var existing model.DeploymentTarget
	if err := h.db.First(&existing, "id = ? and project_id = ? and application_id = ?", ctx.Param("targetId"), app.ProjectID, app.ID).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "deployment target not found")
		return
	}
	var input deploymentTargetInput
	if !bindJSON(ctx, &input) {
		return
	}
	target, ok := h.deploymentTargetFromInput(ctx, user, app, input, existing.ID)
	if !ok {
		return
	}
	target.CreatedBy = existing.CreatedBy
	target.CreatedAt = existing.CreatedAt
	if err := h.db.Save(&target).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, target)
}

func (h *Handlers) DeleteDeploymentTarget(ctx *gin.Context) {
	_, _, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin", "developer")
	if !ok {
		return
	}
	app, ok := h.findApplication(ctx)
	if !ok {
		return
	}
	var target model.DeploymentTarget
	if err := h.db.First(&target, "id = ? and project_id = ? and application_id = ?", ctx.Param("targetId"), app.ProjectID, app.ID).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "deployment target not found")
		return
	}
	if err := h.db.Delete(&target).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (h *Handlers) CreateRelease(ctx *gin.Context) {
	user, _, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin", "developer")
	if !ok {
		return
	}
	var input releaseInput
	if !bindJSON(ctx, &input) {
		return
	}
	release := releaseFromInput(ctx.Param("projectId"), user.ID, input, "")
	if !h.validateReleaseForCreate(ctx, &release) {
		return
	}
	release.ID = id.New("rel")
	if err := h.db.Create(&release).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if !h.enqueueDeployRun(ctx.Request.Context(), release) {
		release.Status = "failed"
		release.Message = "部署任务投递失败，请稍后重试"
		if err := h.db.Save(&release).Error; err != nil {
			writeError(ctx, http.StatusInternalServerError, err.Error())
			return
		}
		writeError(ctx, http.StatusServiceUnavailable, "部署队列暂不可用")
		return
	}
	ctx.JSON(http.StatusCreated, release)
}

func (h *Handlers) RollbackRelease(ctx *gin.Context) {
	user, _, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin", "developer")
	if !ok {
		return
	}
	source, ok := h.findRelease(ctx)
	if !ok {
		return
	}
	target, ok := h.findPreviousSuccessfulRelease(ctx, source)
	if !ok {
		return
	}
	revision, err := h.nextReleaseRevision(source)
	if err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	release := rollbackReleaseFromTarget(source, target, user.ID, revision)
	release.ID = id.New("rel")
	if err := h.db.Create(&release).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if !h.enqueueDeployRun(ctx.Request.Context(), release) {
		release.Status = "failed"
		release.Message = "部署任务投递失败，请稍后重试"
		if err := h.db.Save(&release).Error; err != nil {
			writeError(ctx, http.StatusInternalServerError, err.Error())
			return
		}
		writeError(ctx, http.StatusServiceUnavailable, "部署队列暂不可用")
		return
	}
	ctx.JSON(http.StatusCreated, release)
}

func (h *Handlers) GetReleaseLogs(ctx *gin.Context) {
	if _, ok := h.findProjectForCurrentUser(ctx); !ok {
		return
	}
	release, ok := h.findRelease(ctx)
	if !ok {
		return
	}
	var log model.ReleaseLog
	err := h.db.First(&log, "release_id = ? and project_id = ?", release.ID, release.ProjectID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ctx.JSON(http.StatusOK, model.ReleaseLog{
			ReleaseID: release.ID,
			ProjectID: release.ProjectID,
			Content:   "",
		})
		return
	}
	if err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, log)
}

func (h *Handlers) findPreviousSuccessfulRelease(ctx *gin.Context, source model.Release) (model.Release, bool) {
	var target model.Release
	err := h.db.Where(
		"project_id = ? and application_id = ? and environment_id = ? and status = ? and revision < ?",
		source.ProjectID,
		source.ApplicationID,
		source.EnvironmentID,
		"succeeded",
		source.Revision,
	).Order("revision desc, created_at desc").First(&target).Error
	if err != nil {
		writeError(ctx, http.StatusConflict, "上一成功版本不存在")
		return target, false
	}
	return target, true
}

func (h *Handlers) nextReleaseRevision(source model.Release) (int, error) {
	return nextReleaseRevisionFor(h.db, source.ProjectID, source.ApplicationID, source.EnvironmentID)
}

func (h *Handlers) enqueueDeployRun(ctx context.Context, release model.Release) bool {
	if h.taskClient == nil {
		return false
	}
	_, err := h.taskClient.EnqueueDeployRun(ctx, tasks.DeployRunPayload{
		ReleaseID: release.ID,
		ProjectID: release.ProjectID,
		ActorID:   release.CreatedBy,
	})
	return err == nil
}

func (h *Handlers) enqueueAutoDeploymentsForBuildRun(ctx context.Context, run model.BuildRun) {
	if run.Status != "succeeded" || strings.TrimSpace(run.ImageRef) == "" || strings.TrimSpace(run.ModuleID) == "" {
		return
	}
	var targets []model.DeploymentTarget
	if err := h.db.Where(
		"project_id = ? and application_id = ? and module_id = ? and enabled = ? and auto_deploy = ? and require_approval = ?",
		run.ProjectID,
		run.ApplicationID,
		run.ModuleID,
		true,
		true,
		false,
	).Find(&targets).Error; err != nil {
		return
	}
	var config model.ApplicationModule
	if err := h.db.First(&config, "id = ? and project_id = ? and application_id = ?", run.ModuleID, run.ProjectID, run.ApplicationID).Error; err != nil {
		return
	}
	for _, target := range targets {
		if !moduleMatchesBuildRun(config, run) || !deploymentTargetMatchesBuildRun(target, run) {
			continue
		}
		release, ok := h.createAutoDeployRelease(ctx, run, target)
		if !ok {
			continue
		}
		if !h.enqueueDeployRun(ctx, release) {
			release.Status = "failed"
			release.Message = "部署任务投递失败，请稍后重试"
			_ = h.db.Save(&release).Error
		}
	}
}

func (h *Handlers) createAutoDeployRelease(ctx context.Context, run model.BuildRun, target model.DeploymentTarget) (model.Release, bool) {
	release := model.Release{}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var existing model.Release
		err := tx.First(&existing, "project_id = ? and application_id = ? and environment_id = ? and build_run_id = ?", run.ProjectID, run.ApplicationID, target.EnvironmentID, run.ID).Error
		if err == nil {
			release = existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		revision, err := nextReleaseRevisionFor(tx, run.ProjectID, run.ApplicationID, target.EnvironmentID)
		if err != nil {
			return err
		}
		release = model.Release{
			ID:            id.New("rel"),
			ProjectID:     run.ProjectID,
			ApplicationID: run.ApplicationID,
			EnvironmentID: target.EnvironmentID,
			ModuleID:      run.ModuleID,
			BuildRunID:    run.ID,
			ImageRef:      run.ImageRef,
			Type:          "deploy",
			Status:        "pending",
			Revision:      revision,
			Message:       "auto deploy from build",
			CreatedBy:     run.CreatedBy,
		}
		return tx.Create(&release).Error
	})
	return release, err == nil && release.ID != "" && release.Status == "pending"
}

func moduleMatchesBuildRun(config model.ApplicationModule, run model.BuildRun) bool {
	return matchesDeploymentPattern(config.BranchPattern, run.SourceBranch) && matchesDeploymentPattern(config.TagPattern, run.SourceTag)
}

func deploymentTargetMatchesBuildRun(target model.DeploymentTarget, run model.BuildRun) bool {
	return matchesDeploymentPattern(target.BranchPattern, run.SourceBranch) && matchesDeploymentPattern(target.TagPattern, run.SourceTag)
}

func matchesDeploymentPattern(patterns string, value string) bool {
	normalized := normalizeStringList(strings.Split(patterns, ","))
	if len(normalized) == 0 {
		return true
	}
	value = strings.TrimSpace(value)
	for _, patternValue := range normalized {
		if patternValue == "*" {
			return true
		}
		if value == "" {
			continue
		}
		if patternValue == value {
			return true
		}
		matched, err := path.Match(patternValue, value)
		if err == nil && matched {
			return true
		}
	}
	return false
}

func nextReleaseRevisionFor(tx *gorm.DB, projectID string, applicationID string, environmentID string) (int, error) {
	var maxRevision int
	err := tx.Model(&model.Release{}).
		Where("project_id = ? and application_id = ? and environment_id = ?", projectID, applicationID, environmentID).
		Select("coalesce(max(revision), 0)").
		Scan(&maxRevision).Error
	if err != nil {
		return 0, err
	}
	return maxRevision + 1, nil
}

func (h *Handlers) validateReleaseForCreate(ctx *gin.Context, release *model.Release) bool {
	var application model.Application
	if err := h.db.First(&application, "id = ? and project_id = ?", release.ApplicationID, release.ProjectID).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, "应用不存在或不属于当前项目空间")
		return false
	}
	var environment model.Environment
	if err := h.db.First(&environment, "id = ? and project_id = ?", release.EnvironmentID, release.ProjectID).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, "环境不存在或不属于当前项目空间")
		return false
	}
	if strings.TrimSpace(release.BuildRunID) == "" {
		if strings.TrimSpace(release.ImageRef) == "" {
			writeError(ctx, http.StatusBadRequest, "发布镜像不能为空")
			return false
		}
		if strings.TrimSpace(release.ModuleID) != "" {
			var config model.ApplicationModule
			if err := h.db.First(&config, "id = ? and project_id = ? and application_id = ?", release.ModuleID, release.ProjectID, release.ApplicationID).Error; err != nil {
				writeError(ctx, http.StatusBadRequest, "模块配置不存在")
				return false
			}
		}
		return true
	}
	var run model.BuildRun
	if err := h.db.First(&run, "id = ? and project_id = ? and application_id = ?", release.BuildRunID, release.ProjectID, release.ApplicationID).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, "构建产物不存在或不属于当前应用")
		return false
	}
	if run.Status != "succeeded" {
		writeError(ctx, http.StatusBadRequest, "只能发布成功构建产物")
		return false
	}
	release.ModuleID = run.ModuleID
	imageRef := strings.TrimSpace(run.ImageRef)
	if imageRef == "" && strings.TrimSpace(run.TargetRegistryID) != "" {
		var registry model.ArtifactRegistry
		if err := h.db.First(&registry, "id = ?", run.TargetRegistryID).Error; err == nil {
			imageRef = buildImageRef(registry, run)
		}
	}
	if imageRef == "" {
		writeError(ctx, http.StatusBadRequest, "构建产物缺少镜像引用")
		return false
	}
	if strings.TrimSpace(release.ImageRef) != "" && strings.TrimSpace(release.ImageRef) != imageRef {
		writeError(ctx, http.StatusBadRequest, "发布镜像必须与所选构建产物一致")
		return false
	}
	release.ImageRef = imageRef
	return true
}

func (h *Handlers) findEnvironment(ctx *gin.Context) (model.Environment, bool) {
	var environment model.Environment
	if err := h.db.First(&environment, "id = ? and project_id = ?", ctx.Param("environmentId"), ctx.Param("projectId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "environment not found")
		return environment, false
	}
	return environment, true
}

func (h *Handlers) findRelease(ctx *gin.Context) (model.Release, bool) {
	var release model.Release
	if err := h.db.First(&release, "id = ? and project_id = ?", ctx.Param("releaseId"), ctx.Param("projectId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "release not found")
		return release, false
	}
	return release, true
}

func (h *Handlers) runtimeClusterResponseForUser(user model.User, cluster model.RuntimeCluster) model.RuntimeCluster {
	cluster.KubeconfigSet = cluster.KubeconfigRef != ""
	cluster.Kubeconfig = ""
	if !h.canInspectScopedResourceConfig(user, cluster.Scope, cluster.OwnerRef) {
		cluster.Endpoint = ""
	}
	if h.canInspectRuntimeClusterKubeconfig(user, cluster) {
		cluster.Kubeconfig = h.secrets.Resolve(cluster.KubeconfigRef)
	}
	return cluster
}

func (h *Handlers) canInspectRuntimeClusterKubeconfig(user model.User, cluster model.RuntimeCluster) bool {
	return user.Role == "platform_admin" || cluster.CreatedBy == user.ID
}

func (h *Handlers) canInspectClusterResourceProject(ctx *gin.Context, user model.User, projectID string) bool {
	if user.Role == "platform_admin" {
		return true
	}
	if _, ok := h.findProjectForCurrentUserWithRolesByID(ctx, projectID, "owner", "admin"); ok {
		return true
	}
	return false
}

func (h *Handlers) filterClusterResourceSnapshots(ctx *gin.Context, user model.User, items []kubeprovider.ResourceSnapshot) []kubeprovider.ResourceSnapshot {
	if user.Role == "platform_admin" {
		return items
	}
	allowed := make(map[string]bool)
	filtered := make([]kubeprovider.ResourceSnapshot, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.ProjectID) == "" {
			continue
		}
		allowedProject, ok := allowed[item.ProjectID]
		if !ok {
			_, projectOK := h.findProjectForCurrentUserWithRolesByID(ctx, item.ProjectID, "owner", "admin")
			allowedProject = projectOK
			allowed[item.ProjectID] = projectOK
		}
		if allowedProject {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func (h *Handlers) runtimeClusterFromInput(ctx *gin.Context, user model.User, input runtimeClusterInput, clusterID string) (model.RuntimeCluster, bool) {
	scope, ownerRef, ok := h.normalizeScopedOwner(ctx, user, input.Scope, input.OwnerRef, "只有平台管理员可以维护全局运行集群")
	if !ok {
		return model.RuntimeCluster{}, false
	}
	if input.IsDefault && scope != "global" {
		writeError(ctx, http.StatusBadRequest, "只有全局运行集群可以设为默认集群")
		return model.RuntimeCluster{}, false
	}
	kubeconfigRef := ""
	if strings.TrimSpace(input.Kubeconfig) != "" {
		kubeconfig, err := flattenKubeconfig(input.Kubeconfig)
		if err != nil {
			writeError(ctx, http.StatusBadRequest, err.Error())
			return model.RuntimeCluster{}, false
		}
		kubeconfigRef = h.secrets.Store(kubeconfig, user.ID, "runtime_cluster:"+clusterID+":kubeconfig")
	}
	return model.RuntimeCluster{
		ID:            clusterID,
		Name:          strings.TrimSpace(input.Name),
		Type:          normalizeRuntimeClusterType(input.Type),
		Endpoint:      strings.TrimSpace(input.Endpoint),
		Scope:         scope,
		OwnerRef:      ownerRef,
		KubeconfigRef: kubeconfigRef,
		IsDefault:     input.IsDefault,
		Status:        fallback(strings.TrimSpace(input.Status), "unknown"),
		CreatedBy:     user.ID,
	}, true
}

func flattenKubeconfig(kubeconfig string) (string, error) {
	config, err := clientcmd.Load([]byte(kubeconfig))
	if err != nil {
		return "", fmt.Errorf("kubeconfig 无效，请检查格式")
	}
	if err := api.FlattenConfig(config); err != nil {
		return "", fmt.Errorf("kubeconfig 引用了当前 API 无法读取的证书文件，请导入已内联证书的 kubeconfig: %w", err)
	}
	output, err := clientcmd.Write(*config)
	if err != nil {
		return "", fmt.Errorf("kubeconfig 序列化失败")
	}
	return string(output), nil
}

func (h *Handlers) saveRuntimeClusterWithDefault(cluster model.RuntimeCluster) error {
	return h.db.Transaction(func(tx *gorm.DB) error {
		if cluster.IsDefault {
			if cluster.Scope != "global" {
				return errors.New("只有全局运行集群可以设为默认集群")
			}
			if err := tx.Model(&model.RuntimeCluster{}).Where("scope = ? and id <> ?", "global", cluster.ID).Update("is_default", false).Error; err != nil {
				return err
			}
		} else if cluster.Scope != "global" {
			cluster.IsDefault = false
		}
		return tx.Save(&cluster).Error
	})
}

func environmentFromInput(projectID, userID string, input environmentInput, environmentID string) model.Environment {
	slug := strings.TrimSpace(input.Slug)
	return model.Environment{
		ID:            environmentID,
		ProjectID:     projectID,
		Name:          strings.TrimSpace(input.Name),
		Slug:          slug,
		Stage:         normalizeStage(input.Stage),
		ClusterID:     strings.TrimSpace(input.ClusterID),
		Namespace:     strings.TrimSpace(input.Namespace),
		Replicas:      fallbackInt(input.Replicas, 1),
		CPURequest:    strings.TrimSpace(input.CPURequest),
		MemoryRequest: strings.TrimSpace(input.MemoryRequest),
		EnvVars:       strings.TrimSpace(input.EnvVars),
		ConfigRefs:    strings.TrimSpace(input.ConfigRefs),
		SecretRefs:    strings.TrimSpace(input.SecretRefs),
		CreatedBy:     userID,
	}
}

func validateEnvironmentSlug(ctx *gin.Context, slug string) bool {
	slug = strings.TrimSpace(slug)
	if len(slug) > environmentSlugMaxLength {
		writeError(ctx, http.StatusBadRequest, fmt.Sprintf("环境标识最多 %d 个字符", environmentSlugMaxLength))
		return false
	}
	return true
}

func (h *Handlers) deploymentTargetFromInput(ctx *gin.Context, user model.User, app model.Application, input deploymentTargetInput, targetID string) (model.DeploymentTarget, bool) {
	var environment model.Environment
	if err := h.db.First(&environment, "id = ? and project_id = ?", strings.TrimSpace(input.EnvironmentID), app.ProjectID).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, "环境不存在或不属于当前项目空间")
		return model.DeploymentTarget{}, false
	}
	var config model.ApplicationModule
	if err := h.db.First(&config, "id = ? and project_id = ? and application_id = ?", strings.TrimSpace(input.ModuleID), app.ProjectID, app.ID).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, "模块配置不存在或不属于当前应用")
		return model.DeploymentTarget{}, false
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = config.Name + " -> " + environment.Name
	}
	return model.DeploymentTarget{
		ID:              targetID,
		ProjectID:       app.ProjectID,
		ApplicationID:   app.ID,
		EnvironmentID:   environment.ID,
		ModuleID:        config.ID,
		Name:            name,
		AutoDeploy:      input.AutoDeploy,
		BranchPattern:   strings.TrimSpace(input.BranchPattern),
		TagPattern:      strings.TrimSpace(input.TagPattern),
		RequireApproval: input.RequireApproval,
		Enabled:         input.Enabled,
		CreatedBy:       user.ID,
	}, true
}

func releaseFromInput(projectID, userID string, input releaseInput, releaseID string) model.Release {
	return model.Release{
		ID:            releaseID,
		ProjectID:     projectID,
		ApplicationID: strings.TrimSpace(input.ApplicationID),
		EnvironmentID: strings.TrimSpace(input.EnvironmentID),
		ModuleID:      strings.TrimSpace(input.ModuleID),
		BuildRunID:    strings.TrimSpace(input.BuildRunID),
		ImageRef:      strings.TrimSpace(input.ImageRef),
		Type:          normalizeReleaseType(input.Type),
		Status:        fallback(strings.TrimSpace(input.Status), "pending"),
		Revision:      fallbackInt(input.Revision, 1),
		Message:       strings.TrimSpace(input.Message),
		CreatedBy:     userID,
	}
}

func rollbackReleaseFromTarget(source model.Release, target model.Release, userID string, revision int) model.Release {
	return model.Release{
		ProjectID:      source.ProjectID,
		ApplicationID:  source.ApplicationID,
		EnvironmentID:  source.EnvironmentID,
		ModuleID:       target.ModuleID,
		BuildRunID:     target.BuildRunID,
		ImageRef:       target.ImageRef,
		Type:           "rollback",
		Status:         "pending",
		Revision:       fallbackInt(revision, source.Revision+1),
		RollbackFromID: target.ID,
		CreatedBy:      userID,
	}
}

func normalizeRuntimeClusterType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "docker-compose":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "kubernetes"
	}
}

func normalizeStage(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "prod", "production":
		return "prod"
	case "staging", "test":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "dev"
	}
}

func normalizeReleaseType(value string) string {
	if strings.ToLower(strings.TrimSpace(value)) == "rollback" {
		return "rollback"
	}
	return "deploy"
}

type runtimeClusterInput struct {
	Name       string `json:"name" binding:"required"`
	Type       string `json:"type"`
	Endpoint   string `json:"endpoint"`
	Scope      string `json:"scope"`
	OwnerRef   string `json:"ownerRef"`
	Kubeconfig string `json:"kubeconfig"`
	IsDefault  bool   `json:"isDefault"`
	Status     string `json:"status"`
}

type environmentInput struct {
	Name          string `json:"name" binding:"required"`
	Slug          string `json:"slug" binding:"required"`
	Stage         string `json:"stage"`
	ClusterID     string `json:"clusterId"`
	Namespace     string `json:"namespace"`
	Replicas      int    `json:"replicas"`
	CPURequest    string `json:"cpuRequest"`
	MemoryRequest string `json:"memoryRequest"`
	EnvVars       string `json:"envVars"`
	ConfigRefs    string `json:"configRefs"`
	SecretRefs    string `json:"secretRefs"`
}

type deploymentTargetInput struct {
	Name            string `json:"name"`
	EnvironmentID   string `json:"environmentId" binding:"required"`
	ModuleID        string `json:"moduleId" binding:"required"`
	AutoDeploy      bool   `json:"autoDeploy"`
	BranchPattern   string `json:"branchPattern"`
	TagPattern      string `json:"tagPattern"`
	RequireApproval bool   `json:"requireApproval"`
	Enabled         bool   `json:"enabled"`
}

type releaseInput struct {
	ApplicationID string `json:"applicationId" binding:"required"`
	EnvironmentID string `json:"environmentId" binding:"required"`
	ModuleID      string `json:"moduleId"`
	BuildRunID    string `json:"buildRunId"`
	ImageRef      string `json:"imageRef" binding:"required"`
	Type          string `json:"type"`
	Status        string `json:"status"`
	Revision      int    `json:"revision"`
	Message       string `json:"message"`
}
