ALTER TABLE deployment_targets
  DROP COLUMN IF EXISTS data_volume_mode,
  DROP COLUMN IF EXISTS auto_scaling_memory_percent,
  DROP COLUMN IF EXISTS auto_scaling_cpu_percent,
  DROP COLUMN IF EXISTS auto_scaling_max_replicas,
  DROP COLUMN IF EXISTS auto_scaling_min_replicas,
  DROP COLUMN IF EXISTS auto_scaling_enabled,
  DROP COLUMN IF EXISTS sidecar_containers,
  DROP COLUMN IF EXISTS init_containers,
  DROP COLUMN IF EXISTS lifecycle;
