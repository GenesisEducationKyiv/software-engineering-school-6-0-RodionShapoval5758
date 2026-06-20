# ADR-0004: Use NATS JetStream for async notification delivery

## Author
Rodion Shapoval

## Status
Accepted

## Context
The original implementation sent confirmation and release notification emails synchronously inside the monolith. The API handler called the mailer directly during the subscribe request, and the background worker called it during the release scan. This tied email delivery latency and failure to the HTTP response and the scan loop.

The open design choice is how to decouple email delivery from the API and worker so that a slow or unavailable SMTP server does not block callers and so that the notification service can be operated independently.

The system has one producer — the API and background worker — and one consumer — the notification service. The requirement is async delivery with at-least-once guarantee and durability across notification service restarts. There are two event types: subscription confirmation and release notification. No complex routing, fan-out, or ordering across multiple consumers is needed at this scale.

## Decision
Use NATS JetStream as the message transport between the API service and the notification service.

The API service publishes `ConfirmationRequested` and `ReleasePublished` events to a durable, file-backed JetStream stream named `NOTIFICATIONS`. The notification service runs a durable push consumer with explicit acknowledgment: it ACKs on success, NAKs on transient send failure so the message is redelivered, and Terms on unmarshal failure so a poison message is dropped rather than retried indefinitely.

## Consequences
### Positive
- email delivery is decoupled from the HTTP response and the scan loop
- file-backed stream durability means events survive a notification service restart and are redelivered on reconnect
- the notification service can be deployed, restarted, and scaled independently of the API
- NATS JetStream has a clean Go client and requires no external coordination service such as ZooKeeper

### Negative
- NATS becomes a required infrastructure dependency; the system cannot deliver notifications if NATS is unavailable
- publish failures are now silent from the caller's perspective — the subscriber gets a 200 OK even if the confirmation event was not delivered
- there are no built-in consumer lag metrics; observability requires additional instrumentation

### Tradeoffs
- JetStream's at-least-once guarantee means the notification service must tolerate duplicate deliveries; the current mailer is not idempotent
- event schema is owned by the shared `contract` Go module rather than a schema registry; both services must be deployed with a compatible version of that module
- if the system later needs multiple independent consumers of the same event stream, JetStream durable consumers handle that without changes to the producer

## Alternatives Considered
### Kafka
Rejected because it is designed for high-throughput, multi-consumer event streaming with strict partition-level ordering guarantees. Operating Kafka requires managing brokers, partitions, and either ZooKeeper or KRaft. That operational overhead is not justified for a system that produces a handful of events per minute and has a single consumer.

### RabbitMQ
Rejected because its strength is complex routing through exchanges and bindings, which this system does not need. RabbitMQ also does not provide built-in message replay: if the notification service is down when an event is published, the message is lost unless a durable queue is explicitly configured. JetStream provides log-based durability and replay by default.

### Synchronous in-process email sending
Rejected because it ties email delivery latency and SMTP failures directly to the HTTP response for subscribe and to the scan loop for release notifications. A slow or unavailable SMTP server would block the subscriber's request and delay or interrupt the background scan.
