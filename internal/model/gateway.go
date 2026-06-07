package model

import (
	"time"

	"gorm.io/gorm"
)

type GatewayRoute struct {
	ID                string         `gorm:"primaryKey" json:"id"`
	ProjectID         string         `gorm:"index;not null" json:"projectId"`
	ApplicationID     string         `gorm:"index;not null" json:"applicationId"`
	EnvironmentID     string         `gorm:"index" json:"environmentId"`
	Host              string         `gorm:"index;not null" json:"host"`
	Path              string         `gorm:"not null;default:/" json:"path"`
	ServicePort       int            `gorm:"not null;default:80" json:"servicePort"`
	TLSMode           string         `gorm:"not null;default:http-only" json:"tlsMode"`
	CertificateStatus string         `gorm:"not null;default:disabled" json:"certificateStatus"`
	CNAMEName         string         `json:"cnameName"`
	CNAMETarget       string         `json:"cnameTarget"`
	DNSStatus         string         `gorm:"not null;default:pending" json:"dnsStatus"`
	Status            string         `gorm:"not null;default:pending" json:"status"`
	IsDefault         bool           `gorm:"not null;default:false" json:"isDefault"`
	CreatedBy         string         `gorm:"index" json:"createdBy"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}
