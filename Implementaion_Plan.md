# Implementation Plan — TORQ Modular Monolith

## Overview
This document tracks the phased implementation of the CRM from foundation to production release. Each level builds on the previous, delivering incremental value.

---

## Level 1: Foundation & Core Domain ✅ COMPLETE

### 1.1 Database Layer
- [x] SQLite with WAL mode, foreign keys, busy timeout
- [x] Connection pool (max 10 open, 5 idle)
- [x] Migration system (goose) with embedded SQL files
- [x] PRAGMA tuning (cache_size, page_size, mmap_size, synchronous=NORMAL)
- [x] Health check + integrity verification on startup

### 1.2 Domain Models
- [x] Company, Contact, Deal, Task, Project, Lead
- [x] Agent (MCP), API Key, Webhook, Integration
- [x] Workflow, Pipeline, Stage
- [x] Calendar Event, Email, Activity
- [x] User, Session, Audit Log
- [x] Custom Fields (JSONB), Tags (array)

### 1.3 Repository Layer
- [x] Generic `ListOptions` (pagination, sort, filter)
- [x] CRUD for all 15 entities
- [x] Soft delete (archive) pattern
- [x] Polymorphic relations (Task.related_to, Activity.entity)
- [x] Full-text search repositories (FTS5 virtual tables)

### 1.4 Performance Indexes
- [x] 40+ indexes covering all query patterns
- [x] Composite indexes for common filter combinations
- [x] Partial indexes for soft-delete exclusion
- [x] FTS5 content tables for search

---

## Level 2: API & HTTP Server ✅ COMPLETE

### 2.1 HTTP Server
- [x] net/http with middleware chain
- [x] Graceful shutdown (10s drain)
- [x] Request timeouts (read 15s, write 30s, idle 60s)
- [x] Request ID generation + propagation
- [x] Structured JSON logging (slog)

### 2.2 REST API
- [x] OpenAPI 3.0 spec (api/openapi/openapi.yaml)
- [x] Consistent error format (RFC 7807 style)
- [x] Pagination (limit/offset, max 200)
- [x] Filtering (query params → WHERE clauses)
- [x] Sorting (validated allowlist per resource)
- [x] Idempotency keys (24h TTL, Redis-free in-memory)
- [x] Request validation (size limits, content-type)

### 2.3 Middleware Chain (ordered)
1. Recovery (panic → 500)
2. Request ID
3. Logging (access + latency)
4. CORS
5. Security Headers (CSP, HSTS, XFO, etc.)
6. Request Size Limit
7. Rate Limiting
8. Request Timeout
9. Authentication (session + API key)
10. Authorization (RBAC + workspace)

---

## Level 3: Frontend SPA ✅ COMPLETE

### 3.1 Tech Stack
- [x] React 19 + TypeScript (strict)
- [x] Vite 5 (Rolldown) + HMR
- [x] Tailwind CSS 3.4 + custom design tokens
- [x] React Router 7 (data mode ready)
- [x] Lucide React icons
- [x] TanStack Query (server state) — *deferred to v1.1*

### 3.2 Pages & Components
- [x] Layout: Sidebar, Header, Breadcrumbs, Footer
- [x] Dashboard: metrics cards, pipeline chart, recent activity, my tasks
- [x] Companies: list, detail, create/edit, contacts sub-tab, deals sub-tab
- [x] Contacts: list, detail, create/edit, timeline, deals, tasks
- [x] Deals: kanban board, list, detail, stage drag-drop
- [x] Tasks: list (my/team/overdue), kanban, detail, recurrence
- [x] Projects: list, detail (gantt-lite), members, milestones
- [x] Calendar: month/week/day, CalDAV sync status
- [x] Search: global command palette, faceted results
- [x] Settings: profile, workspace, users, pipelines, custom fields, webhooks, API keys
- [x] Admin: health, memory, audit log, sessions, jobs, schema

### 3.3 Build & Deploy
- [x] TypeScript strict mode, no `any`
- [x] Production build: hashed assets, code splitting
- [x] Embedded in Go binary via //go:embed
- [x] PWA manifest + service worker

---

## Level 4: Authentication & Authorization ✅ COMPLETE

### 4.1 Authentication
- [x] Argon2id password hashing (memory=64MB, iterations=3, parallelism=4)
- [x] Session cookies: HttpOnly, Secure, SameSite=Lax, 24h max age
- [x] CSRF: double-submit cookie (csrf_token) + header (X-CSRF-Token)
- [x] Owner bootstrap (first admin creation)
- [x] Login / Logout / Me endpoints
- [x] Session validation + revocation

### 4.2 API Keys
- [x] Scoped keys (read, write, admin, webhook)
- [x] Prefix identification (sk_live_, sk_test_)
- [x] Expiration + rotation
- [x] Last used tracking

