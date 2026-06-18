package model

import "time"

type ScopedResourceProjectBinding struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	ResourceType string    `gorm:"uniqueIndex:idx_scoped_resource_project;index;not null" json:"resourceType"`
	ResourceID   string    `gorm:"uniqueIndex:idx_scoped_resource_project;index;not null" json:"resourceId"`
	ProjectID    string    `gorm:"uniqueIndex:idx_scoped_resource_project;index;not null" json:"projectId"`
	IsDefault    bool      `gorm:"index;not null;default:false" json:"isDefault"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
