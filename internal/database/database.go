package database

import (
	"fmt"
	"strings"

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
	if err := cleanupLegacySchema(db); err != nil {
		return err
	}
	return db.AutoMigrate(
		&model.User{},
		&model.UserSession{},
		&model.UserRememberToken{},
		&model.AuthProvider{},
		&model.ExternalIdentity{},
		&model.AuthAdmissionPolicy{},
		&model.Project{},
		&model.ProjectMember{},
		&model.ProjectPin{},
		&model.ProjectHookConfig{},
		&model.HookRun{},
		&model.HookRunLog{},
		&model.AccessToken{},
		&model.AuditLog{},
		&model.WorkerTaskEvent{},
		&model.SecretValue{},
		&model.Application{},
		&model.GitProvider{},
		&model.GitAccount{},
		&model.RepositoryBinding{},
		&model.ArtifactRegistry{},
		&model.RegistryCredential{},
		&model.ContainerImage{},
		&model.BuildProvider{},
		&model.ApplicationModule{},
		&model.ApplicationModuleHookBinding{},
		&model.BuildVariableSet{},
		&model.BuildRun{},
		&model.BuildJob{},
		&model.BuildLog{},
		&model.BuilderAgent{},
		&model.RuntimeCluster{},
		&model.Environment{},
		&model.Release{},
		&model.ReleaseLog{},
		&model.DeploymentTarget{},
		&model.GatewayRoute{},
		&model.AppConfig{},
	)
}

func cleanupLegacySchema(db *gorm.DB) error {
	if err := db.Exec("ALTER TABLE deployment_targets DROP COLUMN IF EXISTS build_config_id").Error; err != nil {
		return fmt.Errorf("drop legacy deployment_targets.build_config_id: %w", err)
	}
	return nil
}
