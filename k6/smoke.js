import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { BASE_URL, authHeaders } from './config.js';

export const options = {
  vus: 3,
  duration: '30s',
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<500'],
  },
};

export default function () {
  group('auth guard', () => {
    // Valid key → 200.
    const ok = http.get(`${BASE_URL}/api/validate`, { headers: authHeaders });
    check(ok, { 'validate: status 200': (r) => r.status === 200 });

    // Missing key → 401. responseCallback marks 401 as expected so k6 does
    // not count this request toward http_req_failed (default ≥ 400 = failed).
    const noKey = http.get(`${BASE_URL}/api/validate`, {
      responseCallback: http.expectedStatuses(401),
    });
    check(noKey, { 'no auth: status 401': (r) => r.status === 401 });
  });

  group('read path', () => {
    // DB read path — exercises the pgx pool.
    const subs = http.get(`${BASE_URL}/api/subscriptions?email=smoke@test.com`, {
      headers: authHeaders,
    });
    check(subs, {
      'subscriptions: status 200': (r) => r.status === 200,
      'subscriptions: body is array': (r) => Array.isArray(r.json()),
    });
  });

  sleep(1);
}
