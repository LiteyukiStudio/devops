package api

import (
	"net/http"

	"github.com/LiteyukiStudio/devops/internal/authz"
	dashboardservice "github.com/LiteyukiStudio/devops/internal/dashboard"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) GetDashboard(ctx *gin.Context) {
	user, ok := h.currentUser(ctx)
	if !ok {
		return
	}
	platformAdmin := authz.IsPlatformAdmin(user.Role)
	if platformAdmin {
		if _, err := h.ensurePlatformSystemProject(user); err != nil {
			writeErrorCode(ctx, http.StatusInternalServerError, "dashboard.load_failed", err.Error())
			return
		}
	}
	projectIDs := []string{}
	if !platformAdmin {
		projectIDs = h.projectIDsForUser(user.ID)
	}
	overview, err := dashboardservice.NewService(h.db).Overview(ctx.Request.Context(), dashboardservice.Scope{
		UserID:            user.ID,
		PlatformAdmin:     platformAdmin,
		VisibleProjectIDs: projectIDs,
	})
	if err != nil {
		writeErrorCode(ctx, http.StatusInternalServerError, "dashboard.load_failed", err.Error())
		return
	}
	ctx.JSON(http.StatusOK, overview)
}
