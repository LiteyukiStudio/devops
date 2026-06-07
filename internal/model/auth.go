package model

import (
	"gorm.io/gorm"
	"time"
)

type OIDCAuthState struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	StateHash    string    `gorm:"uniqueIndex;not null" json:"-"`
	Nonce        string    `gorm:"not null" json:"-"`
	ProviderID   string    `gorm:"index;not null" json:"providerId"`
	UserID       string    `gorm:"index" json:"userId"`
	Mode         string    `gorm:"not null" json:"mode"`
	RedirectPath string    `gorm:"not null;default:/projects" json:"redirectPath"`
	ExpiresAt    time.Time `gorm:"index;not null" json:"expiresAt"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (OIDCAuthState) TableName() string {
	return "oidc_auth_states"
}

type AuthProvider struct {
	ID              string         `gorm:"primaryKey" json:"id"`
	Type            string         `gorm:"not null" json:"type"`
	Name            string         `gorm:"not null" json:"name"`
	Enabled         bool           `gorm:"not null;default:true" json:"enabled"`
	IssuerURL       string         `gorm:"not null" json:"issuerUrl"`
	ClientID        string         `gorm:"not null" json:"clientId"`
	ClientSecretRef string         `json:"clientSecretRef"`
	Scopes          string         `gorm:"not null;default:openid profile email" json:"scopes"`
	GroupClaim      string         `gorm:"not null;default:groups" json:"groupClaim"`
	EmailClaim      string         `gorm:"not null;default:email" json:"emailClaim"`
	UsernameClaim   string         `gorm:"not null;default:preferred_username" json:"usernameClaim"`
	IsDefault       bool           `gorm:"not null;default:false" json:"isDefault"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

type ExternalIdentity struct {
	ID            string     `gorm:"primaryKey" json:"id"`
	UserID        string     `gorm:"index:idx_external_identities_user_provider,unique;not null" json:"userId"`
	ProviderID    string     `gorm:"index:idx_external_identities_provider_subject,unique;index:idx_external_identities_user_provider,unique;not null" json:"providerId"`
	Subject       string     `gorm:"index:idx_external_identities_provider_subject,unique;not null" json:"subject"`
	Email         string     `json:"email"`
	EmailVerified bool       `gorm:"not null;default:false" json:"emailVerified"`
	Username      string     `json:"username"`
	LastLoginAt   *time.Time `json:"lastLoginAt"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type AuthAdmissionPolicy struct {
	ID                  string    `gorm:"primaryKey" json:"id"`
	AllowLocalLogin     bool      `gorm:"not null;default:true" json:"allowLocalLogin"`
	AllowOIDCLogin      bool      `gorm:"not null;default:true" json:"allowOidcLogin"`
	AllowedEmailDomains string    `json:"allowedEmailDomains"`
	AllowedOIDCGroups   string    `json:"allowedOidcGroups"`
	InvitedEmails       string    `json:"invitedEmails"`
	DefaultRole         string    `gorm:"not null;default:user" json:"defaultRole"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}
