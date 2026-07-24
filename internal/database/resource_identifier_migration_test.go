package database

import (
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

func TestIdentifierTemplateReferenceMigrationCoversPersistedTemplatesAndEvents(t *testing.T) {
	data, err := sqlmigrations.FS.ReadFile("000050_identifier_template_references.up.sql")
	if err != nil {
		t.Fatalf("read identifier template reference migration: %v", err)
	}
	sql := string(data)

	for _, fragment := range []string{
		"UPDATE registry_credentials",
		"repository_template",
		"tag_template",
		"{projectSlug}', '{projectIdentifier}",
		"{appSlug}', '{appIdentifier}",
		"{applicationSlug}', '{applicationIdentifier}",
		"UPDATE notification_templates",
		".Event.Project.Slug', '.Event.Project.Identifier",
		".Event.Application.Slug', '.Event.Application.Identifier",
		".Event.DeploymentTarget.Slug', '.Event.DeploymentTarget.Identifier",
		"UPDATE notification_channels",
		"UPDATE notification_deliveries",
		"UPDATE platform_events",
		"jsonb_build_object('identifier'",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("identifier template reference migration is missing %q", fragment)
		}
	}
}

func TestIdentifierTemplateReferenceMigrationUpgradesLegacyValuesInPostgres(t *testing.T) {
	databaseURL := os.Getenv("AUTH_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("AUTH_TEST_DATABASE_URL is not configured")
	}

	db := openIdentifierMigrationTestSchema(t, databaseURL)
	if err := db.Exec(`
CREATE TABLE registry_credentials (
  id text PRIMARY KEY,
  repository_template text NOT NULL DEFAULT '',
  tag_template text NOT NULL DEFAULT ''
);
CREATE TABLE notification_templates (
  id text PRIMARY KEY,
  subject_template text NOT NULL DEFAULT '',
  body_template text NOT NULL DEFAULT '',
  json_body_template text NOT NULL DEFAULT ''
);
CREATE TABLE notification_channels (
  id text PRIMARY KEY,
  config_json jsonb NOT NULL DEFAULT '{}'::jsonb
);
CREATE TABLE notification_deliveries (
  id text PRIMARY KEY,
  event_json jsonb NOT NULL DEFAULT '{}'::jsonb
);
CREATE TABLE platform_events (
  id text PRIMARY KEY,
  detail_json jsonb NOT NULL DEFAULT '{}'::jsonb
);

INSERT INTO registry_credentials(id, repository_template, tag_template)
VALUES (
  'rgc_legacy',
  '{registryNamespace}/{projectSlug}-{appSlug}-{applicationSlug}',
  '${projectSlug}-${appSlug}-${applicationSlug}-${targetSlug}'
);
INSERT INTO notification_templates(id, subject_template, body_template, json_body_template)
VALUES (
  'ntpl_legacy',
  '{{.Event.Project.Slug}}',
  '{{.Event.Application.Slug}} / {{.Event.DeploymentTarget.Slug}}',
  '{"project":"{{.Event.Project.Slug}}","application":"{{.Event.Application.Slug}}","target":"{{.Event.DeploymentTarget.Slug}}"}'
);
INSERT INTO notification_channels(id, config_json)
VALUES (
  'nch_legacy',
  '{"bodyTemplate":"{{.Event.Project.Slug}}/{{.Event.Application.Slug}}/{{.Event.DeploymentTarget.Slug}}"}'
);
INSERT INTO notification_deliveries(id, event_json)
VALUES (
  'ndl_legacy',
  '{"project":{"id":"prj_legacy","slug":"legacy-project"},"application":{"id":"app_legacy","slug":"legacy-app"},"deploymentTarget":{"id":"dplt_legacy","slug":"prod"}}'
);
INSERT INTO platform_events(id, detail_json)
VALUES (
  'evt_legacy',
  '{"project":{"id":"prj_legacy","slug":"legacy-project"},"application":{"id":"app_legacy","slug":"legacy-app"},"deploymentTarget":{"id":"dplt_legacy","slug":"prod"}}'
);
`).Error; err != nil {
		t.Fatalf("create identifier migration fixtures: %v", err)
	}

	upMigration, err := sqlmigrations.FS.ReadFile("000050_identifier_template_references.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(string(upMigration)).Error; err != nil {
		t.Fatalf("apply identifier template reference migration: %v", err)
	}

	assertIdentifierTemplateReferences(t, db, "identifier", "projectIdentifier", "appIdentifier", "applicationIdentifier", "target")

	downMigration, err := sqlmigrations.FS.ReadFile("000050_identifier_template_references.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(string(downMigration)).Error; err != nil {
		t.Fatalf("roll back identifier template reference migration: %v", err)
	}

	assertIdentifierTemplateReferences(t, db, "slug", "projectSlug", "appSlug", "applicationSlug", "target")
}

