# GitHub-Release-Notification-API
API that allows users to subscribe to email notifications about new releases of a chosen GitHub repository.

### How it works and what it does(to a user):
- subscribe your email to owner/repo if you have an API key
- confirm subscription in the email with the button
- periodically checks GitHub releases and sends notifications
- go to that exact release with the button in the email
- unsubscribe with the button in the notification email
- list all the subscriptions for an email

### Stack
- Golang
- Chi & net/http
- PostgreSQL
- golang-migrate
- GitHub REST API
- GitHub Actions for CI
- net/smtp / Mailpit for local SMTP server
- Docker / Docker Compose
- Gemini CLI extensions for Code Review and Idiomatic Go check style
- Codex CLI for explanation, some research and trivial/routine code

### Endpoints
- POST /api/subscribe
- GET /api/confirm/{token}
- GET /api/unsubscribe/{token}
- GET /api/subscriptions?email=...

### Configuration

Major env vars:
- GITHUB_TOKEN - without it the API can make only up to 60 req/hour
- API_KEY - the endpoints are not secured without it

### Run Locally
- `docker compose up --build` for the first time, then without the --build option
- `go run test ./...` to run unit tests(you have to be in the root of the project)

### CI
- CI is in `.github/workflows/ci.yml`
- Implemented with GitHub Actions
- Runs on every push/pull-request
- Runs vet, build, test and golangci-lint

### Notes
- migrations run on startup
- Mailpit UI available locally on port 8025
- Swagger contract is in swagger.yaml
- API auth key is *genesis-summer-school*. Header is "Authorization"
### **Workflow + thoughts:**

DB schema:
    Two main entities:
        subscription(email, repo id, tokens, confirmed), 
        repo(name, last_checked)
    subscriptions table has a composite UNIQUE that ensures the same pair of email and repo cannot be added 
    So that the last changes are not updated in all the users separately, but in one row
    + indexes for tokens and email

Errors && logging:
    Wrapping technical failures(db query failed, network failure)
    Sentinel errors for meaningful errors (not found, exists, invalid format)
    Logging:
        ERRORS: log fatal with context when something failed and needs attention
        INFO: when some process started or for providing context, so that I know what has happened

Instead of creating GerOrCreate method for repository that is race-safe
I separated methods Create and Find and implemented manual race condition handling
so that everything is explicit

In case of duplicate tokens the program tries to regenerate it 5 times. 
There is a ridiculously small possibility of getting 
duplicate tokens with 32 token length, so I neglect it

Swagger docs states that /api/subscribe consumes both JSON and form data, so I made a fallback for both

Concurrent worker is supposed to work like that:
    Every tick(4 minutes) get all the repositories. Create a waitGroup
    and a semaphore channel with 10 slots. Add 1 goroutine to the wait group and 1 slot to the semaphore.
    Run that go routine that processes one repository: if repository has no releases - skip,
    if the github rate limit is hit, skip that scan. After finishing of one processing release goroutine
    from the semaphore and waitgroup. Waitgroup makes sure that all the repositories in the current scan
    are scanned, before repeating the cycle. 
    Semaphore makes sure that at most 10 repositories are scanned in parallel