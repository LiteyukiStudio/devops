package api

import (
	"github.com/LiteyukiStudio/devops/internal/model"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) projectIDsForUser(userID string) []string {
	return h.projects.IDsForUser(userID)
}

func (h *Handlers) userHasProject(ctx *gin.Context, userID, projectID string) bool {
	return h.projects.UserHasProject(userID, projectID)
}

func (h *Handlers) findProjectForCurrentUserByID(ctx *gin.Context, projectID string) (model.Project, bool) {
	original := ctx.Param("projectId")
	ctx.Params = append(ctx.Params, gin.Param{Key: "projectId", Value: projectID})
	project, ok := h.findProjectForCurrentUser(ctx)
	ctx.Params = replaceParam(ctx.Params, "projectId", original)
	return project, ok
}

func (h *Handlers) findProjectForCurrentUserWithRolesByID(ctx *gin.Context, projectID string, roles ...string) (model.Project, bool) {
	original := ctx.Param("projectId")
	ctx.Params = append(ctx.Params, gin.Param{Key: "projectId", Value: projectID})
	project, ok := h.findProjectForCurrentUserWithRoles(ctx, roles...)
	ctx.Params = replaceParam(ctx.Params, "projectId", original)
	return project, ok
}

func replaceParam(params gin.Params, key, value string) gin.Params {
	result := gin.Params{}
	for _, param := range params {
		if param.Key == key {
			continue
		}
		result = append(result, param)
	}
	if value != "" {
		result = append(result, gin.Param{Key: key, Value: value})
	}
	return result
}
