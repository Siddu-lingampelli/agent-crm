# TORQ Developer Guide

## Overview

This guide covers setting up a development environment, project structure, coding standards, and contribution workflow for TORQ.

## Prerequisites

- **Go**: 1.23+
- **Node.js**: 20+
- **npm**: 10+
- **SQLite**: 3.45+ (with FTS5)
- **Git**: 2.40+
- **Docker**: 24+ (optional, for containerized dev)
- **Make**: 4.4+ (optional)

## Quick Start

```bash
# Clone repository
git clone https://github.com/yourorg/crm.git
cd crm

# Install Go dependencies
go mod download

# Install frontend dependencies
cd web && npm ci && cd ..

# Create development config
cp config.example.yaml config.yaml  # if exists, or use env vars

# Run database migrations
go run ./cmd/crm migrate

# Start development servers (two terminals)

# Terminal 1: Backend
go run ./cmd/crm

# Terminal 2: Frontend (with HMR)
cd web && npm run dev
```

## Project Structure

```
crm/
├── cmd/
│   └── crm/                 # Main entry point
├── internal/                # Private application code
│   ├── app/                 # Application wiring
│   ├── auth/                # Authentication & authorization
│   ├── config/              # Configuration management
│   ├── db/                  # Database layer
│   ├── httpserver/          # HTTP handlers & routing
│   ├── jobs/                # Background job processing
│   ├── logging/             # Structured logging
│   ├── memtrack/            # Memory tracking & budgets
│   ├── mcp/                 # Model Context Protocol
│   ├── model/               # Domain models
│   ├── permissions/         # RBAC permissions
│   ├── repository/          # Data access layer
│   ├── security/            # Security middleware
│   ├── service/             # Business logic services
│   ├── uuid/                # UUID generation
│   └── jsonutil/            # JSON utilities
├── db/
│   └── migrations/          # SQL migrations (goose)
├── web/                     # React frontend
│   ├── src/
│   │   ├── components/      # React components
│   │   ├── pages/           # Page components
│   │   ├── hooks/           # Custom hooks
│   │   ├── services/        # API clients
│   │   ├── store/           # State management
│   │   ├── types/           # TypeScript types
│   │   └── utils/           # Utilities
│   ├── public/              # Static assets
│   └── dist/                # Production build output
├── api/
│   └── openapi/             # OpenAPI specification
├── k8s/                     # Kubernetes manifests
├── monitoring/              # Prometheus/Grafana configs
├── docs/                    # Documentation
├── scripts/                 # Build & release scripts
├── Dockerfile               # Multi-stage Docker build
├── docker-compose.yml       # Local development stack
├── go.mod / go.sum          # Go dependencies
├── Makefile                 # Build automation
└── README.md
```

## Development Workflow

### Branching Strategy

- `main` - Production-ready code
- `develop` - Integration branch (optional)
- `feature/*` - New features
- `fix/*` - Bug fixes
- `release/*` - Release preparation
- `hotfix/*` - Emergency production fixes

### Commit Messages

Follow Conventional Commits:

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`, `build`, `ci`

Examples:
```
feat(contacts): add bulk import endpoint
fix(auth): handle expired token refresh
docs(api): update webhook payload examples
```

### Pull Request Process

1. Create feature branch from `main`
2. Make changes with tests
3. Run linting and tests locally
4. Open PR with description
5. CI runs: tests, security scans, build
6. Code review (1+ approvals)
7. Squash and merge to `main`

## Backend Development

### Running Tests

```bash
# All tests
go test ./...

# With race detector (requires CGO)
CGO_ENABLED=1 go test -race ./...

# Specific package
go test -v ./internal/auth/...

# Run specific test
go test -v -run TestOwnerBootstrap ./internal/auth/...

# Generate coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Code Generation

```bash
# Generate SQL code (sqlc)
sqlc generate

# Generate OpenAPI client (oapi-codegen)
oapi-codegen -generate types,client -package api api/openapi/openapi.yaml > internal/api/client.go

# Generate mocks (mockery)
mockery --all --output internal/mocks
```

### Database Migrations

```bash
# Create new migration
goose -dir db/migrations create add_user_preferences sql

# Run migrations
goose -dir db/migrations sqlite3 data/crm.db up

# Rollback
goose -dir db/migrations sqlite3 data/crm.db down

# Status
goose -dir db/migrations sqlite3 data/crm.db status
```

### Adding a New Resource

1. **Model** - Add to `internal/model/model.go`
2. **Repository** - Create `internal/repository/{resource}.go`
3. **Service** - Create `internal/service/{resource}/service.go`
4. **Handlers** - Create `internal/httpserver/{resource}_handlers.go`
5. **Routes** - Register in `internal/httpserver/server.go`
6. **Tests** - Add `*_test.go` files
7. **Migrations** - Add SQL migration
8. **OpenAPI** - Update `api/openapi/openapi.yaml`

### Security Middleware

