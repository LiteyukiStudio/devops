package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LiteyukiStudio/devops/internal/id"
	"github.com/LiteyukiStudio/devops/internal/model"
	kubeprovider "github.com/LiteyukiStudio/devops/internal/provider/kubernetes"
	"github.com/LiteyukiStudio/devops/internal/tasks"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/remotecommand"
)

var apiDNSLabelInvalidPattern = regexp.MustCompile(`[^a-z0-9-]+`)

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
	responses, err := h.clusterResourceResponses(items)
	if err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, responses)
}

func (h *Handlers) DeleteRuntimeClusterResource(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	var cluster model.RuntimeCluster
	if err := h.db.First(&cluster, "id = ?", ctx.Param("clusterId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "runtime cluster not found")
		return
	}
	if !h.canManageScopedResource(ctx, user, cluster.Scope, cluster.OwnerRef, "无权维护该集群资源") {
		return
	}
	kubeconfig := h.secrets.Resolve(cluster.KubeconfigRef)
	if strings.TrimSpace(kubeconfig) == "" {
		writeError(ctx, http.StatusBadRequest, "运行集群缺少 kubeconfig，无法维护资源")
		return
	}
	client, err := kubeprovider.NewClientFromKubeconfig(kubeconfig)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, "运行集群 kubeconfig 无效")
		return
	}
	kind := strings.TrimSpace(ctx.Query("kind"))
	namespace := strings.TrimSpace(ctx.Query("namespace"))
	name := strings.TrimSpace(ctx.Query("name"))
	if kind == "" || name == "" {
		writeError(ctx, http.StatusBadRequest, "资源类型和名称不能为空")
		return
	}
	requestCtx, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()
	snapshot, err := client.GetManagedResource(requestCtx, kind, namespace, name)
	if err != nil {
		writeError(ctx, http.StatusBadGateway, "集群资源读取失败，请确认资源仍存在且归属平台管理")
		return
	}
	if !h.canManageClusterResourceSnapshot(ctx, user, snapshot) {
		return
	}
	if err := client.DeleteManagedResource(requestCtx, kind, namespace, name); err != nil {
		writeError(ctx, http.StatusBadGateway, "集群资源删除失败，请确认资源仍存在且归属平台管理")
		return
	}
	h.audit(user.ID, "runtime_cluster_resource.delete", cluster.ID, true, strings.Join([]string{kind, namespace, name}, "/"))
	ctx.Status(http.StatusNoContent)
}

func (h *Handlers) ListRuntimeClusterResourceEvents(ctx *gin.Context) {
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
		writeError(ctx, http.StatusBadRequest, "运行集群缺少 kubeconfig，无法读取资源事件")
		return
	}
	client, err := kubeprovider.NewClientFromKubeconfig(kubeconfig)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, "运行集群 kubeconfig 无效")
		return
	}
	kind := strings.TrimSpace(ctx.Query("kind"))
	namespace := strings.TrimSpace(ctx.Query("namespace"))
	name := strings.TrimSpace(ctx.Query("name"))
	if kind == "" || name == "" {
		writeError(ctx, http.StatusBadRequest, "资源类型和名称不能为空")
		return
	}
	requestCtx, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()
	events, snapshot, err := client.ListManagedResourceEvents(requestCtx, kind, namespace, name)
	if err != nil {
		writeError(ctx, http.StatusBadGateway, "集群资源事件读取失败，请确认资源仍存在且归属平台管理")
		return
	}
	if !h.canInspectClusterResourceSnapshot(ctx, user, snapshot) {
		return
	}
	ctx.JSON(http.StatusOK, events)
}

type clusterResourceResponse struct {
	ID                   string            `json:"id"`
	Kind                 string            `json:"kind"`
	Name                 string            `json:"name"`
	Namespace            string            `json:"namespace"`
	Status               string            `json:"status"`
	Summary              string            `json:"summary"`
	ProjectID            string            `json:"projectId"`
	ApplicationID        string            `json:"applicationId"`
	EnvironmentID        string            `json:"environmentId"`
	DeploymentTargetID   string            `json:"deploymentTargetId"`
	ReleaseID            string            `json:"releaseId"`
	RouteID              string            `json:"routeId"`
	ProjectName          string            `json:"projectName"`
	ApplicationName      string            `json:"applicationName"`
	DeploymentTargetName string            `json:"deploymentTargetName"`
	Labels               map[string]string `json:"labels"`
	CreatedAt            time.Time         `json:"createdAt"`
}

func (h *Handlers) clusterResourceResponses(items []kubeprovider.ResourceSnapshot) ([]clusterResourceResponse, error) {
	responses := make([]clusterResourceResponse, 0, len(items))
	releaseIDs := make(map[string]bool)
	routeIDs := make(map[string]bool)
	for _, item := range items {
		responses = append(responses, clusterResourceResponse{
			ID:                 item.ID,
			Kind:               item.Kind,
			Name:               item.Name,
			Namespace:          item.Namespace,
			Status:             item.Status,
			Summary:            item.Summary,
			ProjectID:          item.ProjectID,
			ApplicationID:      item.ApplicationID,
			EnvironmentID:      item.EnvironmentID,
			DeploymentTargetID: item.DeploymentTargetID,
			ReleaseID:          item.ReleaseID,
			RouteID:            item.RouteID,
			Labels:             item.Labels,
			CreatedAt:          item.CreatedAt,
		})
		addStringID(releaseIDs, item.ReleaseID)
		addStringID(routeIDs, item.RouteID)
	}

	releasesByID := make(map[string]model.Release)
	if ids := stringSetValues(releaseIDs); len(ids) > 0 {
		var releases []model.Release
		if err := h.db.Unscoped().Where("id in ?", ids).Find(&releases).Error; err != nil {
			return nil, err
		}
		for _, release := range releases {
			releasesByID[release.ID] = release
		}
	}

	routesByID := make(map[string]model.GatewayRoute)
	if ids := stringSetValues(routeIDs); len(ids) > 0 {
		var routes []model.GatewayRoute
		if err := h.db.Unscoped().Where("id in ?", ids).Find(&routes).Error; err != nil {
			return nil, err
		}
		for _, route := range routes {
			routesByID[route.ID] = route
		}
	}

	deploymentTargetIDs := make(map[string]bool)
	for index := range responses {
		response := &responses[index]
		if route, ok := routesByID[response.RouteID]; ok {
			if strings.TrimSpace(response.DeploymentTargetID) == "" {
				response.DeploymentTargetID = strings.TrimSpace(route.DeploymentTargetID)
			}
			fillResourceOwnerIDs(response, route.ProjectID, route.ApplicationID)
			if strings.TrimSpace(response.EnvironmentID) == "" {
				response.EnvironmentID = strings.TrimSpace(route.EnvironmentID)
			}
		}
		if release, ok := releasesByID[response.ReleaseID]; ok {
			if strings.TrimSpace(response.DeploymentTargetID) == "" {
				response.DeploymentTargetID = strings.TrimSpace(release.DeploymentTargetID)
			}
			fillResourceOwnerIDs(response, release.ProjectID, release.ApplicationID)
		}
		addStringID(deploymentTargetIDs, response.DeploymentTargetID)
	}

	targetsByID := make(map[string]model.DeploymentTarget)
	if ids := stringSetValues(deploymentTargetIDs); len(ids) > 0 {
		var targets []model.DeploymentTarget
		if err := h.db.Unscoped().Where("id in ?", ids).Find(&targets).Error; err != nil {
			return nil, err
		}
		for _, target := range targets {
			targetsByID[target.ID] = target
		}
	}

	projectIDs := make(map[string]bool)
	applicationIDs := make(map[string]bool)
	deploymentTargetNameByID := make(map[string]string)
	for index := range responses {
		response := &responses[index]
		if target, ok := targetsByID[response.DeploymentTargetID]; ok {
			fillResourceOwnerIDs(response, target.ProjectID, target.ApplicationID)
			deploymentTargetNameByID[target.ID] = target.Name
		}
		addStringID(projectIDs, response.ProjectID)
		addStringID(applicationIDs, response.ApplicationID)
	}

	projectNames, err := h.projectNamesByID(projectIDs)
	if err != nil {
		return nil, err
	}
	applicationNames, err := h.applicationNamesByID(applicationIDs)
	if err != nil {
		return nil, err
	}
	for index := range responses {
		responses[index].ProjectName = projectNames[responses[index].ProjectID]
		responses[index].ApplicationName = applicationNames[responses[index].ApplicationID]
		responses[index].DeploymentTargetName = deploymentTargetNameByID[responses[index].DeploymentTargetID]
	}
	return responses, nil
}

