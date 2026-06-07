CREATE TABLE IF NOT EXISTS build_logs (
  id text PRIMARY KEY,
  build_run_id text NOT NULL,
  build_job_id text NOT NULL,
  project_id text NOT NULL,
  content text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_build_logs_build_run_id ON build_logs(build_run_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_build_logs_build_job_id ON build_logs(build_job_id);
CREATE INDEX IF NOT EXISTS idx_build_logs_project_id ON build_logs(project_id);