All handlers are wrapped with security middleware in `internal/app/app.go`:

```go
// Middleware chain (order matters):
// 1. Recovery panic handler
// 2. Request ID
// 3. Logging
// 4. CORS
// 5. Security headers
// 6. Request size limit
// 7. Rate limiting
// 8. Timeout
// 9. Authentication
// 10. Authorization
```

### Background Jobs

Jobs are processed by a worker pool in `internal/jobs/jobs.go`:

```go
// Submit job
jobPool.Submit(ctx, jobs.Job{
    Type: "send_email",
    Payload: map[string]any{"to": "user@example.com", "template": "welcome"},
    Priority: jobs.PriorityNormal,
})

// Register handler
jobPool.RegisterHandler("send_email", func(ctx context.Context, payload map[string]any) error {
    // Process job
    return nil
})
```

### Memory Tracking

Monitor memory usage with `internal/memtrack/memtrack.go`:

```go
// Get current status
stats := memTracker.Stats()
fmt.Printf("RSS: %d MB\n", stats.RSS/1024/1024)

// Force GC
memTracker.ForceGC()

// Subscribe to budget alerts
memTracker.Subscribe(func(status memtrack.BudgetStatus) {
    if status.Level >= memtrack.Heavy {
        log.Warn("Memory budget exceeded", "level", status.Level)
    }
})
```

## Frontend Development

### Project Setup

```bash
cd web
npm ci          # Install dependencies
npm run dev     # Start Vite dev server (port 5173)
npm run build   # Production build
npm run typecheck  # TypeScript check
npm run lint    # Oxlint
```

### Key Technologies

- **React 19** - UI library
- **TypeScript** - Type safety
- **Vite** - Build tool & dev server
- **Tailwind CSS** - Styling
- **React Router v7** - Routing
- **Lucide React** - Icons
- **TanStack Query** - Server state (if added)
- **Zod** - Validation (if added)

### Project Structure

```
web/src/
├── main.tsx                 # App entry point
├── App.tsx                  # Root component with routes
├── components/
│   ├── ui/                  # Base UI components (Button, Input, Modal, etc.)
│   ├── layout/              # Layout components (Sidebar, Header, Footer)
│   ├── forms/               # Form components
│   └── data-display/        # Tables, Cards, Lists
├── pages/
│   ├── Dashboard.tsx
│   ├── Contacts/
│   ├── Companies/
│   ├── Deals/
│   ├── Tasks/
│   ├── Projects/
│   ├── Calendar/
│   ├── Settings/
│   └── Auth/
├── hooks/
│   ├── useAuth.ts
│   ├── useApi.ts
│   └── useDebounce.ts
├── services/
│   ├── api.ts               # Axios/fetch wrapper
│   ├── auth.ts
│   └── websocket.ts
├── store/
│   ├── authStore.ts         # Zustand/Redux store
│   └── uiStore.ts
├── types/
│   ├── api.ts               # API response types
│   └── models.ts            # Domain models
└── utils/
    ├── formatters.ts
    ├── validators.ts
    └── constants.ts
```

### API Client

```typescript
// web/src/services/api.ts
import axios from 'axios';

const api = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true,  // For session cookies
});

// Request interceptor for CSRF
api.interceptors.request.use((config) => {
  const csrfToken = document.cookie
    .split('; ')
    .find(row => row.startsWith('csrf_token='))
    ?.split('=')[1];
  if (csrfToken && ['post', 'patch', 'delete'].includes(config.method || '')) {
    config.headers['X-CSRF-Token'] = csrfToken;
  }
  return config;
});

// Response interceptor for auth errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Redirect to login
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export default api;
```

### Adding a New Page

1. Create page component in `web/src/pages/{Resource}/`
2. Add route in `web/src/App.tsx`
3. Add navigation link in `web/src/components/layout/Sidebar.tsx`
4. Create API service calls in `web/src/services/`
5. Add TypeScript types in `web/src/types/`

## Coding Standards

### Go

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` / `gofumpt` for formatting
- Run `staticcheck` and `golangci-lint`
- Prefer standard library over dependencies
- Error handling: wrap with context, don't ignore
- Interfaces: small, focused, defined by consumer
- Tests: table-driven, parallel when possible

### TypeScript/React

- Strict TypeScript config
- ESLint + Prettier (or Oxlint + dprint)
- Functional components with hooks
- Colocate components with their hooks/styles
- Use `React.FC` sparingly; prefer inline types
- Custom hooks for reusable logic
- Tailwind for styling; avoid custom CSS

### Database

- Use parameterized queries always
- Index foreign keys and common query columns
- Use transactions for multi-step operations
- Migrations: up + down, idempotent when possible
- FTS5 for searchable text columns

### API Design

- RESTful with plural nouns: `/api/v1/contacts`
- Nested resources: `/api/v1/companies/{id}/contacts`
- Actions as sub-resources: `/api/v1/deals/{id}/move`
- Consistent error format
- Pagination with `limit`/`offset`
- Filtering with query params
- Versioning in URL path

## Testing Strategy

### Unit Tests

- Test business logic in services
- Mock repositories with interfaces
- Test edge cases and error paths
- Target: >80% coverage for services

### Integration Tests

- Test HTTP handlers with real database
- Use test database (temp file)
- Test authentication flows
- Test middleware chain

### Fuzz Tests

- Security-critical functions (SSRF, rate limiting)
- Search query parsing
- Input validation

### Load Tests

- Run with `hey` or `k6`
- Target endpoints: `/health`, `/api/v1/companies`, `/api/v1/search`
- Baseline: P99 < 100ms for simple GET, < 500ms for search

## Debugging

### Backend

```bash
# Delve debugger
dlv debug ./cmd/crm

