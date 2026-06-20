# ADR-0004: Use NATS JetStream for async inter-service messaging

## Author
Rodion Shapoval

## Status
Accepted

## Context
The original implementation sent confirmation and release notification emails synchronously inside the monolith. This tied email delivery latency and failure to the HTTP response and the scan loop.

The system was then split into three independent services — Subscription, Monitoring, and Notification — which introduced two distinct async messaging needs:

1. **Notification delivery**: Subscription and Monitoring publish events that the Notification service must consume to send emails (`ConfirmationRequested`, `ReleaseDetected`, `ReleaseFound`).
2. **Tracking coordination**: Subscription must inform Monitoring which repositories to scan (`RepoTracked`, `RepoUntracked`).

Both require at-least-once delivery with durability across restarts. The system has a small number of event types and producers; no complex routing, fan-out across many consumers, or strict per-entity ordering is needed at this scale.

## Decision
Use NATS JetStream as the message transport between all three services.

Two durable, file-backed streams are provisioned:

- **`NOTIFICATIONS`** (`notifications.>`, WorkQueue retention) — carries `notifications.confirmation`, `notifications.release`, and `notifications.release_found`. Subscription and Monitoring publish via their transactional outbox relays. The Notification service consumes confirmation and release messages; the Subscription fanout consumer consumes `release_found` to fan out per-recipient outbox rows.
- **`TRACKING`** (`tracking.>`, Limits retention) — carries `tracking.repo.tracked` and `tracking.repo.untracked`. Subscription publishes on subscribe/unsubscribe. Monitoring consumes via a durable consumer to maintain `scan_cursors`.

Each publish includes a stable `Msg-Id` header for server-side deduplication within a 2-minute window. Monitoring relay uses `mon-` prefixed IDs to avoid collision with subscription relay IDs.

The Notification service uses explicit acknowledgment: ACK on success, NAK on transient SMTP failure (triggers redelivery), Term on unmarshal failure (message dropped to `NOTIFICATIONS_DLQ`).

All event definitions live in the shared `services/contract` Go module.

## Consequences
### Positive
- email delivery is fully decoupled from HTTP responses and the scan loop
- file-backed stream durability means events survive service restarts and are redelivered on reconnect
- all three services can be deployed, restarted, and scaled independently
- the fanout consumer in Subscription keeps subscriber lookup co-located with the data it owns, avoiding cross-service queries
- the transactional outbox pattern (state change + publish intent in one DB transaction) eliminates the dual-write race

### Negative
- NATS is a required infrastructure dependency; all async flows fail if NATS is unavailable
- eventual consistency: a new subscription does not appear in `scan_cursors` until the `RepoTracked` event is consumed
- the shared `contract` module couples service deployments; both publisher and consumer must use a compatible version
- there are no built-in consumer lag metrics; observability requires additional instrumentation

### Tradeoffs
- JetStream's at-least-once guarantee requires consumers to tolerate duplicate deliveries; the Notification service mailer is not idempotent
- WorkQueue retention on `NOTIFICATIONS` means a message can only be consumed by one consumer group; this is the right model for delivery commands but limits future fan-out to multiple independent consumers of the same event
- `TRACKING` uses Limits retention rather than WorkQueue since only Monitoring consumes it and ordering across restarts is handled by the durable consumer position

## Alternatives Considered
### Kafka
Rejected because of high operational overhead (brokers, ZooKeeper/KRaft, partition management) not justified at this event volume and team size.

### RabbitMQ
Rejected because it lacks built-in log-based replay. If the Notification service is down when an event is published, the message is lost without explicit durable queue configuration. JetStream provides log-based durability and replay by default.

### Direct gRPC calls between services
Rejected because synchronous RPC requires the downstream service to be available at call time, defeating the goal of independent failure domains.

### Synchronous in-process handling (original monolith)
Rejected because it ties email delivery latency and SMTP failures directly to the HTTP response and the scan loop, and it prevents independent scaling and deployment of each service.
