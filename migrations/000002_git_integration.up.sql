CREATE TABLE IF NOT EXISTS git_providers (
  id text PRIMARY KEY,
  type text NOT NULL,
  name text NOT NULL,
  base_url text NOT NULL DEFAULT '',
  auth_type text NOT NULL DEFAULT 'oauth',
  client_id text NOT NULL DEFAULT '',
  client_secret_ref text NOT NULL DEFAULT '',
  enabled boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);

CREATE TABLE IF NOT EXISTS git_accounts (
  id text PRIMARY KEY,
  user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider_id text NOT NULL REFERENCES git_providers(id) ON DELETE RESTRICT,
  external_user_id text NOT NULL DEFAULT '',
  username text NOT NULL,
  avatar_url text NOT NULL DEFAULT '',
  access_token_ref text NOT NULL DEFAULT '',
  refresh_token_ref text NOT NULL DEFAULT '',
  scopes text NOT NULL DEFAULT '',
  expires_at timestamptz,
  status text NOT NULL DEFAULT 'connected',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_git_accounts_user_provider ON git_accounts (user_id, provider_id) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS repository_bindings (
  id text PRIMARY KEY,
  project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  application_id text NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
  git_provider_id text NOT NULL REFERENCES git_providers(id) ON DELETE RESTRICT,
  git_account_id text NOT NULL REFERENCES git_accounts(id) ON DELETE RESTRICT,
  owner text NOT NULL,
  repo text NOT NULL,
  clone_url text NOT NULL DEFAULT '',
  default_branch text NOT NULL DEFAULT 'main',
  webhook_status text NOT NULL DEFAULT 'pending',
  credential_ref text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_repository_bindings_application_active ON repository_bindings (application_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_repository_bindings_project ON repository_bindings (project_id) WHERE deleted_at IS NULL;