func openIdentifierMigrationTestSchema(t *testing.T, databaseURL string) *gorm.DB {
	t.Helper()

	adminDB, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		t.Fatalf("open integration database: %v", err)
	}
	schema := fmt.Sprintf("identifier_template_migration_test_%d", time.Now().UnixNano())
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
	t.Cleanup(func() {
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func assertIdentifierTemplateReferences(t *testing.T, db *gorm.DB, entityKey, projectPlaceholder, appPlaceholder, applicationPlaceholder, targetPlaceholder string) {
	t.Helper()

	var credential struct {
		RepositoryTemplate string
		TagTemplate        string
	}
	if err := db.Raw(`SELECT repository_template, tag_template FROM registry_credentials WHERE id = 'rgc_legacy'`).Scan(&credential).Error; err != nil {
		t.Fatalf("read migrated registry credential: %v", err)
	}
	for _, placeholder := range []string{projectPlaceholder, appPlaceholder, applicationPlaceholder} {
		if !strings.Contains(credential.RepositoryTemplate+credential.TagTemplate, "{"+placeholder+"}") {
			t.Fatalf("registry templates do not contain {%s}: %#v", placeholder, credential)
		}
	}
	if !strings.Contains(credential.TagTemplate, "{"+targetPlaceholder+"}") {
		t.Fatalf("registry tag template does not contain {%s}: %q", targetPlaceholder, credential.TagTemplate)
	}

	var template struct {
		SubjectTemplate  string
		BodyTemplate     string
		JSONBodyTemplate string
	}
	if err := db.Raw(`SELECT subject_template, body_template, json_body_template FROM notification_templates WHERE id = 'ntpl_legacy'`).Scan(&template).Error; err != nil {
		t.Fatalf("read migrated notification template: %v", err)
	}
	for _, entity := range []string{"Project", "Application", "DeploymentTarget"} {
		if !strings.Contains(template.SubjectTemplate+template.BodyTemplate+template.JSONBodyTemplate, ".Event."+entity+"."+strings.ToUpper(entityKey[:1])+entityKey[1:]) {
			t.Fatalf("notification templates do not reference %s.%s: %#v", entity, entityKey, template)
		}
	}

	var channelConfig string
	if err := db.Raw(`SELECT config_json::text FROM notification_channels WHERE id = 'nch_legacy'`).Scan(&channelConfig).Error; err != nil {
		t.Fatalf("read migrated notification channel: %v", err)
	}
	if !strings.Contains(channelConfig, ".Event.Project."+strings.ToUpper(entityKey[:1])+entityKey[1:]) {
		t.Fatalf("notification channel config does not reference %s: %s", entityKey, channelConfig)
	}

	for _, tableAndColumn := range [][2]string{
		{"notification_deliveries", "event_json"},
		{"platform_events", "detail_json"},
	} {
		var keyExists bool
		query := fmt.Sprintf(`SELECT jsonb_exists(%s->'project', ?) FROM %s WHERE id IN ('ndl_legacy', 'evt_legacy')`, tableAndColumn[1], tableAndColumn[0])
		if err := db.Raw(query, entityKey).Scan(&keyExists).Error; err != nil {
			t.Fatalf("read migrated %s: %v", tableAndColumn[0], err)
		}
		if !keyExists {
			t.Fatalf("%s does not contain project.%s", tableAndColumn[0], entityKey)
		}
	}
}
