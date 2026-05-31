import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, authHeaders } from './config.js';

// Ramp to 20 VUs, hold for 1 minute, ramp back down.
// This is the DB read path — watch the USE dashboard while it runs:
//   - db_pool_acquired_connections climbing toward db_pool_max_connections
//   - http_request_duration_seconds p95 in the RED dashboard
export const options = {
  stages: [
    { duration: '30s', target: 20 },
    { duration: '1m', target: 20 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<800'],
    http_req_failed: ['rate<0.01'],
  },
};

export default function () {
  const res = http.get(`${BASE_URL}/api/subscriptions?email=load@test.com`, {
    headers: authHeaders,
  });
  check(res, { 'status 200': (r) => r.status === 200 });
  sleep(1);
}
