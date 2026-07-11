package model

import "time"

type PlatformEvent struct {
	ID                 string    `gorm:"primaryKey" json:"id"`
	Type               string    `gorm:"index;not null" json:"type"`
	Category           string    `gorm:"index;not null" json:"category"`
	Severity           string    `gorm:"index;not null" json:"severity"`
	Status             string    `gorm:"index;not null" json:"status"`
	ProjectID          string    `gorm:"index;not null;default:''" json:"projectId"`
	ApplicationID      string    `gorm:"index;not null;default:''" json:"applicationId"`
	DeploymentTargetID string    `gorm:"index;not null;default:''" json:"deploymentTargetId"`
	ResourceType       string    `gorm:"index;not null;default:''" json:"resourceType"`
	ResourceID         string    `gorm:"index;not null;default:''" json:"resourceId"`
	ActorID            string    `gorm:"index;not null;default:''" json:"actorId"`
	SummaryKey         string    `gorm:"not null;default:''" json:"summaryKey"`
	Message            string    `gorm:"type:text;not null;default:''" json:"message"`
	DetailJSON         string    `gorm:"type:jsonb;not null;default:'{}'" json:"-"`
	LinksJSON          string    `gorm:"type:jsonb;not null;default:'{}'" json:"-"`
	CorrelationID      string    `gorm:"index;not null;default:''" json:"correlationId"`
	TraceID            string    `gorm:"index;not null;default:''" json:"traceId"`
	DedupKey           *string   `gorm:"uniqueIndex" json:"-"`
	OccurredAt         time.Time `gorm:"index;not null" json:"occurredAt"`
	CreatedAt          time.Time `gorm:"index;not null" json:"createdAt"`
}
