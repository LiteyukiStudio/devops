ALTER TABLE IF EXISTS gateway_routes
    DROP COLUMN IF EXISTS hostname_aliases,
    DROP COLUMN IF EXISTS backend_weight,
    DROP COLUMN IF EXISTS request_redirect,
    DROP COLUMN IF EXISTS url_rewrite,
    DROP COLUMN IF EXISTS path_match_type,
    DROP COLUMN IF EXISTS section_name,
    DROP COLUMN IF EXISTS parent_gateway_namespace,
    DROP COLUMN IF EXISTS parent_gateway_name;

ALTER TABLE IF EXISTS gateway_routes
    ADD COLUMN IF NOT EXISTS ingress_class_name text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ingress_annotations text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS strip_path_prefix boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS preserve_host boolean NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS proxy_read_timeout_seconds bigint NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS proxy_send_timeout_seconds bigint NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS body_size_limit text NOT NULL DEFAULT '';

ALTER TABLE IF EXISTS runtime_clusters
    ADD COLUMN IF NOT EXISTS gateway_ingress_class_name text NOT NULL DEFAULT 'traefik';

UPDATE runtime_clusters
SET gateway_ingress_class_name = COALESCE(NULLIF(gateway_ingress_class_name, ''), NULLIF(gateway_class_name, ''), 'traefik'),
    gateway_external_tls_mode = CASE
        WHEN gateway_external_tls_mode = 'gateway' THEN 'ingress'
        WHEN gateway_external_tls_mode IN ('none', 'ingress', 'upstream') THEN gateway_external_tls_mode
        ELSE 'none'
    END;

ALTER TABLE IF EXISTS runtime_clusters
    DROP COLUMN IF EXISTS gateway_namespace,
    DROP COLUMN IF EXISTS gateway_name,
    DROP COLUMN IF EXISTS gateway_class_name,
    DROP COLUMN IF EXISTS gateway_provider;
