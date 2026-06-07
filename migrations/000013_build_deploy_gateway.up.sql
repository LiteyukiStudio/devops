CREATE TABLE IF NOT EXISTS build_providers (
  id text PRIMARY KEY,
  name text NOT NULL,
  type text NOT NULL DEFAULT 'platform',
  scope text NOT NULL DEFAULT 'global',
  owner_ref text NOT NULL DEFAULT '',
  config text NOT NULL DEFAULT '',
  enabled boolean NOT NULL DEFAULT true,
  created_by text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_build_providers_scope ON build_providers(scope);
CREATE INDEX IF NOT EXISTS idx_build_providers_owner_ref ON build_providers(owner_ref);
CREATE INDEX IF NOT EXISTS idx_build_providers_created_by ON build_providers(created_by);
CREATE INDEX IF NOT EXISTS idx_build_providers_deleted_at ON build_providers(deleted_at);

CREATE TABLE IF NOT EXISTS build_runs (
  id text PRIMARY KEY,
  project_id text NOT NULL,
  application_id text NOT NULL DEFAULT '',
  build_provider_id text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'queued',
  trigger_type text NOT NULL DEFAULT 'manual',
  source_branch text NOT NULL DEFAULT '',
  source_tag text NOT NULL DEFAULT '',
  source_commit text NOT NULL DEFAULT '',
  dockerfile_path text NOT NULL DEFAULT 'Dockerfile',
  build_context text NOT NULL DEFAULT '.',
  build_directory text NOT NULL DEFAULT '',
  target_registry_id text NOT NULL DEFAULT '',
  target_repository text NOT NULL DEFAULT '',
  target_tag text NOT NULL DEFAULT '',
  image_ref text NOT NULL DEFAULT '',
  image_digest text NOT NULL DEFAULT '',
  cache_config text NOT NULL DEFAULT '',
  cpu_core_seconds bigint NOT NULL DEFAULT 0,
  memory_mb_seconds bigint NOT NULL DEFAULT 0,
  credit_cost bigint NOT NULL DEFAULT 0,
  started_at timestamptz,
  finished_at timestamptz,
  created_by text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_build_runs_project_id ON build_runs(project_id);
CREATE INDEX IF NOT EXISTS idx_build_runs_application_id ON build_runs(application_id);
CREATE INDEX IF NOT EXISTS idx_build_runs_build_provider_id ON build_runs(build_provider_id);
CREATE INDEX IF NOT EXISTS idx_build_runs_status ON build_runs(status);
CREATE INDEX IF NOT EXISTS idx_build_runs_target_registry_id ON build_runs(target_registry_id);
CREATE INDEX IF NOT EXISTS idx_build_runs_created_by ON build_runs(created_by);
CREATE INDEX IF NOT EXISTS idx_build_runs_deleted_at ON build_runs(deleted_at);

CREATE TABLE IF NOT EXISTS build_jobs (
  id text PRIMARY KEY,
  build_run_id text NOT NULL,
  project_id text NOT NULL,
  type text NOT NULL DEFAULT 'build',
  status text NOT NULL DEFAULT 'queued',
  message text NOT NULL DEFAULT '',
  log_ref text NOT NULL DEFAULT '',
  attempts integer NOT NULL DEFAULT 0,
  started_at timestamptz,
  finished_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_build_jobs_build_run_id ON build_jobs(build_run_id);
CREATE INDEX IF NOT EXISTS idx_build_jobs_project_id ON build_jobs(project_id);
CREATE INDEX IF NOT EXISTS idx_build_jobs_status ON build_jobs(status);
CREATE INDEX IF NOT EXISTS idx_build_jobs_deleted_at ON build_jobs(deleted_at);

CREATE TABLE IF NOT EXISTS runtime_clusters (
  id text PRIMARY KEY,
  name text NOT NULL,
  type text NOT NULL DEFAULT 'kubernetes',
  endpoint text NOT NULL DEFAULT '',
  scope text NOT NULL DEFAULT 'global',
  owner_ref text NOT NULL DEFAULT '',
  kubeconfig_ref text NOT NULL DEFAULT '',
  is_default boolean NOT NULL DEFAULT false,
  status text NOT NULL DEFAULT 'unknown',
  last_checked_at timestamptz,
  created_by text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_runtime_clusters_scope ON runtime_clusters(scope);
CREATE INDEX IF NOT EXISTS idx_runtime_clusters_owner_ref ON runtime_clusters(owner_ref);
CREATE INDEX IF NOT EXISTS idx_runtime_clusters_created_by ON runtime_clusters(created_by);
CREATE INDEX IF NOT EXISTS idx_runtime_clusters_deleted_at ON runtime_clusters(deleted_at);

CREATE TABLE IF NOT EXISTS environments (
  id text PRIMARY KEY,
  project_id text NOT NULL,
  name text NOT NULL,
  slug text NOT NULL,
  stage text NOT NULL DEFAULT 'dev',
  cluster_id text NOT NULL DEFAULT '',
  namespace text NOT NULL DEFAULT '',
  replicas integer NOT NULL DEFAULT 1,
  cpu_request text NOT NULL DEFAULT '',
  memory_request text NOT NULL DEFAULT '',
  env_vars text NOT NULL DEFAULT '',
  config_refs text NOT NULL DEFAULT '',
  secret_refs text NOT NULL DEFAULT '',
  created_by text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_environments_project_id ON environments(project_id);
CREATE INDEX IF NOT EXISTS idx_environments_slug ON environments(slug);
CREATE INDEX IF NOT EXISTS idx_environments_cluster_id ON environments(cluster_id);
CREATE INDEX IF NOT EXISTS idx_environments_created_by ON environments(created_by);
CREATE INDEX IF NOT EXISTS idx_environments_deleted_at ON environments(deleted_at);

CREATE TABLE IF NOT EXISTS releases (
  id text PRIMARY KEY,
  project_id text NOT NULL,
  application_id text NOT NULL,
  environment_id text NOT NULL,
  build_run_id text NOT NULL DEFAULT '',
  image_ref text NOT NULL,
  type text NOT NULL DEFAULT 'deploy',
  status text NOT NULL DEFAULT 'pending',
  revision integer NOT NULL DEFAULT 1,
  rollback_from_id text NOT NULL DEFAULT '',
  message text NOT NULL DEFAULT '',
  started_at timestamptz,
  finished_at timestamptz,
  created_by text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_releases_project_id ON releases(project_id);
CREATE INDEX IF NOT EXISTS idx_releases_application_id ON releases(application_id);
CREATE INDEX IF NOT EXISTS idx_releases_environment_id ON releases(environment_id);
CREATE INDEX IF NOT EXISTS idx_releases_build_run_id ON releases(build_run_id);
CREATE INDEX IF NOT EXISTS idx_releases_status ON releases(status);
CREATE INDEX IF NOT EXISTS idx_releases_rollback_from_id ON releases(rollback_from_id);
CREATE INDEX IF NOT EXISTS idx_releases_created_by ON releases(created_by);
CREATE INDEX IF NOT EXISTS idx_releases_deleted_at ON releases(deleted_at);

CREATE TABLE IF NOT EXISTS gateway_routes (
  id text PRIMARY KEY,
  project_id text NOT NULL,
  application_id text NOT NULL,
  environment_id text NOT NULL DEFAULT '',
  host text NOT NULL,
  path text NOT NULL DEFAULT '/',
  service_port integer NOT NULL DEFAULT 80,
  tls_mode text NOT NULL DEFAULT 'http-only',
  certificate_status text NOT NULL DEFAULT 'disabled',
  cname_name text NOT NULL DEFAULT '',
  cname_target text NOT NULL DEFAULT '',
  dns_status text NOT NULL DEFAULT 'pending',
  status text NOT NULL DEFAULT 'pending',
  is_default boolean NOT NULL DEFAULT false,
  created_by text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_gateway_routes_project_id ON gateway_routes(project_id);
CREATE INDEX IF NOT EXISTS idx_gateway_routes_application_id ON gateway_routes(application_id);
CREATE INDEX IF NOT EXISTS idx_gateway_routes_environment_id ON gateway_routes(environment_id);
CREATE INDEX IF NOT EXISTS idx_gateway_routes_host ON gateway_routes(host);
CREATE INDEX IF NOT EXISTS idx_gateway_routes_created_by ON gateway_routes(created_by);
CREATE INDEX IF NOT EXISTS idx_gateway_routes_deleted_at ON gateway_routes(deleted_at);
