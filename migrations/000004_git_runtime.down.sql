DROP TABLE IF EXISTS git_oauth_states;

ALTER TABLE repository_bindings
  DROP COLUMN IF EXISTS webhook_id,
  DROP COLUMN IF EXISTS webhook_secret,
  DROP COLUMN IF EXISTS last_event,
  DROP COLUMN IF EXISTS last_commit_sha,
  DROP COLUMN IF EXISTS last_webhook_at;
