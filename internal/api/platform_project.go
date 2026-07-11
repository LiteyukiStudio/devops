package api

import (
	"errors"
	"strings"

	"github.com/LiteyukiStudio/devops/internal/id"
	"github.com/LiteyukiStudio/devops/internal/model"
	"gorm.io/gorm"
)

const (
	platformSystemProjectKey  = "platform"
	platformSystemProjectSlug = "platform-system"
)

func isSystemProject(project model.Project) bool {
	return strings.TrimSpace(project.SystemKey) != ""
}

func (h *Handlers) ensurePlatformSystemProject(user model.User) (model.Project, error) {
	var project model.Project
	if err := h.db.First(&project, "system_key = ?", platformSystemProjectKey).Error; err == nil {
		return h.ensurePlatformSystemProjectBillingOwner(project, user)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Project{}, err
	}

	project = model.Project{
		ID:                  id.New("prj"),
		Slug:                platformSystemProjectSlug,
		Name:                "Luna Platform",
		Description:         "Platform-owned applications and probes managed by Luna DevOps.",
		NamespaceStrategy:   "project",
		MaxConcurrentBuilds: 1,
		BillingOwnerUserID:  strings.TrimSpace(user.ID),
		SystemKey:           platformSystemProjectKey,
		DeleteStatus:        "active",
	}
	if err := h.db.Create(&project).Error; err != nil {
		var existing model.Project
		if findErr := h.db.First(&existing, "system_key = ?", platformSystemProjectKey).Error; findErr == nil {
			return h.ensurePlatformSystemProjectBillingOwner(existing, user)
		}
		return model.Project{}, err
	}
	return h.ensurePlatformSystemProjectBillingOwner(project, user)
}

func (h *Handlers) ensurePlatformSystemProjectBillingOwner(project model.Project, user model.User) (model.Project, error) {
	if strings.TrimSpace(project.BillingOwnerUserID) == "" && strings.TrimSpace(user.ID) != "" {
		project.BillingOwnerUserID = user.ID
		if err := h.db.Model(&project).Update("billing_owner_user_id", user.ID).Error; err != nil {
			return model.Project{}, err
		}
	}
	return project, nil
}
