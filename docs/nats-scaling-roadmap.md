# NATS JetStream — Future Scaling Roadmap

Detailed notes for the **future** changes from the NATS setup analysis. None of these are
required for the current homework — they apply when the system actually breaks into
separate services. The condensed version (plus the "now" hardening items) lives in the
plan file; this document expands the future items with the *why*, the *how* in this
codebase, the *tradeoffs*, and the *trigger* that should make you act.

Current shape for reference:
- API binary = HTTP handlers + in-process release **worker** (`internal/monitoring/`) +
  **publisher** (`internal/notifier/`).
- Notification binary = single durable **consumer** (`services/notification/...`) → SMTP.
- One stream `NOTIFICATIONS` (`notifications.>`), JSON events in the shared `contract`
  module (`services/contract/events.go`).

**The key lever is A + B.** Do those first; the rest is hardening that follows from them.

---

## A. Extract the scanner/worker into its own service

**What.** Move the release worker (`internal/monitoring/worker.go`) out of the API process
into its own binary — a `scanner` service that polls GitHub and emits release events.

**Why.** The worker and the email path have *different scaling profiles*. Scanning is
GitHub-rate-limit-bound and runs on a fixed interval across all tracked repos; email
sending is SMTP-throughput-bound and spikes when a popular repo releases. Bundling them in
one binary means you can't scale or deploy them independently, and a slow scan loop shares
a process with the HTTP API. The peer project already does this split (scanner as its own
service).

**How.** The worker already depends only on small interfaces (`catalogClient`,
`githubClient`, `releaseNotifier` in `internal/monitoring/ports.go`). Lift the package into
a `cmd/scanner` / `services/scanner` binary, give it its own config + DB pool, and replace
the in-process `ReleaseNotifier` call with a NATS publish of a domain event (see B).

**Tradeoffs.** + Independent scaling/deploy, isolated failure domains, API process no
longer does background work. − A third binary to operate; the scanner needs its own DB
access or a query path to the catalog; more moving parts in compose.

**Trigger.** When scan latency starts affecting API responsiveness, or when you want to
scale email workers without scaling polling (or vice versa).

---

## B. Move fan-out to the consumer (publish a thin domain event)

**What.** Today the worker looks up confirmed subscribers and `SendReleaseEmails` publishes
**one pre-addressed email command per recipient** (`internal/monitoring/notifier.go` +
`internal/notifier/publisher.go`). Replace that with **one** event per release —
`releases.detected{repo, tag, name, url}` — and let the notification service own "who to
notify" (subscriber lookup + per-recipient send).

**Why.** The current release "event" isn't a domain event, it's a batch of commands the
producer pre-computed. That couples release *detection* to subscription *knowledge*: the
scanner must read the subscription store. A thin domain event keeps each bounded context
responsible for its own data — the scanner says "this happened," the notifier decides "who
cares." It also shrinks the message volume on the stream (1 event vs N commands) and makes
the event reusable by future consumers (analytics, webhooks, etc.).

**How.** Publish `releases.detected` from the scanner. The notification service consumes it,
calls `ListConfirmedByRepositoryID`, and fans out emails on the consumer side. The
subscriber-lookup logic already exists (`subscription` service /
`ListConfirmedByRepositoryID`); it moves from the worker's `ReleaseNotifier` to the
consumer.

**Tradeoffs.** + Clean separation of contexts, smaller/ reusable events, no producer-side
coupling to subscriptions. − The notification context now needs read access to subscription
data: either its own store, a query API/gRPC call back to the subscription service, or
event-carried state. This is the central design decision — pick the coupling you can live
with. Fat events (carry recipients) reduce calls but recreate today's coupling; thin events
+ lookup decouple but add a dependency.

**Trigger.** Do this together with A — splitting the scanner forces the question of where
fan-out lives, and producer-side fan-out across a service boundary is the wrong default.

---

## C. Subject taxonomy + versioning; events vs commands

**What.** Redesign subjects from `notifications.confirmation` / `notifications.release` to a
hierarchical, versioned scheme: `<domain>.<version>.<entity>.<event>`, e.g.
`releases.v1.detected`, `email.v1.send`.

**Why.** Two problems with the current names. First, `notifications.release` is really a
*command* ("send this email"), not an *event* ("a release happened") — mixing the two makes
intent ambiguous as consumers multiply. Second, there's no version token, so any
breaking payload change has nowhere to go. A version in the subject lets old and new
consumers coexist during a rollout.

**How.** Define the taxonomy as constants in the contract artifact (E). Keep "events" as
past-tense facts (`releases.v1.detected`) and "commands" as imperatives (`email.v1.send`).
Route them to appropriate streams (D).

**Tradeoffs.** + Clear intent, safe schema evolution, room for many consumers. − Requires a
migration: existing durable consumers reference the old subjects; you cut over by running
new consumers on new subjects and draining the old.

**Trigger.** Before the second consumer or the first breaking schema change — whichever
comes first.

---

## D. Split streams by domain

**What.** One `NOTIFICATIONS` stream today. As domains grow (releases, email, audit,
billing…), give each its own stream with independent retention/storage/replica policy.

**Why.** A single stream forces one retention and storage policy on everything. Release
events and audit logs have very different lifetimes; email commands are short-lived while
audit may need long retention. Separate streams also isolate load and let you replicate the
critical ones (G) without paying for the rest.

