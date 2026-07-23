ALTER TABLE deployment_targets
  DROP COLUMN kubernetes_name;

ALTER TABLE projects
  DROP COLUMN kubernetes_namespace;

DROP INDEX IF EXISTS idx_applications_identifier;
DROP INDEX IF EXISTS idx_applications_project_identifier_active;
DROP INDEX IF EXISTS idx_projects_identifier_active;

ALTER TABLE applications
  RENAME COLUMN identifier TO slug;

ALTER TABLE projects
  RENAME COLUMN identifier TO slug;

CREATE UNIQUE INDEX idx_projects_slug_active
  ON projects(slug)
  WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX idx_applications_project_slug_active
  ON applications(project_id, slug)
  WHERE deleted_at IS NULL;

CREATE INDEX idx_applications_slug
  ON applications(slug);
