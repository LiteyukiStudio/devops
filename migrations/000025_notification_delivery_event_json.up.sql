ALTER TABLE IF EXISTS notification_deliveries
    ADD COLUMN IF NOT EXISTS event_json jsonb NOT NULL DEFAULT '{}'::jsonb;
