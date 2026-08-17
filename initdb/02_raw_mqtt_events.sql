CREATE TABLE IF NOT EXISTS raw.mqtt_events (
    id          BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    topic       TEXT        NOT NULL,
    payload     JSONB       NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX mqtt_events_topic_index ON raw.mqtt_events (topic);
