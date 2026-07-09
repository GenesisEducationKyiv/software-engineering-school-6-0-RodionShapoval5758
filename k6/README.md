# k6 Load Test Suite

Tests the GitHub Release Notification API across correctness (smoke), SLO-gated sustained load,
saturation ceiling exploration, burst survival, leak detection, an auto-stopping breakpoint finder,
a rate-limited write path, and a full end-to-end user journey.

All protected endpoints now require a real JWT from the auth service — there is no more static
shared API key. Tests obtain tokens by driving the actual signup flow (`lib/auth.js`): register →
scrape the verification link from Mailpit → verify → log in. **Mailpit must be reachable** for
every test now, not just `journey.js`.

---

## Prerequisites

```bash
# k6 must be installed on the host — it is NOT containerized in this stack.
brew install k6            # macOS
sudo apt install k6        # Debian/Ubuntu
# or: https://k6.io/docs/get-started/installation/

# Stack must be running with Prometheus remote-write enabled (already in compose.yaml)
# and Mailpit reachable (published at :8025 by compose.yaml).
task up-full
```

---

## Test files

| File | Executor | Rate | Duration | Purpose |
|---|---|---|---|---|
| `smoke.js` | constant-vus | 3 VUs | 30s | Correctness + auth guard — run first |
| `read-load.js` | constant-arrival-rate | **500 RPS** | 3m | SLO baseline — hard-fail if breached |
| `read-stress.js` | ramping-arrival-rate | 200 → **10 000 RPS** | ~6.5m | Find the saturation knee |
| `breakpoint.js` | ramping-arrival-rate | 200 → **12 000 RPS** | ≤10m | Auto-stop at the breaking point |
| `read-spike.js` | ramping-arrival-rate | 0 → **3 000 RPS** burst | ~40s | Burst survival |
| `read-soak.js` | constant-arrival-rate | **200 RPS** | 30m | Leak detection over time |
| `write-load.js` | constant-arrival-rate | 1 RPS | 3m | Write path, GitHub-rate-limited |
| `journey.js` | per-vu-iterations | 3 VUs × 5 iter | ≤10m | Full lifecycle via mailpit |

### How each test gets its JWT

- **`smoke.js`, `read-load.js`, `read-stress.js`, `read-spike.js`, `breakpoint.js`**: provision one
  account in `setup()` and share that single access token across every VU for the whole run. These
  all measure DB read throughput, not auth throughput, and each run comfortably finishes inside the
  access token's 15-minute TTL, so no refresh is needed.
- **`read-soak.js`** (30 minutes — longer than the 15-minute TTL): each VU lazily provisions its
  own account on first use and refreshes independently on a 401. Module-scope state in k6 is
  per-VU, so there's no cross-VU race on the single-use, rotating refresh token.
- **`write-load.js`, `journey.js`**: provision a fresh account **per iteration**. A single shared
  identity would collapse every subscribe onto one email, and with only 10 repos in `lib/data.js`
  to cycle through, most calls would 409 instead of exercising a genuinely new (email, repo) pair —
  defeating the point of both tests.

### Why these rates?

The previous suite topped out at 200 RPS and the service answered with p95 = **1.7ms** using a
single VU — it was never stressed. At ~1.7ms per query with a default pgx pool of ~8 connections
(`internal/db/pool.go` uses `pgxpool.New` with no `MaxConns`), the theoretical pool ceiling is
roughly **4 000–5 000 RPS**. The new rates are set to actually reach and exceed that ceiling.

---

## Running via Task (recommended)

All `task k6-*` targets automatically push results to Prometheus so dashboards fill without
extra flags. Variables are overridable on the command line.

```bash
task k6-smoke           # correctness — always run first
task k6-load            # 500 RPS SLO gate
task k6-stress          # ramp to 10k — find the knee
task k6-breakpoint      # auto-stop at the breaking point
task k6-spike           # 3k RPS burst
task k6-soak            # 200 RPS × 30 min leak check
task k6-write           # write path (needs GITHUB_TOKEN in the service)
task k6-journey         # full lifecycle via mailpit

task k6-suite           # runs smoke → load → stress → spike back-to-back
task k6-clean           # delete @loadtest.local rows after write/journey

# Override the target URL or Mailpit address:
task k6-stress K6_BASE_URL=http://staging K6_MAILPIT_URL=http://staging:8025
```

---

## Running manually with Prometheus output

