-- Up migration
CREATE TABLE IF NOT EXISTS repositories (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL, -- format: owner/repo
    last_seen_tag VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS subscriptions (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    repository_id BIGINT NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    confirmed BOOLEAN NOT NULL DEFAULT FALSE,
    confirmation_token VARCHAR(255) UNIQUE,
    unsubscribe_token VARCHAR(255) UNIQUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    confirmed_at TIMESTAMPTZ,
    UNIQUE (email, repository_id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_subscriptions_email ON subscriptions(email);
CREATE INDEX IF NOT EXISTS idx_subscriptions_confirmation_token ON subscriptions(confirmation_token);
CREATE INDEX IF NOT EXISTS idx_subscriptions_unsubscribe_token ON subscriptions(unsubscribe_token);
