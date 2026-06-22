CREATE TABLE IF NOT EXISTS app_template_installations (
  id text PRIMARY KEY,
  template_id text NOT NULL,
  template_version text NOT NULL DEFAULT '',
  project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  application_id text NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
  deployment_target_id text NOT NULL DEFAULT '',
  release_id text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'installed',
  message text NOT NULL DEFAULT '',
  values_snapshot text NOT NULL DEFAULT '{}',
  created_by text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_app_template_installations_template_id ON app_template_installations(template_id);
CREATE INDEX IF NOT EXISTS idx_app_template_installations_project_id ON app_template_installations(project_id);
CREATE INDEX IF NOT EXISTS idx_app_template_installations_application_id ON app_template_installations(application_id);
CREATE INDEX IF NOT EXISTS idx_app_template_installations_deployment_target_id ON app_template_installations(deployment_target_id);
CREATE INDEX IF NOT EXISTS idx_app_template_installations_release_id ON app_template_installations(release_id);
CREATE INDEX IF NOT EXISTS idx_app_template_installations_status ON app_template_installations(status);
CREATE INDEX IF NOT EXISTS idx_app_template_installations_created_by ON app_template_installations(created_by);
CREATE INDEX IF NOT EXISTS idx_app_template_installations_deleted_at ON app_template_installations(deleted_at);
