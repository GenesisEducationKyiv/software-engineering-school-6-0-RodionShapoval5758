import http from 'k6/http';
import { check, sleep } from 'k6';
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
  // Auth path — no DB, no external calls.
  const validate = http.get(`${BASE_URL}/api/validate`, { headers: authHeaders });
  check(validate, { 'validate: status 200': (r) => r.status === 200 });

  // DB read path — exercises the pgx pool.
  const subs = http.get(`${BASE_URL}/api/subscriptions?email=smoke@test.com`, {
    headers: authHeaders,
  });
  check(subs, { 'subscriptions: status 200': (r) => r.status === 200 });

  // Auth guard — request without a key must be rejected.
  const unauthorized = http.get(`${BASE_URL}/api/validate`);
  check(unauthorized, { 'no auth: status 401': (r) => r.status === 401 });

  sleep(1);
}
