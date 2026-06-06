CREATE TABLE IF NOT EXISTS users (
  id text PRIMARY KEY,
  email text NOT NULL UNIQUE,
  name text NOT NULL,
  auth_type text NOT NULL DEFAULT 'local',
  role text NOT NULL DEFAULT 'user',
  language text NOT NULL DEFAULT 'zh-CN',
  password text NOT NULL DEFAULT '',
  disabled boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);

CREATE TABLE IF NOT EXISTS user_sessions (
  id text PRIMARY KEY,
  user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash text NOT NULL UNIQUE,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS oidc_auth_states (
  id text PRIMARY KEY,
  state_hash text NOT NULL UNIQUE,
  nonce text NOT NULL,
  provider_id text NOT NULL,
  user_id text NOT NULL DEFAULT '',
  mode text NOT NULL,
  redirect_path text NOT NULL DEFAULT '/projects',
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS auth_providers (
  id text PRIMARY KEY,
  type text NOT NULL,
  name text NOT NULL,
  enabled boolean NOT NULL DEFAULT true,
  issuer_url text NOT NULL,
  client_id text NOT NULL,
  client_secret_ref text NOT NULL DEFAULT '',
  scopes text NOT NULL DEFAULT 'openid profile email',
  group_claim text NOT NULL DEFAULT 'groups',
  email_claim text NOT NULL DEFAULT 'email',
  username_claim text NOT NULL DEFAULT 'preferred_username',
  is_default boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);

CREATE TABLE IF NOT EXISTS external_identities (
  id text PRIMARY KEY,
  user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider_id text NOT NULL REFERENCES auth_providers(id) ON DELETE RESTRICT,
  subject text NOT NULL,
  email text NOT NULL DEFAULT '',
  email_verified boolean NOT NULL DEFAULT false,
  username text NOT NULL DEFAULT '',
  last_login_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (provider_id, subject),
  UNIQUE (user_id, provider_id)
);

CREATE TABLE IF NOT EXISTS auth_admission_policies (
  id text PRIMARY KEY,
  allow_local_login boolean NOT NULL DEFAULT true,
  allow_oidc_login boolean NOT NULL DEFAULT true,
  allowed_email_domains text NOT NULL DEFAULT '',
  allowed_oidc_groups text NOT NULL DEFAULT '',
  invited_emails text NOT NULL DEFAULT '',
  default_role text NOT NULL DEFAULT 'user',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS projects (
  id text PRIMARY KEY,
  slug text NOT NULL,
  name text NOT NULL,
  description text NOT NULL DEFAULT '',
  namespace_strategy text NOT NULL DEFAULT 'project',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_projects_slug_active ON projects (slug) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS project_members (
  id text PRIMARY KEY,
  project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (project_id, user_id)
);

CREATE TABLE IF NOT EXISTS access_tokens (
  id text PRIMARY KEY,
  user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name text NOT NULL,
  scope text NOT NULL,
  token_hash text NOT NULL,
  expires_at timestamptz,
  revoked_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS audit_logs (
  id text PRIMARY KEY,
  user_id text NOT NULL DEFAULT '',
  action text NOT NULL,
  resource text NOT NULL,
  success boolean NOT NULL DEFAULT true,
  message text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS applications (
  id text PRIMARY KEY,
  project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  slug text NOT NULL,
  name text NOT NULL,
  source_type text NOT NULL,
  repository_url text NOT NULL DEFAULT '',
  image_reference text NOT NULL DEFAULT '',
  dockerfile_path text NOT NULL DEFAULT 'Dockerfile',
  build_context text NOT NULL DEFAULT '.',
  service_port integer NOT NULL DEFAULT 8080,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  UNIQUE (project_id, slug)
);

CREATE TABLE IF NOT EXISTS app_configs (
  key text PRIMARY KEY,
  value text NOT NULL DEFAULT '',
  updated_at timestamptz NOT NULL DEFAULT now()
);
