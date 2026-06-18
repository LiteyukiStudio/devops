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
