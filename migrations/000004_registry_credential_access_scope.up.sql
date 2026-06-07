ALTER TABLE registry_credentials
  ADD COLUMN IF NOT EXISTS access_scope text NOT NULL DEFAULT 'personal';
