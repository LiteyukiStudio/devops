CREATE TABLE IF NOT EXISTS secret_values (
  id text PRIMARY KEY,
  cipher_ref text NOT NULL,
  created_by text NOT NULL DEFAULT '',
  resource text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_secret_values_created_by ON secret_values (created_by);
CREATE INDEX IF NOT EXISTS idx_secret_values_resource ON secret_values (resource);
