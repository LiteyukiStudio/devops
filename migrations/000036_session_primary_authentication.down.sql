DROP INDEX IF EXISTS idx_user_sessions_primary_authenticated_at;
ALTER TABLE user_sessions DROP COLUMN IF EXISTS primary_authenticated_at;
