-- Down migration
DROP INDEX IF EXISTS idx_subscriptions_unsubscribe_token;
DROP INDEX IF EXISTS idx_subscriptions_confirmation_token;
DROP INDEX IF EXISTS idx_subscriptions_email;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS repositories;