func fillResourceOwnerIDs(response *clusterResourceResponse, projectID string, applicationID string) {
	if strings.TrimSpace(response.ProjectID) == "" {
		response.ProjectID = strings.TrimSpace(projectID)
	}
	if strings.TrimSpace(response.ApplicationID) == "" {
		response.ApplicationID = strings.TrimSpace(applicationID)
	}
}

func (h *Handlers) projectNamesByID(ids map[string]bool) (map[string]string, error) {
	names := make(map[string]string)
	if values := stringSetValues(ids); len(values) > 0 {
		var projects []model.Project
		if err := h.db.Unscoped().Where("id in ?", values).Find(&projects).Error; err != nil {
			return nil, err
		}
		for _, project := range projects {
			names[project.ID] = project.Name
		}
	}
	return names, nil
}

func (h *Handlers) applicationNamesByID(ids map[string]bool) (map[string]string, error) {
	names := make(map[string]string)
	if values := stringSetValues(ids); len(values) > 0 {
		var applications []model.Application
		if err := h.db.Unscoped().Where("id in ?", values).Find(&applications).Error; err != nil {
			return nil, err
		}
		for _, application := range applications {
			names[application.ID] = application.Name
		}
	}
	return names, nil
}

func addStringID(ids map[string]bool, value string) {
	normalized := strings.TrimSpace(value)
	if normalized != "" {
		ids[normalized] = true
	}
}

