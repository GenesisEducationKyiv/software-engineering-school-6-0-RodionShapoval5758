CREATE TABLE scan_cursors (
    repo_id      BIGINT      PRIMARY KEY,
    full_name    TEXT        NOT NULL,
    last_seen_tag TEXT       NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
