CREATE TABLE detected_releases (
    id           BIGSERIAL PRIMARY KEY,
    repo_id      BIGINT      NOT NULL,
    repo_name    TEXT        NOT NULL,
    release_tag  TEXT        NOT NULL,
    release_name TEXT        NOT NULL,
    release_url  TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at TIMESTAMPTZ
);

CREATE INDEX detected_releases_pending ON detected_releases (id) WHERE processed_at IS NULL;
