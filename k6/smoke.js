import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { BASE_URL, authHeaders } from './config.js';
import { provisionToken } from './lib/auth.js';

export const options = {
  vus: 3,
  duration: '30s',
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<500'],
  },
};

// One shared account for the whole run.
export function setup() {
  const { accessToken } = provisionToken();
  return { token: accessToken };
}

export default function (data) {
  group('auth guard', () => {
    // There's no more standalone /api/validate — the JWT check now lives on
    // the same DB-backed endpoint every read hits, so the auth-guard check
    // and the read-path check below necessarily share an endpoint.
    const withToken = http.get(`${BASE_URL}/api/subscriptions`, { headers: authHeaders(data.token) });
    check(withToken, { 'valid JWT: status 200': (r) => r.status === 200 });

    // Missing token → 401. responseCallback marks 401 as expected so k6 does
    // not count this request toward http_req_failed (default ≥ 400 = failed).
    const noToken = http.get(`${BASE_URL}/api/subscriptions`, {
      responseCallback: http.expectedStatuses(401),
    });
    check(noToken, { 'no auth: status 401': (r) => r.status === 401 });
  });

  group('read path', () => {
    // DB read path — exercises the pgx pool.
    const subs = http.get(`${BASE_URL}/api/subscriptions`, { headers: authHeaders(data.token) });
    check(subs, {
      'subscriptions: status 200': (r) => r.status === 200,
      'subscriptions: body is array': (r) => Array.isArray(r.json()),
    });
  });

  sleep(1);
}
