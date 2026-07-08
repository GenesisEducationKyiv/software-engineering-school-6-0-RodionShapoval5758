ALTER TABLE repositories ALTER COLUMN last_seen_tag DROP NOT NULL;
ALTER TABLE repositories ALTER COLUMN last_seen_tag DROP DEFAULT;
