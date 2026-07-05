ALTER TABLE gateway_routes
  DROP COLUMN IF EXISTS domain_suffix;

ALTER TABLE runtime_clusters
  DROP COLUMN IF EXISTS gateway_domain_suffixes;
