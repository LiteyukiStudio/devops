package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/LiteyukiStudio/devops/internal/id"
	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/LiteyukiStudio/devops/internal/tasks"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) ListGatewayRoutes(ctx *gin.Context) {
	if _, ok := h.findProjectForCurrentUser(ctx); !ok {
		return
	}
	query := h.db.Where("project_id = ?", ctx.Param("projectId")).Order("created_at desc")
	query = applySearch(ctx, query, "host", "path", "status")
	var routes []model.GatewayRoute
	if err := query.Find(&routes).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, routes)
}

func (h *Handlers) CreateGatewayRoute(ctx *gin.Context) {
	user, project, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin", "developer")
	if !ok {
		return
	}
	var input gatewayRouteInput
	if !bindJSON(ctx, &input) {
		return
	}
	route, ok := h.gatewayRouteFromInput(ctx, project, user.ID, input, "")
	if !ok {
		return
	}
	route.ID = id.New("gwr")
	if err := h.db.Create(&route).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if !h.enqueueGatewayApply(ctx.Request.Context(), route) {
		route.Status = "failed"
		if err := h.db.Save(&route).Error; err != nil {
			writeError(ctx, http.StatusInternalServerError, err.Error())
			return
		}
		writeError(ctx, http.StatusServiceUnavailable, "网关任务投递失败，请稍后重试")
		return
	}
	ctx.JSON(http.StatusCreated, route)
}

func (h *Handlers) UpdateGatewayRoute(ctx *gin.Context) {
	_, project, ok := h.projectAndCurrentUserWithRoles(ctx, "owner", "admin", "developer")
	if !ok {
		return
	}
	route, ok := h.findGatewayRoute(ctx)
	if !ok {
		return
	}
	var input gatewayRouteInput
	if !bindJSON(ctx, &input) {
		return
	}
	next, ok := h.gatewayRouteFromInput(ctx, project, route.CreatedBy, input, route.ID)
	if !ok {
		return
	}
	route.ApplicationID = next.ApplicationID
	route.EnvironmentID = next.EnvironmentID
	route.Host = next.Host
	route.Path = next.Path
	route.ServicePort = next.ServicePort
	route.TLSMode = next.TLSMode
	route.CertificateStatus = next.CertificateStatus
	route.CNAMEName = next.CNAMEName
	route.CNAMETarget = next.CNAMETarget
	route.DNSStatus = next.DNSStatus
	route.Status = next.Status
	route.IsDefault = next.IsDefault
	if err := h.db.Save(&route).Error; err != nil {
		writeError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if !h.enqueueGatewayApply(ctx.Request.Context(), route) {
		route.Status = "failed"
		if err := h.db.Save(&route).Error; err != nil {
			writeError(ctx, http.StatusInternalServerError, err.Error())
			return
		}
		writeError(ctx, http.StatusServiceUnavailable, "网关任务投递失败，请稍后重试")
		return
	}
	ctx.JSON(http.StatusOK, route)
}

func (h *Handlers) DeleteGatewayRoute(ctx *gin.Context) {
	if _, ok := h.findProjectForCurrentUserWithRoles(ctx, "owner", "admin"); !ok {
		return
	}
	route, ok := h.findGatewayRoute(ctx)
	if !ok {
		return
	}
	if err := h.db.Delete(&route).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (h *Handlers) CheckGatewayDomain(ctx *gin.Context) {
	if _, ok := h.findProjectForCurrentUser(ctx); !ok {
		return
	}
	host := strings.ToLower(strings.TrimSpace(ctx.Query("host")))
	if host == "" {
		writeError(ctx, http.StatusBadRequest, "请输入域名")
		return
	}
	var count int64
	if err := h.db.Model(&model.GatewayRoute{}).
		Where("host = ? and project_id <> ?", host, ctx.Param("projectId")).
		Count(&count).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"available": count == 0, "host": host})
}

func (h *Handlers) findGatewayRoute(ctx *gin.Context) (model.GatewayRoute, bool) {
	var route model.GatewayRoute
	if err := h.db.First(&route, "id = ? and project_id = ?", ctx.Param("routeId"), ctx.Param("projectId")).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "gateway route not found")
		return route, false
	}
	return route, true
}

