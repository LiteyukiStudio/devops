ALTER TABLE git_providers
  ADD COLUMN IF NOT EXISTS scope text NOT NULL DEFAULT 'user';

ALTER TABLE git_providers
  ADD COLUMN IF NOT EXISTS owner_ref text;

CREATE INDEX IF NOT EXISTS idx_git_providers_scope_owner_ref ON git_providers (scope, owner_ref);

ALTER TABLE git_accounts
  ADD COLUMN IF NOT EXISTS scope text NOT NULL DEFAULT 'user';

ALTER TABLE git_accounts
  ADD COLUMN IF NOT EXISTS owner_ref text;

CREATE INDEX IF NOT EXISTS idx_git_accounts_scope_owner_ref ON git_accounts (scope, owner_ref);
