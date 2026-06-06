package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        string         `gorm:"primaryKey" json:"id"`
	Email     string         `gorm:"uniqueIndex;not null" json:"email"`
	Name      string         `gorm:"not null" json:"name"`
	AuthType  string         `gorm:"not null;default:local" json:"authType"`
	Role      string         `gorm:"not null;default:user" json:"role"`
	Language  string         `gorm:"not null;default:zh-CN" json:"language"`
	Password  string         `json:"-"`
	Disabled  bool           `gorm:"not null;default:false" json:"disabled"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type UserSession struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	UserID    string    `gorm:"index;not null" json:"userId"`
	TokenHash string    `gorm:"uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time `gorm:"index;not null" json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

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

type Project struct {
	ID                string         `gorm:"primaryKey" json:"id"`
	Slug              string         `gorm:"index;not null" json:"slug"`
	Name              string         `gorm:"not null" json:"name"`
	Description       string         `json:"description"`
	NamespaceStrategy string         `gorm:"not null" json:"namespaceStrategy"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

type ProjectMember struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	ProjectID string    `gorm:"index;not null" json:"projectId"`
	UserID    string    `gorm:"index;not null" json:"userId"`
	Role      string    `gorm:"not null" json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type AccessToken struct {
	ID        string     `gorm:"primaryKey" json:"id"`
	UserID    string     `gorm:"index" json:"userId"`
	Name      string     `gorm:"not null" json:"name"`
	Scope     string     `gorm:"not null" json:"scope"`
	TokenHash string     `gorm:"not null" json:"-"`
	ExpiresAt *time.Time `json:"expiresAt"`
	RevokedAt *time.Time `json:"revokedAt"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type AuditLog struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	UserID    string    `gorm:"index" json:"userId"`
	Action    string    `gorm:"index;not null" json:"action"`
	Resource  string    `gorm:"index;not null" json:"resource"`
	Success   bool      `gorm:"not null;default:true" json:"success"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
}

type Application struct {
	ID             string         `gorm:"primaryKey" json:"id"`
	ProjectID      string         `gorm:"index;not null" json:"projectId"`
	Slug           string         `gorm:"index;not null" json:"slug"`
	Name           string         `gorm:"not null" json:"name"`
	SourceType     string         `gorm:"not null" json:"sourceType"`
	RepositoryURL  string         `json:"repositoryUrl"`
	ImageReference string         `json:"imageReference"`
	DockerfilePath string         `json:"dockerfilePath"`
	BuildContext   string         `json:"buildContext"`
	ServicePort    int            `json:"servicePort"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

type GitProvider struct {
	ID              string         `gorm:"primaryKey" json:"id"`
	Type            string         `gorm:"not null" json:"type"`
	Name            string         `gorm:"not null" json:"name"`
	BaseURL         string         `gorm:"not null" json:"baseUrl"`
	AuthType        string         `gorm:"not null;default:oauth" json:"authType"`
	ClientID        string         `json:"clientId"`
	ClientSecretRef string         `json:"clientSecretRef"`
	Enabled         bool           `gorm:"not null;default:true" json:"enabled"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

type GitAccount struct {
	ID              string         `gorm:"primaryKey" json:"id"`
	UserID          string         `gorm:"index;not null" json:"userId"`
	ProviderID      string         `gorm:"index;not null" json:"providerId"`
	ExternalUserID  string         `json:"externalUserId"`
	Username        string         `gorm:"not null" json:"username"`
	AvatarURL       string         `json:"avatarUrl"`
	AccessTokenRef  string         `json:"accessTokenRef"`
	RefreshTokenRef string         `json:"refreshTokenRef"`
	Scopes          string         `json:"scopes"`
	ExpiresAt       *time.Time     `json:"expiresAt"`
	Status          string         `gorm:"not null;default:connected" json:"status"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

type GitOAuthState struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	StateHash    string    `gorm:"uniqueIndex;not null" json:"-"`
	ProviderID   string    `gorm:"index;not null" json:"providerId"`
	UserID       string    `gorm:"index;not null" json:"userId"`
	RedirectPath string    `gorm:"not null;default:/projects" json:"redirectPath"`
	ExpiresAt    time.Time `gorm:"index;not null" json:"expiresAt"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type RepositoryBinding struct {
	ID            string         `gorm:"primaryKey" json:"id"`
	ProjectID     string         `gorm:"index;not null" json:"projectId"`
	ApplicationID string         `gorm:"index;not null" json:"applicationId"`
	GitProviderID string         `gorm:"index;not null" json:"gitProviderId"`
	GitAccountID  string         `gorm:"index;not null" json:"gitAccountId"`
	Owner         string         `gorm:"not null" json:"owner"`
	Repo          string         `gorm:"not null" json:"repo"`
	CloneURL      string         `gorm:"not null" json:"cloneUrl"`
	DefaultBranch string         `gorm:"not null;default:main" json:"defaultBranch"`
	WebhookStatus string         `gorm:"not null;default:pending" json:"webhookStatus"`
	WebhookID     string         `json:"webhookId"`
	WebhookSecret string         `json:"-"`
	CredentialRef string         `json:"credentialRef"`
	LastEvent     string         `json:"lastEvent"`
	LastCommitSHA string         `json:"lastCommitSha"`
	LastWebhookAt *time.Time     `json:"lastWebhookAt"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

type ArtifactRegistry struct {
	ID            string         `gorm:"primaryKey" json:"id"`
	Name          string         `gorm:"not null" json:"name"`
	Provider      string         `gorm:"not null" json:"provider"`
	Endpoint      string         `gorm:"not null" json:"endpoint"`
	Namespace     string         `json:"namespace"`
	Scope         string         `gorm:"index;not null;default:global" json:"scope"`
	OwnerRef      string         `gorm:"index" json:"ownerRef"`
	CredentialRef string         `json:"credentialRef"`
	IsDefault     bool           `gorm:"not null;default:false" json:"isDefault"`
	Capabilities  string         `json:"capabilities"`
	CreatedBy     string         `gorm:"index" json:"createdBy"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

type RegistryCredential struct {
	ID          string         `gorm:"primaryKey" json:"id"`
	RegistryID  string         `gorm:"index;not null" json:"registryId"`
	Name        string         `gorm:"not null" json:"name"`
	Username    string         `json:"username"`
	PasswordRef string         `json:"-"`
	TokenRef    string         `json:"-"`
	Scope       string         `gorm:"not null;default:push-pull" json:"scope"`
	CreatedBy   string         `gorm:"index" json:"createdBy"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type ContainerImage struct {
	ID            string         `gorm:"primaryKey" json:"id"`
	ProjectID     string         `gorm:"index" json:"projectId"`
	ApplicationID string         `gorm:"index" json:"applicationId"`
	RegistryID    string         `gorm:"index;not null" json:"registryId"`
	Repository    string         `gorm:"not null" json:"repository"`
	Tag           string         `gorm:"not null" json:"tag"`
	Digest        string         `json:"digest"`
	ImageRef      string         `gorm:"not null" json:"imageRef"`
	SourceCommit  string         `json:"sourceCommit"`
	BuildRunID    string         `gorm:"index" json:"buildRunId"`
	SourceType    string         `gorm:"not null;default:manual-image" json:"sourceType"`
	ScanStatus    string         `gorm:"not null;default:unknown" json:"scanStatus"`
	CreatedBy     string         `gorm:"index" json:"createdBy"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

type AppConfig struct {
	Key       string    `gorm:"primaryKey" json:"key"`
	Value     string    `gorm:"not null" json:"value"`
	UpdatedAt time.Time `json:"updatedAt"`
}
