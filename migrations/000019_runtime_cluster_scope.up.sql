ALTER TABLE runtime_clusters ADD COLUMN IF NOT EXISTS scope text NOT NULL DEFAULT 'global';
ALTER TABLE runtime_clusters ADD COLUMN IF NOT EXISTS owner_ref text NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_runtime_clusters_scope ON runtime_clusters(scope);
CREATE INDEX IF NOT EXISTS idx_runtime_clusters_owner_ref ON runtime_clusters(owner_ref);

UPDATE runtime_clusters
SET scope = 'global', owner_ref = ''
WHERE scope IS NULL OR scope = '';
