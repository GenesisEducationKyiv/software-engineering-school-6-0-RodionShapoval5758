CREATE TABLE subscribe_sagas (
    id              BIGSERIAL    PRIMARY KEY,
    saga_id         TEXT         NOT NULL UNIQUE,
    subscription_id BIGINT,
    repository_id   BIGINT       NOT NULL,
    email           VARCHAR(255) NOT NULL,
    state           TEXT         NOT NULL,
    deadline_at     TIMESTAMPTZ  NOT NULL,
    last_error      TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX subscribe_sagas_subscription_id ON subscribe_sagas (subscription_id);
CREATE INDEX subscribe_sagas_active_deadlines
    ON subscribe_sagas (deadline_at)
    WHERE state IN ('STARTED', 'AWAITING_CONFIRMATION');
