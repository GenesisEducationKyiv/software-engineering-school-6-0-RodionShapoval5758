// breakpoint.js — auto-find the exact RPS at which the service breaks.
//
// Executor: ramping-arrival-rate, smooth linear ramp 200 → 12 000 RPS over 10 min.
// The test **aborts automatically** the moment either threshold is breached for
// 10 continuous seconds. k6 prints the current rate and VU count at abort time —
// that number is your service's breaking point.
//
// Unlike read-stress.js (which runs to completion so you see the whole curve),
// this test gives you a single precise headline: "broke at N RPS".
//
// How to read the abort:
//   k6 prints: "thresholds on metrics 'http_req_duration' were breached; stopping test."
//   Look at the last "iteration_duration" line in stdout — the rate at that timestamp
//   is the ceiling. Cross-reference the Grafana dashboards to attribute the root cause:
//
//   db_pool_empty_acquire_total spiking  →  DB connection pool is the bottleneck
//     Fix: add ?pool_max_conns=50 to DATABASE_URL in docker-compose.yaml
//
//   502s / connection refused, pool looks idle  →  nginx connection churn
//     Fix: add upstream keepalive block to nginx/nginx.conf
//
//   Pool + nginx both look fine, host CPU is pegged  →  host-generator contention
//     Fix: run k6 on a separate machine against the service
//
// delayAbortEval: '10s' — ignores transient first-second spikes, only aborts on
// sustained (10s) threshold breach. Remove it to make the abort more sensitive.

import http from 'k6/http';
import { check } from 'k6';
import { BASE_URL, authHeaders } from './config.js';

export const options = {
  scenarios: {
    breakpoint: {
      executor: 'ramping-arrival-rate',
      startRate: 200,
      timeUnit: '1s',
      preAllocatedVUs: 500,
      maxVUs: 4000,
      stages: [
        { duration: '10m', target: 12000 },  // smooth linear ramp — abort stops it early
      ],
    },
  },
  thresholds: {
    // Abort the instant p95 exceeds 1 s for 10 continuous seconds.
    http_req_duration: [{ threshold: 'p(95)<1000', abortOnFail: true, delayAbortEval: '10s' }],
    // Also abort if error rate exceeds 5% for 10 continuous seconds.
    http_req_failed: [{ threshold: 'rate<0.05', abortOnFail: true, delayAbortEval: '10s' }],
  },
};

export default function () {
  const res = http.get(`${BASE_URL}/api/subscriptions?email=breakpoint@test.com`, {
    headers: authHeaders,
  });
  check(res, { 'status 200': (r) => r.status === 200 });
}
