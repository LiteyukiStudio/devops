DROP TABLE IF EXISTS builder_agents;

DROP INDEX IF EXISTS idx_build_jobs_builder_id;
DROP INDEX IF EXISTS idx_build_jobs_lease_until;

ALTER TABLE build_jobs DROP COLUMN IF EXISTS builder_id;
ALTER TABLE build_jobs DROP COLUMN IF EXISTS lease_until;
