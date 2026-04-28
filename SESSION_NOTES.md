# Session Notes

## Desired Project Structure and Package Responsibilities

For this MVP, keep the structure simple and explicit:

```text
cmd/api/
internal/config/
internal/http/
internal/service/
internal/store/
internal/github/
internal/mail/
internal/worker/
migrations/
```

Responsibilities:

- `cmd/api/`
  - Application entrypoint.
  - Load config, connect DB, run migrations, build dependencies, start HTTP server, start background scanner.
  - Keep business logic out of here.

- `internal/config/`
  - Parse env vars.
  - Build a typed config struct.
  - Example: DB URL, GitHub token, SMTP settings, scan interval.

- `internal/http/`
  - Router, handlers, request parsing, response writing.
  - Only HTTP concerns.
  - It should call services, not query the DB directly unless the project is extremely small.

- `internal/service/`
  - Core business logic.
  - This is the most important package.
  - Examples:
    - create subscription
    - confirm token
    - unsubscribe by token
    - list subscriptions by email
    - scan repos for new releases
    - notify subscribers

- `internal/store/`
  - DB access.
  - SQL queries and persistence logic.
  - No HTTP knowledge.
  - Keep this boring and predictable.

- `internal/github/`
  - GitHub API client.
  - Validate repo existence.
  - Fetch latest release/tag.
  - Handle `404` and `429` carefully.

- `internal/mail/`
  - Send confirmation and release notification emails.
  - Build email content and delivery abstraction.

- `internal/worker/`
  - Periodic background jobs.
  - For MVP: scanner loop that checks active subscriptions / repositories and triggers notifications.

- `migrations/`
  - SQL files for schema creation and changes.
  - Run on startup.

A good MVP data model is roughly:

- `subscriptions`
  - `id`
  - `email`
  - `repo`
  - `confirmed`
  - `confirm_token`
  - `unsubscribe_token`
  - `created_at`
- `repositories` or `tracked_repositories`
  - `repo`
  - `last_seen_tag`
  - `updated_at`

You can also keep `last_seen_tag` directly on `subscriptions`, but that duplicates repo state across subscribers. Better options are:

1. Simpler MVP option:
- Store `last_seen_tag` on each subscription.
- Easier to implement.
- Some duplication.

2. Cleaner option:
- Store repository tracking separately.
- Better if many users subscribe to the same repo.
- Slightly more design work.

Recommendation:

- If your goal is fastest correct MVP: put `last_seen_tag` on `subscriptions`.
- If you want a better structure without much added complexity: create a `repositories` table and link subscriptions to it logically by `repo`.

Suggested implementation order:

1. Read `swagger.yaml` carefully and write down each endpoint’s inputs and status codes.
2. Design the DB schema.
3. Create the basic project skeleton under `internal/`.
4. Implement config loading and app bootstrap in `cmd/api/main.go`.
5. Implement store layer.
6. Implement service layer.
7. Add HTTP handlers.
8. Add GitHub client integration.
9. Add mail sending abstraction.
10. Add scanner worker.
11. Add unit tests for service logic.

Best first concrete task:

- define the DB schema
- define service interfaces
- implement `POST /api/subscribe` first

That endpoint forces you to solve the important foundations:

- validation
- GitHub lookup
- duplicate handling
- token generation
- persistence
- email trigger

## Service Interfaces and Why to Use Them

Service interfaces are contracts. In Go, an interface says: “anything that has these methods can be used here.”

Example:

```go
type SubscriptionService interface {
	Subscribe(ctx context.Context, email, repo string) error
	Confirm(ctx context.Context, token string) error
	Unsubscribe(ctx context.Context, token string) error
	ListByEmail(ctx context.Context, email string) ([]Subscription, error)
}
```

Then your HTTP handler depends on that interface, not on a concrete struct:

```go
type Handler struct {
	service SubscriptionService
}
```

Why this is useful:

- It separates HTTP code from business logic.
- Your handler does not need to know how subscriptions are stored or how GitHub/email work.
- It makes testing much easier, because in handler tests you can pass a fake implementation.
- It reduces coupling, so you can change internals without rewriting everything.

Why not use interfaces everywhere:

- In Go, beginners often overuse them.
- If you create interfaces for every struct “just in case”, the code becomes abstract and harder to understand.
- Interfaces should appear where they solve a real boundary problem, not as decoration.

A good rule for this project:

- Use interfaces at boundaries.
- Do not use interfaces for plain data structs or simple internal helpers.

Good places for interfaces here:

- HTTP handler depends on `SubscriptionService`
- Service depends on `SubscriptionStore`
- Service depends on `GitHubClient`
- Service depends on `Mailer`

Example:

```go
type SubscriptionStore interface {
	Create(ctx context.Context, sub Subscription) error
	FindByEmailAndRepo(ctx context.Context, email, repo string) (*Subscription, error)
	FindByConfirmToken(ctx context.Context, token string) (*Subscription, error)
	Confirm(ctx context.Context, id int64) error
	ListConfirmedByEmail(ctx context.Context, email string) ([]Subscription, error)
}
```

```go
type GitHubClient interface {
	RepoExists(ctx context.Context, repo string) (bool, error)
	LatestReleaseTag(ctx context.Context, repo string) (string, error)
}
```

```go
type Mailer interface {
	SendConfirmation(ctx context.Context, email, token string) error
	SendReleaseNotification(ctx context.Context, email, repo, tag, unsubscribeToken string) error
}
```

Why this helps in tests:

- In service tests, you can fake GitHub and Mailer behavior.
- You can simulate `404`, `429`, email failures, duplicates, invalid tokens.
- That is much cleaner than hitting real GitHub or SMTP.

Strict advice:

- Define interfaces where they are consumed, not automatically where structs are implemented.
- Start with concrete structs if you are unsure.
- Introduce an interface when you need mocking, swapping implementations, or cleaner boundaries.
- For this project, interfaces around store, GitHub client, and mailer are justified.
- An interface for every package or every struct is not justified.

A practical starting point:

- Make a concrete `subscriptionService struct`.
- Let it depend on 3 interfaces:
  - `SubscriptionStore`
  - `GitHubClient`
  - `Mailer`

That gives you clean architecture without unnecessary abstraction.
