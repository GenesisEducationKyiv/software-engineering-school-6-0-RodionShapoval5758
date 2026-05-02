.PHONY: lint lint-core lint-style lint-all lint-style-fix

lint-core:
	golangci-lint run -c .golangci.core.yaml

lint-style:
	golangci-lint run -c .golangci.style.yaml

lint-style-fix:
	golangci-lint run -c .golangci.style.yaml --fix

lint-all: lint-core lint-style

lint: lint-core
