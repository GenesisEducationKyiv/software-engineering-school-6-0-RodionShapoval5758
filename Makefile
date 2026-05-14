.PHONY: lint lint-fix lint-all format format-check

lint:
	golangci-lint run -c .golangci.yaml

lint-fix:
	golangci-lint run -c .golangci.yaml --fix

format:
	golangci-lint fmt -c .golangci.yaml

format-check:
	golangci-lint fmt -c .golangci.yaml --diff

lint-all: lint
