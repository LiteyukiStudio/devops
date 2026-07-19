DROP TABLE IF EXISTS oauth_refresh_tokens;
DROP TABLE IF EXISTS oauth_authorization_codes;
DROP TABLE IF EXISTS oauth_grants;
DROP TABLE IF EXISTS oauth_applications;

DROP INDEX IF EXISTS idx_access_tokens_oauth_grant_id;
DROP INDEX IF EXISTS idx_access_tokens_oauth_application_id;
DROP INDEX IF EXISTS idx_access_tokens_source;
DROP INDEX IF EXISTS idx_access_tokens_token_hash;

ALTER TABLE access_tokens
  DROP COLUMN IF EXISTS oauth_grant_id,
  DROP COLUMN IF EXISTS oauth_application_id,
  DROP COLUMN IF EXISTS source;
