CREATE TABLE IF NOT EXISTS platform_events (
    id text PRIMARY KEY,
    type text NOT NULL,
    category text NOT NULL,
    severity text NOT NULL,
    status text NOT NULL,
    project_id text NOT NULL DEFAULT '',
    application_id text NOT NULL DEFAULT '',
    deployment_target_id text NOT NULL DEFAULT '',
    resource_type text NOT NULL DEFAULT '',
    resource_id text NOT NULL DEFAULT '',
    actor_id text NOT NULL DEFAULT '',
    summary_key text NOT NULL DEFAULT '',
    message text NOT NULL DEFAULT '',
    detail_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    links_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    correlation_id text NOT NULL DEFAULT '',
    trace_id text NOT NULL DEFAULT '',
    dedup_key text,
    occurred_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_platform_events_type ON platform_events(type);
CREATE INDEX IF NOT EXISTS idx_platform_events_category ON platform_events(category);
CREATE INDEX IF NOT EXISTS idx_platform_events_severity ON platform_events(severity);
CREATE INDEX IF NOT EXISTS idx_platform_events_status ON platform_events(status);
CREATE INDEX IF NOT EXISTS idx_platform_events_project_id ON platform_events(project_id);
CREATE INDEX IF NOT EXISTS idx_platform_events_application_id ON platform_events(application_id);
CREATE INDEX IF NOT EXISTS idx_platform_events_deployment_target_id ON platform_events(deployment_target_id);
CREATE INDEX IF NOT EXISTS idx_platform_events_resource ON platform_events(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_platform_events_actor_id ON platform_events(actor_id);
CREATE INDEX IF NOT EXISTS idx_platform_events_correlation_id ON platform_events(correlation_id);
CREATE INDEX IF NOT EXISTS idx_platform_events_occurred_at ON platform_events(occurred_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_platform_events_dedup_key ON platform_events(dedup_key) WHERE dedup_key IS NOT NULL;
