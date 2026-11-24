.PHONY: help run test test-race test-integration vet verify check
.DEFAULT_GOAL := help

help:
	@printf 'Available commands:\n  make run              Start the server (not for agent verification)\n  make test             Run unit tests\n  make test-race        Run unit tests with the race detector\n  make test-integration Run PostgreSQL integration tests via Testcontainers\n  make vet              Run go vet\n  make verify           Run native Go guardrails and go vet\n  make check            Run verify and unit tests\n'

run:
	go run ./cmd/api

test:
	go test -count=1 ./...

test-race:
	go test -count=1 -race ./...

test-integration:
	go test -tags=integration -count=1 -timeout 15m ./test/integration/...

vet:
	go vet ./...

verify:
	go test ./test/architecture/...
	go vet ./...

check: verify test
