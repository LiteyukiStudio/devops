ALTER TABLE runtime_clusters
  ADD COLUMN IF NOT EXISTS gateway_domain_suffixes text NOT NULL DEFAULT '';

UPDATE runtime_clusters
SET gateway_domain_suffixes = gateway_root_domain
WHERE COALESCE(NULLIF(gateway_domain_suffixes, ''), '') = ''
  AND COALESCE(NULLIF(gateway_root_domain, ''), '') <> '';

ALTER TABLE gateway_routes
  ADD COLUMN IF NOT EXISTS domain_suffix text NOT NULL DEFAULT '';
