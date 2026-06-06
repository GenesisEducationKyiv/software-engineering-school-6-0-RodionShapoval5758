UPDATE repositories SET last_seen_tag = '' WHERE last_seen_tag IS NULL;
ALTER TABLE repositories ALTER COLUMN last_seen_tag SET DEFAULT '';
ALTER TABLE repositories ALTER COLUMN last_seen_tag SET NOT NULL;
