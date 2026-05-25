.PHONY: build up down test test-integration test-e2e lint lint-fix format format-check

build:
	docker compose up --build -d
up:
	docker compose up -d
down:
	docker compose down
test:
	go test ./...

test-integration:
	go test -tags=integration ./test/integration/... -v

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