func (h *Handlers) enqueueGatewayApply(ctx context.Context, route model.GatewayRoute) bool {
	if h.taskClient == nil {
		return false
	}
	_, err := h.taskClient.EnqueueGatewayApply(ctx, tasks.GatewayApplyPayload{
		GatewayRouteID: route.ID,
		ProjectID:      route.ProjectID,
		ActorID:        route.CreatedBy,
	})
	return err == nil
}

func (h *Handlers) gatewayRouteFromInput(ctx *gin.Context, project model.Project, userID string, input gatewayRouteInput, routeID string) (model.GatewayRoute, bool) {
	host := strings.ToLower(strings.TrimSpace(input.Host))
	if host == "" {
		host = h.defaultGatewayHost(project, input.Stage, input.ApplicationSlug)
	}
	if host == "" {
		writeError(ctx, http.StatusBadRequest, "请输入域名或选择应用")
		return model.GatewayRoute{}, false
	}
	var count int64
	query := h.db.Model(&model.GatewayRoute{}).Where("host = ? and id <> ?", host, routeID)
	if err := query.Count(&count).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return model.GatewayRoute{}, false
	}
	if count > 0 {
		writeError(ctx, http.StatusBadRequest, "域名已被占用")
		return model.GatewayRoute{}, false
	}

	tlsMode := normalizeTLSMode(input.TLSMode)
	certStatus := "disabled"
	if tlsMode != "http-only" {
		certStatus = fallback(strings.TrimSpace(input.CertificateStatus), "pending")
	}
	return model.GatewayRoute{
		ID:                routeID,
		ProjectID:         project.ID,
		ApplicationID:     strings.TrimSpace(input.ApplicationID),
		EnvironmentID:     strings.TrimSpace(input.EnvironmentID),
		Host:              host,
		Path:              fallback(strings.TrimSpace(input.Path), "/"),
		ServicePort:       fallbackInt(input.ServicePort, 80),
		TLSMode:           tlsMode,
		CertificateStatus: certStatus,
		CNAMEName:         host,
		CNAMETarget:       h.gatewayCNAMETarget(project),
		DNSStatus:         fallback(strings.TrimSpace(input.DNSStatus), "pending"),
		Status:            fallback(strings.TrimSpace(input.Status), "pending"),
		IsDefault:         input.IsDefault,
		CreatedBy:         userID,
	}, true
}

func (h *Handlers) defaultGatewayHost(project model.Project, stage, applicationSlug string) string {
	rootDomain := strings.TrimSpace(h.configValue("gateway.rootDomain"))
	if rootDomain == "" {
		rootDomain = "apps.local"
	}
	appSlug := strings.Trim(strings.ToLower(strings.TrimSpace(applicationSlug)), "-")
	if appSlug == "" {
		return ""
	}
	return fmt.Sprintf("%s-%s.%s.%s", appSlug, project.Slug, normalizeStage(stage), strings.TrimPrefix(rootDomain, "."))
}

func (h *Handlers) gatewayCNAMETarget(project model.Project) string {
	rootDomain := strings.TrimSpace(h.configValue("gateway.rootDomain"))
	if rootDomain == "" {
		rootDomain = "apps.local"
	}
	return fmt.Sprintf("*.%s.%s", project.Slug, strings.TrimPrefix(rootDomain, "."))
}

func (h *Handlers) configValue(key string) string {
	values := h.configs.get([]string{key})
	return values[key]
}

func normalizeTLSMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "http-challenge", "manual-cert":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "http-only"
	}
}

type gatewayRouteInput struct {
	ApplicationID     string `json:"applicationId" binding:"required"`
	ApplicationSlug   string `json:"applicationSlug"`
	EnvironmentID     string `json:"environmentId"`
	Stage             string `json:"stage"`
	Host              string `json:"host"`
	Path              string `json:"path"`
	ServicePort       int    `json:"servicePort"`
	TLSMode           string `json:"tlsMode"`
	CertificateStatus string `json:"certificateStatus"`
	DNSStatus         string `json:"dnsStatus"`
	Status            string `json:"status"`
	IsDefault         bool   `json:"isDefault"`
}
