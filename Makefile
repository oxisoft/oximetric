VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
BINARY := oximetric
GOFLAGS := -ldflags "-s -w -X main.Version=$(VERSION)"

.DEFAULT_GOAL := help

.PHONY: help dev seed test test-unit test-integration test-integration-sqlite test-integration-postgres test-coverage test-all build docker-build clean

## dev: run locally with SQLite (for development)
dev:
	OXIMETRIC_ADMIN_USERNAME=admin \
	OXIMETRIC_ADMIN_PASSWORD=admin \
	OXIMETRIC_JWT_SECRET=dev-secret-that-is-at-least-32-characters-long \
	OXIMETRIC_DB_DRIVER=sqlite \
	OXIMETRIC_DB_DSN=./dev.db \
	OXIMETRIC_PORT=6940 \
	go run $(GOFLAGS) ./cmd/oximetric/

## seed: populate dev instance with sample data (requires OXIMETRIC_TOKEN)
seed:
	./scripts/seed.sh

## test-unit: run unit tests
test-unit:
	go test -v -count=1 ./internal/...

## test-integration-sqlite: run integration tests with SQLite in Docker
test-integration-sqlite:
	docker compose -f tests/integration/docker-compose.sqlite.yml up --build --abort-on-container-exit --exit-code-from integration-test
	docker compose -f tests/integration/docker-compose.sqlite.yml down -v

## test-integration-postgres: run integration tests with PostgreSQL in Docker
test-integration-postgres:
	docker compose -f tests/integration/docker-compose.postgres.yml up --build --abort-on-container-exit --exit-code-from integration-test
	docker compose -f tests/integration/docker-compose.postgres.yml down -v

## test-integration: run integration tests with both SQLite and PostgreSQL
test-integration: test-integration-sqlite test-integration-postgres

## test-coverage: run all tests with coverage report
test-coverage:
	go test -coverprofile=coverage.raw -coverpkg=./internal/... -count=1 ./internal/...
	grep -v -e 'internal/storage/postgres.go' -e 'internal/geoip/updater.go' coverage.raw > coverage.out
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo ""
	@echo "Coverage report: coverage.html"

## test: alias for test-unit
test: test-unit

## test-all: run unit tests + integration tests + coverage
test-all: test-unit test-integration test-coverage

## build: build binary with version from latest git tag
build:
	go build $(GOFLAGS) -o $(BINARY) ./cmd/oximetric/

## docker-build: build Docker image with version from latest git tag
docker-build:
	docker build --build-arg VERSION=$(VERSION) -t oximetric:$(VERSION) -t oximetric:latest .

## clean: remove build artifacts
clean:
	rm -f $(BINARY) coverage.out coverage.raw coverage.html dev.db

## help: show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
