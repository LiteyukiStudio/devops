ALTER TABLE deployment_targets
  DROP COLUMN IF EXISTS auto_scaling_behavior,
  DROP COLUMN IF EXISTS workload_type;
