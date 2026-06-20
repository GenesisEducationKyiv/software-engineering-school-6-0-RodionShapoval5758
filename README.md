# GitHub-Release-Notification-API
API that allows users to subscribe to email notifications about new releases of a chosen GitHub repository.

### How it works (from a user perspective)
- subscribe your email to `owner/repo` (requires API key)
- confirm the subscription via the link in the confirmation email
- receive email notifications when the repository publishes a new release
- follow the release link directly from the notification email
- unsubscribe with the button in any notification email
- list all active subscriptions for an email address

### Architecture

Three independently deployable Go services:

| Service | Responsibility | Scaling |
|---|---|---|
| **Subscription** | REST API, subscription lifecycle, fanout consumer | Horizontal |
| **Monitoring** | GitHub poller, release detection, outbox relay | Singleton (`replicas: 1`) |
| **Notification** | NATS consumer → SMTP email delivery | Horizontal |

Services communicate via NATS JetStream:
- `NOTIFICATIONS` stream (`notifications.>`) — confirmation requests, per-recipient release events, repo-level release-found events
- `TRACKING` stream (`tracking.>`) — repo tracked / untracked events from Subscription to Monitoring

State is persisted in PostgreSQL. Each service owns its own tables and runs its own migrations on startup.

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
- `POST /api/subscribe`
- `GET /api/confirm/{token}`
- `GET /api/unsubscribe/{token}`
- `GET /api/subscriptions?email=...`

### Configuration

Key environment variables:

| Variable | Service | Notes |
|---|---|---|
| `DATABASE_URL` | Subscription, Monitoring | Postgres connection string |
| `NATS_URL` | All | e.g. `nats://nats:4222` |
| `GITHUB_TOKEN` | Subscription, Monitoring | Without it: 60 req/hour rate limit |
| `API_KEY` | Subscription | Secures `POST /api/subscribe` and `GET /api/subscriptions` |
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
- Default API key for local development: `genesis-summer-school` (set in `.env`)
- `repositories.last_seen_tag` is a legacy column; release tracking now lives in Monitoring's `scan_cursors` table
