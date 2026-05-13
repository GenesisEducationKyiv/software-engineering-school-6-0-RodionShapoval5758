# ADR-0003: Share repository state across subscriptions

## Author
Rodion Shapoval

## Status
Accepted

## Context
Many users can subscribe to the same GitHub repository. The system needs to store release tracking state such as `last_seen_tag` and avoid duplicating the same repository-level data for every subscriber.

## Decision
Store repository tracking state once per repository and let multiple subscriptions reference the same repository record.

## Consequences
### Positive
- avoids duplicating repository data across subscriptions
- `last_seen_tag` is tracked once per repository, which simplifies release detection
- reduces repeated GitHub release checks for the same repository
- allows database constraints to protect repository identity cleanly

### Negative
- unsubscribe flow becomes slightly more complex because orphaned repository rows must be cleaned up
- repository creation must handle concurrent subscribe requests safely

### Tradeoffs
- slightly more coordination logic in exchange for cleaner persistence and less duplicated state

## Alternatives Considered
### Store repository state separately for each subscription
Rejected because it duplicates `last_seen_tag` and other repository-level information, increases storage redundancy, and makes release detection less consistent.

### Recompute repository state without storing shared records
Rejected because the worker needs durable repository-level tracking state between scan intervals.
