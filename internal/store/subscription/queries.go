package subscription

const createSubscriptionQuery = `
	INSERT INTO subscriptions (
		email,
		repository_id,
		confirmation_token,
		unsubscribe_token
	)
	VALUES ($1, $2, $3, $4)
`

const findSubscriptionByUnsubscribeTokenQuery = `
	SELECT id, email, repository_id, confirmed, confirmation_token, unsubscribe_token, created_at, confirmed_at
	FROM subscriptions
	WHERE unsubscribe_token = $1
`

const confirmSubscriptionByTokenQuery = `
	UPDATE subscriptions
	SET confirmed = TRUE, confirmed_at = NOW()
	WHERE confirmation_token = $1
`

const deleteSubscriptionByUnsubscribeTokenQuery = `
	DELETE FROM subscriptions
	WHERE unsubscribe_token = $1
`

const hasAnySubscriptionsByRepositoryIDQuery = `
	SELECT EXISTS(
		SELECT 1
		FROM subscriptions
	WHERE repository_id = $1
	)
`

const listConfirmedSubscriptionsByRepositoryIDQuery = `
	SELECT
		id,
		email,
		repository_id,
		confirmed,
		confirmation_token,
		unsubscribe_token,
		created_at,
		confirmed_at
	FROM subscriptions
	WHERE repository_id = $1 AND confirmed = TRUE
`

const listSubscriptionDetailsByEmailQuery = `
	SELECT subscriptions.email, repositories.name, subscriptions.confirmed, repositories.last_seen_tag
	FROM subscriptions
	JOIN repositories ON subscriptions.repository_id = repositories.id
	WHERE subscriptions.email = $1 AND subscriptions.confirmed = TRUE
`
