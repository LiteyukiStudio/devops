ALTER TABLE projects
  RENAME COLUMN slug TO identifier;

ALTER TABLE applications
  RENAME COLUMN slug TO identifier;

DROP INDEX IF EXISTS idx_projects_slug_active;
DROP INDEX IF EXISTS idx_applications_project_slug_active;
DROP INDEX IF EXISTS idx_applications_slug;

CREATE UNIQUE INDEX idx_projects_identifier_active
  ON projects(identifier)
  WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX idx_applications_project_identifier_active
  ON applications(project_id, identifier)
  WHERE deleted_at IS NULL;

CREATE INDEX idx_applications_identifier
  ON applications(identifier);

ALTER TABLE projects
  ADD COLUMN kubernetes_namespace text NOT NULL DEFAULT '';

ALTER TABLE deployment_targets
  ADD COLUMN kubernetes_name text NOT NULL DEFAULT '';

UPDATE projects
SET kubernetes_namespace = 'ns-' || LEFT(SPLIT_PART(id, '_', 2), 10)
WHERE kubernetes_namespace = '';

UPDATE deployment_targets
SET kubernetes_name = 'dplt-' || LEFT(SPLIT_PART(id, '_', 2), 10)
WHERE kubernetes_name = '';
