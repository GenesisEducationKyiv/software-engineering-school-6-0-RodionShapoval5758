// read-stress.js — ramp RPS far past the expected ceiling to find the knee point.
//
// Executor: ramping-arrival-rate (200 → 10 000 RPS over ~6.5 min).
//
// The test deliberately does NOT abort on latency — we want to *observe* the
// degradation curve, not stop at the first sign of trouble. That's what breakpoint.js
// is for. This test completes fully so you get the whole shape on the dashboards.
//
// Only hits GET /api/subscriptions (one DB query per request) to maximise pool
// pressure. Mixing in /api/validate (no DB) would dilute the signal.
//
// What to look for:
//   RED dashboard  — find the RPS stage where p95 starts climbing steeply. That
//                    stage is your saturation knee. 5xx appearing = service broken.
//   USE dashboard  — db_pool_empty_acquire_total rising + db_pool_acquired/max → 1.0
//                    = DB pool is the bottleneck.
//                    If you see 502s while pool looks idle → nginx connection churn is
//                    the bottleneck (no upstream keepalive in nginx.conf).
//   k6 dashboard   — dropped_iterations climbing = maxVUs hit first, not the service.
//                    Raise maxVUs further to push past it.

import http from 'k6/http';
import { check } from 'k6';
import { BASE_URL, authHeaders } from './config.js';

export const options = {
  scenarios: {
    read_stress: {
      executor: 'ramping-arrival-rate',
      startRate: 200,
      timeUnit: '1s',
      preAllocatedVUs: 300,
      maxVUs: 3000,
      stages: [
        { duration: '30s', target: 500   },  // warm up
        { duration: '1m',  target: 1000  },  // moderate load
        { duration: '1m',  target: 2500  },  // approaching suspected ceiling
        { duration: '1m',  target: 5000  },  // pool should saturate around here
        { duration: '1m',  target: 7500  },  // nginx churn territory
        { duration: '1m',  target: 10000 },  // theoretical host ceiling
        { duration: '30s', target: 0     },  // drain
      ],
    },
  },
  thresholds: {
    // Non-aborting soft gate — we want to see where it breaks, not bail early.
    http_req_failed: ['rate<0.05'],
  },
};

export default function () {
  const res = http.get(`${BASE_URL}/api/subscriptions?email=stress@test.com`, {
    headers: authHeaders,
  });
  check(res, { 'status 200': (r) => r.status === 200 });
}
