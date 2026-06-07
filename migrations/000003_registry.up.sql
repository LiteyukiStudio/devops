CREATE TABLE IF NOT EXISTS artifact_registries (
  id text PRIMARY KEY,
  name text NOT NULL,
  provider text NOT NULL,
  endpoint text NOT NULL,
  namespace text NOT NULL DEFAULT '',
  scope text NOT NULL DEFAULT 'global',
  owner_ref text NOT NULL DEFAULT '',
  credential_ref text NOT NULL DEFAULT '',
  is_default boolean NOT NULL DEFAULT false,
  capabilities text NOT NULL DEFAULT '',
  created_by text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_artifact_registries_scope_owner ON artifact_registries (scope, owner_ref) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_artifact_registries_default_global ON artifact_registries (scope) WHERE deleted_at IS NULL AND scope = 'global' AND is_default;
CREATE UNIQUE INDEX IF NOT EXISTS idx_artifact_registries_default_project ON artifact_registries (scope, owner_ref) WHERE deleted_at IS NULL AND scope = 'project' AND is_default;
CREATE UNIQUE INDEX IF NOT EXISTS idx_artifact_registries_default_user ON artifact_registries (scope, owner_ref) WHERE deleted_at IS NULL AND scope = 'user' AND is_default;

CREATE TABLE IF NOT EXISTS registry_credentials (
  id text PRIMARY KEY,
  registry_id text NOT NULL REFERENCES artifact_registries(id) ON DELETE CASCADE,
  name text NOT NULL,
  username text NOT NULL DEFAULT '',
  password_ref text NOT NULL DEFAULT '',
  token_ref text NOT NULL DEFAULT '',
  scope text NOT NULL DEFAULT 'push-pull',
  access_scope text NOT NULL DEFAULT 'personal',
  created_by text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_registry_credentials_registry ON registry_credentials (registry_id) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS container_images (
  id text PRIMARY KEY,
  project_id text NOT NULL DEFAULT '',
  application_id text NOT NULL DEFAULT '',
  registry_id text NOT NULL REFERENCES artifact_registries(id) ON DELETE RESTRICT,
  repository text NOT NULL,
  tag text NOT NULL,
  digest text NOT NULL DEFAULT '',
  image_ref text NOT NULL,
  source_commit text NOT NULL DEFAULT '',
  build_run_id text NOT NULL DEFAULT '',
  source_type text NOT NULL DEFAULT 'manual-image',
  scan_status text NOT NULL DEFAULT 'unknown',
  created_by text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_container_images_project ON container_images (project_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_container_images_application ON container_images (application_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_container_images_registry_repo ON container_images (registry_id, repository) WHERE deleted_at IS NULL;
