# Use fish-compatible commands

APP_NAME=moodle
PKG=github.com/Dubjay18/moodle
DB_DSN?=$(DATABASE_URL)
MIGRATIONS_DIR=./migrations
MIGRATIONS_TABLE?=goose_db_version

.PHONY: build run test tidy migrate-up migrate-down migrate-status migrate-create migrate-reset migrate-drop-table goose

build:
	GO111MODULE=on go build -o bin/$(APP_NAME) ./cmd/api

run: build
	./bin/$(APP_NAME)

test:
	go test ./...

tidy:
	go mod tidy

# Helper to compute DSN for goose: append flags to avoid pgx stmtcache issues
define RESOLVE_DSN
DB_DSN="$${DB_DSN:-$${DATABASE_URL}}"; \
if [ -z "$$DB_DSN" ] && [ -f .env ]; then \
	DB_DSN="$$(grep -E '^DATABASE_URL=' .env | head -n1 | cut -d= -f2- | sed -e 's/^"//' -e 's/"$$//')"; \
fi; \
if [ -z "$$DB_DSN" ]; then echo "DB_DSN/DATABASE_URL not set. Put it in .env or pass DB_DSN=postgres://user:pass@host:5432/dbname?sslmode=require" >&2; exit 2; fi; \
case "$$DB_DSN" in \
*\?*) DSN_FOR_GOOSE="$$DB_DSN&prefer_simple_protocol=true&statement_cache_mode=none" ;; \
*) DSN_FOR_GOOSE="$$DB_DSN?prefer_simple_protocol=true&statement_cache_mode=none" ;; \
esac; \
command -v goose >/dev/null 2>&1 || go install github.com/pressly/goose/v3/cmd/goose@latest;
endef

migrate-up:
	@$(RESOLVE_DSN) \
	goose -table $(MIGRATIONS_TABLE) -dir $(MIGRATIONS_DIR) postgres "$$DSN_FOR_GOOSE" up

migrate-down:
	@$(RESOLVE_DSN) \
	goose -table $(MIGRATIONS_TABLE) -dir $(MIGRATIONS_DIR) postgres "$$DSN_FOR_GOOSE" down

migrate-status:
	@$(RESOLVE_DSN) \
	goose -table $(MIGRATIONS_TABLE) -dir $(MIGRATIONS_DIR) postgres "$$DSN_FOR_GOOSE" status

migrate-create:
	@if test -z "$(name)"; then echo "Usage: make migrate-create name=<name>"; exit 1; fi
	goose -dir $(MIGRATIONS_DIR) create $(name) sql

migrate-reset:
	@$(RESOLVE_DSN) \
	goose -table $(MIGRATIONS_TABLE) -dir $(MIGRATIONS_DIR) postgres "$$DSN_FOR_GOOSE" reset

# Drop the goose migrations table (requires psql). Handy when the table is left in a bad state.
migrate-drop-table:
	@DB_DSN="$${DB_DSN:-$${DATABASE_URL}}"; \
if [ -z "$$DB_DSN" ] && [ -f .env ]; then DB_DSN="$$(grep -E '^DATABASE_URL=' .env | head -n1 | cut -d= -f2- | sed -e 's/^"//' -e 's/"$$//')"; fi; \
if ! command -v psql >/dev/null 2>&1; then echo "psql not found. Install PostgreSQL client tools or drop table $(MIGRATIONS_TABLE) manually." >&2; exit 2; fi; \
psql "$$DB_DSN" -c 'DROP TABLE IF EXISTS $(MIGRATIONS_TABLE);'

goose:
	@command -v goose >/dev/null 2>&1 || go install github.com/pressly/goose/v3/cmd/goose@latest