func stringSetValues(ids map[string]bool) []string {
	values := make([]string, 0, len(ids))
	for value := range ids {
		values = append(values, value)
	}
	return values
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
	if targetID := strings.TrimSpace(ctx.Query("deploymentTargetId")); targetID != "" {
		query = query.Where("deployment_target_id = ?", targetID)
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
	if err := h.attachDeploymentTargetHookBindings(targets); err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, deploymentTargetResponses(targets))
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
	if !applicationCanMutate(app) {
		writeErrorCode(ctx, http.StatusConflict, "application.delete_in_progress", "应用正在删除中，不能新增部署配置")
		return
	}
	var input deploymentTargetInput
	if !bindJSON(ctx, &input) {
		return
	}
	input.Enabled = true
	target, ok := h.deploymentTargetFromInput(ctx, user, app, input, id.New("dplt"), nil)
	if !ok {
		return
	}
	if err := h.saveDeploymentTarget(target, input.BuildHookBindings); err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if !h.syncDeploymentTargetDataVolume(ctx, target) {
		return
	}
	target, _ = h.deploymentTargetWithHookBindings(target)
	ctx.JSON(http.StatusCreated, deploymentTargetResponseFromModel(target))
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
	if !applicationCanMutate(app) {
		writeErrorCode(ctx, http.StatusConflict, "application.delete_in_progress", "应用正在删除中，不能修改部署配置")
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
	target, ok := h.deploymentTargetFromInput(ctx, user, app, input, existing.ID, decodeSecretRefs(existing.SecretFiles))
	if !ok {
		return
	}
	target.CreatedBy = existing.CreatedBy
	target.CreatedAt = existing.CreatedAt
	if strings.TrimSpace(input.SecretRefs) == "" {
		target.SecretRefs = existing.SecretRefs
	}
	if err := h.saveDeploymentTarget(target, input.BuildHookBindings); err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if !h.syncDeploymentTargetDataVolume(ctx, target) {
		return
	}
	target, _ = h.deploymentTargetWithHookBindings(target)
	ctx.JSON(http.StatusOK, deploymentTargetResponseFromModel(target))
}

func (h *Handlers) ExportDeploymentTargetData(ctx *gin.Context) {
	user, _, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin")
	if !ok {
		return
	}
	app, ok := h.findApplication(ctx)
	if !ok {
		return
	}
	if !applicationCanMutate(app) {
		writeErrorCode(ctx, http.StatusConflict, "application.delete_in_progress", "应用正在删除中，不能删除部署配置")
		return
	}
	var target model.DeploymentTarget
	if err := h.db.First(&target, "id = ? and project_id = ? and application_id = ?", ctx.Param("targetId"), app.ProjectID, app.ID).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "deployment target not found")
		return
	}
	if !target.DataRetentionEnabled {
		writeError(ctx, http.StatusBadRequest, "该部署配置未启用运行数据保留")
		return
	}
	var project model.Project
	if err := h.db.First(&project, "id = ?", target.ProjectID).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "project not found")
		return
	}
	var environment model.Environment
	if err := h.db.First(&environment, "id = ? and project_id = ?", target.EnvironmentID, target.ProjectID).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "environment not found")
		return
	}
	client, namespace, ok := h.kubernetesClientForEnvironment(ctx, project, environment, "运行集群不可用，无法导出运行数据")
	if !ok {
		return
	}
	filename := fmt.Sprintf("%s-%s-data.tar.gz", app.Slug, target.ID)
	ctx.Header("Content-Type", "application/gzip")
	ctx.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	ctx.Header("X-Content-Type-Options", "nosniff")
	requestCtx, cancel := context.WithTimeout(ctx.Request.Context(), 5*time.Minute)
	defer cancel()
	if err := client.StreamDataArchive(requestCtx, kubeprovider.DataExportSpec{
		Name:      "lyd-export-" + shortResourceID(target.ID),
		Namespace: namespace,
		MountPath: deploymentTargetDataMountPath(target),
		Volumes:   deploymentTargetDataExportVolumes(target),
	}, ctx.Writer); err != nil {
		h.audit(user.ID, "deployment_target.data_export", target.ID, false, err.Error())
		return
	}
	h.audit(user.ID, "deployment_target.data_export", target.ID, true, filename)
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
	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("target_id = ?", target.ID).Delete(&model.DeploymentTargetHookBinding{}).Error; err != nil {
			return err
		}
		return tx.Delete(&target).Error
	}); err != nil {
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

func (h *Handlers) GetReleaseRuntimeLogs(ctx *gin.Context) {
	if _, ok := h.findProjectForCurrentUser(ctx); !ok {
		return
	}
	release, ok := h.findRelease(ctx)
	if !ok {
		return
	}
	client, namespace, target, ok := h.releaseRuntimeClient(ctx, release)
	if !ok {
		return
	}
	tailLines := int64(500)
	if value := strings.TrimSpace(ctx.Query("tailLines")); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil && parsed > 0 && parsed <= 5000 {
			tailLines = parsed
		}
	}
	requestCtx, cancel := context.WithTimeout(ctx.Request.Context(), 12*time.Second)
	defer cancel()
	result, err := client.RuntimePodLogs(requestCtx, kubeprovider.RuntimePodLogsOptions{
		Namespace:          namespace,
		DeploymentTargetID: target.ID,
		Container:          strings.TrimSpace(ctx.Query("container")),
		TailLines:          tailLines,
	})
	if err != nil {
		writeError(ctx, http.StatusBadGateway, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, result)
}

func (h *Handlers) ExecReleaseRuntimeCommand(ctx *gin.Context) {
	user, _, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin", "developer")
	if !ok {
		return
	}
	release, ok := h.findRelease(ctx)
	if !ok {
		return
	}
	var input releaseRuntimeExecInput
	if !bindJSON(ctx, &input) {
		return
	}
	command := strings.TrimSpace(input.Command)
	if command == "" {
		writeError(ctx, http.StatusBadRequest, "command is required")
		return
	}
	if len(command) > 2000 {
		writeError(ctx, http.StatusBadRequest, "command is too long")
		return
	}
	client, namespace, target, ok := h.releaseRuntimeClient(ctx, release)
	if !ok {
		return
	}
	requestCtx, cancel := context.WithTimeout(ctx.Request.Context(), 30*time.Second)
	defer cancel()
	result, err := client.RuntimeExec(requestCtx, kubeprovider.RuntimeExecOptions{
		Namespace:          namespace,
		DeploymentTargetID: target.ID,
		Container:          strings.TrimSpace(input.Container),
		Command:            command,
	})
	if err != nil {
		h.audit(user.ID, "release_runtime.exec", release.ID, false, err.Error())
		writeError(ctx, http.StatusBadGateway, err.Error())
		return
	}
	h.audit(user.ID, "release_runtime.exec", release.ID, result.ExitCode == 0, command)
	ctx.JSON(http.StatusOK, result)
}

func (h *Handlers) StreamReleaseRuntimeTerminal(ctx *gin.Context) {
	user, _, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin", "developer")
	if !ok {
		return
	}
	release, ok := h.findRelease(ctx)
	if !ok {
		return
	}
	client, namespace, target, ok := h.releaseRuntimeClient(ctx, release)
	if !ok {
		return
	}
	upgrader := websocket.Upgrader{
		CheckOrigin: func(request *http.Request) bool {
			origin := strings.TrimSpace(request.Header.Get("Origin"))
			if origin == "" {
				return true
			}
			return containsString(configuredAllowedOrigins(), origin)
		},
	}
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		h.audit(user.ID, "release_runtime.terminal", release.ID, false, err.Error())
		return
	}
	defer conn.Close()

	sessionCtx, cancel := context.WithCancel(ctx.Request.Context())
	defer cancel()
	stdinReader, stdinWriter := io.Pipe()
	defer stdinReader.Close()
	defer stdinWriter.Close()
	sizeQueue := newRuntimeTerminalSizeQueue()
	wsWriter := &runtimeTerminalWebSocketWriter{conn: conn}

	go h.readRuntimeTerminalMessages(sessionCtx, conn, stdinWriter, sizeQueue, cancel)
	err = client.RuntimeTerminal(sessionCtx, kubeprovider.RuntimeTerminalOptions{
		Namespace:          namespace,
		DeploymentTargetID: target.ID,
		Container:          strings.TrimSpace(ctx.Query("container")),
		Stdin:              stdinReader,
		Stdout:             wsWriter,
		SizeQueue:          sizeQueue,
	})
	if err != nil && sessionCtx.Err() == nil {
		_, _ = wsWriter.Write([]byte("\r\nterminal disconnected: " + err.Error() + "\r\n"))
		h.audit(user.ID, "release_runtime.terminal", release.ID, false, err.Error())
		return
	}
	h.audit(user.ID, "release_runtime.terminal", release.ID, true, strings.TrimSpace(ctx.Query("container")))
}

func (h *Handlers) readRuntimeTerminalMessages(ctx context.Context, conn *websocket.Conn, stdin *io.PipeWriter, sizeQueue *runtimeTerminalSizeQueue, cancel context.CancelFunc) {
	defer cancel()
	defer stdin.Close()
	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
			continue
		}
		if messageType == websocket.TextMessage {
			var message runtimeTerminalClientMessage
			if err := json.Unmarshal(data, &message); err == nil && message.Type == "resize" {
				sizeQueue.Push(message.Cols, message.Rows)
				continue
			}
		}
		if _, err := stdin.Write(data); err != nil {
			return
		}
	}
}

func (h *Handlers) releaseRuntimeClient(ctx *gin.Context, release model.Release) (*kubeprovider.Client, string, model.DeploymentTarget, bool) {
	var project model.Project
	if err := h.db.First(&project, "id = ?", release.ProjectID).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "project not found")
		return nil, "", model.DeploymentTarget{}, false
	}
	var environment model.Environment
	if err := h.db.First(&environment, "id = ? and project_id = ?", release.EnvironmentID, release.ProjectID).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "environment not found")
		return nil, "", model.DeploymentTarget{}, false
	}
	var target model.DeploymentTarget
	if err := h.db.First(&target, "id = ? and project_id = ? and application_id = ?", release.DeploymentTargetID, release.ProjectID, release.ApplicationID).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "deployment target not found")
		return nil, "", model.DeploymentTarget{}, false
	}
	cluster, ok := h.runtimeClusterForEnvironment(ctx, environment)
	if !ok {
		return nil, "", model.DeploymentTarget{}, false
	}
	kubeconfig := h.secrets.Resolve(cluster.KubeconfigRef)
	if strings.TrimSpace(kubeconfig) == "" {
		writeError(ctx, http.StatusBadRequest, "运行集群缺少 kubeconfig，无法读取运行时")
		return nil, "", model.DeploymentTarget{}, false
	}
	client, err := kubeprovider.NewClientFromKubeconfig(kubeconfig)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, "运行集群 kubeconfig 无效")
		return nil, "", model.DeploymentTarget{}, false
	}
	return client, runtimeProjectNamespace(project), target, true
}

