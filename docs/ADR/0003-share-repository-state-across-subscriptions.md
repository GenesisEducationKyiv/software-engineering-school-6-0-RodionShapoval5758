# ADR-0003: Model repositories separately from subscriptions

## Author
Rodion Shapoval

## Status
Accepted

## Context
Many users can subscribe to the same GitHub repository. The system needs to represent that relationship without duplicating repository identity and release tracking data for every subscription.

The important modeling question is whether a repository should be embedded into each subscription row or stored as its own entity that subscriptions reference.

## Decision
Store repositories in a separate table and let subscriptions reference them by ID.

Repository-level tracking state, including `last_seen_tag`, is stored once per repository. Subscription-level data, including email, confirmation state, and tokens, remains on the subscription record.

## Consequences
### Positive
- avoids duplicating repository identity and tracking data across subscriptions
- `last_seen_tag` is tracked once per repository, which simplifies release detection
- reduces repeated GitHub release checks for the same repository
- keeps repository-level and subscription-level responsibilities separate
- allows database constraints to protect repository identity cleanly

### Negative
- subscription queries need a join when repository data is required
- unsubscribe flow becomes slightly more complex because orphaned repository rows are cleaned up when the last subscription is removed
- deleting orphaned repositories loses repository-level scan state if someone subscribes to the same repository again later
- repository creation must handle concurrent subscribe requests safely

### Tradeoffs
- slightly more coordination logic in exchange for a clearer data model and less duplicated state
- deleting orphaned repositories avoids tracking repositories with no subscribers, but keeping them would preserve `last_seen_tag` for possible future subscribers

## Alternatives Considered
### Store repository data directly on each subscription
Rejected because it duplicates repository identity and `last_seen_tag`, increases storage redundancy, and makes release detection less consistent.

### Recompute repository state without storing shared records
Rejected because the worker needs durable repository-level tracking state between scan intervals.
