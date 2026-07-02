ALTER TABLE deployment_targets
  ADD COLUMN IF NOT EXISTS lifecycle text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS init_containers text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS sidecar_containers text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS auto_scaling_enabled boolean NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS auto_scaling_min_replicas integer NOT NULL DEFAULT 1,
  ADD COLUMN IF NOT EXISTS auto_scaling_max_replicas integer NOT NULL DEFAULT 1,
  ADD COLUMN IF NOT EXISTS auto_scaling_cpu_percent integer NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS auto_scaling_memory_percent integer NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS data_volume_mode text NOT NULL DEFAULT '';
