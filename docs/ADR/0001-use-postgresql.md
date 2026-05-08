# ADR-0001: Use PostgreSQL as the source of truth

## Context
The service needs durable storage for subscriptions, repository tracking state, and token-based lifecycle flows. It also needs schema constraints for uniqueness and reliable relational queries between subscriptions and repositories.

The Go ecosystem has strong PostgreSQL support through `pgx`, including a native connection pool already used in this project.

## Decision
Use PostgreSQL as the main and only persistent data store for the service.

## Consequences
### Positive
- PostgreSQL is widely used, mature, and well understood in production systems
- `pgx` provides good native Go support and connection pooling
- relational modeling fits subscriptions, repositories, and token constraints well
- unique constraints and indexes can enforce important invariants at the database level

### Negative
- the service depends on an external database even for local development
- schema changes require migrations and careful compatibility handling

### Tradeoffs
- simpler and safer than mixing multiple storage systems at the current project size

## Alternatives Considered
### In-memory storage
Rejected because application state would be lost on restart and uniqueness guarantees would be weak.

### A different SQL or NoSQL database
Rejected because the current data model is relational and PostgreSQL has better fit, tooling, and native Go support for this project.