func (h *Handlers) runtimeClusterForEnvironment(ctx *gin.Context, environment model.Environment) (model.RuntimeCluster, bool) {
	var cluster model.RuntimeCluster
	if clusterID := strings.TrimSpace(environment.ClusterID); clusterID != "" {
		err := h.db.First(&cluster, "id = ? and type in ?", clusterID, []string{"kubernetes", "k3s"}).Error
		if err != nil {
			writeError(ctx, http.StatusNotFound, "runtime cluster not found")
			return cluster, false
		}
		return cluster, true
	}
	err := h.db.Where("scope = ? and is_default = ? and type in ?", "global", true, []string{"kubernetes", "k3s"}).First(&cluster).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = h.db.Where("scope = ? and type in ?", "global", []string{"kubernetes", "k3s"}).Order("created_at asc").First(&cluster).Error
	}
	if err != nil {
		writeError(ctx, http.StatusNotFound, "runtime cluster not found")
		return cluster, false
	}
	return cluster, true
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
	if run.Status != "succeeded" || strings.TrimSpace(run.ImageRef) == "" || strings.TrimSpace(run.DeploymentTargetID) == "" {
		return
	}
	var application model.Application
	if err := h.db.First(&application, "id = ? and project_id = ?", run.ApplicationID, run.ProjectID).Error; err != nil {
		return
	}
	if !applicationCanMutate(application) {
		return
	}
	var target model.DeploymentTarget
	if err := h.db.First(
		&target,
		"id = ? and project_id = ? and application_id = ? and enabled = ? and auto_deploy = ? and require_approval = ?",
		run.DeploymentTargetID,
		run.ProjectID,
		run.ApplicationID,
		true,
		true,
		false,
	).Error; err != nil {
		return
	}
	if !deploymentTargetMatchesBuildRun(target, run) {
		return
	}
	release, ok := h.createAutoDeployRelease(ctx, run, target)
	if !ok {
		return
	}
	if !h.enqueueDeployRun(ctx, release) {
		release.Status = "failed"
		release.Message = "部署任务投递失败，请稍后重试"
		_ = h.db.Save(&release).Error
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
			ID:                 id.New("rel"),
			ProjectID:          run.ProjectID,
			ApplicationID:      run.ApplicationID,
			EnvironmentID:      target.EnvironmentID,
			DeploymentTargetID: target.ID,
			BuildRunID:         run.ID,
			ImageRef:           run.ImageRef,
			Type:               "deploy",
			Status:             "pending",
			Revision:           revision,
			Message:            "auto deploy from build",
			CreatedBy:          run.CreatedBy,
		}
		return tx.Create(&release).Error
	})
	return release, err == nil && release.ID != "" && release.Status == "pending"
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
	if !applicationCanMutate(application) {
		writeErrorCode(ctx, http.StatusConflict, "application.delete_in_progress", "应用正在删除中，不能创建发布")
		return false
	}
	var target model.DeploymentTarget
	if err := h.db.First(&target, "id = ? and project_id = ? and application_id = ? and enabled = ?", strings.TrimSpace(release.DeploymentTargetID), release.ProjectID, release.ApplicationID, true).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, "部署配置不存在或不可用")
		return false
	}
	release.EnvironmentID = target.EnvironmentID
	if strings.TrimSpace(release.BuildRunID) == "" {
		if strings.TrimSpace(release.ImageRef) == "" {
			release.ImageRef = strings.TrimSpace(target.ImageRef)
		}
		if strings.TrimSpace(release.ImageRef) == "" {
			writeError(ctx, http.StatusBadRequest, "发布镜像不能为空")
			return false
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
	if run.DeploymentTargetID != release.DeploymentTargetID {
		writeError(ctx, http.StatusBadRequest, "构建产物不属于当前部署配置")
		return false
	}
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

func (h *Handlers) canInspectClusterResourceSnapshot(ctx *gin.Context, user model.User, item kubeprovider.ResourceSnapshot) bool {
	if user.Role == "platform_admin" {
		return true
	}
	if strings.TrimSpace(item.ProjectID) == "" {
		writeError(ctx, http.StatusForbidden, "无权查看无项目空间归属的集群资源")
		return false
	}
	return h.canInspectClusterResourceProject(ctx, user, item.ProjectID)
}

func (h *Handlers) canManageClusterResourceSnapshot(ctx *gin.Context, user model.User, item kubeprovider.ResourceSnapshot) bool {
	if user.Role == "platform_admin" {
		return true
	}
	if strings.TrimSpace(item.ProjectID) == "" {
		writeError(ctx, http.StatusForbidden, "无权维护无项目空间归属的集群资源")
		return false
	}
	if _, ok := h.findProjectForCurrentUserWithRolesByID(ctx, item.ProjectID, "owner", "admin"); ok {
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

func (h *Handlers) deploymentTargetFromInput(ctx *gin.Context, user model.User, app model.Application, input deploymentTargetInput, targetID string, existingSecretFiles map[string]string) (model.DeploymentTarget, bool) {
	var environment model.Environment
	if err := h.db.First(&environment, "id = ? and project_id = ?", strings.TrimSpace(input.EnvironmentID), app.ProjectID).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, "环境不存在或不属于当前项目空间")
		return model.DeploymentTarget{}, false
	}
	sourceType := normalizeDeploymentSourceType(input.SourceType)
	repositoryBindingID := strings.TrimSpace(input.RepositoryBindingID)
	if sourceType == "repository" {
		if repositoryBindingID == "" {
			writeError(ctx, http.StatusBadRequest, "代码仓库不能为空")
			return model.DeploymentTarget{}, false
		}
		var binding model.RepositoryBinding
		if err := h.db.First(&binding, "id = ? and project_id = ? and application_id = ?", repositoryBindingID, app.ProjectID, app.ID).Error; err != nil {
			writeError(ctx, http.StatusBadRequest, "代码仓库绑定不存在")
			return model.DeploymentTarget{}, false
		}
	}
	if strings.TrimSpace(input.BuildProviderID) != "" {
		var provider model.BuildProvider
		if err := h.db.First(&provider, "id = ? and enabled = ?", input.BuildProviderID, true).Error; err != nil {
			writeError(ctx, http.StatusBadRequest, "构建提供方不可用")
			return model.DeploymentTarget{}, false
		}
	}
	targetRepository, targetTag := splitTargetImageRef(input.TargetImageRef)
	if targetRepository == "" {
		targetRepository = strings.Trim(strings.TrimSpace(input.TargetRepository), "/")
		targetTag = strings.TrimSpace(input.TargetTag)
	}
	buildHooksEnabled := true
	if input.BuildHooksEnabled != nil {
		buildHooksEnabled = *input.BuildHooksEnabled
	}
	dataCapacity, ok := normalizeDataCapacity(ctx, input.DataCapacity, input.DataRetentionEnabled)
	if !ok {
		return model.DeploymentTarget{}, false
	}
	dataMountPath, ok := normalizeDataMountPath(ctx, input.DataMountPath, input.DataRetentionEnabled)
	if !ok {
		return model.DeploymentTarget{}, false
	}
	dataVolumes, ok := normalizeDataVolumes(ctx, input.DataVolumes, input.DataRetentionEnabled, dataMountPath, dataCapacity)
	if !ok {
		return model.DeploymentTarget{}, false
	}
	if len(dataVolumes) > 0 {
		dataMountPath = dataVolumes[0].MountPath
		dataCapacity = dataVolumes[0].Capacity
	}
	runtimeConfigSetIDs := normalizeStringList(input.RuntimeConfigSetIDs)
	if len(runtimeConfigSetIDs) > 0 {
		var count int64
		if err := h.db.Model(&model.ProjectRuntimeConfigSet{}).Where("project_id = ? and id in ?", app.ProjectID, runtimeConfigSetIDs).Count(&count).Error; err != nil {
			writeError(ctx, http.StatusInternalServerError, err.Error())
			return model.DeploymentTarget{}, false
		}
		if count != int64(len(runtimeConfigSetIDs)) {
			writeError(ctx, http.StatusBadRequest, "运行配置集不存在或不属于当前项目空间")
			return model.DeploymentTarget{}, false
		}
	}
	configFiles, ok := normalizeRuntimeConfigFilesInput(ctx, input.ConfigFiles)
	if !ok {
		return model.DeploymentTarget{}, false
	}
	secretFiles, ok := h.runtimeSecretFilesFromInput(ctx, user, targetID, input.SecretFiles, existingSecretFiles)
	if !ok {
		return model.DeploymentTarget{}, false
	}
	secretFilesContent, err := json.Marshal(secretFiles)
	if err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return model.DeploymentTarget{}, false
	}
	for _, volume := range dataVolumes {
		if runtimeDataPathConflicts(volume.MountPath, configFiles, string(secretFilesContent)) {
			writeError(ctx, http.StatusBadRequest, "运行数据目录不能与配置文件或密钥文件挂载路径重叠")
			return model.DeploymentTarget{}, false
		}
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = environment.Name
	}
	return model.DeploymentTarget{
		ID:                   targetID,
		ProjectID:            app.ProjectID,
		ApplicationID:        app.ID,
		EnvironmentID:        environment.ID,
		Name:                 name,
		SourceType:           sourceType,
		RepositoryBindingID:  repositoryBindingID,
		BuildProviderID:      strings.TrimSpace(input.BuildProviderID),
		DockerfilePath:       fallback(strings.TrimSpace(input.DockerfilePath), "Dockerfile"),
		BuildContext:         fallback(strings.TrimSpace(input.BuildContext), "."),
		BuildDirectory:       strings.TrimSpace(input.BuildDirectory),
		TargetRegistryID:     strings.TrimSpace(input.TargetRegistryID),
		TargetRepository:     targetRepository,
		TargetTag:            fallback(targetTag, "latest"),
		ImageRef:             strings.TrimSpace(input.ImageRef),
		BuildLabels:          strings.Join(normalizeBuildSelectorList(strings.Split(input.BuildLabels, ",")), ","),
		BuildVariableSetIDs:  encodeBuildVariableSetIDs(input.BuildVariableSetIDs),
		BuildHooksEnabled:    buildHooksEnabled,
		AutoDeploy:           input.AutoDeploy,
		BranchPattern:        strings.TrimSpace(input.BranchPattern),
		TagPattern:           strings.TrimSpace(input.TagPattern),
		ConcurrencyPolicy:    normalizeBuildConcurrencyPolicy(input.ConcurrencyPolicy),
		RuntimeConfigSetIDs:  encodeBuildVariableSetIDs(runtimeConfigSetIDs),
		EnvVars:              strings.TrimSpace(input.EnvVars),
		ConfigRefs:           strings.TrimSpace(input.ConfigRefs),
		SecretRefs:           normalizeSecretRefsInput(input.SecretRefs),
		ConfigFiles:          configFiles,
		SecretFiles:          string(secretFilesContent),
		DataRetentionEnabled: input.DataRetentionEnabled,
		DataCapacity:         dataCapacity,
		DataMountPath:        dataMountPath,
		DataVolumes:          encodeDataVolumes(dataVolumes),
		RequireApproval:      input.RequireApproval,
		Enabled:              input.Enabled,
		CreatedBy:            user.ID,
	}, true
}

func normalizeDataCapacity(ctx *gin.Context, value string, enabled bool) (string, bool) {
	normalized := strings.TrimSpace(value)
	if !enabled {
		return "", true
	}
	if normalized == "" {
		normalized = "1Gi"
	}
	quantity, err := resource.ParseQuantity(normalized)
	if err != nil || quantity.Sign() <= 0 {
		writeError(ctx, http.StatusBadRequest, "运行数据容量格式无效，例如 1Gi 或 10Gi")
		return "", false
	}
	return normalized, true
}

func normalizeDataMountPath(ctx *gin.Context, value string, enabled bool) (string, bool) {
	normalized := strings.TrimSpace(value)
	if !enabled {
		return "", true
	}
	if normalized == "" {
		normalized = "/data"
	}
	if !strings.HasPrefix(normalized, "/") {
		writeError(ctx, http.StatusBadRequest, "运行数据目录必须使用容器内绝对路径，例如 /data")
		return "", false
	}
	cleaned := path.Clean(normalized)
	if cleaned == "/" {
		writeError(ctx, http.StatusBadRequest, "运行数据目录不能是根目录")
		return "", false
	}
	return cleaned, true
}

func normalizeDataVolumes(ctx *gin.Context, value string, enabled bool, fallbackMountPath string, fallbackCapacity string) ([]deploymentTargetDataVolumeInput, bool) {
	if !enabled {
		return nil, true
	}
	normalized := strings.TrimSpace(value)
	if normalized == "" || normalized == "[]" {
		return []deploymentTargetDataVolumeInput{{
			Name:      "data",
			MountPath: fallback(fallbackMountPath, "/data"),
			Capacity:  fallback(fallbackCapacity, "1Gi"),
		}}, true
	}
	if !strings.HasPrefix(normalized, "[") {
		writeError(ctx, http.StatusBadRequest, "运行数据卷必须使用数组格式")
		return nil, false
	}
	var raw []deploymentTargetDataVolumeInput
	if err := json.Unmarshal([]byte(normalized), &raw); err != nil {
		writeError(ctx, http.StatusBadRequest, "运行数据卷格式无效")
		return nil, false
	}
	if len(raw) == 0 {
		writeError(ctx, http.StatusBadRequest, "启用运行数据后至少需要一个数据卷")
		return nil, false
	}
	seenNames := map[string]bool{}
	seenMountPaths := []string{}
	volumes := make([]deploymentTargetDataVolumeInput, 0, len(raw))
	for index, item := range raw {
		mountPath, ok := normalizeDataMountPath(ctx, item.MountPath, true)
		if !ok {
			return nil, false
		}
		name := normalizeDataVolumeName(item.Name, mountPath, index)
		if seenNames[name] {
			writeError(ctx, http.StatusBadRequest, "运行数据卷标识不能重复")
			return nil, false
		}
		for _, existingPath := range seenMountPaths {
			if mountPath == existingPath || strings.HasPrefix(mountPath, existingPath+"/") || strings.HasPrefix(existingPath, mountPath+"/") {
				writeError(ctx, http.StatusBadRequest, "运行数据目录不能重复或互相嵌套")
				return nil, false
			}
		}
		capacity := fallback(strings.TrimSpace(item.Capacity), "1Gi")
		if _, ok := normalizeDataCapacity(ctx, capacity, true); !ok {
			return nil, false
		}
		seenNames[name] = true
		seenMountPaths = append(seenMountPaths, mountPath)
		volumes = append(volumes, deploymentTargetDataVolumeInput{Name: name, MountPath: mountPath, Capacity: capacity})
	}
	return volumes, true
}

func normalizeDataVolumeName(value string, mountPath string, index int) string {
	if strings.TrimSpace(value) != "" {
		return runtimeDNSLabel(value)
	}
	base := path.Base(mountPath)
	if base == "." || base == "/" || base == "" {
		base = fmt.Sprintf("data-%d", index+1)
	}
	return runtimeDNSLabel(base)
}

func encodeDataVolumes(volumes []deploymentTargetDataVolumeInput) string {
	if len(volumes) == 0 {
		return ""
	}
	content, err := json.Marshal(volumes)
	if err != nil {
		return ""
	}
	return string(content)
}

func runtimeDataPathConflicts(mountPath string, configValues ...string) bool {
	for _, value := range configValues {
		for _, filePath := range runtimeConfigFilePaths(value) {
			if filePath == mountPath || strings.HasPrefix(filePath, mountPath+"/") || strings.HasPrefix(mountPath, filePath+"/") {
				return true
			}
		}
	}
	return false
}

func runtimeConfigFilePaths(value string) []string {
	normalized := strings.TrimSpace(value)
	if normalized == "" || normalized == "[]" || normalized == "{}" || !strings.HasPrefix(normalized, "[") {
		return nil
	}
	var raw []runtimeConfigFileInput
	if err := json.Unmarshal([]byte(normalized), &raw); err != nil {
		return nil
	}
	paths := make([]string, 0, len(raw))
	for _, item := range raw {
		filePath := strings.TrimSpace(item.Path)
		if filePath == "" || !strings.HasPrefix(filePath, "/") {
			continue
		}
		paths = append(paths, path.Clean(filePath))
	}
	return paths
}

func (h *Handlers) syncDeploymentTargetDataVolume(ctx *gin.Context, target model.DeploymentTarget) bool {
	if !target.DataRetentionEnabled {
		return true
	}
	var project model.Project
	if err := h.db.First(&project, "id = ?", target.ProjectID).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "project not found")
		return false
	}
	var environment model.Environment
	if err := h.db.First(&environment, "id = ? and project_id = ?", target.EnvironmentID, target.ProjectID).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "environment not found")
		return false
	}
	client, namespace, ok := h.kubernetesClientForEnvironment(ctx, project, environment, "运行集群不可用，无法同步运行数据容量")
	if !ok {
		return false
	}
	requestCtx, cancel := context.WithTimeout(ctx.Request.Context(), 10*time.Second)
	defer cancel()
	if err := client.EnsureNamespace(requestCtx, namespace, kubeprovider.ProjectNamespaceLabels(project.ID)); err != nil {
		writeError(ctx, http.StatusBadGateway, "运行数据命名空间同步失败，请检查集群权限")
		return false
	}
	if err := client.ApplyPersistentDataVolume(requestCtx, kubeprovider.ApplicationResourcesSpec{
		Name:                 deploymentTargetResourceName(target),
		Namespace:            namespace,
		ProjectID:            target.ProjectID,
		ApplicationID:        target.ApplicationID,
		EnvironmentID:        target.EnvironmentID,
		DeploymentTargetID:   target.ID,
		DataRetentionEnabled: true,
		DataCapacity:         target.DataCapacity,
		DataMountPath:        deploymentTargetDataMountPath(target),
		DataVolumes:          deploymentTargetKubernetesDataVolumes(target),
	}); err != nil {
		writeError(ctx, http.StatusBadGateway, "运行数据容量同步失败，请检查集群是否支持扩容")
		return false
	}
	return true
}

