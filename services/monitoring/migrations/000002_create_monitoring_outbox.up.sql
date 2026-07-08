CREATE TABLE monitoring_outbox (
    id           BIGSERIAL   PRIMARY KEY,
    subject      TEXT        NOT NULL,
    payload      BYTEA       NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);

CREATE INDEX monitoring_outbox_pending ON monitoring_outbox (id) WHERE published_at IS NULL;
