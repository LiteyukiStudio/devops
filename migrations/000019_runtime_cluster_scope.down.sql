DROP INDEX IF EXISTS idx_runtime_clusters_owner_ref;
DROP INDEX IF EXISTS idx_runtime_clusters_scope;
ALTER TABLE runtime_clusters DROP COLUMN IF EXISTS owner_ref;
ALTER TABLE runtime_clusters DROP COLUMN IF EXISTS scope;