func (h *Handlers) kubernetesClientForEnvironment(ctx *gin.Context, project model.Project, environment model.Environment, errorMessage string) (*kubeprovider.Client, string, bool) {
	managerCluster, ok := h.runtimeClusterForEnvironment(ctx, environment)
	if !ok {
		return nil, "", false
	}
	kubeconfig := h.secrets.Resolve(managerCluster.KubeconfigRef)
	if strings.TrimSpace(kubeconfig) == "" {
		writeError(ctx, http.StatusBadRequest, errorMessage)
		return nil, "", false
	}
	client, err := kubeprovider.NewClientFromKubeconfig(kubeconfig)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, "运行集群 kubeconfig 无效")
		return nil, "", false
	}
	namespace := runtimeProjectNamespace(project)
	return client, namespace, true
}

func normalizeDeploymentSourceType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "image":
		return "image"
	default:
		return "repository"
	}
}

func (h *Handlers) saveDeploymentTarget(target model.DeploymentTarget, hookInputs []deploymentTargetHookBindingInput) error {
	return h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&target).Error; err != nil {
			return err
		}
		return h.replaceDeploymentTargetHookBindings(tx, target, hookInputs)
	})
}

func (h *Handlers) attachDeploymentTargetHookBindings(targets []model.DeploymentTarget) error {
	if len(targets) == 0 {
		return nil
	}
	targetIDs := make([]string, 0, len(targets))
	targetIndex := make(map[string]int, len(targets))
	for index := range targets {
		targetIDs = append(targetIDs, targets[index].ID)
		targetIndex[targets[index].ID] = index
	}
	var bindings []model.DeploymentTargetHookBinding
	if err := h.db.Where("target_id in ?", targetIDs).Order("run_order asc, created_at asc").Find(&bindings).Error; err != nil {
		return err
	}
	for _, binding := range bindings {
		index, ok := targetIndex[binding.TargetID]
		if !ok {
			continue
		}
		targets[index].BuildHookBindings = append(targets[index].BuildHookBindings, binding)
	}
	return nil
}

