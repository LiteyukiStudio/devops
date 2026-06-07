CREATE TABLE IF NOT EXISTS worker_task_events (
  id text PRIMARY KEY,
  task_id text NOT NULL,
  task_type text NOT NULL,
  dedupe_key text NOT NULL,
  actor_id text,
  resource_ref text,
  status text NOT NULL,
  message text,
  attempt integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_worker_task_events_task_id ON worker_task_events (task_id);
CREATE INDEX IF NOT EXISTS idx_worker_task_events_task_type ON worker_task_events (task_type);
CREATE INDEX IF NOT EXISTS idx_worker_task_events_dedupe_key ON worker_task_events (dedupe_key);
CREATE INDEX IF NOT EXISTS idx_worker_task_events_actor_id ON worker_task_events (actor_id);
CREATE INDEX IF NOT EXISTS idx_worker_task_events_resource_ref ON worker_task_events (resource_ref);
CREATE INDEX IF NOT EXISTS idx_worker_task_events_status ON worker_task_events (status);
