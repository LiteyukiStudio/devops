ALTER TABLE build_jobs ADD COLUMN IF NOT EXISTS builder_id text;
ALTER TABLE build_jobs ADD COLUMN IF NOT EXISTS lease_until timestamptz;

CREATE INDEX IF NOT EXISTS idx_build_jobs_builder_id ON build_jobs (builder_id);
CREATE INDEX IF NOT EXISTS idx_build_jobs_lease_until ON build_jobs (lease_until);

CREATE TABLE IF NOT EXISTS builder_agents (
  id text PRIMARY KEY,
  name text NOT NULL,
  labels text,
  executor text,
  status text NOT NULL DEFAULT 'online',
  max_concurrency integer NOT NULL DEFAULT 1,
  current_concurrency integer NOT NULL DEFAULT 0,
  last_heartbeat_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_builder_agents_status ON builder_agents (status);
