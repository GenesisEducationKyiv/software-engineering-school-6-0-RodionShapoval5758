// write-load.js — bounded write path at GitHub-safe throughput.
//
// Executor: constant-arrival-rate at 1 RPS.
// Why capped: with a GitHub token, the API allows ~5000 req/hr ≈ 1.4 req/s.
// Driving above this deliberately produces 429s. This test stays below the limit
// and asserts on the *expected* outcome distribution (200, 409, 429 each mean
// something specific — none are silent failures; 5xx is the real alarm).
//
// Custom outcome counters appear in the k6 dashboard under:
//   subscribe_created_total    — fresh subscribe succeeded (200)
//   subscribe_conflict_total   — email+repo already subscribed (409)
//   subscribe_rate_limited_total — GitHub rate limit hit (429)
//   subscribe_repo_not_found_total — GitHub returned 404 for the repo (404)
//   subscribe_bad_token_total  — GitHub token invalid/expired (502)
//   subscribe_server_error_total — unexpected 5xx (triggers threshold failure)
//
// Watch: RED dashboard → /api/subscribe row; USE → pool utilisation.

import http from 'k6/http';
import { check, group } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import { BASE_URL, jsonAuthHeaders } from './config.js';
import { uniqueEmail, randomRepo } from './lib/data.js';

// Per-outcome counters — each outcome is semantically distinct.
const created        = new Counter('subscribe_created_total');
const conflict       = new Counter('subscribe_conflict_total');
const rateLimited    = new Counter('subscribe_rate_limited_total');
const repoNotFound   = new Counter('subscribe_repo_not_found_total');
const badToken       = new Counter('subscribe_bad_token_total');
const serverError    = new Counter('subscribe_server_error_total');

// End-to-end subscribe latency (includes GitHub round-trip + SMTP send).
const subscribeDuration = new Trend('subscribe_duration_ms', true);

export const options = {
  scenarios: {
    write_load: {
      executor: 'constant-arrival-rate',
      rate: 1,           // 1 RPS — well under the 1.4 RPS GitHub rate limit
      timeUnit: '1s',
      duration: '3m',
      preAllocatedVUs: 5,
      maxVUs: 10,
    },
  },
  thresholds: {
    // Any 5xx is a real server bug — zero tolerance.
    subscribe_server_error_total: ['count<1'],
    // Subscribe (including GitHub API round-trip) should complete in < 3s p95.
    subscribe_duration_ms: ['p(95)<3000'],
    // k6-level transport failures should never happen.
    http_req_failed: ['rate<0.01'],
  },
};

export default function () {
  const email = uniqueEmail();
  const repo  = randomRepo();

  group('subscribe', () => {
    const res = http.post(
      `${BASE_URL}/api/subscribe`,
      JSON.stringify({ email, repo }),
      { headers: jsonAuthHeaders }
    );

    subscribeDuration.add(res.timings.duration);

    // Record the outcome — each status code has a distinct semantic.
    switch (res.status) {
      case 200: created.add(1);       break;
      case 409: conflict.add(1);      break;
      case 429: rateLimited.add(1);   break;
      case 404: repoNotFound.add(1);  break;
      case 502: badToken.add(1);      break;
      default:
        if (res.status >= 500) serverError.add(1);
        break;
    }

    check(res, {
      'subscribe: not a server error': (r) => r.status < 500,
      'subscribe: has json body':      (r) => r.headers['Content-Type'] &&
                                              r.headers['Content-Type'].includes('application/json'),
    });
  });
}
