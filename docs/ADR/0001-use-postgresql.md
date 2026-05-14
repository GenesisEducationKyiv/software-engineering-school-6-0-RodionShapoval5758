# ADR-0001: Use pgx for PostgreSQL access

## Author
Rodion Shapoval

## Status
Accepted

## Context
The service uses PostgreSQL for durable storage of subscriptions, repository tracking state, and token-based lifecycle flows. The database schema uses relational constraints for uniqueness and reliable queries between subscriptions and repositories.

The open design choice is how the Go application should access PostgreSQL. The current persistence logic is small and does not contain heavy query-generation, reporting, or complex database access patterns. A simple, explicit PostgreSQL client is enough for the current system.

## Decision
Use `pgx` directly for PostgreSQL access instead of `sqlc`, `sqlx`, or GORM.

`pgx` is fast, simple, and relatively low-level. It gives direct control over SQL queries and PostgreSQL behavior without adding a larger abstraction layer. For this project size, that is a reasonable tradeoff because the persistence layer is still small and does not have data-heavy business logic.

## Consequences
### Positive
- `pgx` provides fast native PostgreSQL support and connection pooling
- direct SQL keeps database behavior explicit and easy to inspect
- the dependency surface stays small compared to ORM-based approaches
- the project avoids generated code and ORM conventions while the SQL surface is still small
- the approach is easy to understand for learning PostgreSQL access in Go

### Negative
- query mistakes are usually caught at runtime or by tests, not during development-time code generation
- handwritten scanning and SQL mapping can become repetitive if the persistence layer grows
- developers must keep SQL, Go structs, and migrations aligned manually

### Tradeoffs
- `pgx` is simpler and lighter than heavier abstractions, but it does not validate SQL queries at compile time
- this is acceptable for the current project because the SQL surface is small and can be covered with focused repository tests
- if query complexity grows significantly, switching to generated query code may become worth the added tooling

## Alternatives Considered
### sqlc
Rejected for now because it adds a generation step and more structure than the current persistence layer needs.

The main advantage of `sqlc` is that it can catch many query and type mistakes during development by generating Go code from SQL. That is stronger than plain `pgx`, where query bugs are more likely to appear in tests or at runtime. This may become attractive later if the project gains many queries, more complex joins, or stronger compile-time guarantees become worth the tooling cost.

### sqlx
Rejected because it still keeps handwritten SQL but adds an additional abstraction layer over scanning and mapping. The current project can stay explicit with `pgx` without needing that helper layer.

### GORM
Rejected because an ORM is heavier than needed for this project. It hides more SQL behavior behind model mapping and conventions, which is not a good tradeoff while the schema and queries are still simple and important to understand directly.

### database/sql
Rejected because the project benefits from PostgreSQL-native behavior and ergonomics provided by `pgx`, including its native connection pool and type support.
