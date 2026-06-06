// read-soak.js — sustained load over a long duration to surface slow leaks.
//
// Executor: constant-arrival-rate (200 RPS for 30 min).
// Purpose: find things that ramp tests miss — goroutine accumulation, heap growth,
// DB connection pool exhaustion after many requests, increasing GC pause times.
// 200 RPS is high enough to exercise the pool meaningfully but well under saturation
// so the service stays healthy and any degradation trend is clearly a leak, not load.
//
// Watch while running (USE dashboard):
//   go_goroutines              — should be flat; steady climb = goroutine leak.
//   go_memstats_heap_inuse_bytes — sawtooth (GC) is normal; upward trend = memory leak.
//   db_pool_acquired_connections — should stay well below db_pool_max_connections.
//   watcher_scan_duration_seconds — should not be affected by HTTP load (background job).
//   gc pause p99               — should not trend upward over 30 min.

import http from 'k6/http';
import { check } from 'k6';
import { BASE_URL, authHeaders } from './config.js';

export const options = {
  scenarios: {
    read_soak: {
      executor: 'constant-arrival-rate',
      rate: 200,
      timeUnit: '1s',        // 200 RPS for 30 min
      duration: '30m',
      preAllocatedVUs: 50,
      maxVUs: 400,
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<800'],
    http_req_failed: ['rate<0.01'],
    checks: ['rate>0.99'],
  },
};

export default function () {
  const res = http.get(`${BASE_URL}/api/subscriptions?email=soak@test.com`, {
    headers: authHeaders,
  });
  check(res, { 'status 200': (r) => r.status === 200 });
}
