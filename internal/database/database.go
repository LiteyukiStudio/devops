package database

import (
	"fmt"
	"strings"

	"github.com/LiteyukiStudio/devops/internal/billing"
	"github.com/LiteyukiStudio/devops/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Open(databaseURL string) (*gorm.DB, error) {
	if strings.HasPrefix(databaseURL, "postgres://") || strings.HasPrefix(databaseURL, "postgresql://") {
		return gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	}

	return nil, fmt.Errorf("unsupported database url: %s", databaseURL)
}

func Migrate(db *gorm.DB) error {
	if err := cleanupApplicationDeliveryColumns(db); err != nil {
		return err
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.UserSession{},
		&model.UserRememberToken{},
		&model.AuthProvider{},
		&model.ExternalIdentity{},
		&model.AuthAdmissionPolicy{},
		&model.Project{},
		&model.ProjectMember{},
		&model.ProjectPin{},
		&model.ProjectWallet{},
		&model.ProjectHookConfig{},
		&model.HookRun{},
		&model.HookRunLog{},
		&model.AccessToken{},
		&model.AuditLog{},
		&model.WorkerTaskEvent{},
		&model.SecretValue{},
		&model.ScopedResourceProjectBinding{},
		&model.Application{},
		&model.GitProvider{},
		&model.GitAccount{},
		&model.RepositoryBinding{},
		&model.ArtifactRegistry{},
		&model.RegistryCredential{},
		&model.ContainerImage{},
		&model.DeploymentTargetHookBinding{},
		&model.BuildVariableSet{},
		&model.BuildRun{},
		&model.BuildJob{},
		&model.BuildLog{},
		&model.BillingRateRule{},
		&model.BillingUsageRecord{},
		&model.BillingLedgerEntry{},
		&model.RuntimeCluster{},
		&model.Environment{},
		&model.Release{},
		&model.ReleaseLog{},
		&model.ProjectRuntimeConfigSet{},
		&model.DeploymentTarget{},
		&model.GatewayRoute{},
		&model.AppConfig{},
	); err != nil {
		return err
	}
	return (billing.Service{DB: db}).EnsureDefaultRateRules()
}

func cleanupApplicationDeliveryColumns(db *gorm.DB) error {
	for _, statement := range cleanupApplicationDeliveryStatements() {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("cleanup application delivery columns: %w", err)
		}
	}
	return nil
}

func cleanupApplicationDeliveryStatements() []string {
	return []string{
		"DROP INDEX IF EXISTS idx_applications_git_account",
		"ALTER TABLE IF EXISTS applications DROP COLUMN IF EXISTS source_type",
		"ALTER TABLE IF EXISTS applications DROP COLUMN IF EXISTS repository_url",
		"ALTER TABLE IF EXISTS applications DROP COLUMN IF EXISTS image_reference",
		"ALTER TABLE IF EXISTS applications DROP COLUMN IF EXISTS git_account_id",
		"ALTER TABLE IF EXISTS applications DROP COLUMN IF EXISTS service_port",
		"ALTER TABLE IF EXISTS deployment_targets DROP COLUMN IF EXISTS build_config_id",
	}
}
