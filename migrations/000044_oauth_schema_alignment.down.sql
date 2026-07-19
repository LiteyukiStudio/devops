ALTER TABLE access_tokens
  ADD COLUMN IF NOT EXISTS o_auth_application_id text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS o_auth_grant_id text NOT NULL DEFAULT '';

UPDATE access_tokens
SET o_auth_application_id = oauth_application_id,
    o_auth_grant_id = oauth_grant_id
WHERE oauth_application_id <> '' AND oauth_grant_id <> '';

CREATE TABLE IF NOT EXISTS o_auth_applications (LIKE oauth_applications INCLUDING ALL);
CREATE TABLE IF NOT EXISTS o_auth_grants (LIKE oauth_grants INCLUDING ALL);
CREATE TABLE IF NOT EXISTS o_auth_authorization_codes (LIKE oauth_authorization_codes INCLUDING ALL);
CREATE TABLE IF NOT EXISTS o_auth_refresh_tokens (LIKE oauth_refresh_tokens INCLUDING ALL);

INSERT INTO o_auth_applications SELECT * FROM oauth_applications ON CONFLICT (id) DO NOTHING;
INSERT INTO o_auth_grants SELECT * FROM oauth_grants ON CONFLICT (id) DO NOTHING;
INSERT INTO o_auth_authorization_codes SELECT * FROM oauth_authorization_codes ON CONFLICT (id) DO NOTHING;
INSERT INTO o_auth_refresh_tokens SELECT * FROM oauth_refresh_tokens ON CONFLICT (id) DO NOTHING;
