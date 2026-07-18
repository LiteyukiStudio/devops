package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/LiteyukiStudio/devops/internal/dependency"
	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/LiteyukiStudio/devops/internal/notification"
	kubeprovider "github.com/LiteyukiStudio/devops/internal/provider/kubernetes"
	"github.com/LiteyukiStudio/devops/internal/resourcename"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) dependencyService() *dependency.Service {
	return dependency.NewService(dependency.NewGormRepository(h.db))
}

func (h *Handlers) ListServiceBindings(ctx *gin.Context) {
	if _, ok := h.findProjectForCurrentUser(ctx); !ok {
		return
	}
	pagination := dependencyPagination(ctx, map[string]bool{
		"createdAt": true, "updatedAt": true, "protocol": true, "enabled": true,
	}, "createdAt")
	bindings, total, err := h.dependencyService().ListServiceBindings(ctx.Request.Context(), ctx.Param("projectId"), dependencyListOptions(pagination))
	if err != nil {
		writeDependencyError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, paginatedResponse(bindings, total, pagination))
}

func (h *Handlers) CreateServiceBinding(ctx *gin.Context) {
	user, project, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin")
	if !ok {
		return
	}
	var input dependency.ServiceBindingInput
	if !bindJSON(ctx, &input) {
		return
	}
	binding, err := h.dependencyService().CreateServiceBinding(ctx.Request.Context(), project.ID, user.ID, input)
	if err != nil {
		h.audit(user.ID, "service_binding.create", project.ID, false, dependencyAuditMessage(err))
		writeDependencyError(ctx, err)
		return
	}
	h.audit(user.ID, "service_binding.create", binding.ID, true, binding.SourceDeploymentTargetID)
	h.emitServiceBindingEvent(ctx.Request.Context(), user, project, binding, "created", notification.SeverityInfo)
	ctx.JSON(http.StatusCreated, serviceBindingMutationResponseFor(binding))
}

func (h *Handlers) UpdateServiceBinding(ctx *gin.Context) {
	user, project, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin")
	if !ok {
		return
	}
	var input dependency.ServiceBindingInput
	if !bindJSON(ctx, &input) {
		return
	}
	bindingID := ctx.Param("bindingId")
	binding, err := h.dependencyService().UpdateServiceBinding(ctx.Request.Context(), project.ID, bindingID, input)
	if err != nil {
		h.audit(user.ID, "service_binding.update", bindingID, false, dependencyAuditMessage(err))
		writeDependencyError(ctx, err)
		return
	}
	h.audit(user.ID, "service_binding.update", binding.ID, true, binding.SourceDeploymentTargetID)
	h.emitServiceBindingEvent(ctx.Request.Context(), user, project, binding, "updated", notification.SeverityInfo)
	ctx.JSON(http.StatusOK, serviceBindingMutationResponseFor(binding))
}

func (h *Handlers) DeleteServiceBinding(ctx *gin.Context) {
	user, project, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin")
	if !ok {
		return
	}
	bindingID := ctx.Param("bindingId")
	binding, err := h.dependencyService().DeleteServiceBinding(ctx.Request.Context(), project.ID, bindingID)
	if err != nil {
		h.audit(user.ID, "service_binding.delete", bindingID, false, dependencyAuditMessage(err))
		writeDependencyError(ctx, err)
		return
	}
	h.audit(user.ID, "service_binding.delete", binding.ID, true, binding.SourceDeploymentTargetID)
	h.emitServiceBindingEvent(ctx.Request.Context(), user, project, binding, "deleted", notification.SeverityInfo)
	ctx.JSON(http.StatusOK, deletedServiceBindingMutationResponse(binding))
}