func (h *Handlers) deploymentTargetWithHookBindings(target model.DeploymentTarget) (model.DeploymentTarget, error) {
	targets := []model.DeploymentTarget{target}
	if err := h.attachDeploymentTargetHookBindings(targets); err != nil {
		return target, err
	}
	return targets[0], nil
}

func (h *Handlers) replaceDeploymentTargetHookBindings(tx *gorm.DB, target model.DeploymentTarget, inputs []deploymentTargetHookBindingInput) error {
	if err := tx.Where("target_id = ?", target.ID).Delete(&model.DeploymentTargetHookBinding{}).Error; err != nil {
		return err
	}
	if len(inputs) == 0 {
		return nil
	}
	hookIDs := make([]string, 0, len(inputs))
	seen := make(map[string]bool, len(inputs))
	for _, input := range inputs {
		hookID := strings.TrimSpace(input.HookConfigID)
		phase := normalizeHookPhase(input.Phase)
		if hookID == "" || phase == "" {
			continue
		}
		key := phase + "\x00" + hookID
		if seen[key] {
			continue
		}
		seen[key] = true
		hookIDs = append(hookIDs, hookID)
	}
	if len(hookIDs) == 0 {
		return nil
	}
	var hooks []model.ProjectHookConfig
	if err := tx.Where("project_id = ? and id in ?", target.ProjectID, hookIDs).Find(&hooks).Error; err != nil {
		return err
	}
	validHookIDs := make(map[string]bool, len(hooks))
	for _, hook := range hooks {
		validHookIDs[hook.ID] = true
	}
	bindings := make([]model.DeploymentTargetHookBinding, 0, len(seen))
	created := make(map[string]bool, len(seen))
	for index, input := range inputs {
		hookID := strings.TrimSpace(input.HookConfigID)
		phase := normalizeHookPhase(input.Phase)
		if hookID == "" || phase == "" {
			continue
		}
		key := phase + "\x00" + hookID
		if created[key] {
			continue
		}
		created[key] = true
		if !validHookIDs[hookID] {
			return errors.New("构建钩子不存在")
		}
		bindings = append(bindings, model.DeploymentTargetHookBinding{
			ID:            id.New("dtmhb"),
			ProjectID:     target.ProjectID,
			ApplicationID: target.ApplicationID,
			TargetID:      target.ID,
			HookConfigID:  hookID,
			Phase:         phase,
			RunOrder:      index + 1,
		})
	}
	return tx.Create(&bindings).Error
}

