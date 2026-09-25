# ADR-0003: Model repositories separately from subscriptions

## Author
Rodion Shapoval

## Status
Accepted (amended)

## Context
Many users can subscribe to the same GitHub repository. The system needs to represent that relationship without duplicating repository identity for every subscription.

The important modeling question is whether a repository should be embedded into each subscription row or stored as its own entity that subscriptions reference.

## Decision
Store repositories in a separate table and let subscriptions reference them by ID.

The Subscription service owns the `repositories` table for catalog purposes: find-or-create on subscribe, orphan cleanup on last unsubscribe, uniqueness constraint on `name`. Subscription-level data — email, confirmation state, and tokens — stays on the subscription record.

**Amendment (Phase 2):** Release tracking state (`last_seen_tag`) was moved out of `repositories` and into Monitoring's `scan_cursors` table. Monitoring maintains its own cursor per repo. The `repositories.last_seen_tag` column remains in the schema as a legacy artifact but is no longer written by any service.

**Amendment (Phase 3):** How Monitoring learns which repos to track changed from `RepoTracked`/`RepoUntracked` events to a synchronous gRPC pull against Subscription's catalog service, which also closed the bootstrap gap noted below. See [ADR-0005](0005-pull-tracked-repositories-via-grpc.md).

## Consequences
### Positive
- avoids duplicating repository identity across subscriptions
- subscription queries need only a join on `repository_id` for repo metadata
- release tracking state is now fully owned by the bounded context that produces it (Monitoring), not mixed into the subscription catalog
- Monitoring can independently evolve its cursor schema without touching subscription migrations
- database constraints protect repository identity cleanly

### Negative
- subscription queries still need a join when repository name is required
- unsubscribe flow is slightly more complex because orphaned repository rows must be cleaned up when the last subscription is removed
- deleting orphaned repositories loses the repo record; a new subscription triggers GitHub validation again
- repository creation must handle concurrent subscribe requests safely (find-or-create with conflict recovery)
- `repositories.last_seen_tag` is a dead column that requires a future cleanup migration

### Tradeoffs
- clean bounded-context ownership of tracking state at the cost of a legacy column in the schema
- `scan_cursors` starts empty; repos subscribed before Monitoring was event-driven must re-subscribe to appear in the scanner (known bootstrap gap — resolved by the Phase 3 amendment above, see ADR-0005)

## Alternatives Considered
### Store repository data directly on each subscription
Rejected because it duplicates repository identity, increases storage redundancy, and makes release detection inconsistent.

### Keep `last_seen_tag` in `repositories` and have Monitoring write it
Rejected because it breaks data ownership boundaries: Monitoring would write to a table owned by Subscription's bounded context. The `scan_cursors` table gives Monitoring a clean, independently-migrated place to own its state.

### Recompute repository state without storing shared records
Rejected because the scanner needs durable per-repo tracking state between scan intervals.
