# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Shared CRM design for Companies, Contacts, Leads, and Deals, with card/table views, native editors, page exports, and page-scoped summaries.
- Production readiness verification complete
- All 10 levels implemented and verified

### Fixed
- CRM card/board pagination, lead source/priority filtering, mixed-currency summaries, stale search responses, and partial bulk archive reporting.
- SQLite journal_mode persistence issue (fixed by setting PRAGMA after connection)
- Rate limiter fuzz test panic (fixed config initialization)
- Search fuzz test missing sqlite driver import
- gosec SQL concat false positives (all validated by allowlist functions)
- staticcheck unused types and context key collisions
- Memory leak detection test

## [1.0.0] - 2026-11-08

### Added
- Level 1: Foundation & Core Domain (SQLite, migrations, 15 entities, 40+ indexes)
- Level 2: API & HTTP Server (REST, OpenAPI, middleware chain, idempotency)
- Level 3: Frontend SPA (React 19, TypeScript, Vite, Tailwind, PWA)
- Level 4: Authentication & Authorization (Argon2id, session cookies, CSRF, RBAC, API keys)
- Level 5: Background Jobs & Async (worker pool, priority queue, retry/backoff, DLQ)
- Level 6: Integrations & Extensibility (Webhooks, MCP, Email IMAP/SMTP, CalDAV)
- Level 7: Observability (Prometheus metrics, health endpoints, structured logging)
- Level 8: Security Hardening (SSRF protection, rate limiting, secure HTTP client, crypto)
- Level 9: Memory & Performance (MemoryTracker, budget tiers, GOMEMLIMIT, leak detection)
- Level 10: Production Release (multi-arch builds, Docker, K8s, monitoring stack, docs)

### Security
- Fuzz tests for SSRF, rate limiting, search parsing
- govulncheck clean
- gosec: 22 issues (all SQL concat with nolint justification)
- staticcheck clean
