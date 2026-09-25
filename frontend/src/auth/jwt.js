// Decodes a JWT payload without verifying its signature. The browser can't
// verify an ES256 signature and doesn't need to — the backend already did
// that before issuing the token. This is only used to read the `email`
// claim for display and session bookkeeping.
export function decodeJwtPayload(token) {
  if (!token) return null
  const parts = token.split('.')
  if (parts.length !== 3) return null

  try {
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const padded = base64.padEnd(base64.length + (4 - base64.length % 4) % 4, '=')
    return JSON.parse(atob(padded))
  } catch {
    return null
  }
}

export function emailFromToken(token) {
  return decodeJwtPayload(token)?.email ?? null
}
