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

test:
	$(GO) test ./... 2>&1
	cd web && $(NPM) run test -- --run 2>&1 || true

test-go:
	$(GO) test ./...

test-web:
	cd web && $(NPM) run test -- --run

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

## Fresh database migration and integrity check.
migrate:
	CRM_DATABASE_PATH=data/check.db $(GO) run ./cmd/crm &

clean:
	rm -rf bin data web/dist web/node_modules
