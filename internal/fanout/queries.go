package fanout

const enqueueDetectedReleaseQuery = `
	INSERT INTO detected_releases (repo_id, repo_name, release_tag, release_name, release_url)
	VALUES ($1, $2, $3, $4, $5)
`

const fetchDetectedReleasesForUpdateQuery = `
	SELECT id, repo_id, repo_name, release_tag, release_name, release_url
	FROM detected_releases
	WHERE processed_at IS NULL
	ORDER BY id
	LIMIT $1
	FOR UPDATE SKIP LOCKED
`

const markDetectedReleasesProcessedQuery = `
	UPDATE detected_releases SET processed_at = now() WHERE id = ANY($1)
`
