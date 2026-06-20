# NATS JetStream — Scaling Roadmap

This document tracks future scaling and hardening work for the NATS messaging layer. Items marked **✅ Done** were completed during the monolith-to-services migration (Phases 1–3).

Current shape for reference:
- **Subscription** service: HTTP API + fanout consumer (`notifications.release_found`) + outbox relay → NATS.
- **Monitoring** service (singleton): GitHub scanner + tracking consumer (`tracking.>`) + outbox relay → NATS.
- **Notification** service (stateless): durable consumer (`notifications.confirmation`, `notifications.release`) → SMTP + DLQ.
- Two streams: `NOTIFICATIONS` (`notifications.>`, WorkQueue) and `TRACKING` (`tracking.>`, Limits).
- Shared `services/contract` Go module carries all event types and subjects.

---

## A. Extract the scanner/worker into its own service — ✅ Done (Phase 1 + 2)

The monitoring worker runs in `services/monitoring` as a standalone binary deployed with `replicas: 1`. It connects to Postgres and NATS independently of the Subscription service. The API process no longer runs background work.

---

## B. Move fan-out to the consumer (publish a thin domain event) — ✅ Done (Phase 2)

Monitoring publishes one `ReleaseFound` event per new release (not one per recipient). The Subscription fanout consumer receives it, queries its own `subscriptions` table for confirmed recipients, and inserts per-recipient `ReleaseDetected` rows into its own outbox. The Notification service only sends email; it does not touch subscription data. Each bounded context owns its own data.

---

## C. Subject taxonomy + versioning; events vs commands

**What.** Redesign subjects from `notifications.confirmation` / `notifications.release` to a hierarchical, versioned scheme: `<domain>.<version>.<entity>.<event>`, e.g. `releases.v1.detected`, `email.v1.send`.

**Why.** Two problems with the current names: `notifications.release` is really a command, not an event; and there is no version token, so a breaking payload change has nowhere to land. A version token lets old and new consumers coexist during a rollout.

**How.** Define the taxonomy as constants in `services/contract`. Keep events as past-tense facts and commands as imperatives. Requires cutting over durable consumers — run new consumers on new subjects, drain old ones.

**Trigger.** Before the second consumer or the first breaking schema change.

---

## D. Split streams by domain — Partial ✅ (Phase 2)

`NOTIFICATIONS` and `TRACKING` are now separate streams with independent subjects and retention policies. What remains: if more domains appear (audit, billing, webhooks), they should get their own streams rather than being added to `NOTIFICATIONS`.

**Trigger.** When a second domain of events appears with different retention or HA requirements.

---

## E. Contract as a versioned, published artifact

**What.** The shared `services/contract` Go module is imported directly by all services. Move to a versioned, published contract: protobuf + buf, or a properly versioned Go module with a compatibility policy.

**Why.** A directly-imported module couples deploys — all services must move together on any change, defeating independent deployment. Protobuf additionally provides payload validation and automated compat checks.

**How.** Define events in `.proto`, generate Go, validate with `protovalidate` on consume. Publish as a versioned artifact. Adopt additive-only evolution; pair with subject versioning (C).

**Trigger.** When services move to separate repos or teams, or when an accidental schema break causes an incident.

---

## F. Horizontal consumer scaling

**What.** Run N Notification instances sharing one durable consumer; JetStream load-balances messages across all `Consume` subscribers on that durable.

**Why.** A single consumer instance caps email throughput. Sharing one durable across instances scales horizontally without duplicate delivery.

**How.** Deploy multiple replicas of the notification binary, all binding the same durable name. Set `MaxAckPending` to bound in-flight work. Handlers must be idempotent.

**Trigger.** When consumer lag grows during release spikes.

---

## G. NATS HA / clustering

**What.** Run a 3-node NATS cluster and set stream `Replicas: 3`. Currently a single node, `Replicas: 1`.

**Why.** A single NATS node is a single point of failure. R3 keeps the stream available and durable across a node loss.

**Trigger.** When the async path becomes business-critical enough that a NATS outage is unacceptable.

---

## H. Transactional outbox — ✅ Done (Phase 2)

Both Subscription and Monitoring use the transactional outbox pattern: state changes and "intent to publish" are committed in one database transaction. A relay goroutine polls for unpublished rows with `FOR UPDATE SKIP LOCKED`, publishes with a stable `Msg-Id` for deduplication, and marks rows published. The dual-write race is eliminated.

---

## I. Per-entity ordering when needed

**What.** JetStream guarantees ordering only per subject. If a future consumer needs per-entity ordering (e.g. all events for one repo in sequence), partition by entity in the subject: `tracking.repo.tracked.<repo_id>`, consumed by a single worker per partition.

**Why.** With horizontal scaling (F), messages for the same entity can land on different instances out of order.

**Trigger.** Only if/when a consumer appears that needs ordered per-entity processing. Do not build it speculatively.

---

## Suggested sequencing for remaining work

1. **C / D / E** — subject versioning, stream taxonomy, versioned contract — the plumbing for safe independent evolution.
2. **F / G** — scale and harden the runtime as actual load demands.
3. **I** — per-entity ordering only if a concrete consumer needs it.
