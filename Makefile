.PHONY: build up down up-metrics up-logging up-full observability-setup \
        test test-integration test-e2e lint lint-fix format format-check

build:
	docker compose up --build -d

up:
	docker compose up -d

down:
	docker compose down

up-metrics:
	docker compose --profile metrics up -d --wait

up-logging:
	docker compose --profile logging up -d --wait

up-full:
	docker compose --profile metrics --profile logging up -d --wait

observability-setup:
	docker compose --profile setup run --rm kibana-setup

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
