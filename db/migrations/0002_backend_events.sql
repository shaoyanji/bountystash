BEGIN;

CREATE TABLE backend_events (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    event_type TEXT NOT NULL,
    work_item_id UUID,
    work_version_id UUID,
    payload JSONB
);

CREATE INDEX backend_events_work_item_idx
    ON backend_events (work_item_id);

CREATE INDEX backend_events_created_at_idx
    ON backend_events (created_at DESC);

COMMIT;