func (h *Handlers) CheckServiceBinding(ctx *gin.Context) {
	user, project, ok := h.projectAndCurrentUser(ctx)
	if !ok {
		return
	}
	service := h.dependencyService()
	binding, err := service.ServiceBinding(ctx.Request.Context(), project.ID, ctx.Param("bindingId"))
	if err != nil {
		writeDependencyError(ctx, err)
		return
	}
	result, err := service.CheckServiceBinding(ctx.Request.Context(), project.ID, binding.ID)
	if err != nil {
		writeDependencyError(ctx, err)
		return
	}
	if result.Status != "invalid" {
		var sourceTarget, targetTarget model.DeploymentTarget
		if err := h.db.WithContext(ctx).First(&sourceTarget, "id = ? and project_id = ?", binding.SourceDeploymentTargetID, project.ID).Error; err != nil {
			writeErrorCode(ctx, http.StatusNotFound, dependency.CodeNotFound, "source deployment target not found")
			return
		}
		if err := h.db.WithContext(ctx).First(&targetTarget, "id = ? and project_id = ?", binding.TargetDeploymentTargetID, project.ID).Error; err != nil {
			writeErrorCode(ctx, http.StatusNotFound, dependency.CodeNotFound, "target deployment target not found")
			return
		}
		client, _, clientOK := h.kubernetesClientForDeploymentTarget(ctx, project, targetTarget, "runtime_cluster_unavailable")
		if !clientOK {
			return
		}
		readContext, cancel := context.WithTimeout(ctx.Request.Context(), 12*time.Second)
		diagnostic, diagnosticErr := client.CheckServiceDependency(readContext, kubeprovider.ServiceDependencyCheckOptions{
			SourceNamespace: runtimeProjectNamespace(project),
			TargetNamespace: runtimeProjectNamespace(project),
			ServiceName:     resourcename.DeploymentTarget(targetTarget.ID),
			PortName:        binding.TargetPortName,
			PortNumber:      int32(binding.TargetPort),
		})
		cancel()
		if diagnosticErr != nil {
			result.Status = "unavailable"
			result.Checks = append(result.Checks, dependency.BindingCheckItem{Code: "kubernetes_check", Status: "failed", Detail: diagnosticErr.Error()})
		} else {
			for _, check := range diagnostic.Checks {
				result.Checks = append(result.Checks, dependency.BindingCheckItem{
					Code: check.Code, Status: string(check.Status), Resource: fmt.Sprintf("%s/%s", diagnostic.TargetNamespace, diagnostic.ServiceName),
				})
				switch {
				case check.Code == kubeprovider.ServiceDependencyCheckServicePortResolved && check.Status == kubeprovider.ServiceDependencyCheckFailed:
					result.Status = "invalid"
				case (check.Code == kubeprovider.ServiceDependencyCheckServiceExists || check.Code == kubeprovider.ServiceDependencyCheckEndpointReady) && check.Status == kubeprovider.ServiceDependencyCheckFailed && result.Status != "invalid":
					result.Status = "unavailable"
				}
			}
		}
	}
	previousStatus := strings.TrimSpace(binding.LastCheckStatus)
	if err := service.RecordServiceBindingCheck(ctx.Request.Context(), binding.ID, result.Status, result.CheckedAt); err == nil {
		if result.Status == "invalid" && previousStatus != "invalid" {
			h.emitServiceBindingEvent(ctx.Request.Context(), user, project, binding, "invalid", notification.SeverityError)
		} else if previousStatus == "invalid" && (result.Status == "ready" || result.Status == "pending_release") {
			h.emitServiceBindingEvent(ctx.Request.Context(), user, project, binding, "recovered", notification.SeverityInfo)
		}
	}
	ctx.JSON(http.StatusOK, result)
}

func (h *Handlers) ListProjectTopologyEdges(ctx *gin.Context) {
	if _, ok := h.findProjectForCurrentUser(ctx); !ok {
		return
	}
	pagination := dependencyPagination(ctx, map[string]bool{
		"createdAt": true, "updatedAt": true, "relationType": true, "protocol": true,
	}, "createdAt")
	edges, total, err := h.dependencyService().ListTopologyEdges(ctx.Request.Context(), ctx.Param("projectId"), dependencyListOptions(pagination))
	if err != nil {
		writeDependencyError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, paginatedResponse(edges, total, pagination))
}

