import http from 'k6/http';
import { BASE_URL } from '../config.js';
import { findMessage, getBody, extractToken } from './mailpit.js';

const PASSWORD = 'k6-load-test-passw0rd';

// Unique per call — deliberately independent of k6/execution's VU/iteration
// counters (unlike lib/data.js's uniqueEmail) so this is safe to call from
// setup(), which runs outside any VU/scenario context.
function provisioningEmail() {
  const rand = Math.random().toString(36).slice(2, 10);
  return `k6-auth-${Date.now()}-${rand}@loadtest.local`;
}

// Registers a fresh throwaway account, verifies it via the link mailed by
// the auth service (scraped from Mailpit), logs in, and returns the
// resulting session. Each call provisions one brand-new account — callers
// decide whether to call this once (a shared token is enough for pure read
// load under the 15m access-token TTL) or per iteration (a fresh identity
// per write, to avoid artificial 409 conflicts under one shared account).
export function provisionToken() {
  const email = provisioningEmail();

  const registerRes = http.post(
    `${BASE_URL}/auth/register`,
    JSON.stringify({ email, password: PASSWORD }),
    { headers: { 'Content-Type': 'application/json' } }
  );
  if (registerRes.status !== 201) {
    throw new Error(`provisionToken: register failed (${registerRes.status}): ${registerRes.body}`);
  }

  const msgId = findMessage(email, 'Verify your email address', { retries: 15, intervalMs: 1000 });
  if (!msgId) {
    throw new Error(`provisionToken: verification email never arrived for ${email}`);
  }

  const verifyToken = extractToken(getBody(msgId), 'auth/verify-email');
  if (!verifyToken) {
    throw new Error(`provisionToken: could not extract verify token for ${email}`);
  }

  const verifyRes = http.get(`${BASE_URL}/auth/verify-email/${verifyToken}`);
  if (verifyRes.status !== 200) {
    throw new Error(`provisionToken: verify-email failed (${verifyRes.status}): ${verifyRes.body}`);
  }

  const loginRes = http.post(
    `${BASE_URL}/auth/login`,
    JSON.stringify({ email, password: PASSWORD }),
    { headers: { 'Content-Type': 'application/json' } }
  );
  if (loginRes.status !== 200) {
    throw new Error(`provisionToken: login failed (${loginRes.status}): ${loginRes.body}`);
  }

  const pair = loginRes.json();
  return { accessToken: pair.access_token, refreshToken: pair.refresh_token, email };
}

// Exchanges a refresh token for a new pair. Used by tests whose duration
// exceeds the access token's 15m TTL (read-soak.js). Returns null if the
// refresh fails — e.g. the refresh token was already rotated away by a
// concurrent use, which callers should treat as "reprovision from scratch."
export function refreshSession(refreshToken) {
  const res = http.post(
    `${BASE_URL}/auth/refresh`,
    JSON.stringify({ refresh_token: refreshToken }),
    { headers: { 'Content-Type': 'application/json' } }
  );
  if (res.status !== 200) return null;

  const pair = res.json();
  return { accessToken: pair.access_token, refreshToken: pair.refresh_token };
}
