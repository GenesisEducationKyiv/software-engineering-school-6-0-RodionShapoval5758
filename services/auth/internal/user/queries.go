package user

const insertUserQuery = `
	INSERT INTO users (email, password_hash)
	VALUES ($1, $2)
	RETURNING id
`

const getUserByEmailQuery = `
	SELECT id, email, password_hash, email_verified
	FROM users
	WHERE email = $1
`

const insertVerificationQuery = `
	INSERT INTO email_verifications (token, user_id, expires_at)
	VALUES ($1, $2, $3)
`

const getVerificationForUpdateQuery = `
	SELECT user_id, expires_at, used_at
	FROM email_verifications
	WHERE token = $1
	FOR UPDATE
`

const markVerificationUsedQuery = `
	UPDATE email_verifications SET used_at = now() WHERE token = $1
`

const markUserVerifiedQuery = `
	UPDATE users SET email_verified = true WHERE id = $1
`

const insertRefreshTokenQuery = `
	INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
	VALUES ($1, $2, $3)
`

const getRefreshTokenForUpdateQuery = `
	SELECT rt.user_id, rt.expires_at, rt.revoked_at, u.email
	FROM refresh_tokens rt
	JOIN users u ON u.id = rt.user_id
	WHERE rt.token_hash = $1
	FOR UPDATE OF rt
`

const revokeRefreshTokenQuery = `
	UPDATE refresh_tokens SET revoked_at = now()
	WHERE token_hash = $1 AND revoked_at IS NULL
`
