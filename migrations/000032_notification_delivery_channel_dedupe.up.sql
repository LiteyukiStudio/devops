DELETE FROM notification_deliveries
WHERE id IN (
    SELECT id
    FROM (
        SELECT
            id,
            row_number() OVER (
                PARTITION BY event_id, channel_id
                ORDER BY
                    CASE WHEN status = 'succeeded' THEN 0 ELSE 1 END,
                    finished_at DESC NULLS LAST,
                    created_at DESC NULLS LAST,
                    id ASC
            ) AS occurrence
        FROM notification_deliveries
    ) ranked
    WHERE ranked.occurrence > 1
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_notification_deliveries_event_channel
    ON notification_deliveries(event_id, channel_id);
