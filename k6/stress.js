import http from 'k6/http';
import { check, sleep } from 'k6';
import { BASE_URL, authHeaders } from './config.js';

// Ramp aggressively past expected load to find the saturation ceiling.
// No hard latency threshold — we want to observe where it degrades, not gate on it.
// Watch: db_pool_empty_acquire_total and db_pool_acquired_connections / db_pool_max_connections
// in the USE dashboard. Latency on the RED dashboard will climb as the pool saturates.
export const options = {
  stages: [
    { duration: '1m', target: 50 },
    { duration: '2m', target: 150 },
    { duration: '1m', target: 0 },
  ],
  thresholds: {
    // Soft gate: allow latency to degrade but hard fail if requests error out.
    http_req_failed: ['rate<0.05'],
  },
};

export default function () {
  const res = http.get(`${BASE_URL}/api/subscriptions?email=stress@test.com`, {
    headers: authHeaders,
  });
  check(res, { 'status 200': (r) => r.status === 200 });
  sleep(0.5);
}
