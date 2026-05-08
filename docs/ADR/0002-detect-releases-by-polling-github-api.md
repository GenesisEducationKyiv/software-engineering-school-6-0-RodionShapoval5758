# ADR-0002: Detect new releases by polling GitHub API

## Context
The system must detect new releases for tracked repositories and notify subscribers. The current product is a single service and needs a simple, controllable integration model that works without additional external event infrastructure.

## Decision
Detect new releases by periodically polling the GitHub REST API from a background worker inside the monolith.

## Consequences
### Positive
- simple integration model with no webhook receiver or signature verification flow
- works within the current monolith architecture
- polling cadence is controlled by the service
- easy to combine with persisted `last_seen_tag` state

### Negative
- notifications are not instant and depend on scan interval
- GitHub rate limits directly affect scan freshness
- repeated polling creates unnecessary requests compared to push-based integration

### Tradeoffs
- simpler operationally than webhooks, but less real-time and less efficient

## Alternatives Considered
### GitHub webhooks
Rejected for now because they require additional inbound integration, delivery verification, and more operational complexity than the current project needs.

### Manual refresh or user-triggered checks
Rejected because the product requirement is automatic release notification.
