# GitHub-Release-Notification-API
API that allows users to subscribe to email notifications about new releases of a chosen GitHub repository.

### How it works (from a user perspective)
- register an account with your email and verify it via the emailed link
- log in to get a JWT access token
- subscribe to `owner/repo` (requires the JWT; the subscription email is your account email)
- confirm the subscription via the link in the confirmation email
- receive email notifications when the repository publishes a new release
- follow the release link directly from the notification email
- unsubscribe with the button in any notification email
- list all active subscriptions for your account

### Architecture

Four independently deployable Go services:

| Service | Responsibility | Scaling |
|---|---|---|
| **Auth** | User accounts, email verification, JWT issuing, signing-key gRPC API | Horizontal |
| **Subscription** | REST API, subscription lifecycle, fanout consumer | Horizontal |
| **Monitoring** | GitHub poller, release detection, outbox relay | Singleton (`replicas: 1`) |
| **Notification** | NATS consumer → SMTP email delivery | Horizontal |

Services communicate via NATS JetStream:
- `NOTIFICATIONS` stream (`notifications.>`) — confirmation requests, email-verification requests, per-recipient release events, repo-level release-found events
- `SAGA` stream (`saga.>`) — email-sent/failed replies from Notification back to Subscription's saga orchestrator (compensation + orphan repo cleanup)
- `NOTIFICATIONS_DLQ` stream (`dlq.notifications`) — poison / max-retry messages from the notification consumer

Monitoring discovers which repos to scan via a synchronous gRPC call to Subscription's catalog service (mTLS) rather than a NATS tracking stream — see [ADR-0005](docs/ADR/0005-pull-tracked-repositories-via-grpc.md).

State is persisted in PostgreSQL. Each service owns its own tables and runs its own migrations on startup; Auth runs against its own Postgres instance so password hashes stay in a separate failure/backup domain.

Subscription verifies JWTs locally (ES256): it fetches Auth's public signing key once over mTLS gRPC and caches it, so request handling never blocks on the Auth service.

### Stack
- Go, Chi, net/http
- PostgreSQL + pgx + golang-migrate
- NATS JetStream (transactional outbox pattern)
- GitHub REST API
- net/smtp / Mailpit for local SMTP
- Docker / Docker Compose
- GitHub Actions for CI
- Gemini CLI / Codex CLI for code review and research

### Endpoints

Subscription (`/api`, JWT-protected unless noted):
- `POST /api/subscribe`
- `GET /api/confirm/{token}` (public, from email link)
- `GET /api/unsubscribe/{token}` (public, from email link)
- `GET /api/subscriptions`

Auth (`/auth` via nginx):
- `POST /auth/register`
- `GET /auth/verify-email/{token}` (public, from email link)
- `POST /auth/login`
- `POST /auth/refresh`
- `POST /auth/logout`

### Configuration

Key environment variables:

| Variable | Service | Notes |
|---|---|---|
| `DATABASE_URL` | Auth, Subscription, Monitoring | Postgres connection string |
| `NATS_URL` | All | e.g. `nats://nats:4222` |
| `GITHUB_TOKEN` | Subscription, Monitoring | Without it: 60 req/hour rate limit |
| `AUTH_GRPC_ADDR` | Subscription | Auth gRPC address for signing-key fetch |
| `JWT_PRIVATE_KEY` | Auth | Path to the ES256 private key PEM |
| `SUBSCRIPTION_GRPC_ADDR` | Monitoring | Subscription gRPC address for catalog pull (ADR-0005) |
| `SMTP_HOST` / `SMTP_PORT` | Notification | SMTP server |
| `SCAN_INTERVAL` | Monitoring | Release scan interval (default: 25s) |

### Run Locally
```
docker compose up --build
```
Mailpit UI is available at `http://localhost:8025`.

### Run Tests
```
go test ./...
```

### CI
- `.github/workflows/ci.yml`
- Runs on every push and pull request
- Runs `vet`, `build`, `test`, and `golangci-lint`

### Notes
- Swagger contract is in `swagger.yaml`
- Migrations run automatically on startup for each service
- `repositories.last_seen_tag` is a legacy column; release tracking now lives in Monitoring's `scan_cursors` table