func (h *Handlers) CreateProjectTopologyEdge(ctx *gin.Context) {
	user, project, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin")
	if !ok {
		return
	}
	var input dependency.TopologyEdgeInput
	if !bindJSON(ctx, &input) {
		return
	}
	edge, err := h.dependencyService().CreateTopologyEdge(ctx.Request.Context(), project.ID, user.ID, input)
	if err != nil {
		h.audit(user.ID, "project_topology_edge.create", project.ID, false, dependencyAuditMessage(err))
		writeDependencyError(ctx, err)
		return
	}
	h.audit(user.ID, "project_topology_edge.create", edge.ID, true, edge.RelationType)
	ctx.JSON(http.StatusCreated, edge)
}

func (h *Handlers) UpdateProjectTopologyEdge(ctx *gin.Context) {
	user, project, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin")
	if !ok {
		return
	}
	var input dependency.TopologyEdgeInput
	if !bindJSON(ctx, &input) {
		return
	}
	edgeID := ctx.Param("edgeId")
	edge, err := h.dependencyService().UpdateTopologyEdge(ctx.Request.Context(), project.ID, edgeID, input)
	if err != nil {
		h.audit(user.ID, "project_topology_edge.update", edgeID, false, dependencyAuditMessage(err))
		writeDependencyError(ctx, err)
		return
	}
	h.audit(user.ID, "project_topology_edge.update", edge.ID, true, edge.RelationType)
	ctx.JSON(http.StatusOK, edge)
}

func (h *Handlers) DeleteProjectTopologyEdge(ctx *gin.Context) {
	user, project, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin")
	if !ok {
		return
	}
	edgeID := ctx.Param("edgeId")
	edge, err := h.dependencyService().DeleteTopologyEdge(ctx.Request.Context(), project.ID, edgeID)
	if err != nil {
		h.audit(user.ID, "project_topology_edge.delete", edgeID, false, dependencyAuditMessage(err))
		writeDependencyError(ctx, err)
		return
	}
	h.audit(user.ID, "project_topology_edge.delete", edge.ID, true, edge.RelationType)
	ctx.Status(http.StatusNoContent)
}

func dependencyListOptions(pagination paginationParams) dependency.ListOptions {
	return dependency.ListOptions{
		Page: pagination.Page, PageSize: pagination.PageSize, SortBy: pagination.SortBy, SortOrder: pagination.SortOrder,
	}
}

func serviceBindingMutationResponseFor(binding model.ServiceBinding) gin.H {
	return gin.H{
		"item":             binding,
		"requiresRedeploy": true,
		"affectedDeploymentTargets": []gin.H{{
			"applicationId": binding.SourceApplicationID, "deploymentTargetId": binding.SourceDeploymentTargetID,
		}},
	}
}

func deletedServiceBindingMutationResponse(binding model.ServiceBinding) gin.H {
	return gin.H{
		"requiresRedeploy": true,
		"affectedDeploymentTargets": []gin.H{{
			"applicationId": binding.SourceApplicationID, "deploymentTargetId": binding.SourceDeploymentTargetID,
		}},
	}
}

func dependencyPagination(ctx *gin.Context, allowed map[string]bool, fallback string) paginationParams {
	pagination := paginationFromQuery(ctx)
	if !allowed[pagination.SortBy] {
		pagination.SortBy = fallback
	}
	return pagination
}

func dependencyAuditMessage(err error) string {
	if code := dependency.ErrorCode(err); code != "" {
		return code
	}
	return "dependency operation failed"
}

func writeDependencyError(ctx *gin.Context, err error) {
	code := dependency.ErrorCode(err)
	status := http.StatusInternalServerError
	switch code {
	case dependency.CodeNotFound:
		status = http.StatusNotFound
	case dependency.CodeEnvConflict, dependency.CodeTopologyDuplicate:
		status = http.StatusConflict
	case dependency.CodeInvalidInput, dependency.CodeCrossProject, dependency.CodeCrossCluster,
		dependency.CodeSourceTargetSame, dependency.CodePortNotFound, dependency.CodeReservedEnv:
		status = http.StatusBadRequest
	}
	if code == "" {
		code = "dependency_operation_failed"
	}
	writeErrorCode(ctx, status, code, err.Error())
}
