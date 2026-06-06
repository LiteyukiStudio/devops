CREATE TABLE IF NOT EXISTS git_oauth_states (
  id text PRIMARY KEY,
  state_hash text NOT NULL UNIQUE,
  provider_id text NOT NULL REFERENCES git_providers(id) ON DELETE CASCADE,
  user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  redirect_path text NOT NULL DEFAULT '/projects',
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_git_oauth_states_expires_at ON git_oauth_states (expires_at);

ALTER TABLE repository_bindings
  ADD COLUMN IF NOT EXISTS webhook_id text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS webhook_secret text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS last_event text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS last_commit_sha text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS last_webhook_at timestamptz;
