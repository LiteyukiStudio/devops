DROP INDEX IF EXISTS idx_applications_git_account;

ALTER TABLE applications
  DROP COLUMN IF EXISTS git_account_id;
