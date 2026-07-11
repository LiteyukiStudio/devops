package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/LiteyukiStudio/devops/internal/billing"
	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/gin-gonic/gin"
)

type gatewayTrafficStatusResponse struct {
	Available             bool       `json:"available"`
	Installed             bool       `json:"installed"`
	Status                string     `json:"status"`
	ComponentID           string     `json:"componentId"`
	InstallableTemplateID string     `json:"installableTemplateId"`
	LastHeartbeatAt       *time.Time `json:"lastHeartbeatAt"`
	LastReportedAt        *time.Time `json:"lastReportedAt"`
	LastWindowStart       *time.Time `json:"lastWindowStart"`
	LastWindowEnd         *time.Time `json:"lastWindowEnd"`
	LastError             string     `json:"lastError"`
}

func (h *Handlers) GetGatewayTrafficStatus(ctx *gin.Context) {
	if _, ok := h.currentUser(ctx); !ok {
		return
	}
	state, ok, err := h.gatewayTrafficRuntimeStore().Summary(ctx.Request.Context())
	if err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		ctx.JSON(http.StatusOK, gatewayTrafficStatusResponse{
			Available:             false,
			Installed:             false,
			Status:                "not_installed",
			ComponentID:           systemComponentGatewayTrafficProbe,
			InstallableTemplateID: "luna-gateway-traffic-probe",
		})
		return
	}
	ctx.JSON(http.StatusOK, gatewayTrafficStatusResponse{
		Available:             state.Status == "ready",
		Installed:             true,
		Status:                state.Status,
		ComponentID:           systemComponentGatewayTrafficProbe,
		InstallableTemplateID: "luna-gateway-traffic-probe",
		LastHeartbeatAt:       &state.LastHeartbeatAt,
		LastReportedAt:        state.LastReportedAt,
		LastWindowStart:       state.LastWindowStart,
		LastWindowEnd:         state.LastWindowEnd,
		LastError:             state.LastError,
	})
}

func (h *Handlers) CreateGatewayTrafficUsage(ctx *gin.Context) {
	actorID := ""
	var component model.SystemComponentInstallation
	componentAuthenticated := false
	if token := bearerTokenFromHeader(ctx.GetHeader("Authorization")); token != "" {
		if item, ok := h.systemComponentForBearerToken(token, systemComponentGatewayTrafficProbe); ok {
			component = item
			componentAuthenticated = true
			actorID = item.ID
		}
	}
	if !componentAuthenticated {
		user, ok := h.currentUser(ctx)
		if !ok {
			return
		}
		if user.Role != "platform_admin" {
			writeErrorKey(ctx, http.StatusForbidden, user.Language, "config.admin.required")
			return
		}
		actorID = user.ID
	}
	var input gatewayTrafficUsageInput
	if !bindJSON(ctx, &input) {
		return
	}
	routeID := strings.TrimSpace(input.RouteID)
	if routeID == "" {
		writeErrorCode(ctx, http.StatusBadRequest, "billing.gateway_route_required", "gateway route is required")
		return
	}
	if input.ResponseBytes <= 0 {
		writeErrorCode(ctx, http.StatusBadRequest, "billing.gateway_response_bytes_invalid", "gateway response bytes must be positive")
		return
	}
	periodStart, err := time.Parse(time.RFC3339, strings.TrimSpace(input.PeriodStart))
	if err != nil {
		writeErrorCode(ctx, http.StatusBadRequest, "billing.period_start_invalid", "periodStart must be RFC3339 time")
		return
	}
	periodEnd, err := time.Parse(time.RFC3339, strings.TrimSpace(input.PeriodEnd))
	if err != nil || !periodEnd.After(periodStart) {
		writeErrorCode(ctx, http.StatusBadRequest, "billing.period_end_invalid", "periodEnd must be RFC3339 time after periodStart")
		return
	}
	var route model.GatewayRoute
	if err := h.db.First(&route, "id = ? and delete_status = ?", routeID, "active").Error; err != nil {
		writeError(ctx, http.StatusNotFound, "gateway route not found")
		return
	}
	if componentAuthenticated && !h.gatewayRouteBelongsToRuntimeCluster(route, component.RuntimeClusterID) {
		writeErrorCode(ctx, http.StatusForbidden, "billing.gateway_route_cluster_forbidden", "gateway route does not belong to the probe runtime cluster")
		return
	}
	err = (billing.Service{DB: h.db}).SettleGatewayTrafficWindow(billing.GatewayTrafficUsageInput{
		Route:         route,
		ResponseBytes: input.ResponseBytes,
		RequestCount:  input.RequestCount,
		PeriodStart:   periodStart,
		PeriodEnd:     periodEnd,
		ActorID:       actorID,
	})
	if errors.Is(err, billing.ErrAlreadySettled) {
		if componentAuthenticated {
			h.markGatewayTrafficReported(ctx, component.RuntimeClusterID, periodStart, periodEnd)
		}
		ctx.JSON(http.StatusOK, gin.H{"status": "already_settled"})
		return
	}
	if err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	if componentAuthenticated {
		h.markGatewayTrafficReported(ctx, component.RuntimeClusterID, periodStart, periodEnd)
	}
	h.audit(actorID, "billing.gateway_traffic", route.ID, true, "")
	ctx.JSON(http.StatusCreated, gin.H{"status": "settled"})
}

func (h *Handlers) CreateGatewayTrafficProbeHello(ctx *gin.Context) {
	token := bearerTokenFromHeader(ctx.GetHeader("Authorization"))
	component, ok := h.systemComponentForBearerToken(token, systemComponentGatewayTrafficProbe)
	if !ok {
		writeError(ctx, http.StatusUnauthorized, "gateway traffic probe token is invalid")
		return
	}
	if err := h.gatewayTrafficRuntimeStore().MarkHello(ctx.Request.Context(), component.RuntimeClusterID); err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handlers) markGatewayTrafficReported(ctx *gin.Context, runtimeClusterID string, periodStart time.Time, periodEnd time.Time) {
	if err := h.gatewayTrafficRuntimeStore().MarkReport(ctx.Request.Context(), runtimeClusterID, periodStart, periodEnd); err != nil {
		h.audit("", "billing.gateway_traffic_status", runtimeClusterID, false, err.Error())
	}
}

func bearerTokenFromHeader(header string) string {
	header = strings.TrimSpace(header)
	if len(header) < len("Bearer ") || !strings.EqualFold(header[:len("Bearer ")], "Bearer ") {
		return ""
	}
	return strings.TrimSpace(header[len("Bearer "):])
}

func (h *Handlers) gatewayRouteBelongsToRuntimeCluster(route model.GatewayRoute, clusterID string) bool {
	clusterID = strings.TrimSpace(clusterID)
	if clusterID == "" {
		return false
	}
	var target model.DeploymentTarget
	if err := h.db.Select("id", "cluster_id").First(&target, "id = ? and project_id = ?", route.DeploymentTargetID, route.ProjectID).Error; err != nil {
		return false
	}
	targetClusterID := strings.TrimSpace(target.ClusterID)
	if targetClusterID == "" {
		targetClusterID = h.defaultRuntimeClusterID()
	}
	return targetClusterID == clusterID
}
