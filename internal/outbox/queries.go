package outbox

const insertOutboxQuery = `INSERT INTO outbox (subject, payload) VALUES ($1, $2)`

const fetchOutboxForUpdateQuery = `
	SELECT id, subject, payload
	FROM outbox
	WHERE published_at IS NULL
	ORDER BY id
	LIMIT $1
	FOR UPDATE SKIP LOCKED
`

const markOutboxPublishedQuery = `
	UPDATE outbox SET published_at = now() WHERE id = ANY($1)
`
