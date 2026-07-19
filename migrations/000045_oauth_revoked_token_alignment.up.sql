UPDATE access_tokens AS token
SET revoked_at = oauth_grant.revoked_at
FROM oauth_grants AS oauth_grant
WHERE token.oauth_grant_id = oauth_grant.id
  AND token.revoked_at IS NULL
  AND oauth_grant.revoked_at IS NOT NULL;
