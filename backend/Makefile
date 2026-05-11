ENV_FILE ?= .env
MIGRATIONS_DIR ?= ./migrations
DB_DRIVER ?= postgres
GOOSE ?= goose

.PHONY: run build test test-integration lint migrate-up migrate-down migrate-status migrate-reset migrate-create

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test -race ./...

# Needs a PostgreSQL server, e.g. `docker compose up -d postgres`.
test-integration:
	TEST_DATABASE_URL=$${TEST_DATABASE_URL:-postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable} \
	go test -race -count=1 ./...

lint:
	golangci-lint run ./...

migrate-up:
	set -a; [ -f "$(ENV_FILE)" ] && . "$(ENV_FILE)"; set +a; \
	$(GOOSE) -dir "$(MIGRATIONS_DIR)" "$(DB_DRIVER)" "$$DATABASE_URL" up

migrate-down:
	set -a; [ -f "$(ENV_FILE)" ] && . "$(ENV_FILE)"; set +a; \
	$(GOOSE) -dir "$(MIGRATIONS_DIR)" "$(DB_DRIVER)" "$$DATABASE_URL" down

migrate-status:
	set -a; [ -f "$(ENV_FILE)" ] && . "$(ENV_FILE)"; set +a; \
	$(GOOSE) -dir "$(MIGRATIONS_DIR)" "$(DB_DRIVER)" "$$DATABASE_URL" status

migrate-reset:
	set -a; [ -f "$(ENV_FILE)" ] && . "$(ENV_FILE)"; set +a; \
	$(GOOSE) -dir "$(MIGRATIONS_DIR)" "$(DB_DRIVER)" "$$DATABASE_URL" reset

migrate-create:
ifndef name
	$(error Usage: make migrate-create name=add_something)
endif
	$(GOOSE) -dir "$(MIGRATIONS_DIR)" create "$(name)" sql