### 4.3 Authorization (RBAC)
- [x] Roles: Admin, Manager, User, Viewer
- [x] Permissions matrix (resource × action)
- [x] Workspace isolation (all queries scoped)
- [x] Resource-level checks (owner, team, public)

---

## Level 5: Background Jobs & Async ✅ COMPLETE

### 5.1 Job System
- [x] Worker pool (configurable workers, default 8)
- [x] Priority queue (High/Normal/Low)
- [x] Job persistence in SQLite (jobs table)
- [x] Retry with exponential backoff (max 3, then DLQ)
- [x] Dead letter queue with admin retry
- [x] Job status: pending → running → completed/failed
- [x] Timeout enforcement (30s default)

### 5.2 Built-in Job Types
- [x] Email sending (templates, attachments)
- [x] Webhook delivery (HMAC, retry, logging)
- [x] Search index updates (FTS5)
- [x] Backup (daily, point-in-time)
- [x] Cleanup (expired sessions, old audit logs, DLQ)

---

## Level 6: Integrations & Extensibility ✅ COMPLETE

### 6.1 Webhooks
- [x] Event registry (20+ event types)
- [x] HMAC-SHA256 signature verification
- [x] Retry with exponential backoff (max 5)
- [x] Delivery log (status, response, latency)
- [x] Test endpoint (send sample payload)
- [x] Disable on repeated failures

### 6.2 MCP (Model Context Protocol)
- [x] Streamable HTTP transport
- [x] Resources: entities, search, dashboard
- [x] Tools: create/read/update/delete for all entities
- [x] Agent authentication (API key scopes)
- [x] Capability negotiation

### 6.3 Third-Party Integrations
- [x] Email: IMAP sync + SMTP send (per-user)
- [x] Calendar: CalDAV read/write
- [x] Credential encryption: AES-256-GCM (per-workspace key)

---

## Level 7: Observability ✅ COMPLETE

### 7.1 Metrics (Prometheus)
- [x] HTTP: requests_total, duration_seconds, in_flight
- [x] Business: entities_created, deals_won/lost, pipeline_value
- [x] Database: connections, wait_count, query_duration
- [x] Jobs: queued, running, completed, failed, duration
- [x] Memory: RSS, heap, goroutines, GC pauses, budget level
- [x] Security: rate_limit_exceeded, ssrf_blocked, auth_failed

### 7.2 Endpoints
- [x] GET /health — liveness (always 200 if process alive)
- [x] GET /ready — readiness (DB + deps)
- [x] GET /metrics — Prometheus scrape
- [x] Admin: /admin/memory, /admin/gc, /admin/audit-logs, /admin/sessions, /admin/jobs, /admin/schema

### 7.3 Logging
- [x] JSON format with request_id
- [x] Levels: debug, info, warn, error
- [x] Structured fields (no string interpolation)
- [x] Correlation across HTTP → job → DB

---

## Level 8: Security Hardening ✅ COMPLETE

### 8.1 Input & Output
- [x] Request body limit: 10 MB (configurable)
- [x] Upload limit: 50 MB
- [x] Parameterized queries everywhere (sqlc + manual)
- [x] CSP: restrictive, nonce-based for inline scripts
- [x] Security headers: XFO, HSTS, Referrer-Policy, Permissions-Policy

### 8.2 Network Security
- [x] CORS: allowlist origins, credentials support
- [x] Rate limiting: token bucket per IP (100 rps / 200 burst)
- [x] SSRF protection (middleware):
  - Block RFC1918, loopback, link-local, metadata (169.254.169.254)
  - DNS rebinding protection (resolve at request time)
  - Allowlist for internal services
- [x] Request timeout: 30s (configurable)

### 8.3 Secure HTTP Client (outbound)
- [x] Custom dialer with IP validation
- [x] Redirect validation (same SSRF rules)
- [x] Response size limit (10 MB)
- [x] TLS verification enforced
- [x] Timeout enforcement (30s)

### 8.4 Cryptography
- [x] Argon2id for passwords
- [x] AES-256-GCM for integration credentials
- [x] crypto/rand for tokens, API keys, CSRF
- [x] Constant-time comparisons (subtle.ConstantTimeCompare)

### 8.5 Security Testing
- [x] Fuzz: SSRF URL validation (14 seeds)
- [x] Fuzz: Rate limit key extraction
- [x] Fuzz: Search query parsing (14 seeds)
- [x] govulncheck: 0 vulnerabilities in called code
- [x] gosec: 22 issues (all SQL concat, nolint with justification)
- [x] staticcheck: clean

---

## Level 9: Memory & Performance ✅ COMPLETE

### 9.1 Memory Tracking
- [x] Periodic sampling (10s interval): RSS, Heap, Goroutines, GC
- [x] Budget tiers:
  - Normal: < 250 MiB
  - Heavy: 250–400 MiB (warn)
  - Ceiling: 400–500 MiB (alert, restrict allocations)
  - Critical: > 500 MiB (force GC, reject non-critical)
