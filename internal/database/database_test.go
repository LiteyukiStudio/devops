package database

import (
	"strings"
	"testing"
)

func TestCleanupApplicationDeliveryStatementsDropLegacyServicePort(t *testing.T) {
	statements := strings.Join(cleanupApplicationDeliveryStatements(), "\n")
	if !strings.Contains(statements, "applications DROP COLUMN IF EXISTS service_port") {
		t.Fatalf("cleanup statements do not drop legacy application service_port: %s", statements)
	}
}

func TestShouldAdoptLegacyMigrationState(t *testing.T) {
	state := legacyMigrationState{
		HasUsers:                true,
		HasProjects:             true,
		HasDeploymentTargets:    true,
		HasBillingLedgerEntries: true,
	}
	if !shouldAdoptLegacyMigrationState(state) {
		t.Fatalf("expected complete legacy schema to be adopted")
	}
}

func TestShouldNotAdoptWhenMigrationTableExists(t *testing.T) {
	state := legacyMigrationState{
		HasMigrationTable:       true,
		HasUsers:                true,
		HasProjects:             true,
		HasDeploymentTargets:    true,
		HasBillingLedgerEntries: true,
	}
	if shouldAdoptLegacyMigrationState(state) {
		t.Fatalf("expected schema with migration table to keep normal migration flow")
	}
}

func TestShouldNotAdoptIncompleteSchema(t *testing.T) {
	state := legacyMigrationState{
		HasUsers:          true,
		HasProjects:       true,
		HasMigrationTable: false,
	}
	if shouldAdoptLegacyMigrationState(state) {
		t.Fatalf("expected incomplete schema to run migrations from the beginning")
	}
}
