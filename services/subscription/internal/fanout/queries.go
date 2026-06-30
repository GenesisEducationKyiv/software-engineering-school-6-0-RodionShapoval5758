package fanout

const updateLastSeenTagQuery = `
	UPDATE repositories SET last_seen_tag = $2, updated_at = now() WHERE id = $1
`
