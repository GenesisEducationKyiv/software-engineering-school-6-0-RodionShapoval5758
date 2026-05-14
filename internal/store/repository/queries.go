package repository

const createRepoQuery = `
	INSERT INTO repositories (name)
	VALUES ($1)
	RETURNING id, name, last_seen_tag, created_at, updated_at
`

const findByNameQuery = `
	SELECT id, name, last_seen_tag, created_at, updated_at
	FROM repositories
	WHERE name = $1
`

const updateLastSeenTagByIDQuery = `
	UPDATE repositories
		SET last_seen_tag = $2, updated_at = now()
	WHERE id = $1
`

const deleteByIDQuery = `
	DELETE FROM repositories
	WHERE id = $1
`

const listTrackedReposQuery = `
	SELECT id, name, last_seen_tag, created_at, updated_at
	FROM repositories
`
