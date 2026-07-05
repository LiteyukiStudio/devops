package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/LiteyukiStudio/devops/internal/appstore"
	"github.com/LiteyukiStudio/devops/internal/id"
	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/LiteyukiStudio/devops/internal/tasks"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	systemComponentGatewayTrafficProbe = "gateway-traffic-probe"
	systemComponentNamespaceDefault    = "liteyuki-system"
)

type systemComponentInstallInput struct {
	ClusterID  string `json:"clusterId"`
	Namespace  string `json:"namespace"`
	Mode       string `json:"mode"`
	APIBaseURL string `json:"apiBaseUrl"`
}

type systemComponentInstallResponse struct {
	Installation model.SystemComponentInstallation `json:"installation"`
}

type systemComponentStatusResponse struct {
	Items                      []model.SystemComponentInstallation `json:"items"`
	GatewayTrafficProbeEnabled bool                                `json:"gatewayTrafficProbeEnabled"`
}

func (h *Handlers) ListSystemComponents(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	if user.Role != "platform_admin" {
		writeErrorKey(ctx, http.StatusForbidden, user.Language, "config.admin.required")
		return
	}
	var items []model.SystemComponentInstallation
	query := h.db.Order("component_id asc, runtime_cluster_id asc")
	if componentID := strings.TrimSpace(ctx.Query("componentId")); componentID != "" {
		query = query.Where("component_id = ?", componentID)
	}
	if clusterID := strings.TrimSpace(ctx.Query("clusterId")); clusterID != "" {
		query = query.Where("runtime_cluster_id = ?", clusterID)
	}
	if err := query.Find(&items).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, systemComponentStatusResponse{
		Items:                      items,
		GatewayTrafficProbeEnabled: hasReadySystemComponent(items, systemComponentGatewayTrafficProbe),
	})
}

func (h *Handlers) InstallSystemAppTemplate(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	if user.Role != "platform_admin" {
		writeErrorKey(ctx, http.StatusForbidden, user.Language, "config.admin.required")
		return
	}
	template, found, err := appstore.Find(ctx.Param("templateId"))
	if err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	if !found || strings.TrimSpace(template.Kind) != "system_component" || strings.TrimSpace(template.SystemComponent) == "" {
		writeError(ctx, http.StatusNotFound, "system component template not found")
		return
	}
	var input systemComponentInstallInput
	if !bindJSON(ctx, &input) {
		return
	}
	clusterID := strings.TrimSpace(input.ClusterID)
	if clusterID == "" {
		writeErrorCode(ctx, http.StatusBadRequest, "runtime_cluster.required", "runtime cluster is required")
		return
	}
	var cluster model.RuntimeCluster
	if err := h.db.First(&cluster, "id = ?", clusterID).Error; err != nil {
		writeError(ctx, http.StatusNotFound, "runtime cluster not found")
		return
	}
	if cluster.Type != "kubernetes" && cluster.Type != "k3s" {
		writeErrorCode(ctx, http.StatusBadRequest, "runtime_cluster.unsupported", "only kubernetes/k3s runtime clusters are supported")
		return
	}
	if h.taskClient == nil {
		writeError(ctx, http.StatusServiceUnavailable, "task queue is not configured")
		return
	}
	namespace := strings.TrimSpace(input.Namespace)
	if namespace == "" {
		namespace = systemComponentNamespaceDefault
	}
	mode := strings.TrimSpace(input.Mode)
	if mode == "" {
		mode = "traefik-metrics"
	}
	apiBaseURL := strings.TrimRight(strings.TrimSpace(input.APIBaseURL), "/")
	if apiBaseURL == "" {
		writeErrorCode(ctx, http.StatusBadRequest, "system_component.api_base_url_required", "API base URL is required")
		return
	}
	configJSON, err := json.Marshal(map[string]string{"apiBaseUrl": apiBaseURL})
	if err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	componentID := strings.TrimSpace(template.SystemComponent)
	reportToken := "lyd_probe_" + randomHex(32)
	installation := model.SystemComponentInstallation{
		ID:               id.New("scmp"),
		ComponentID:      componentID,
		ComponentVersion: template.Version,
		RuntimeClusterID: cluster.ID,
		Namespace:        namespace,
		Status:           "pending",
		Message:          "system component install queued",
		ControllerType:   firstNonEmpty(cluster.GatewayControllerType, "traefik"),
		Mode:             mode,
		Config:           string(configJSON),
		ReportTokenHash:  hashToken(reportToken),
		InstalledBy:      user.ID,
	}
	err = h.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "component_id"}, {Name: "runtime_cluster_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"component_version": template.Version,
			"namespace":         namespace,
			"status":            "pending",
			"message":           "system component install queued",
			"controller_type":   firstNonEmpty(cluster.GatewayControllerType, "traefik"),
			"mode":              mode,
			"config":            string(configJSON),
			"report_token_hash": hashToken(reportToken),
			"last_error":        "",
			"installed_by":      user.ID,
		}),
	}).Create(&installation).Error
	if err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	var persistedInstallation model.SystemComponentInstallation
	if err := h.db.First(&persistedInstallation, "component_id = ? and runtime_cluster_id = ?", componentID, cluster.ID).Error; err != nil {
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	installation = persistedInstallation
	_, err = h.taskClient.EnqueueSystemComponentApply(ctx.Request.Context(), tasks.SystemComponentApplyPayload{
		InstallationID: installation.ID,
		ComponentID:    componentID,
		ClusterID:      cluster.ID,
		ActorID:        user.ID,
		ReportToken:    reportToken,
	})
	if err != nil {
		_ = h.db.Model(&installation).Updates(map[string]any{"status": "failed", "message": "system component task enqueue failed", "last_error": err.Error()}).Error
		writeError(ctx, http.StatusInternalServerError, err.Error())
		return
	}
	h.audit(user.ID, "system_component.install", installation.ID, true, componentID)
	ctx.JSON(http.StatusCreated, systemComponentInstallResponse{Installation: installation})
}

func hasReadySystemComponent(items []model.SystemComponentInstallation, componentID string) bool {
	for _, item := range items {
		if item.ComponentID == componentID && (item.Status == "ready" || item.Status == "deployed") {
			return true
		}
	}
	return false
}

func (h *Handlers) systemComponentForBearerToken(token string, componentID string) (model.SystemComponentInstallation, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return model.SystemComponentInstallation{}, false
	}
	var installation model.SystemComponentInstallation
	err := h.db.First(&installation, "component_id = ? and report_token_hash = ?", componentID, hashToken(token)).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.SystemComponentInstallation{}, false
	}
	return installation, err == nil
}
