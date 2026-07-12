ALTER TABLE user_sessions ADD COLUMN IF NOT EXISTS primary_authenticated_at timestamptz;
CREATE INDEX IF NOT EXISTS idx_user_sessions_primary_authenticated_at ON user_sessions(primary_authenticated_at);
