// read-spike.js — sudden 3 000 RPS burst to test cold-start / burst survival.
//
// Executor: ramping-arrival-rate (0 → 3000 RPS in 5s, hold 30s, drop to 0).
// Models a traffic spike (deploy, viral post, cron fan-out).
//
// At this scale the pgx pool saturates immediately (default ≈ 8 connections at
// ~1.7ms/query gives a theoretical ceiling of ~4 700 RPS, and the burst is a
// sharp step rather than a ramp). The test tells you:
//   - Does the service stay below 5% error rate during the burst?
//   - Does latency recover to baseline after the burst ends?
//   - Does the pool drain cleanly (db_pool_acquired drops back to idle)?
//
// Watch:
//   RED dashboard  — tall spike on request-rate; p95 climbing during burst is expected.
//   USE dashboard  — db_pool_empty_acquire_total spikes then recovers post-burst.
//   k6 dashboard   — dropped_iterations during peak = maxVUs was the ceiling, not the
//                    service. Raise maxVUs to 4000 if you see heavy drops.

import http from 'k6/http';
import { check } from 'k6';
import { BASE_URL, authHeaders } from './config.js';

export const options = {
  scenarios: {
    read_spike: {
      executor: 'ramping-arrival-rate',
      startRate: 0,
      timeUnit: '1s',
      preAllocatedVUs: 500,
      maxVUs: 3000,
      stages: [
        { duration: '5s',  target: 3000 },  // instant spike
        { duration: '30s', target: 3000 },  // hold at peak
        { duration: '5s',  target: 0    },  // drop off — watch pool recovery
      ],
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.05'],
  },
};

export default function () {
  const res = http.get(`${BASE_URL}/api/subscriptions?email=spike@test.com`, {
    headers: authHeaders,
  });
  check(res, { 'status 200': (r) => r.status === 200 });
}
