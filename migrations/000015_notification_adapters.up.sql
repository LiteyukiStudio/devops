CREATE TABLE IF NOT EXISTS notification_channels (
    id text PRIMARY KEY,
    project_id text NOT NULL DEFAULT '',
    name text NOT NULL,
    adapter_kind text NOT NULL,
    config_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    secret_refs_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    enabled boolean NOT NULL DEFAULT true,
    last_delivery_status text NOT NULL DEFAULT '',
    last_delivery_error text NOT NULL DEFAULT '',
    last_delivered_at timestamptz,
    created_by text,
    created_at timestamptz,
    updated_at timestamptz,
    deleted_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_notification_channels_project_id ON notification_channels(project_id);
CREATE INDEX IF NOT EXISTS idx_notification_channels_adapter_kind ON notification_channels(adapter_kind);
CREATE INDEX IF NOT EXISTS idx_notification_channels_deleted_at ON notification_channels(deleted_at);

CREATE TABLE IF NOT EXISTS notification_templates (
    id text PRIMARY KEY,
    project_id text NOT NULL DEFAULT '',
    name text NOT NULL,
    event_type text NOT NULL,
    adapter_kind text NOT NULL,
    locale text NOT NULL DEFAULT '',
    subject_template text NOT NULL DEFAULT '',
    body_template text NOT NULL DEFAULT '',
    json_body_template text NOT NULL DEFAULT '',
    enabled boolean NOT NULL DEFAULT true,
    created_by text,
    created_at timestamptz,
    updated_at timestamptz,
    deleted_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_notification_templates_project_id ON notification_templates(project_id);
CREATE INDEX IF NOT EXISTS idx_notification_templates_event_type ON notification_templates(event_type);
CREATE INDEX IF NOT EXISTS idx_notification_templates_adapter_kind ON notification_templates(adapter_kind);
CREATE INDEX IF NOT EXISTS idx_notification_templates_locale ON notification_templates(locale);
CREATE INDEX IF NOT EXISTS idx_notification_templates_deleted_at ON notification_templates(deleted_at);

CREATE TABLE IF NOT EXISTS notification_rules (
    id text PRIMARY KEY,
    project_id text NOT NULL DEFAULT '',
    name text NOT NULL,
    event_types_json jsonb NOT NULL DEFAULT '[]'::jsonb,
    filter_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    channel_ids_json jsonb NOT NULL DEFAULT '[]'::jsonb,
    template_id text NOT NULL DEFAULT '',
    locale text NOT NULL DEFAULT '',
    enabled boolean NOT NULL DEFAULT true,
    last_matched_event_id text NOT NULL DEFAULT '',
    created_by text,
    created_at timestamptz,
    updated_at timestamptz,
    deleted_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_notification_rules_project_id ON notification_rules(project_id);
CREATE INDEX IF NOT EXISTS idx_notification_rules_template_id ON notification_rules(template_id);
CREATE INDEX IF NOT EXISTS idx_notification_rules_deleted_at ON notification_rules(deleted_at);

CREATE TABLE IF NOT EXISTS notification_deliveries (
    id text PRIMARY KEY,
    project_id text NOT NULL DEFAULT '',
    event_id text NOT NULL,
    event_type text NOT NULL,
    severity text NOT NULL DEFAULT '',
    channel_id text NOT NULL,
    adapter_kind text NOT NULL,
    rule_id text NOT NULL DEFAULT '',
    template_id text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'pending',
    attempt_count bigint NOT NULL DEFAULT 0,
    duration_millis bigint NOT NULL DEFAULT 0,
    error_message text NOT NULL DEFAULT '',
    request_snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
    response_snippet text NOT NULL DEFAULT '',
    queued_at timestamptz,
    started_at timestamptz,
    finished_at timestamptz,
    created_at timestamptz,
    updated_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_notification_deliveries_project_id ON notification_deliveries(project_id);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_event_id ON notification_deliveries(event_id);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_event_type ON notification_deliveries(event_type);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_severity ON notification_deliveries(severity);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_channel_id ON notification_deliveries(channel_id);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_adapter_kind ON notification_deliveries(adapter_kind);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_rule_id ON notification_deliveries(rule_id);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_template_id ON notification_deliveries(template_id);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_status ON notification_deliveries(status);
