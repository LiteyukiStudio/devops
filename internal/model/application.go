package model

import (
	"gorm.io/gorm"
	"time"
)

type Application struct {
	ID          string         `gorm:"primaryKey" json:"id"`
	ProjectID   string         `gorm:"uniqueIndex:idx_applications_project_slug_active,where:deleted_at IS NULL;index;not null" json:"projectId"`
	Slug        string         `gorm:"uniqueIndex:idx_applications_project_slug_active,where:deleted_at IS NULL;index;not null" json:"slug"`
	Name        string         `gorm:"not null" json:"name"`
	Icon        string         `gorm:"not null;default:'box'" json:"icon"`
	ServicePort int            `json:"servicePort"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type AppConfig struct {
	Key       string    `gorm:"primaryKey" json:"key"`
	Value     string    `gorm:"not null" json:"value"`
	UpdatedAt time.Time `json:"updatedAt"`
}
