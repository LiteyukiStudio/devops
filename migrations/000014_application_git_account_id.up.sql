ALTER TABLE applications
  ADD COLUMN IF NOT EXISTS git_account_id text NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_applications_git_account
  ON applications (git_account_id)
  WHERE deleted_at IS NULL;

UPDATE applications
SET git_account_id = repository_bindings.git_account_id
FROM repository_bindings
WHERE applications.id = repository_bindings.application_id
  AND applications.project_id = repository_bindings.project_id
  AND repository_bindings.deleted_at IS NULL
  AND applications.git_account_id = '';
