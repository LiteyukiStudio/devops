ALTER TABLE deployment_targets
  ADD COLUMN IF NOT EXISTS workload_type text NOT NULL DEFAULT 'Deployment',
  ADD COLUMN IF NOT EXISTS auto_scaling_behavior text NOT NULL DEFAULT '';
