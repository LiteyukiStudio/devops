package database

import (
	"strings"
	"testing"

	sqlmigrations "github.com/LiteyukiStudio/devops/migrations"
)

func TestImmutableResourceIdentifierMigrationPreservesLegacyKubernetesNames(t *testing.T) {
	data, err := sqlmigrations.FS.ReadFile("000049_immutable_resource_identifiers.up.sql")
	if err != nil {
		t.Fatalf("read resource identifier migration: %v", err)
	}
	sql := string(data)

	for _, fragment := range []string{
		"RENAME COLUMN slug TO identifier",
		"ADD COLUMN kubernetes_namespace",
		"ADD COLUMN kubernetes_name",
		"SET kubernetes_namespace = 'ns-'",
		"SET kubernetes_name = 'dplt-'",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("resource identifier migration is missing %q", fragment)
		}
	}
	if strings.Contains(sql, "SET kubernetes_namespace = 'luna-'") || strings.Contains(sql, "SET kubernetes_name = 'luna-'") {
		t.Fatal("migration must preserve legacy Kubernetes names instead of renaming deployed resources")
	}
}
