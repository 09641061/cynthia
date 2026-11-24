.PHONY: help run test vet
.DEFAULT_GOAL := help

help:
	@printf 'Available commands:\n  make run   Start the server\n  make test  Run tests\n  make vet   Run go vet\n'

run:
	go run ./cmd/api

test:
	go test ./...

vet:
	go vet ./...