**How.** e.g. `RELEASES` stream for `releases.>`, `EMAIL` stream for `email.>`. Each with
its own `MaxAge`/`MaxBytes`/`Replicas`. Provisioning owned in one place (see the "single
owner" item in the now-hardening list).

**Tradeoffs.** + Independent policies, isolation, targeted HA. − More streams to manage;
cross-stream workflows need care (a consumer can't filter across streams).

**Trigger.** When a second domain of events appears, or when one event type needs a
retention/HA policy the others don't.

---

## E. Contract as a versioned, published artifact

**What.** The shared `contract` Go module (`services/contract/`) is imported directly by
both binaries. Move to a versioned, published contract: protobuf + buf (as the peer does)
or a properly versioned/published Go module.

**Why.** A directly-imported module couples deploys — both binaries must move together on
any change, which defeats independent service deployment. Protobuf + buf additionally buys
payload validation at the edge and automated backward/forward-compat checks (buf breaking).
Note: this isn't about cross-language support; it's about IDL discipline and compatibility
guarantees across independently-deployed services.

**How.** Define events in `.proto`, generate Go, validate with `protovalidate` on consume.
Publish the generated package as a versioned artifact. Adopt an additive-only / backward-
compatible evolution policy; pair with subject versioning (C).

**Tradeoffs.** + Independent deploys, schema validation, compat enforcement. − buf toolchain
and codegen step; more ceremony than hand-written JSON (which is fine while it's one repo,
one team).

**Trigger.** When services move to separate repos/teams, or when an accidental schema break
causes an incident.

---

## F. Horizontal consumer scaling

**What.** Run N notification instances sharing **one** durable consumer; JetStream
load-balances messages across all `Consume` subscribers on that durable.

**Why.** A single consumer instance caps email throughput. Sharing one durable across
instances scales horizontally without duplicating delivery (each message goes to exactly
one instance).

**How.** Deploy multiple replicas of the notification binary, all binding the same durable
name (`notification-consumer`). Set `MaxAckPending` to bound in-flight work per consumer
(backpressure). Handlers **must be idempotent** — this depends on the producer-side
`Msg-Id`/dedup work from the now-hardening list, plus consumer-side idempotency for
duplicates beyond the dedup window.

**Tradeoffs.** + Linear throughput scaling. − No global ordering across instances
(acceptable for independent emails; not for per-entity ordered processing — see I).
Requires idempotency to be solved first.

**Trigger.** When consumer lag (the metric to add in hardening) grows during release
spikes.

---

## G. NATS HA / clustering

**What.** Run a 3-node NATS cluster and set stream `Replicas: 3`. Today it's a single node,
`Replicas: 1`.

**Why.** A single NATS node is a single point of failure — lose it and the whole async path
stops, and (without replicas) un-acked messages on that node are at risk. R3 keeps the
stream available and durable across a node loss.

**How.** Cluster config in compose/k8s; bump `Replicas` in the stream config. Quorum needs
an odd number of nodes (3 minimum).

**Tradeoffs.** + Availability and durability under node failure. − 3× the NATS footprint,
cross-node replication latency on publish-ack, more ops.

**Trigger.** When the async path becomes business-critical enough that a NATS outage is
unacceptable.

---

## H. Transactional outbox

**What.** Eliminate the dual-write between the DB state change and the NATS publish. Write
the event to an `outbox` table in the **same transaction** as the state change; a relay
process reads the outbox and publishes to NATS.

**Why.** Today `UpdateLastSeenTag` and the publish are two separate writes
(`worker.go processRepository`). Any ordering loses something: update-then-publish can drop
a release on publish failure; publish-then-update can duplicate on a crash between them. The
outbox makes the state change and the "intent to publish" atomic, so the relay can retry the
publish safely with no loss and bounded duplication.

**How.** Add an `outbox` table; in the same tx that advances `last_seen_tag`, insert the
`releases.detected` row. A relay (poller or CDC) publishes pending rows with a stable
`Msg-Id` (dedup handles relay retries) and marks them sent.

**Tradeoffs.** + Exactly the right correctness guarantee for the dual-write. − A relay
component to build and run; small publish latency from the poll interval (or add CDC).

**Trigger.** When the "now" reorder fix (publish-before-update, at-least-once) is no longer
good enough — i.e. when duplicate emails become a real problem and you need true atomicity.

---

## I. Per-entity ordering when needed

**What.** JetStream guarantees ordering only *per subject*. If a future consumer must
process events for a given entity in order (e.g. all events for one repo), partition by an
entity token in the subject: `releases.v1.detected.<repo>`, with per-subject consumers or
a partitioned consumer scheme.

**Why.** With horizontal scaling (F), messages for the same entity can land on different
instances out of order. Most flows here (independent confirmation/release emails) don't
care, but anything that mutates per-entity state in sequence does.

**How.** Encode the entity in the subject token and ensure all events for one entity share a
subject, consumed by a single worker for that partition.

**Tradeoffs.** + Correct per-entity ordering. − Limits parallelism within a partition; hot
entities become bottlenecks.

**Trigger.** Only if/when a consumer appears that needs ordered per-entity processing. Don't
build it speculatively.

---

## Suggested sequencing
1. **A → B → H** — the architectural core: split the scanner, make the release a real
   domain event, then make the dual-write atomic.
2. **C / D / E** — taxonomy, stream split, versioned contract — the plumbing that supports
   multiple services and safe evolution.
3. **F / G / I** — scale and harden the runtime as actual load demands.
