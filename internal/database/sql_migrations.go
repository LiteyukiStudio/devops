package database

import (
	"errors"
	"fmt"

	sqlmigrations "github.com/LiteyukiStudio/devops/migrations"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"
)

const legacyAutoMigrateBaselineVersion = 8

func runSQLMigrations(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("open sql database for migrations: %w", err)
	}
	adoptLegacySchema, err := shouldAdoptLegacySchema(db)
	if err != nil {
		return err
	}
	sourceDriver, err := iofs.New(sqlmigrations.FS, ".")
	if err != nil {
		return fmt.Errorf("open embedded migrations: %w", err)
	}
	databaseDriver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("open postgres migration driver: %w", err)
	}
	runner, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", databaseDriver)
	if err != nil {
		return fmt.Errorf("create migration runner: %w", err)
	}

	if adoptLegacySchema {
		if err := runner.Force(legacyAutoMigrateBaselineVersion); err != nil {
			return fmt.Errorf("adopt legacy schema at migration %d: %w", legacyAutoMigrateBaselineVersion, err)
		}
	}
	if err := runner.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run sql migrations: %w", err)
	}
	return nil
}

func shouldAdoptLegacySchema(db *gorm.DB) (bool, error) {
	var state legacyMigrationState
	if err := db.Raw(`SELECT
  to_regclass('schema_migrations') IS NOT NULL AS has_migration_table,
  to_regclass('users') IS NOT NULL AS has_users,
  to_regclass('projects') IS NOT NULL AS has_projects,
  to_regclass('deployment_targets') IS NOT NULL AS has_deployment_targets,
  to_regclass('billing_ledger_entries') IS NOT NULL AS has_billing_ledger_entries`).Scan(&state).Error; err != nil {
		return false, fmt.Errorf("inspect legacy schema state: %w", err)
	}
	return shouldAdoptLegacyMigrationState(state), nil
}

func shouldAdoptLegacyMigrationState(state legacyMigrationState) bool {
	if state.HasMigrationTable {
		return false
	}
	return state.HasUsers &&
		state.HasProjects &&
		state.HasDeploymentTargets &&
		state.HasBillingLedgerEntries
}

type legacyMigrationState struct {
	HasMigrationTable       bool
	HasUsers                bool
	HasProjects             bool
	HasDeploymentTargets    bool
	HasBillingLedgerEntries bool
}
