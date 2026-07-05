ALTER TABLE deployment_targets ADD COLUMN IF NOT EXISTS build_args text NOT NULL DEFAULT '';
ALTER TABLE build_runs ADD COLUMN IF NOT EXISTS build_args text NOT NULL DEFAULT '';
