package database

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	sqlmigrations "github.com/LiteyukiStudio/devops/migrations"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestSessionPrimaryAuthenticationMigrationLeavesLegacySessionsUntrusted(t *testing.T) {
	data, err := sqlmigrations.FS.ReadFile("000036_session_primary_authentication.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := string(data)
	if !strings.Contains(sql, "primary_authenticated_at timestamptz") {
		t.Fatalf("migration does not add primary authentication time:\n%s", sql)
	}
	if strings.Contains(strings.ToLower(sql), "update user_sessions") || strings.Contains(strings.ToLower(sql), "default now()") {
		t.Fatalf("legacy sessions must not be backfilled as freshly authenticated:\n%s", sql)
	}
}

func TestSessionPrimaryAuthenticationMigrationPreservesNullLegacyValueInPostgres(t *testing.T) {
	databaseURL := os.Getenv("AUTH_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AUTH_TEST_DATABASE_URL is not configured")
	}
	adminDB, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	schema := fmt.Sprintf("session_primary_auth_migration_test_%d", time.Now().UnixNano())
	if err := adminDB.Exec(`CREATE SCHEMA "` + schema + `"`).Error; err != nil {
		t.Fatalf("create integration schema: %v", err)
	}
	t.Cleanup(func() {
		_ = adminDB.Exec(`DROP SCHEMA IF EXISTS "` + schema + `" CASCADE`).Error
		if sqlDB, dbErr := adminDB.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	})

	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse integration database URL: %v", err)
	}
	query := parsedURL.Query()
	query.Set("search_path", schema)
	parsedURL.RawQuery = query.Encode()
	db, err := gorm.Open(postgres.Open(parsedURL.String()), &gorm.Config{})
	if err != nil {
		t.Fatalf("open integration schema: %v", err)
	}
	defer func() {
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	}()

	if err := db.Exec(`
CREATE TABLE user_sessions (
  id text PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO user_sessions(id) VALUES ('ses_legacy');
`).Error; err != nil {
		t.Fatalf("create migration prerequisite: %v", err)
	}
	upMigration, err := sqlmigrations.FS.ReadFile("000036_session_primary_authentication.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(string(upMigration)).Error; err != nil {
		t.Fatalf("apply session primary-authentication migration: %v", err)
	}
	var primaryAuthenticatedAt sql.NullTime
	if err := db.Raw(`SELECT primary_authenticated_at FROM user_sessions WHERE id = 'ses_legacy'`).Scan(&primaryAuthenticatedAt).Error; err != nil {
		t.Fatalf("read upgraded legacy session: %v", err)
	}
	if primaryAuthenticatedAt.Valid {
		t.Fatalf("legacy session was incorrectly marked fresh: %v", primaryAuthenticatedAt)
	}

	downMigration, err := sqlmigrations.FS.ReadFile("000036_session_primary_authentication.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(string(downMigration)).Error; err != nil {
		t.Fatalf("roll back session primary-authentication migration: %v", err)
	}
}
