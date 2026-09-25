CREATE TABLE outbox (
    id           BIGSERIAL PRIMARY KEY,
    subject      TEXT        NOT NULL,
    payload      BYTEA       NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);

CREATE INDEX outbox_pending ON outbox (id) WHERE published_at IS NULL;
