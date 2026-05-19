.PHONY: test test-integration test-e2e lint-fix lint-all format format-check

build:
	docker compose up --build -d
up:
	docker compose up -d
down:
	docker compose down
test:
	go test ./...

test-integration:
	docker compose up -d --wait postgres_db mailpit
	docker compose exec -T postgres_db psql -U postgres -c "CREATE DATABASE github_release_notifications_test;" 2>/dev/null || true
	TEST_DATABASE_URL="postgres://postgres:password@localhost:5432/github_release_notifications_test?sslmode=disable" \
		go test -tags=integration ./test/integration/... -v
	docker compose down

test-e2e:
	API_KEY=test-api-key docker compose up -d --wait --build
	cd frontend && API_KEY=test-api-key npx playwright test
	docker compose down -v

lint:
	golangci-lint run -c .golangci.yaml

lint-fix:
	golangci-lint run -c .golangci.yaml --fix

format:
	golangci-lint fmt -c .golangci.yaml

format-check:
	golangci-lint fmt -c .golangci.yaml --diff
