DROP INDEX IF EXISTS idx_git_accounts_scope_owner_ref;
DROP INDEX IF EXISTS idx_git_providers_scope_owner_ref;

ALTER TABLE git_accounts DROP COLUMN IF EXISTS scope;
ALTER TABLE git_accounts DROP COLUMN IF EXISTS owner_ref;
ALTER TABLE git_providers DROP COLUMN IF EXISTS scope;
ALTER TABLE git_providers DROP COLUMN IF EXISTS owner_ref;
