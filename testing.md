# Running Tests

## Prerequisites

- Git
- Docker
- Go 1.26+
- Make

No other tools required. All infrastructure is managed by Docker.

---

## Unit Tests

Covers business logic in `service`, `handler`, `notifier`, `domain`, `github`, and `config` packages.
No infrastructure needed.

```bash
make test
```

---

## Integration Tests

Covers all API endpoints:

| Method | Path |
|--------|------|
| POST | `/api/subscribe` |
| GET | `/api/confirm/{token}` |
| GET | `/api/unsubscribe/{token}` |
| GET | `/api/subscriptions` |

Requires Postgres and Mailpit. The command starts both automatically, runs the tests against a dedicated test database (`github_release_notifications_test`), then tears everything down.

```bash
make test-integration
```

---

## E2E Tests

Covers the full subscribe → confirm → view subscription happy path via a real browser (Chromium).

Requires Docker. The command builds and starts the full stack (app, postgres, mailpit, nginx), runs Playwright, then tears everything down.

```bash
make test-e2e
```

---

## Running All Tests

```bash
make test && make test-integration && make test-e2e
```
