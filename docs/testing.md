# Running Tests

## Prerequisites

- Git
- Docker
- Go 1.26+
- [Task](https://taskfile.dev) (`go install github.com/go-task/task/v3/cmd/task@latest` or your package manager)

All commands below are `task` targets defined in the root `Taskfile.yaml`. No other tools are
required — infrastructure (Postgres, Mailpit, NATS) is managed by Docker.

---

## Unit Tests

Covers business logic across all service modules: `services/auth`, `services/subscription`,
`services/monitoring`, `services/notification`, and the shared `services/contract` module. No
infrastructure needed.

```bash
task test-unit
```

---

## Integration Tests

Covers the Subscription REST API against a real Postgres and Mailpit:

| Method | Path |
|--------|------|
| POST | `/api/subscribe` |
| GET | `/api/confirm/{token}` |
| GET | `/api/unsubscribe/{token}` |
| GET | `/api/subscriptions` |

Also covers the Notification consumer and the Monitoring scan loop. The command starts Postgres
and Mailpit automatically, runs the tests against a dedicated test database, then tears everything
down.

```bash
task test-integration
```

**Gap:** the Auth service's HTTP endpoints (`/register`, `/verify-email/{token}`, `/login`,
`/refresh`, `/logout`) are currently covered only by unit tests (`go test ./services/auth/...`, run
as part of `task test-unit`), not by an integration suite against a real database. There is no
`services/auth/test/integration/...` package yet.

---

## E2E Tests

Drives the frontend through a real browser (Chromium) via Playwright. The command builds and
starts the full stack (all services, Postgres ×2, NATS, Mailpit, nginx), runs Playwright, then
tears everything down.

```bash
task test-e2e
```

**Known gap:** the Playwright specs (`frontend/e2e/*.spec.ts`) still drive the pre-JWT UI (a
static API-key box) and have not been updated for the register → verify → login → profile flow.
They will fail against the current frontend until they're rewritten to match.

---

## k6 Load Tests

Covers read/write throughput, saturation limits, and a full user-lifecycle journey against the
live stack. Every protected call now needs a real JWT — tests get one by driving the actual signup
flow (register → verify via Mailpit → log in) rather than a static API key. See
[`k6/README.md`](../k6/README.md) for the full test matrix and how each test provisions its token.

```bash
task k6-smoke      # quick correctness pass — run first
task k6-journey    # full register → subscribe → confirm → unsubscribe lifecycle
```

---

## Running All Tests

```bash
task test-all
# equivalent to: task test-unit && task test-integration && task test-e2e
```
