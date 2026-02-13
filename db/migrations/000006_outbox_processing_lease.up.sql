BEGIN;

ALTER TABLE outbox_events
    ADD COLUMN IF NOT EXISTS processing_started_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_outbox_events_processing_started
    ON outbox_events(status, processing_started_at)
    WHERE status = 'processing';

COMMIT;