- [x] /admin/memory — current status + history
- [x] /admin/gc — forced GC endpoint
- [x] Leak detection test (goroutine count stable over time)

### 9.2 Performance Optimization
- [x] Database indexes for all query patterns (Level 1)
- [x] Prepared statement reuse (sqlc + manual)
- [x] Connection pooling tuned
- [x] FTS5 for search (no LIKE scans)
- [x] Frontend: code splitting, lazy routes, hashed assets
- [x] GOMEMLIMIT=400MiB (matches container limit)

### 9.3 Benchmarks
- [x] Memory budget tests (unit)
- [x] GC pressure tests
- [x] Leak detection (1hr soak)
- [x] Load test script (hey) for CI

---

## Level 10: Production Release ✅ COMPLETE

### 10.1 Build & Release
- [x] `scripts/release.sh` — reproducible, multi-arch (amd64/arm64)
- [x] Version embedding (git SHA, build ID, timestamp)
- [x] Stripped binaries (-s -w)
- [x] Frontend production build
- [x] SBOM (Syft: SPDX + CycloneDX)
- [x] SHA256 checksums
- [x] Cosign signing (optional)
- [x] Release manifest JSON

### 10.2 Container & Orchestration
- [x] Dockerfile: 3-stage (frontend → Go builder → Alpine runtime)
- [x] Non-root user (UID 1000), read-only rootfs
- [x] docker-compose.yml: CRM + nginx + Prometheus + Grafana + Loki + Promtail
- [x] K8s manifests:
  - Namespace, ConfigMap, Secret
  - PVC (10Gi, fast-ssd)
  - Deployment (rolling, maxSurge=1, maxUnavailable=0)
  - Service (ClusterIP)
  - Ingress (TLS, rate-limit, body-size)
  - HPA (CPU 70%, Memory 80%, min=2, max=10)
  - RBAC (ServiceAccount, Role, RoleBinding)
  - PodDisruptionBudget (minAvailable=1)
  - NetworkPolicy (ingress from nginx, egress DNS + monitoring)

### 10.3 Monitoring Stack
- [x] Prometheus config (service discovery, relabeling)
- [x] Alert rules (25+ rules, 7 categories):
  - Application (down, errors, latency, rate)
  - Memory (budget, goroutines, GC)
  - Database (connections, locks)
  - Jobs (backlog, failure rate)
  - Infrastructure (disk, CPU, memory limit)
  - Security (rate limit, SSRF, auth)
  - Backup (freshness, integrity, size)
- [x] Grafana dashboard provisioning
- [x] Loki + Promtail for log aggregation

### 10.4 Operations Documentation
- [x] docs/user-guide.md
- [x] docs/admin-guide.md
- [x] docs/api-docs.md
- [x] docs/developer-guide.md
- [x] BACKUP_RESTORE.md + BACKUP_RESTORE_TEST.md
- [x] FAULT_INJECTION_TESTS.md (10 scenarios)
- [x] REGRESSION_AUTOMATION.md (GitHub Actions)
- [x] THREAT_MODEL.md (11 threat categories)
- [x] PRODUCTION_READINESS_CHECKLIST.md

### 10.5 Disaster Recovery
- [x] Backup strategy (daily automated, manual snapshot)
- [x] Restore procedure (documented, tested)
- [x] WAL checkpoint management
- [x] Integrity verification (PRAGMA integrity_check)
- [x] DR checklist: RPO < 1hr, RTO < 30min

---

## Dependencies & Ordering

```
Level 1 (DB + Models) 
    → Level 2 (API + Server) 
        → Level 3 (Frontend) 
        → Level 4 (Auth) 
            → Level 5 (Jobs) 
                → Level 6 (Integrations) 
                    → Level 7 (Observability) 
                        → Level 8 (Security) 
                            → Level 9 (Memory/Perf) 
                                → Level 10 (Release)
```

Each level's tests must pass before advancing. CI runs full suite on every PR.

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| SQLite write throughput bottleneck | Medium | High | Vertical scaling first; plan PG migration at 1000+ users |
| Single-instance SPOF | Low | High | PDB + fast restart; blue-green deploy |
| Memory leak in long-running process | Low | High | MemoryTracker + leak tests + GOMEMLIMIT |
| SSRF bypass via DNS rebinding | Low | Critical | Resolve at request time, short TTL cache |
| Supply chain vulnerability | Medium | High | govulncheck in CI, pinned dependencies, SBOM |

---

## Definition of Done per Level

1. All unit tests pass (`go test ./...`)
2. Fuzz tests pass (security-critical functions)
3. Integration tests pass (DB-backed handlers)
4. govulncheck clean
4. gosec clean (or justified nolint)
5. staticcheck clean
6. Build succeeds (Go + frontend)
7. Documentation updated
8. CHANGELOG entry