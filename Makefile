.PHONY: test test-integration lint-fix lint-all format format-check

test:
	go test ./...

test-integration:
	docker compose up -d --wait postgres_db mailpit
	docker compose exec -T postgres_db psql -U postgres -c "CREATE DATABASE github_release_notifications_test;" 2>/dev/null || true
	TEST_DATABASE_URL="postgres://postgres:password@localhost:5432/github_release_notifications_test?sslmode=disable" \
		go test -tags=integration ./test/integration/... -v
	docker compose down

lint:
	golangci-lint run -c .golangci.yaml

lint-fix:
	golangci-lint run -c .golangci.yaml --fix

format:
	golangci-lint fmt -c .golangci.yaml

format-check:
	golangci-lint fmt -c .golangci.yaml --diff
