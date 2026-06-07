ALTER TABLE git_accounts
  ADD COLUMN IF NOT EXISTS access_scope text NOT NULL DEFAULT 'personal';
