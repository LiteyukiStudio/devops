CREATE TABLE service_bindings (
  id text PRIMARY KEY,
  project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  source_application_id text NOT NULL REFERENCES applications(id) ON DELETE RESTRICT,
  source_deployment_target_id text NOT NULL REFERENCES deployment_targets(id) ON DELETE RESTRICT,
  target_application_id text NOT NULL REFERENCES applications(id) ON DELETE RESTRICT,
  target_deployment_target_id text NOT NULL REFERENCES deployment_targets(id) ON DELETE RESTRICT,
  target_port_name text NOT NULL,
  target_port integer NOT NULL CHECK (target_port BETWEEN 1 AND 65535),
  protocol text NOT NULL CHECK (protocol IN ('http', 'https', 'tcp')),
  path text NOT NULL DEFAULT '',
  injection_mode text NOT NULL CHECK (injection_mode IN ('url', 'host_port')),
  url_env_var text NOT NULL DEFAULT '',
  host_env_var text NOT NULL DEFAULT '',
  port_env_var text NOT NULL DEFAULT '',
  enabled boolean NOT NULL DEFAULT true,
  last_check_status text NOT NULL DEFAULT '',
  last_checked_at timestamptz,
  created_by text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (source_deployment_target_id <> target_deployment_target_id),
  CHECK (
    (injection_mode = 'url' AND url_env_var <> '' AND host_env_var = '' AND port_env_var = '') OR
    (injection_mode = 'host_port' AND url_env_var = '' AND host_env_var <> '' AND port_env_var <> '')
  )
);

CREATE INDEX idx_service_bindings_project_id ON service_bindings(project_id);
CREATE INDEX idx_service_bindings_source_application_id ON service_bindings(source_application_id);
CREATE INDEX idx_service_bindings_source_target_id ON service_bindings(source_deployment_target_id);
CREATE INDEX idx_service_bindings_target_application_id ON service_bindings(target_application_id);
CREATE INDEX idx_service_bindings_target_target_id ON service_bindings(target_deployment_target_id);
CREATE INDEX idx_service_bindings_project_enabled ON service_bindings(project_id, enabled);
CREATE UNIQUE INDEX idx_service_bindings_source_url_env
  ON service_bindings(source_deployment_target_id, url_env_var)
  WHERE url_env_var <> '';
CREATE UNIQUE INDEX idx_service_bindings_source_host_env
  ON service_bindings(source_deployment_target_id, host_env_var)
  WHERE host_env_var <> '';
CREATE UNIQUE INDEX idx_service_bindings_source_port_env
  ON service_bindings(source_deployment_target_id, port_env_var)
  WHERE port_env_var <> '';

CREATE TABLE project_topology_edges (
  id text PRIMARY KEY,
  project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  source_application_id text NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
  source_deployment_target_id text NOT NULL DEFAULT '',
  target_application_id text NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
  target_deployment_target_id text NOT NULL DEFAULT '',
  relation_type text NOT NULL CHECK (relation_type IN ('depends_on', 'calls', 'reads_writes', 'publishes_to', 'consumes_from')),
  protocol text NOT NULL DEFAULT '' CHECK (protocol IN ('', 'http', 'https', 'tcp')),
  port integer NOT NULL DEFAULT 0 CHECK (port BETWEEN 0 AND 65535),
  description text NOT NULL DEFAULT '',
  created_by text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (source_application_id <> target_application_id)
);

CREATE INDEX idx_project_topology_edges_project_id ON project_topology_edges(project_id);
CREATE INDEX idx_project_topology_edges_source_application_id ON project_topology_edges(source_application_id);
CREATE INDEX idx_project_topology_edges_target_application_id ON project_topology_edges(target_application_id);
CREATE INDEX idx_project_topology_edges_source_target_id ON project_topology_edges(source_deployment_target_id);
CREATE INDEX idx_project_topology_edges_target_target_id ON project_topology_edges(target_deployment_target_id);
CREATE UNIQUE INDEX idx_project_topology_edges_identity
  ON project_topology_edges(
    project_id,
    source_application_id,
    source_deployment_target_id,
    target_application_id,
    target_deployment_target_id,
    relation_type,
    protocol,
    port
  );
