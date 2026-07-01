ALTER TABLE IF EXISTS runtime_clusters
    ADD COLUMN IF NOT EXISTS gateway_controller_type text NOT NULL DEFAULT 'traefik',
    ADD COLUMN IF NOT EXISTS gateway_ingress_class_name text NOT NULL DEFAULT 'traefik',
    ADD COLUMN IF NOT EXISTS gateway_external_tls_mode text NOT NULL DEFAULT 'none',
    ADD COLUMN IF NOT EXISTS gateway_forwarded_headers_mode text NOT NULL DEFAULT 'preserve',
    ADD COLUMN IF NOT EXISTS gateway_trusted_proxy_cidrs text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS gateway_default_request_headers text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS gateway_default_response_headers text NOT NULL DEFAULT '';

ALTER TABLE IF EXISTS gateway_routes
    ADD COLUMN IF NOT EXISTS ingress_class_name text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS request_headers text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS response_headers text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ingress_annotations text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS strip_path_prefix boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS preserve_host boolean NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS proxy_read_timeout_seconds bigint NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS proxy_send_timeout_seconds bigint NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS body_size_limit text NOT NULL DEFAULT '';
