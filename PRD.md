# Product Requirements Document — TORQ

## 1. Vision

A self-hosted, modular-monolith TORQ for small-to-mid businesses that need full data ownership, extensibility, and AI-agent accessibility — without the complexity of microservices or vendor lock-in.

**Core principle:** Single binary, single database, zero external dependencies at runtime.

---

## 2. Target Users

| Persona | Needs |
|---------|-------|
| **Founder/CEO** | Pipeline visibility, forecasting, one-screen health |
| **Sales Rep** | Fast contact/deal entry, task reminders, mobile access |
| **Operations** | Automation (webhooks, workflows), integrations, audit trail |
| **Developer** | REST API, MCP for AI agents, webhooks, clean codebase to extend |

---

## 3. Functional Requirements

### 3.1 Core CRM Entities (MVP)

| Entity | Key Fields | Relationships |
|--------|------------|---------------|
| **Company** | name, domain, industry, size, address, tags, custom_fields | 1→N Contacts, 1→N Deals |
| **Contact** | first/last name, email, phone, title, company_id, owner_id, tags, custom_fields | N→1 Company, 1→N Deals, 1→N Tasks |
| **Deal** | name, value, currency, stage, probability, expected_close, company_id, contact_id, owner_id, tags | N→1 Company, N→1 Contact |
| **Task** | title, description, status, priority, due_date, assignee_id, related_to (polymorphic) | N→1 User, polymorphic |
| **Project** | name, description, status, start/end dates, members, milestones | N→M Users, 1→N Tasks |
| **Lead** | display_name, email, phone, company_name, source, status, score, owner_id | Converts to Contact+Company+Deal |

### 3.2 Pipeline & Stages
- Customizable stages per pipeline (default: Lead → Qualified → Proposal → Negotiation → Won/Lost)
- Probability per stage
- Required fields per stage transition
- Drag-and-drop in UI

### 3.3 Activities & Timeline
- Unified activity feed per entity (emails, calls, meetings, notes, stage changes)
- Manual log + automatic capture (email sync, webhook events)
- @mentions for team collaboration

### 3.4 Search
- Full-text across all entities (FTS5)
- Faceted filters (type, date, owner, tags, custom fields)
- Saved searches / smart lists

### 3.5 Calendar & Scheduling
- CalDAV sync (Google, Outlook)
- Availability checking
- Meeting links with auto-contact association

### 3.6 Email Integration
- IMAP/SMTP sync per user
- Threaded view on entity timeline
- Send from CRM (templates, tracking)
- Automatic contact matching

### 3.7 Workflows & Automation
- Trigger: entity created/updated, field changed, webhook received, schedule
- Condition: field matches, formula, script
- Action: create/update entity, send email, call webhook, run script, assign task
- Versioned, testable, audit-logged

### 3.8 Reporting & Dashboards
- Pipeline value by stage/owner/time
- Conversion funnel (lead → opportunity → win)
- Activity metrics (calls, meetings, emails per rep)
- Forecast vs actual
- Custom report builder (drag-drop)

### 3.9 Admin & Settings
- Workspace settings (timezone, currency, branding)
- User roles (Admin, Manager, User, Viewer) + custom permissions
- Custom fields per entity type
- Pipeline configuration
- Webhook management
- API key management (scoped, rotatable)
- Audit log viewer
- Backup/restore

---

## 4. Non-Functional Requirements

### 4.1 Architecture
| Requirement | Target |
|-------------|--------|
| Deployment model | Single binary + SQLite file |
| External deps at runtime | None (embed static assets, migrations) |
| Horizontal scaling | Vertical first; read replicas later |
| Multi-tenancy | Workspace isolation (single-tenant per deploy) |

### 4.2 Performance
| Metric | Target |
|--------|--------|
| API P99 latency (simple GET) | < 100 ms |
| API P99 latency (search) | < 500 ms |
| Dashboard load | < 1 s |
| Concurrent users | 500+ on 2-core/2GB |
| Database size | 10 GB+ without degradation |