```bash
export K6_PROMETHEUS_RW_SERVER_URL=http://localhost:9090/api/v1/write
export K6_PROMETHEUS_RW_TREND_STATS="p(95),p(99),avg,min,max"
export BASE_URL=http://localhost
export MAILPIT_URL=http://localhost:8025

k6 run -o experimental-prometheus-rw --tag testid=read-stress \
  -e BASE_URL=$BASE_URL -e MAILPIT_URL=$MAILPIT_URL \
  k6/read-stress.js
```

The `--tag testid=<name>` value drives the `$testid` filter on the k6 Grafana dashboard,
isolating each run's data from other runs.

---

## What to watch on each dashboard

### k6 dashboard (Grafana → "k6 Load Tests")
- **RPS** — actual arrival rate k6 achieved vs target
- **Active VUs** — climbing VUs = service is slower than expected; sudden plateau = maxVUs hit
- **Dropped iterations** — non-zero = maxVUs was the bottleneck, not the service; raise `maxVUs`
- **Client p95/p99 latency** — client-observed; compare to server-side RED p95 (gap = network + nginx)
- **Check pass rate** — below 99% = business logic errors, not just latency
- **Subscribe outcome breakdown** — visible only during `write-load.js`

### RED dashboard (Grafana → "RED Metrics")
- **Request rate by path** — confirms traffic is reaching the service through nginx
- **Error ratio / 5xx rate** — first 5xx in stress/breakpoint = service has broken
- **p95/p99 by path** — the knee where this climbs steeply is the saturation point

### USE dashboard (Grafana → "Service USE Metrics")
- **db_pool_acquired / max** — > 0.8 means pool is near saturation; at 1.0 = fully saturated
- **db_pool_empty_acquire_total** — rising here = DB pool is the bottleneck (most likely first)
- **go_goroutines** — flat in soak; climbing trend = goroutine leak
- **go_memstats_heap_inuse_bytes** — sawtooth (GC) is normal; upward trend over 30 min = memory leak
- **watcher_scan_duration** — should be unaffected by HTTP load (background job)

---

## Expected failure points

These are grounded in the actual code — intended discoveries, not bugs:

| # | Bottleneck | When it appears | Dashboard signal | Quick fix |
|---|---|---|---|---|
| 1 | **pgx pool exhaustion** | ~4 000–5 000 RPS | `db_pool_empty_acquire_total` rises; pool util → 1.0 | Add `?pool_max_conns=50` to `DATABASE_URL` |
| 2 | **nginx connection churn** | 1 000–3 000+ RPS | 502s on RED while pool looks idle (no upstream keepalive in `nginx/nginx.conf`) | Add `upstream` keepalive block |
| 3 | **Host/generator contention** | Any high-RPS test | Flat pool + climbing client latency + pegged CPU | Run k6 on a separate machine |

### Reading the `task k6-breakpoint` abort

k6 prints: `"thresholds on metrics '...' were breached; stopping test."`
The RPS value visible on the k6 Grafana dashboard at that moment is your ceiling.

- `db_pool_empty_acquire_total` spiking → bottleneck is **#1 (pgx pool)**
- 502s appearing, pool idle → bottleneck is **#2 (nginx)**
- Both fine, host CPU pegged → bottleneck is **#3 (host)**

---

## Notes

### Write load and the GitHub rate limit
`write-load.js` caps at 1 RPS (below GitHub's ~1.4 RPS sustainable limit with a token). 429
responses are expected above that rate — the `subscribe_rate_limited_total` counter tracks them
separately. The `subscribe_server_error_total` threshold (count < 1) only trips on real 5xx.

### Journey and the unsubscribe step
The unsubscribe token is only sent in release notification emails (not confirmation emails). Since
no real releases occur during a test run, the unsubscribe step polls mailpit briefly and logs a
skip if no release email is found. This is by design — an honest representation of the system.

### DB cleanup after write/journey tests
```bash
task k6-clean
# equivalent: DELETE FROM subscriptions WHERE email LIKE '%@loadtest.local';
```

This only cleans up `subscriptions`. Auth accounts provisioned by these tests (also
`%@loadtest.local`) accumulate in the auth database; there is currently no equivalent cleanup task
for the `users` table.

### Prometheus remote-write receiver
The `--web.enable-remote-write-receiver` flag is already in `compose.yaml`. If k6 shows
`ERRO[...] remote write error`, verify Prometheus is running and reachable:
```bash
curl -s localhost:9090/-/ready
```
