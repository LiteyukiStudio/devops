DO $$
BEGIN
  IF to_regclass('public.o_auth_applications') IS NOT NULL THEN
    INSERT INTO oauth_applications (
      id, owner_user_id, name, description, homepage_url, logo_url, client_id,
      client_secret_hash, redirect_uris, allowed_scopes, access_token_lifetime_days,
      revoked_at, created_at, updated_at
    )
    SELECT
      id, owner_user_id, name, COALESCE(description, ''), COALESCE(homepage_url, ''),
      COALESCE(logo_url, ''), client_id, client_secret_hash, redirect_uris,
      allowed_scopes, access_token_lifetime_days::integer, revoked_at,
      COALESCE(created_at, now()), COALESCE(updated_at, now())
    FROM o_auth_applications
    ON CONFLICT (id) DO NOTHING;
  END IF;

  IF to_regclass('public.o_auth_grants') IS NOT NULL THEN
    INSERT INTO oauth_grants (
      id, application_id, user_id, scope, revoked_at, created_at, updated_at
    )
    SELECT
      id, application_id, user_id, scope, revoked_at,
      COALESCE(created_at, now()), COALESCE(updated_at, now())
    FROM o_auth_grants
    ON CONFLICT (id) DO NOTHING;
  END IF;

  IF to_regclass('public.o_auth_authorization_codes') IS NOT NULL THEN
    INSERT INTO oauth_authorization_codes (
      id, application_id, grant_id, user_id, code_hash, redirect_uri, scope,
      code_challenge, code_challenge_method, expires_at, consumed_at, created_at
    )
    SELECT
      id, application_id, grant_id, user_id, code_hash, redirect_uri, scope,
      code_challenge, code_challenge_method, expires_at, consumed_at,
      COALESCE(created_at, now())
    FROM o_auth_authorization_codes
    ON CONFLICT (id) DO NOTHING;
  END IF;

  IF to_regclass('public.o_auth_refresh_tokens') IS NOT NULL THEN
    INSERT INTO oauth_refresh_tokens (
      id, application_id, grant_id, user_id, token_hash, scope, expires_at,
      consumed_at, revoked_at, created_at, updated_at
    )
    SELECT
      id, application_id, grant_id, user_id, token_hash, scope, expires_at,
      consumed_at, revoked_at, COALESCE(created_at, now()), COALESCE(updated_at, now())
    FROM o_auth_refresh_tokens
    ON CONFLICT (id) DO NOTHING;
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'access_tokens'
      AND column_name = 'o_auth_application_id'
  ) THEN
    EXECUTE $sql$
      UPDATE access_tokens
      SET oauth_application_id = o_auth_application_id,
          oauth_grant_id = o_auth_grant_id
      WHERE oauth_application_id = ''
        AND oauth_grant_id = ''
        AND o_auth_application_id <> ''
        AND o_auth_grant_id <> ''
    $sql$;
  END IF;
END $$;

ALTER TABLE access_tokens
  DROP COLUMN IF EXISTS o_auth_application_id,
  DROP COLUMN IF EXISTS o_auth_grant_id;

UPDATE access_tokens AS token
SET revoked_at = oauth_grant.revoked_at
FROM oauth_grants AS oauth_grant
WHERE token.oauth_grant_id = oauth_grant.id
  AND token.revoked_at IS NULL
  AND oauth_grant.revoked_at IS NOT NULL;

DROP TABLE IF EXISTS o_auth_refresh_tokens;
DROP TABLE IF EXISTS o_auth_authorization_codes;
DROP TABLE IF EXISTS o_auth_grants;
DROP TABLE IF EXISTS o_auth_applications;
