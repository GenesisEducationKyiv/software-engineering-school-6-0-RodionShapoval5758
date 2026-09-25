import { getTokens, setTokens, clearTokens } from './tokens'

const REFRESH_PATH = '/auth/refresh'

async function refreshAccessToken(refreshToken) {
  const res = await fetch(REFRESH_PATH, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: refreshToken }),
  })
  if (!res.ok) return null
  return res.json()
}

// Fetch wrapper for endpoints that require the current access token. Access
// tokens are short-lived (15m); on a 401 we attempt exactly one
// refresh-and-retry before giving up, since an expired access token is the
// expected case, not an error worth surfacing to the caller.
export async function authedFetch(path, options = {}) {
  const { accessToken, refreshToken } = getTokens()

  const attempt = (token) => fetch(path, {
    ...options,
    headers: { ...options.headers, Authorization: `Bearer ${token}` },
  })

  const res = await attempt(accessToken)
  if (res.status !== 401 || !refreshToken) return res

  const pair = await refreshAccessToken(refreshToken)
  if (!pair) {
    clearTokens()
    return res
  }

  setTokens(pair)
  return attempt(pair.access_token)
}
