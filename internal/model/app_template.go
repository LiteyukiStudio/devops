package model

import (
	"time"

	"gorm.io/gorm"
)

type AppTemplateInstallation struct {
	ID                 string         `gorm:"primaryKey" json:"id"`
	TemplateID         string         `gorm:"index;not null" json:"templateId"`
	TemplateVersion    string         `gorm:"not null;default:''" json:"templateVersion"`
	ProjectID          string         `gorm:"index;not null" json:"projectId"`
	ApplicationID      string         `gorm:"index;not null" json:"applicationId"`
	DeploymentTargetID string         `gorm:"index;not null;default:''" json:"deploymentTargetId"`
	ReleaseID          string         `gorm:"index;not null;default:''" json:"releaseId"`
	Status             string         `gorm:"index;not null;default:installed" json:"status"`
	Message            string         `gorm:"type:text;not null;default:''" json:"message"`
	ValuesSnapshot     string         `gorm:"type:text;not null;default:'{}'" json:"valuesSnapshot"`
	CreatedBy          string         `gorm:"index;not null;default:''" json:"createdBy"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}
