.PHONY: run run-local build migrate-up migrate-down tidy deps dev test test-cover

# Start local Postgres + Redis (requires Docker)
deps:
	docker compose up -d postgres redis

# Start deps then run the gateway
dev: deps
	@sleep 2
	$(MAKE) run

run:
	go run ./cmd/server

# Run with embedded in-memory databases (no Docker required)
run-local:
	DATABASES_MEMORY=LOCAL go run ./cmd/server

build:
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

tidy:
	go mod tidy

migrate-up:
	@echo "Run migrations via the server (auto-applied on startup)"

migrate-down:
	@echo "Manual rollback: apply migrations/*.down.sql against your database"

seed-org-users:
	go run ./cmd/seed-org-users $(ARGS)

test:
	go test ./... -v -count=1

test-cover:
	go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out
