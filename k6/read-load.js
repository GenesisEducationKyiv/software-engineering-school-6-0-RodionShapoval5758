// read-load.js — sustained read load at a fixed, SLO-gated arrival rate.
//
// Executor: constant-arrival-rate (500 RPS for 3 min).
// This is the "normal production load" baseline — a rate that should be well under
// the saturation ceiling so the SLO thresholds are expected to pass.
//
// 500 RPS at ~1.7ms p95 needs only ~1 connection on average, so the pgx pool
// (default ≈ 8) has plenty of headroom. If this test fails thresholds, something
// is seriously broken before you even get to stress testing.
//
// Watch while running:
//   RED dashboard  — request rate sits near 500/s; p95 should stay flat and low.
//   USE dashboard  — db_pool_acquired_connections / db_pool_max_connections < 0.5.
//   k6 dashboard   — VUs < 5 at this latency; dropped_iterations should be 0.

import http from 'k6/http';
import { check } from 'k6';
import { BASE_URL, authHeaders } from './config.js';

export const options = {
  scenarios: {
    read_load: {
      executor: 'constant-arrival-rate',
      rate: 500,
      timeUnit: '1s',        // 500 RPS
      duration: '3m',
      preAllocatedVUs: 100,  // rate × expected_p99_s = 500 × ~0.005s = 2.5; 100 is generous
      maxVUs: 800,
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<800', 'p(99)<1500'],
    http_req_failed: ['rate<0.01'],
    checks: ['rate>0.99'],
  },
};

export default function () {
  const res = http.get(`${BASE_URL}/api/subscriptions?email=load@test.com`, {
    headers: authHeaders,
  });
  check(res, { 'status 200': (r) => r.status === 200 });
}