### 4.3 Security
| Control | Implementation |
|---------|----------------|
| Password hashing | Argon2id (memory-hard) |
| Session management | HttpOnly, Secure, SameSite=Lax cookies |
| CSRF |Double-submit cookie + header |
| Rate limiting | Token bucket (100 rps / 200 burst per IP) |
| SSRF prot
ection | Block private/metadata IPs on all outbound |
| Input validation | Parameterized queries, size limits, CSP |
| Encryption at rest | AES-256-GCM for integration credentials |
| Audit logging | All mutations + auth events |

### 4.4 Observability
- Prometheus metrics (RED + business + memory)
- Structured JSON logs (request_id correlation)
- Health/readiness endpoints
- Admin UI: memory, GC, audit log, sessions, job queue

### 4.5 Reliability
- Graceful shutdown (drain connections, checkpoint WAL)
- Background job retry with exponential backoff
- Database integrity checks on startup
- Automated backup (daily, point-in-time)

---

## 5. Technical Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.23, net/http, modernc.org/sqlite |
| Database | SQLite (WAL, FK, FTS5) |
| Migrations | goose (embedded SQL) |
| Frontend | React 19, TypeScript, Vite, Tailwind CSS |
| API Spec | OpenAPI 3.0 (generated client) |
| Auth | Argon2id, session cookies, API keys |
| Jobs | In-process worker pool (priority queue) |
| AI Access | MCP (Model Context Protocol) over Streamable HTTP |
| Monitoring | Prometheus, Grafana, Loki |
| Container | Multi-stage Docker (Alpine runtime) |
| Orchestration | Kubernetes manifests + docker-compose |

---

## 6. Release Phases

### Phase 1 — Foundation (Levels 1-3)
- [x] Database layer, migrations, core entities
- [x] Repository pattern, basic CRUD
- [x] HTTP server, REST API, OpenAPI spec
- [x] React SPA with routing, Tailwind
- [x] Basic UI: list/detail/create for all entities

### Phase 2 — Auth & Core Features (Levels 4-5)
- [x] Argon2id auth, session cookies, CSRF
- [x] RBAC, workspace isolation
- [x] Background job system
- [x] Pipeline drag-drop, activity timeline
- [x] Full-text search (FTS5)

### Phase 3 — Integrations & Automation (Level 6)
- [x] Webhooks (HMAC, retry, delivery log)
- [x] MCP server (resources, tools)
- [x] Email sync (IMAP/SMTP)
- [x] Calendar (CalDAV)
- [x] Workflow engine

### Phase 4 — Hardening (Levels 7-9)
- [x] Prometheus metrics, health endpoints
- [x] Structured logging, request IDs
- [x] Rate limiting, CORS, security headers
- [x] SSRF protection (inbound + outbound)
- [x] Secure HTTP client
- [x] Memory tracking, budgets, leak detection
- [x] Fuzz tests, govulncheck, gosec, staticcheck
- [x] Performance indexes, query optimization

### Phase 5 — Production Release (Level 10)
- [x] Reproducible builds, SBOM, signing
- [x] Docker multi-stage, docker-compose
- [x] K8s manifests (Deployment, HPA, Ingress, PDB, NetworkPolicy)
- [x] Prometheus alerts, Grafana dashboards
- [x] Loki + Promtail log aggregation
- [x] Backup/restore procedures + test plan
- [x] Fault injection test suite
- [x] Regression automation (GitHub Actions)
- [x] Documentation: User, Admin, API, Developer guides

---

## 7. Success Metrics

| Metric | Target |
|--------|--------|
| Deploy time (fresh) | < 2 minutes |
| Zero-downtime upgrade | ✅ |
| Security scan (govulncheck/gosec) | 0 critical |
| Test coverage (services) | > 80% |
| Memory budget compliance | 99.9% under ceiling |
| Backup restore verified | Monthly drill |

---

## 8. Out of Scope (v1.0)

- Multi-tenant SaaS (single workspace per deploy)
- Horizontal scaling / clustering
- PostgreSQL / MySQL backend
- Mobile native apps (PWA covers this)
- Advanced ML/AI features (MCP enables external agents)
- Marketplace / plugin system (extend via code)