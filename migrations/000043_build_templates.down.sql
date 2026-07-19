DROP INDEX IF EXISTS idx_build_runs_build_template_id;
ALTER TABLE build_runs
    DROP COLUMN IF EXISTS build_template_checksum,
    DROP COLUMN IF EXISTS build_template_dockerfile,
    DROP COLUMN IF EXISTS build_template_values,
    DROP COLUMN IF EXISTS build_template_version,
    DROP COLUMN IF EXISTS build_template_id,
    DROP COLUMN IF EXISTS build_definition_mode;

DROP INDEX IF EXISTS idx_deployment_targets_build_template_id;
ALTER TABLE deployment_targets
    DROP COLUMN IF EXISTS build_template_values,
    DROP COLUMN IF EXISTS build_template_version,
    DROP COLUMN IF EXISTS build_template_id,
    DROP COLUMN IF EXISTS build_definition_mode;
