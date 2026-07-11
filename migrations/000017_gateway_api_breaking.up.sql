-- Breaking change before first public release:
-- Luna DevOps now uses Kubernetes Gateway API HTTPRoute as the access route runtime.
-- Existing pre-release gateway_routes rows referenced the old Ingress path and are not migrated.
DELETE FROM gateway_routes;

ALTER TABLE IF EXISTS runtime_clusters
    ADD COLUMN IF NOT EXISTS gateway_provider text NOT NULL DEFAULT 'gateway-api',
    ADD COLUMN IF NOT EXISTS gateway_class_name text NOT NULL DEFAULT 'traefik',
    ADD COLUMN IF NOT EXISTS gateway_name text NOT NULL DEFAULT 'luna-gateway',
    ADD COLUMN IF NOT EXISTS gateway_namespace text NOT NULL DEFAULT 'kube-system';

UPDATE runtime_clusters
SET gateway_provider = 'gateway-api',
    gateway_class_name = COALESCE(NULLIF(gateway_class_name, ''), NULLIF(gateway_ingress_class_name, ''), 'traefik'),
    gateway_name = COALESCE(NULLIF(gateway_name, ''), 'luna-gateway'),
    gateway_namespace = COALESCE(NULLIF(gateway_namespace, ''), 'kube-system'),
    gateway_external_tls_mode = CASE
        WHEN gateway_external_tls_mode = 'ingress' THEN 'gateway'
        WHEN gateway_external_tls_mode IN ('none', 'gateway', 'upstream') THEN gateway_external_tls_mode
        ELSE 'none'
    END,
    gateway_controller_type = CASE
        WHEN gateway_controller_type = 'traefik' THEN 'traefik'
        ELSE 'generic'
    END;

ALTER TABLE IF EXISTS runtime_clusters
    DROP COLUMN IF EXISTS gateway_ingress_class_name;

ALTER TABLE IF EXISTS gateway_routes
    ADD COLUMN IF NOT EXISTS parent_gateway_name text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS parent_gateway_namespace text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS section_name text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS path_match_type text NOT NULL DEFAULT 'PathPrefix',
    ADD COLUMN IF NOT EXISTS url_rewrite text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS request_redirect text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS backend_weight bigint NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS hostname_aliases text NOT NULL DEFAULT '';

ALTER TABLE IF EXISTS gateway_routes
    DROP COLUMN IF EXISTS ingress_class_name,
    DROP COLUMN IF EXISTS ingress_annotations,
    DROP COLUMN IF EXISTS strip_path_prefix,
    DROP COLUMN IF EXISTS preserve_host,
    DROP COLUMN IF EXISTS proxy_read_timeout_seconds,
    DROP COLUMN IF EXISTS proxy_send_timeout_seconds,
    DROP COLUMN IF EXISTS body_size_limit;
