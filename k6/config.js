export const BASE_URL = __ENV.BASE_URL || 'http://localhost';
export const MAILPIT_URL = __ENV.MAILPIT_URL || 'http://localhost:8025';

// There is no more static shared API key — the subscription service now
// requires a real JWT issued by the auth service (see lib/auth.js for how
// tests obtain one). These builders take that token per call since it's
// provisioned dynamically, not read from an env var.
export function authHeaders(token) {
  return { Authorization: `Bearer ${token}` };
}

export function jsonAuthHeaders(token) {
  return { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` };
}