# Attach to running process
dlv attach <pid>

# VS Code launch.json
{
  "type": "go",
  "request": "launch",
  "name": "Debug TORQ",
  "program": "${workspaceFolder}/cmd/crm",
  "env": {
    "CRM_DATABASE_PATH": "data/crm.db"
  }
}
```

### Frontend

- React DevTools browser extension
- Vite HMR for instant updates
- Console logging with `debug` package
- Network tab for API calls

### Database

```bash
# SQLite CLI
sqlite3 data/crm.db

# Useful commands
.tables
.schema contacts
PRAGMA index_list(contacts);
EXPLAIN QUERY PLAN SELECT * FROM contacts WHERE email = 'test@example.com';
```

## Performance Profiling

### CPU Profile

```bash
go test -cpuprofile=cpu.prof -bench=. ./internal/service/...
go tool pprof cpu.prof
```

### Memory Profile

```bash
go test -memprofile=mem.prof -bench=. ./internal/service/...
go tool pprof mem.prof
```

### HTTP Profile

```bash
# Add to main.go for runtime profiling
import _ "net/http/pprof"
# Then: go tool pprof http://localhost:6060/debug/pprof/profile
```

## Release Process

### Versioning

Semantic Versioning: `MAJOR.MINOR.PATCH`
- MAJOR: Breaking changes
- MINOR: New features (backward compatible)
- PATCH: Bug fixes

### Release Checklist

1. Update version in `internal/config/config.go`
2. Update CHANGELOG.md
3. Create release branch: `release/v1.2.0`
4. Run full test suite: `go test -race ./...`
5. Run security scans: `gosec`, `govulncheck`, `staticcheck`
6. Build release artifacts: `./scripts/release.sh`
7. Create Git tag: `git tag v1.2.0`
8. Push tag: `git push origin v1.2.0`
9. GitHub Actions builds and publishes release
10. Deploy to staging
11. Smoke tests on staging
12. Deploy to production
13. Monitor metrics
14. Announce release

### Build Script

```bash
# Local build
./scripts/release.sh

# Environment variables
VERSION=v1.2.0
BUILD_ID=$(date +%Y%m%d%H%M%S)
GIT_SHA=$(git rev-parse HEAD)
```

## Troubleshooting Development

### Common Issues

**Port already in use**
```bash
# Find and kill process
lsof -ti:8080 | xargs kill -9
```

**Database locked**
```bash
# Check for stale processes
ps aux | grep crm
# Kill and remove -wal/-shm files
rm data/crm.db-wal data/crm.db-shm
```

**Frontend not proxying to backend**
```bash
# Check vite.config.ts proxy config
# Ensure backend running on port 8080
```

**Module not found**
```bash
go mod tidy
go mod download
```

**TypeScript errors**
```bash
cd web && npm run typecheck
# Fix errors, often missing types or API mismatch
```

## Useful Commands

```bash
# Format all Go code
gofumpt -w .

# Lint Go code
golangci-lint run

# Lint frontend
cd web && npm run lint

# Type check frontend
cd web && npm run typecheck

# Run all tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Generate mocks
mockery --all

# Update dependencies
go get -u ./...
go mod tidy

# Docker build
docker build -t crm:dev .

# Run with docker-compose
docker-compose up -d
```

## Contributing

1. Fork repository
2. Create feature branch
3. Make changes with tests
4. Ensure CI passes
5. Open PR with clear description
6. Address review feedback
7. Merge after approval

## Resources

- [Go Documentation](https://go.dev/doc/)
- [React Documentation](https://react.dev/)
- [TypeScript Handbook](https://www.typescriptlang.org/docs/)
- [Tailwind CSS](https://tailwindcss.com/docs)
- [Vite Guide](https://vitejs.dev/guide/)
- [SQLite Documentation](https://www.sqlite.org/docs.html)
- [sqlc Documentation](https://sqlc.dev/docs/)
- [goose Migrations](https://github.com/pressly/goose)

---

*For user documentation, see [User Guide](user-guide.md). For admin documentation, see [Admin Guide](admin-guide.md). For API reference, see [API Documentation](api-docs.md).*