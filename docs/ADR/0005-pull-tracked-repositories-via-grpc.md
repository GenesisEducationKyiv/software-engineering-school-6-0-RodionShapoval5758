# ADR-0005: Pull tracked repositories via gRPC instead of NATS tracking events

## Author
Rodion Shapoval

## Status
Accepted (retroactive — the code already reflects this decision; this ADR was written after the fact to close the documentation gap)

## Context
ADR-0004 introduced a `TRACKING` NATS JetStream stream (`tracking.>`) carrying `RepoTracked`/`RepoUntracked` events, published by Subscription on subscribe/unsubscribe and consumed by Monitoring to incrementally maintain its `scan_cursors` table. ADR-0004 explicitly rejected direct gRPC calls between services for inter-service coordination, reasoning that synchronous RPC requires the downstream service to be available at call time, defeating the goal of independent failure domains.

That NATS-based design carried two costs already acknowledged in ADR-0003's Phase 2 amendment:
- **Bootstrap gap**: `scan_cursors` starts empty; a repo subscribed before Monitoring existed (or before it went event-driven) never appears in the scanner unless someone resubscribes.
- **Eventual consistency / drift risk**: Monitoring's local mirror of "which repos exist" depends on having consumed every `RepoTracked`/`RepoUntracked` event correctly, with no independent way to reconcile if one was lost, delayed, or the consumer fell behind.

Two commits reversed the design:
- `281f5f2` ("feat(monitoring): remove nats tracking consumer") deleted the consumer side (`services/monitoring/internal/consumer`).
- `05e3d58` ("refactor: remove NATS tracking events, update SDD to reflect gRPC catalog") deleted the publish side, the `RepoTracked`/`RepoUntracked` contract types, and the `TRACKING` stream declaration.

Monitoring now calls Subscription's `CatalogService.ListTrackedRepos` gRPC endpoint once per scan cycle (`services/monitoring/internal/monitoring/worker.go:runOneScan` → `catalogclient.Adapter.ListTracked`), merging the returned repo list with its own locally-owned `last_seen_tag` cursor before scanning. `last_seen_tag` itself is unaffected — it stays exclusively Monitoring-owned state, written directly to `scan_cursors`; only the "which repos currently exist" question moved from push/event to pull/RPC.

## Decision
Replace the NATS `TRACKING` stream and its consumer/publisher with a synchronous gRPC call: Monitoring pulls the full list of tracked repositories from Subscription's catalog service at the start of every scan cycle, instead of maintaining an incrementally-updated local mirror driven by tracking events.

This is judged safe despite ADR-0004's original objection to synchronous RPC because the call site is fundamentally different from a request-response path:
- Monitoring is a singleton background scanner on a fixed interval, not an HTTP handler — a failed or slow `ListTrackedRepos` call delays that scan cycle but doesn't block a client response or lose data; the next tick retries.
- The call is read-only and idempotent, bounded by a 5s timeout (`grpcListTrackedTimeout`) with `grpc.WaitForReady(true)` to ride out brief Subscription unavailability without hard-failing the loop.
- Pulling fresh state every cycle removes the eventual-consistency window and the bootstrap gap entirely, at the cost of one small RPC per scan interval.

## Consequences

### Positive
- No bootstrap gap: any tracked repo appears on the very next scan cycle, no resubscription needed.
- No drift risk between Subscription's `repositories` table and Monitoring's view of it — Monitoring always reads current state, not a replayed event log.
- One fewer NATS stream, consumer, and contract event pair to operate, version, and reason about.
- `last_seen_tag` ownership is unchanged — still exclusively Monitoring's — so this doesn't reopen the ADR-0003 boundary question.

### Negative
- Reintroduces a synchronous runtime dependency from Monitoring on Subscription's availability, which ADR-0004 originally avoided by design. Extended Subscription downtime means Monitoring silently stops discovering new/removed repos until it recovers.
- Couples Monitoring's availability story to Subscription's in a way the event-driven design didn't.
- No fallback if `ListTrackedRepos` fails mid-scan — `runOneScan` skips the whole cycle rather than scanning against a stale repo list.

### Tradeoffs
- Simplicity and read consistency (single source of truth, pulled fresh) traded against decoupling and independent failure domains — the same tradeoff ADR-0004 weighed the other way for the notification-delivery path. The difference is call shape: this is a background poll, not a request/response path, so an availability coupling here is far less costly than it would be on, say, `Subscribe`.

## Alternatives Considered

### Keep the NATS TRACKING stream, fix the bootstrap gap separately (e.g. periodic reconciliation job)
Rejected as more moving parts than removing the stream outright — you'd run a pull-based reconciliation *and* a push-based event stream just to keep them consistent with each other.

### Hybrid: events for immediacy, periodic gRPC resync for drift correction
Rejected because Monitoring is already fully cycle-based — it never reacts to an individual subscribe/unsubscribe in real time. There's no incremental case worth the extra event-plumbing when every consumer of "which repos" is already polling on a timer.

### Status quo (NATS events only)
Rejected due to the acknowledged bootstrap gap and drift risk from ADR-0003, and because it kept `services/contract` carrying event types with exactly one producer and one consumer — minimal value for the coupling cost.