func normalizeSecretRefsInput(value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "{}" {
		return ""
	}
	return normalized
}

func runtimeProjectNamespace(project model.Project) string {
	return runtimeIDResourceName("ns", project.ID)
}

func deploymentTargetResourceName(target model.DeploymentTarget) string {
	return runtimeIDResourceName("dplt", target.ID)
}

func deploymentTargetDataMountPath(target model.DeploymentTarget) string {
	return fallback(strings.TrimSpace(target.DataMountPath), "/data")
}

func deploymentTargetDataVolumes(target model.DeploymentTarget) []deploymentTargetDataVolumeInput {
	normalized := strings.TrimSpace(target.DataVolumes)
	if normalized == "" || normalized == "[]" {
		if !target.DataRetentionEnabled {
			return nil
		}
		return []deploymentTargetDataVolumeInput{{
			Name:      "data",
			MountPath: deploymentTargetDataMountPath(target),
			Capacity:  fallback(strings.TrimSpace(target.DataCapacity), "1Gi"),
		}}
	}
	var volumes []deploymentTargetDataVolumeInput
	if err := json.Unmarshal([]byte(normalized), &volumes); err != nil {
		return nil
	}
	return volumes
}

func deploymentTargetKubernetesDataVolumes(target model.DeploymentTarget) []kubeprovider.ApplicationDataVolume {
	volumes := deploymentTargetDataVolumes(target)
	output := make([]kubeprovider.ApplicationDataVolume, 0, len(volumes))
	for _, volume := range volumes {
		output = append(output, kubeprovider.ApplicationDataVolume{
			Name:      volume.Name,
			MountPath: volume.MountPath,
			Capacity:  volume.Capacity,
		})
	}
	return output
}

func deploymentTargetDataExportVolumes(target model.DeploymentTarget) []kubeprovider.DataExportVolume {
	resourceName := deploymentTargetResourceName(target)
	volumes := deploymentTargetDataVolumes(target)
	if len(volumes) == 0 && target.DataRetentionEnabled {
		volumes = []deploymentTargetDataVolumeInput{{Name: "data"}}
	}
	output := make([]kubeprovider.DataExportVolume, 0, len(volumes))
	for _, volume := range volumes {
		name := normalizeDataVolumeName(volume.Name, volume.MountPath, len(output))
		pvcName := resourceName + "-data"
		if name != "data" {
			pvcName = runtimeDNSLabel(resourceName + "-" + name + "-data")
		}
		output = append(output, kubeprovider.DataExportVolume{Name: name, PVCName: pvcName})
	}
	return output
}

func shortResourceID(value string) string {
	return runtimeShortID(value)
}

func runtimeIDResourceName(prefix string, value string) string {
	suffix := runtimeShortID(value)
	if suffix == "" {
		return runtimeDNSLabel(prefix)
	}
	return runtimeDNSLabel(prefix + "-" + suffix)
}

func runtimeShortID(value string) string {
	value = strings.TrimSpace(value)
	if index := strings.Index(value, "_"); index >= 0 {
		value = value[index+1:]
	}
	value = runtimeDNSLabel(value)
	if len(value) > 10 {
		return value[:10]
	}
	return value
}

func runtimeDNSLabel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = apiDNSLabelInvalidPattern.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "app"
	}
	if len(value) > 63 {
		value = strings.Trim(value[:63], "-")
	}
	if value == "" {
		return "app"
	}
	return value
}

type deploymentTargetResponse struct {
	ID                   string                              `json:"id"`
	ProjectID            string                              `json:"projectId"`
	ApplicationID        string                              `json:"applicationId"`
	EnvironmentID        string                              `json:"environmentId"`
	Name                 string                              `json:"name"`
	SourceType           string                              `json:"sourceType"`
	RepositoryBindingID  string                              `json:"repositoryBindingId"`
	BuildProviderID      string                              `json:"buildProviderId"`
	DockerfilePath       string                              `json:"dockerfilePath"`
	BuildContext         string                              `json:"buildContext"`
	BuildDirectory       string                              `json:"buildDirectory"`
	TargetRegistryID     string                              `json:"targetRegistryId"`
	TargetRepository     string                              `json:"targetRepository"`
	TargetTag            string                              `json:"targetTag"`
	ImageRef             string                              `json:"imageRef"`
	BuildLabels          string                              `json:"buildLabels"`
	BuildVariableSetIDs  string                              `json:"buildVariableSetIds"`
	BuildHooksEnabled    bool                                `json:"buildHooksEnabled"`
	BuildHookBindings    []model.DeploymentTargetHookBinding `json:"buildHookBindings"`
	AutoDeploy           bool                                `json:"autoDeploy"`
	BranchPattern        string                              `json:"branchPattern"`
	TagPattern           string                              `json:"tagPattern"`
	ConcurrencyPolicy    string                              `json:"concurrencyPolicy"`
	RuntimeConfigSetIDs  string                              `json:"runtimeConfigSetIds"`
	EnvVars              string                              `json:"envVars"`
	ConfigRefs           string                              `json:"configRefs"`
	SecretRefsSet        bool                                `json:"secretRefsSet"`
	ConfigFiles          string                              `json:"configFiles"`
	SecretFilesSet       bool                                `json:"secretFilesSet"`
	DataRetentionEnabled bool                                `json:"dataRetentionEnabled"`
	DataCapacity         string                              `json:"dataCapacity"`
	DataMountPath        string                              `json:"dataMountPath"`
	DataVolumes          string                              `json:"dataVolumes"`
	RequireApproval      bool                                `json:"requireApproval"`
	Enabled              bool                                `json:"enabled"`
	CreatedBy            string                              `json:"createdBy"`
	CreatedAt            time.Time                           `json:"createdAt"`
}

func deploymentTargetResponses(targets []model.DeploymentTarget) []deploymentTargetResponse {
	responses := make([]deploymentTargetResponse, 0, len(targets))
	for _, target := range targets {
		responses = append(responses, deploymentTargetResponseFromModel(target))
	}
	return responses
}

