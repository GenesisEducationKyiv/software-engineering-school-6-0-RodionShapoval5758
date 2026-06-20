# Subscribe Saga — Technical Implementation Report

## The Problem

Before this change, `POST /subscribe` ran a single local transaction and forgot the result:

```
BEGIN
  INSERT subscriptions (pending)
  INSERT outbox(ConfirmationRequested)
  INSERT outbox(RepoTracked)
COMMIT
```

The outbox relay then published both events and the system moved on. Two failure modes were silently ignored:

1. **Email permanently undeliverable** — the Notification service would exhaust retries and write the message to the DLQ. Nobody cleaned up the pending subscription. It sat in the database forever, and Monitoring kept scanning the repository's cursor for a subscription that could never be confirmed.

2. **User never confirms** — no TTL or timeout existed. A pending subscription from months ago would still be there, holding a repository in Monitoring's scan list.

## The Solution: Orchestrated SAGA

An orchestrated saga converts the subscribe workflow into an explicit multi-step distributed transaction with compensating actions. A dedicated saga row tracks state; the Subscription service drives every transition; failures trigger cleanup.

### Why orchestration, not choreography?

Choreography (services reacting to each other's events) works well for workflows without central authority. Subscribe is different: only the Subscription service knows whether a pending subscription exists and owns the cleanup. An orchestrator with a state table gives us a single authoritative view of where each subscribe attempt stands, which is required for the TTL reaper and for the confirm-vs-timeout race.

## State Machine

```
[POST /subscribe]
      │
      ▼
   STARTED ──── EmailFailed ──────────────────────────────┐
      │                                                    │
   EmailSent                                               ▼
      │                                             COMPENSATING ──── subscription confirmed? ──► COMPLETED
      ▼                                                    │
AWAITING_CONFIRMATION ──── EmailFailed or TTL expired ──► │
      │                                                    ▼
   /confirm                                             FAILED
      │
      ▼
   COMPLETED
```

| State | Meaning |
|---|---|
| `STARTED` | Pending subscription created; waiting for email delivery outcome |
| `AWAITING_CONFIRMATION` | Email delivered; waiting for user to click the link |
| `COMPLETED` | Terminal success — subscription is confirmed |
| `COMPENSATING` | Failure detected; running cleanup |
| `FAILED` | Terminal failure — subscription deleted, repo untracked if orphaned |

## Files Changed

### `services/contract/events.go`
- Added `SagaID string` field to `ConfirmationRequested` — the correlation key that Notification echoes back in its reply events.
- Added stream `SAGA` (`saga.>`) and two new subjects: `saga.email.sent`, `saga.email.failed`.
- Added two new event types: `EmailSent{SagaID}`, `EmailFailed{SagaID, Reason}`.

**Why:** The contract is the shared language between services. Notification needs to know the correlation ID to reply, and both services need to agree on what subjects those replies land on.

### `services/subscription/migrations/000006_create_subscribe_sagas.up.sql`
New table `subscribe_sagas`:

```sql
CREATE TABLE subscribe_sagas (
    id              BIGSERIAL    PRIMARY KEY,
    saga_id         TEXT         NOT NULL UNIQUE,   -- random correlation key
    subscription_id BIGINT,                         -- FK to subscriptions (soft)
    repository_id   BIGINT       NOT NULL,
    email           VARCHAR(255) NOT NULL,
    state           TEXT         NOT NULL,
    deadline_at     TIMESTAMPTZ  NOT NULL,
    last_error      TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);
-- partial index: only active rows need to be checked by the reaper
CREATE INDEX subscribe_sagas_active_deadlines
    ON subscribe_sagas (deadline_at)
    WHERE state IN ('STARTED', 'AWAITING_CONFIRMATION');
```

**Why:** The saga state must survive process restarts. Storing it in Postgres gives us the same durability guarantee as the subscription row itself, and lets the reaper use a simple polling query with `FOR UPDATE SKIP LOCKED`.

### `services/subscription/internal/saga/store.go`
All raw Postgres access for the saga table:
- `InsertInTx` — called inside the subscribe transaction.
- `FindBySagaIDForUpdate` — used by the orchestrator; locks the row for safe state transitions.
- `FindBySubscriptionIDForUpdate` — used by the confirm path to advance the saga to COMPLETED.
- `FindExpiredForUpdate(SKIP LOCKED)` — used by the reaper to claim expired sagas without blocking.
- `UpdateState` — transitions state; sets `last_error` on failure.
- `DeleteUnconfirmedSubscription` — compensation step; only deletes if `confirmed = FALSE`. Returns a boolean to detect the confirm-wins race.
- `ConfirmSubscription` / `FindSubscriptionIDByToken` — moved here to support the confirm path.

**Why `db.DBTX` instead of `pgx.Tx`:** The orchestrator receives a `db.Tx` from `db.TxBeginner`; the reaper receives `pgx.Tx` from `pgxpool.Pool`. Both satisfy `db.DBTX` (which only requires `Exec`/`Query`/`QueryRow`). Using the wider interface makes the store callable from both contexts without any adapter.

### `services/subscription/internal/saga/orchestrator.go`
Business logic for all saga transitions:
- `HandleEmailSent` — transitions `STARTED → AWAITING_CONFIRMATION` inside a locked transaction. No-ops if the saga is already past `STARTED` (idempotency for duplicate deliveries).
- `HandleEmailFailed` — transitions to `COMPENSATING`, commits, then calls `compensate()`.
- `HandleConfirmed` — called by the confirm usecase; transitions to `COMPLETED`.
- `CompensateExpired` — called by the reaper after marking rows `COMPENSATING`.
- `compensate()` — deletes the pending subscription (`confirmed = FALSE` guard), updates saga to `FAILED`, commits, then calls `catalog.DeleteIfOrphaned` (which emits `RepoUntracked`). If `DeleteUnconfirmedSubscription` returns `false` (user confirmed first), the saga is marked `COMPLETED` instead.

**The confirm-vs-timeout race is safe:** Compensation deletes only where `confirmed = FALSE`. If the confirm handler ran first, `deleted = false`, so compensation marks the saga `COMPLETED`, not `FAILED`. Only one terminal state is ever written.

### `services/subscription/internal/saga/consumer.go`
JetStream consumer on the `SAGA` stream. Consumes `EmailSent` and `EmailFailed`, routes each to the orchestrator's handlers. Uses the same ack/nak pattern as `fanout.Worker`.

### `services/subscription/internal/saga/reaper.go`
Ticker goroutine (30 s interval). Runs `FindExpiredForUpdate` with `SKIP LOCKED` so multiple replicas of the Subscription service don't double-compensate the same saga. Marks all found rows `COMPENSATING` in a single tx, then compensates each sequentially. Individual failures are logged and skipped without aborting the rest.

### `services/subscription/internal/subscription/internal/store/queries.go`
- `createSubscriptionQuery` — added `RETURNING id` so the subscribe tx can record the subscription ID in the saga row.
- `confirmSubscriptionByTokenQuery` — added `RETURNING id` so the confirm path can advance the saga without a second query.

### `services/subscription/internal/subscription/internal/store/store.go`
- `Create` / `CreateInTx` — return `(int64, error)` instead of `error` (the new subscription ID).
- `Confirm` — returns `(int64, error)` instead of `error` (the confirmed subscription ID).

### `services/subscription/internal/subscription/usecase/subscribe.go`
- Added `subscribeSaga` interface and `sagaStore` / `sagaTTL` fields.
- `NewSubscribe` takes two new parameters: `subscribeSaga` and `time.Duration`.
- `createAndEnqueue` now also inserts the saga row atomically (same transaction as the subscription and outbox inserts).
- `ConfirmationRequested` payload now carries `SagaID` so Notification can echo it back.
- A `saga_id` is generated via `idgen.New()` — a random hex string, not the confirm token (keeps sensitive tokens out of saga events and NATS logs).

### `services/subscription/internal/subscription/usecase/confirm.go`
- `confirmRepository.Confirm` now returns `(int64, error)` — the subscription ID needed to look up the saga row.
- `NewConfirm` takes a new `confirmSagaOrchestrator` parameter.
- After confirming, calls `orchestrator.HandleConfirmed(subscriptionID)` to complete the saga.

### `services/subscription/internal/config/config.go`
- Added `SagaConfirmTTL time.Duration`.
- Reads `SAGA_CONFIRM_TTL` env var (`time.ParseDuration` format, e.g. `24h`, `30s`). Defaults to `24h`.

### `services/subscription/internal/app/app.go`
- Creates the `SAGA` stream on startup (alongside `NOTIFICATIONS` and `TRACKING`).
- Wires `saga.Store`, `saga.Orchestrator`, `saga.Consumer`, and `saga.Reaper`.
- Starts the saga consumer and reaper as goroutines in `Serve` alongside the existing relay and fanout.

### `services/notification/internal/consumer/consumer.go`
- `processMessage` now returns `(outcome, string, string)` — the third value is `sagaID` (empty for non-confirmation messages).
- After an `actionAck` for a confirmation message: publishes `EmailSent{SagaID}` to `saga.email.sent`.
- After an `actionDLQ` for a confirmation message: publishes `EmailFailed{SagaID, Reason}` to `saga.email.failed` (in addition to the existing DLQ write).
- Release-notification messages are unchanged — `sagaID` is always empty for them.

**Why direct publish (no outbox) on the Notification side:** Notification is intentionally stateless (no DB). Adding an outbox would require giving it a database, which changes its operational model. The orchestrator's state-guarded transitions are idempotent, so a duplicate or delayed reply is a no-op. A lost reply (NATS briefly down) is covered by the TTL reaper — the saga will eventually be compensated. This tradeoff is documented as a Known Limitation in the SDD.

## New NATS Stream

| Stream | Subject | Retention | Duplicates TTL |
|---|---|---|---|
| `SAGA` | `saga.>` | Limits | 2 min |

Limits retention (not WorkQueue) because the saga consumer is the sole subscriber and the stream only serves as a delivery buffer — messages don't need to be kept after delivery.

## Idempotency and Concurrency Guarantees

| Scenario | Protection |
|---|---|
| Duplicate `EmailSent` | `HandleEmailSent` no-ops if state ≠ `STARTED` |
| Duplicate `EmailFailed` | `HandleEmailFailed` no-ops if state is already terminal |
| Confirm races with compensation | `DeleteUnconfirmedSubscription` deletes only `confirmed = FALSE`; 0 rows deleted → saga → `COMPLETED` |
| Two reaper instances | `SKIP LOCKED` ensures each expired saga is claimed by exactly one worker |
| NATS restart mid-publish | Saga reply is at-least-once; duplicates are no-ops via state guards |

## Environment Variable

| Variable | Default | Description |
|---|---|---|
| `SAGA_CONFIRM_TTL` | `24h` | Go duration string. Controls how long a saga waits for user confirmation before the reaper compensates. Set to `30s` to test timeout compensation locally. |

## Follow-up Items

- **Outbox on Notification side** — eliminate the "lost reply" risk by giving the Notification service a lightweight DB for saga reply outbox. Higher ops overhead; justified if the TTL gap is unacceptable.
- **Saga metrics** — Prometheus counters for `STARTED`, `COMPLETED`, `FAILED` transitions to make saga health observable.
- **Cleanup job** — `COMPLETED` and `FAILED` saga rows accumulate. Add a periodic DELETE on rows older than N days to keep the table small.
- **`RepoUntracked` in the same tx** — currently `catalog.DeleteIfOrphaned` emits `RepoUntracked` outside the compensation transaction (same issue as before the saga). A future migration could make this fully atomic.
