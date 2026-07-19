ALTER TABLE deployment_targets
    ADD COLUMN IF NOT EXISTS build_definition_mode text NOT NULL DEFAULT 'repository_dockerfile',
    ADD COLUMN IF NOT EXISTS build_template_id text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS build_template_version text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS build_template_values text NOT NULL DEFAULT '{}';

CREATE INDEX IF NOT EXISTS idx_deployment_targets_build_template_id
    ON deployment_targets (build_template_id);

ALTER TABLE build_runs
    ADD COLUMN IF NOT EXISTS build_definition_mode text NOT NULL DEFAULT 'repository_dockerfile',
    ADD COLUMN IF NOT EXISTS build_template_id text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS build_template_version text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS build_template_values text NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS build_template_dockerfile text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS build_template_checksum text NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_build_runs_build_template_id
    ON build_runs (build_template_id);
