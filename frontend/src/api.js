const BASE = '/api'

export async function validateKey(apiKey) {
  return fetch(`${BASE}/validate`, {
    headers: { 'Authorization': `Bearer ${apiKey}` },
  })
}

export async function subscribe(email, repo, apiKey) {
  const body = new URLSearchParams({ email, repo })
  return fetch(`${BASE}/subscribe`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded',
      'Authorization': `Bearer ${apiKey}`,
    },
    body,
  })
}

export async function confirmSubscription(token) {
  return fetch(`${BASE}/confirm/${token}`)
}

export async function unsubscribe(token) {
  return fetch(`${BASE}/unsubscribe/${token}`)
}

export async function getSubscriptions(email, apiKey) {
  return fetch(`${BASE}/subscriptions?email=${encodeURIComponent(email)}`, {
    headers: { 'Authorization': `Bearer ${apiKey}` },
  })
}
