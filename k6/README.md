# k6 Load Tests

Three scripts covering smoke, load, and stress scenarios against the GitHub Release
Notification API.

## Prerequisites

- **k6 installed** — `dnf install k6` / `brew install k6` / [k6.io/docs/get-started/installation](https://k6.io/docs/get-started/installation/)
- **App running** — `go run ./cmd/api` (uses `.env`, listens on port 2433 by default)
- **Postgres running** — used by the subscriptions read path

## Run

```bash
# Smoke — 3 VUs for 30s, proves correctness under trivial load
k6 run k6/smoke.js

# Load — ramps to 20 VUs over the DB read path; watch Grafana dashboards while it runs
k6 run k6/load.js

# Stress — ramps to 150 VUs to find the saturation ceiling
k6 run k6/stress.js
```

Override the base URL or API key without editing the scripts:

```bash
BASE_URL=http://localhost:8080 API_KEY=mykey k6 run k6/load.js
```

## What to watch in Grafana (localhost:3000)

**While running load.js or stress.js:**

| Dashboard | Panel | What it tells you |
|---|---|---|
| RED Metrics | Request Rate | Requests/sec arriving at the app |
| RED Metrics | p95 Latency | How long the slow requests are taking |
| RED Metrics | Error Rate | Fraction of non-2xx responses |
| USE Metrics | Connection State | Acquired vs total vs max — pool fill level |
| USE Metrics | Connection Utilization | % of max pool in use (yellow >70%, red >90%) |
| USE Metrics | Empty Acquire Rate | Requests waiting for a free connection |

**Stress test tells the full story:** as VUs climb, watch `db_pool_acquired_connections`
approach `db_pool_max_connections`. When they meet, `db_pool_empty_acquire_total`
starts climbing and p95 latency spikes — that's your saturation point.

## Thresholds

| Script | Threshold | Meaning |
|---|---|---|
| smoke.js | `p(95)<500ms`, `error_rate<1%` | Hard gate — smoke must pass cleanly |
| load.js | `p(95)<800ms`, `error_rate<1%` | SLO at expected load |
| stress.js | `error_rate<5%` | Soft gate — degradation expected, errors should not |

A non-zero k6 exit code means a threshold was breached.
