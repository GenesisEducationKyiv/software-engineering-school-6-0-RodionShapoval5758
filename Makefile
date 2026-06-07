.PHONY: build up down down-all up-metrics up-logging up-full observability-setup \
        test test-integration test-e2e lint lint-fix format format-check \
        k6-smoke k6-smoke-ci k6-load k6-stress k6-spike k6-soak k6-breakpoint k6-write k6-journey k6-suite k6-clean

build:
	docker compose up --build -d

up:
	docker compose up -d

down:
	docker compose down

down-all:
	docker compose --profile metrics --profile logging --profile setup down

up-metrics:
	docker compose --profile metrics up -d --wait

up-logging:
	docker compose --profile logging up -d --wait

up-full:
	docker compose --profile metrics --profile logging up -d --wait

observability-setup:
	docker compose --profile logging --profile setup run --rm kibana-setup

test:
	go test ./...
	go test -C contract ./...
	go test -C services/notification ./...

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

K6_PROM_URL    ?= http://localhost:9090/api/v1/write
K6_BASE_URL    ?= http://localhost
K6_API_KEY     ?= genesis-summer-school
K6_TREND_STATS ?= p(95),p(99),avg,min,max

define run_k6
K6_PROMETHEUS_RW_SERVER_URL=$(K6_PROM_URL) K6_PROMETHEUS_RW_TREND_STATS="$(K6_TREND_STATS)" \
k6 run -o experimental-prometheus-rw --tag testid=$(1) -e BASE_URL=$(K6_BASE_URL) -e API_KEY=$(K6_API_KEY) $(2)
endef

k6-smoke:
	$(call run_k6,smoke,k6/smoke.js)

k6-smoke-ci:
	k6 run -e BASE_URL=$(K6_BASE_URL) -e API_KEY=$(K6_API_KEY) k6/smoke.js

k6-load:
	$(call run_k6,read-load,k6/read-load.js)

k6-stress:
	$(call run_k6,read-stress,k6/read-stress.js)

k6-spike:
	$(call run_k6,read-spike,k6/read-spike.js)

k6-soak:
	$(call run_k6,read-soak,k6/read-soak.js)

k6-breakpoint:
	$(call run_k6,breakpoint,k6/breakpoint.js)

k6-write:
	$(call run_k6,write-load,k6/write-load.js)

k6-journey:
	K6_PROMETHEUS_RW_SERVER_URL=$(K6_PROM_URL) K6_PROMETHEUS_RW_TREND_STATS="$(K6_TREND_STATS)" \
	k6 run -o experimental-prometheus-rw --tag testid=journey \
	  -e BASE_URL=$(K6_BASE_URL) -e API_KEY=$(K6_API_KEY) -e MAILPIT_URL=http://localhost:8025 \
	  k6/journey.js

k6-suite: k6-smoke k6-load k6-stress k6-spike

k6-clean:
	docker compose exec -T postgres_db psql -U postgres -d github_release_notifications \
	  -c "DELETE FROM subscriptions WHERE email LIKE '%@loadtest.local';"
