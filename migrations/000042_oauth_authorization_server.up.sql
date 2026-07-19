ALTER TABLE access_tokens
  ADD COLUMN IF NOT EXISTS source text NOT NULL DEFAULT 'personal',
  ADD COLUMN IF NOT EXISTS oauth_application_id text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS oauth_grant_id text NOT NULL DEFAULT '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_access_tokens_token_hash ON access_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_access_tokens_source ON access_tokens(source);
CREATE INDEX IF NOT EXISTS idx_access_tokens_oauth_application_id ON access_tokens(oauth_application_id);
CREATE INDEX IF NOT EXISTS idx_access_tokens_oauth_grant_id ON access_tokens(oauth_grant_id);

CREATE TABLE IF NOT EXISTS oauth_applications (
  id text PRIMARY KEY,
  owner_user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name text NOT NULL,
  description text NOT NULL DEFAULT '',
  homepage_url text NOT NULL DEFAULT '',
  logo_url text NOT NULL DEFAULT '',
  client_id text NOT NULL,
  client_secret_hash text NOT NULL,
  redirect_uris text NOT NULL,
  allowed_scopes text NOT NULL,
  access_token_lifetime_days integer NOT NULL DEFAULT 30,
  revoked_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_oauth_applications_client_id ON oauth_applications(client_id);
CREATE INDEX IF NOT EXISTS idx_oauth_applications_owner_user_id ON oauth_applications(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_applications_revoked_at ON oauth_applications(revoked_at);

CREATE TABLE IF NOT EXISTS oauth_grants (
  id text PRIMARY KEY,
  application_id text NOT NULL REFERENCES oauth_applications(id) ON DELETE CASCADE,
  user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  scope text NOT NULL,
  revoked_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_oauth_grants_application_id ON oauth_grants(application_id);
CREATE INDEX IF NOT EXISTS idx_oauth_grants_user_id ON oauth_grants(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_grants_revoked_at ON oauth_grants(revoked_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_oauth_grants_active_application_user
  ON oauth_grants(application_id, user_id) WHERE revoked_at IS NULL;

CREATE TABLE IF NOT EXISTS oauth_authorization_codes (
  id text PRIMARY KEY,
  application_id text NOT NULL REFERENCES oauth_applications(id) ON DELETE CASCADE,
  grant_id text NOT NULL REFERENCES oauth_grants(id) ON DELETE CASCADE,
  user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  code_hash text NOT NULL,
  redirect_uri text NOT NULL,
  scope text NOT NULL,
  code_challenge text NOT NULL,
  code_challenge_method text NOT NULL,
  expires_at timestamptz NOT NULL,
  consumed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_oauth_authorization_codes_code_hash ON oauth_authorization_codes(code_hash);
CREATE INDEX IF NOT EXISTS idx_oauth_authorization_codes_application_id ON oauth_authorization_codes(application_id);
CREATE INDEX IF NOT EXISTS idx_oauth_authorization_codes_grant_id ON oauth_authorization_codes(grant_id);
CREATE INDEX IF NOT EXISTS idx_oauth_authorization_codes_user_id ON oauth_authorization_codes(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_authorization_codes_expires_at ON oauth_authorization_codes(expires_at);
CREATE INDEX IF NOT EXISTS idx_oauth_authorization_codes_consumed_at ON oauth_authorization_codes(consumed_at);

CREATE TABLE IF NOT EXISTS oauth_refresh_tokens (
  id text PRIMARY KEY,
  application_id text NOT NULL REFERENCES oauth_applications(id) ON DELETE CASCADE,
  grant_id text NOT NULL REFERENCES oauth_grants(id) ON DELETE CASCADE,
  user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash text NOT NULL,
  scope text NOT NULL,
  expires_at timestamptz NOT NULL,
  consumed_at timestamptz,
  revoked_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_token_hash ON oauth_refresh_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_application_id ON oauth_refresh_tokens(application_id);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_grant_id ON oauth_refresh_tokens(grant_id);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_user_id ON oauth_refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_expires_at ON oauth_refresh_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_consumed_at ON oauth_refresh_tokens(consumed_at);
CREATE INDEX IF NOT EXISTS idx_oauth_refresh_tokens_revoked_at ON oauth_refresh_tokens(revoked_at);
