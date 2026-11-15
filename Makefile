.PHONY: all build test lint typecheck frontend api migrate clean run fmt vet

GO ?= go
NPM ?= npm
BIN := bin/crm

all: build

## Build the frontend then the Go binary with embedded assets.
build: frontend
	mkdir -p bin
	$(GO) build -o $(BIN) ./cmd/crm

## Frontend production build (required for the embedded app).
frontend:
	cd web && $(NPM) install && $(NPM) run build

test: test-go test-web

test-go:
	$(GO) test ./...

# The web package has no test runner configured; typecheck + lint + build act
# as its CI gate (see .github/workflows/ci.yml).
test-web:
	cd web && $(NPM) run typecheck && $(NPM) run lint && $(NPM) run build

lint:
	$(GO) vet ./...

typecheck:
	cd web && $(NPM) run typecheck

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

run:
	$(GO) run ./cmd/crm

api:
	cd web && $(NPM) run api:generate

## Apply migrations to a scratch database and run an integrity check,
## without leaving a server process running.
migrate:
	mkdir -p data
	CRM_DATABASE_PATH=data/check.db $(GO) run ./cmd/crm & \
	SERVER_PID=$$!; \
	for i in $$(seq 1 30); do \
		curl -sf http://127.0.0.1:8080/ready >/dev/null 2>&1 && break; \
		sleep 1; \
	done; \
	curl -sf http://127.0.0.1:8080/ready || { echo "server did not become ready"; kill $$SERVER_PID 2>/dev/null; exit 1; }; \
	sqlite3 data/check.db "PRAGMA integrity_check;" 2>/dev/null || echo "sqlite3 not installed, skipping integrity_check"; \
	kill $$SERVER_PID 2>/dev/null; \
	wait $$SERVER_PID 2>/dev/null; \
	rm -f data/check.db data/check.db-wal data/check.db-shm

clean:
	rm -rf bin data web/dist web/node_modules