func deploymentTargetResponseFromModel(target model.DeploymentTarget) deploymentTargetResponse {
	return deploymentTargetResponse{
		ID:                   target.ID,
		ProjectID:            target.ProjectID,
		ApplicationID:        target.ApplicationID,
		EnvironmentID:        target.EnvironmentID,
		Name:                 target.Name,
		SourceType:           normalizeDeploymentSourceType(target.SourceType),
		RepositoryBindingID:  target.RepositoryBindingID,
		BuildProviderID:      target.BuildProviderID,
		DockerfilePath:       target.DockerfilePath,
		BuildContext:         target.BuildContext,
		BuildDirectory:       target.BuildDirectory,
		TargetRegistryID:     target.TargetRegistryID,
		TargetRepository:     target.TargetRepository,
		TargetTag:            target.TargetTag,
		ImageRef:             target.ImageRef,
		BuildLabels:          target.BuildLabels,
		BuildVariableSetIDs:  target.BuildVariableSetIDs,
		BuildHooksEnabled:    target.BuildHooksEnabled,
		BuildHookBindings:    target.BuildHookBindings,
		AutoDeploy:           target.AutoDeploy,
		BranchPattern:        target.BranchPattern,
		TagPattern:           target.TagPattern,
		ConcurrencyPolicy:    target.ConcurrencyPolicy,
		RuntimeConfigSetIDs:  target.RuntimeConfigSetIDs,
		EnvVars:              target.EnvVars,
		ConfigRefs:           target.ConfigRefs,
		SecretRefsSet:        strings.TrimSpace(target.SecretRefs) != "",
		ConfigFiles:          target.ConfigFiles,
		SecretFilesSet:       strings.TrimSpace(target.SecretFiles) != "" && strings.TrimSpace(target.SecretFiles) != "{}",
		DataRetentionEnabled: target.DataRetentionEnabled,
		DataCapacity:         target.DataCapacity,
		DataMountPath:        deploymentTargetDataMountPath(target),
		DataVolumes:          encodeDataVolumes(deploymentTargetDataVolumes(target)),
		RequireApproval:      target.RequireApproval,
		Enabled:              target.Enabled,
		CreatedBy:            target.CreatedBy,
		CreatedAt:            target.CreatedAt,
	}
}

func releaseFromInput(projectID, userID string, input releaseInput, releaseID string) model.Release {
	return model.Release{
		ID:                 releaseID,
		ProjectID:          projectID,
		ApplicationID:      strings.TrimSpace(input.ApplicationID),
		EnvironmentID:      strings.TrimSpace(input.EnvironmentID),
		DeploymentTargetID: strings.TrimSpace(input.DeploymentTargetID),
		BuildRunID:         strings.TrimSpace(input.BuildRunID),
		ImageRef:           strings.TrimSpace(input.ImageRef),
		Type:               normalizeReleaseType(input.Type),
		Status:             fallback(strings.TrimSpace(input.Status), "pending"),
		Revision:           fallbackInt(input.Revision, 1),
		Message:            strings.TrimSpace(input.Message),
		CreatedBy:          userID,
	}
}

func rollbackReleaseFromTarget(source model.Release, target model.Release, userID string, revision int) model.Release {
	return model.Release{
		ProjectID:          source.ProjectID,
		ApplicationID:      source.ApplicationID,
		EnvironmentID:      source.EnvironmentID,
		DeploymentTargetID: source.DeploymentTargetID,
		BuildRunID:         target.BuildRunID,
		ImageRef:           target.ImageRef,
		Type:               "rollback",
		Status:             "pending",
		Revision:           fallbackInt(revision, source.Revision+1),
		RollbackFromID:     target.ID,
		CreatedBy:          userID,
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
	Name                 string                             `json:"name"`
	EnvironmentID        string                             `json:"environmentId" binding:"required"`
	SourceType           string                             `json:"sourceType"`
	RepositoryBindingID  string                             `json:"repositoryBindingId"`
	BuildProviderID      string                             `json:"buildProviderId"`
	DockerfilePath       string                             `json:"dockerfilePath"`
	BuildContext         string                             `json:"buildContext"`
	BuildDirectory       string                             `json:"buildDirectory"`
	TargetRegistryID     string                             `json:"targetRegistryId"`
	TargetImageRef       string                             `json:"targetImageRef"`
	TargetRepository     string                             `json:"targetRepository"`
	TargetTag            string                             `json:"targetTag"`
	ImageRef             string                             `json:"imageRef"`
	BuildLabels          string                             `json:"buildLabels"`
	BuildVariableSetIDs  []string                           `json:"buildVariableSetIds"`
	BuildHooksEnabled    *bool                              `json:"buildHooksEnabled"`
	BuildHookBindings    []deploymentTargetHookBindingInput `json:"buildHookBindings"`
	AutoDeploy           bool                               `json:"autoDeploy"`
	BranchPattern        string                             `json:"branchPattern"`
	TagPattern           string                             `json:"tagPattern"`
	ConcurrencyPolicy    string                             `json:"concurrencyPolicy"`
	RuntimeConfigSetIDs  []string                           `json:"runtimeConfigSetIds"`
	EnvVars              string                             `json:"envVars"`
	ConfigRefs           string                             `json:"configRefs"`
	SecretRefs           string                             `json:"secretRefs"`
	ConfigFiles          string                             `json:"configFiles"`
	SecretFiles          string                             `json:"secretFiles"`
	DataRetentionEnabled bool                               `json:"dataRetentionEnabled"`
	DataCapacity         string                             `json:"dataCapacity"`
	DataMountPath        string                             `json:"dataMountPath"`
	DataVolumes          string                             `json:"dataVolumes"`
	RequireApproval      bool                               `json:"requireApproval"`
	Enabled              bool                               `json:"enabled"`
}

type deploymentTargetDataVolumeInput struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	Capacity  string `json:"capacity"`
}

type releaseInput struct {
	ApplicationID      string `json:"applicationId" binding:"required"`
	EnvironmentID      string `json:"environmentId"`
	DeploymentTargetID string `json:"deploymentTargetId" binding:"required"`
	BuildRunID         string `json:"buildRunId"`
	ImageRef           string `json:"imageRef"`
	Type               string `json:"type"`
	Status             string `json:"status"`
	Revision           int    `json:"revision"`
	Message            string `json:"message"`
}

type releaseRuntimeExecInput struct {
	Command   string `json:"command"`
	Container string `json:"container"`
}

type runtimeTerminalClientMessage struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

type runtimeTerminalSizeQueue struct {
	ch chan remotecommand.TerminalSize
}

func newRuntimeTerminalSizeQueue() *runtimeTerminalSizeQueue {
	return &runtimeTerminalSizeQueue{ch: make(chan remotecommand.TerminalSize, 8)}
}

func (q *runtimeTerminalSizeQueue) Push(cols uint16, rows uint16) {
	if q == nil || cols == 0 || rows == 0 {
		return
	}
	size := remotecommand.TerminalSize{Width: cols, Height: rows}
	select {
	case q.ch <- size:
	default:
		select {
		case <-q.ch:
		default:
		}
		q.ch <- size
	}
}

func (q *runtimeTerminalSizeQueue) Next() *remotecommand.TerminalSize {
	if q == nil {
		return nil
	}
	size, ok := <-q.ch
	if !ok {
		return nil
	}
	return &size
}

type runtimeTerminalWebSocketWriter struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (w *runtimeTerminalWebSocketWriter) Write(data []byte) (int, error) {
	if w == nil || w.conn == nil {
		return 0, io.ErrClosedPipe
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
		return 0, err
	}
	return len(data), nil
}

type deploymentTargetHookBindingInput struct {
	HookConfigID string `json:"hookConfigId"`
	Phase        string `json:"phase"`
	RunOrder     int    `json:"runOrder"`
}
