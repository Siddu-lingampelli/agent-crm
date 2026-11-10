# CRM security and bug review — 10 November 2026

The source was reviewed and repaired across authentication, REST/MCP routing,
SQLite, repositories, background work, the React client and deployment files.
This is a code review with local regression tests, not a certification that every
possible defect has been eliminated.

## Security fixes

| Finding | Change |
| --- | --- |
| Public MCP endpoint and hardcoded privileged identity | Require a valid API key or an authenticated owner session; pass the verified actor, workspace and scopes into each tool call. |
| Revoked, expired or disabled agent credentials | Check the key and its agent on each request; intersect their scopes and enforce configured key limits. |
| Owner pages denied valid operations | Supply the complete owner scope vocabulary for this single-owner application. |
| Cross-workspace references and search access | Validate linked records against their workspace; limit search to entities the actor can read. |
| Agent approval bypass for its own email drafts | Require human approval for agent sends and prevent agents from approving drafts. |
| Stored XSS in HTML email preview | Render HTML inside an isolated sandboxed iframe with a restrictive document CSP. |
| Initial-request SSRF and DNS rebinding | Validate resolved addresses inside the dialer, connect to that exact address and validate redirects. Block private and special-use destinations unless explicitly allowed. |
| Unenforced outbound response limit | Bound streamed response reads and reject oversized declared bodies. |
| Optional unverified outbound TLS | Enforce certificate verification and TLS 1.2 or newer. |
| Cookie options ignored; raw session tokens stored | Honor configured cookie settings and lifetime; issue 256-bit random tokens and store SHA-256 digests. Remove expired sessions during login. |
| Missing browser CSRF enforcement | Apply Go's Origin/Fetch Metadata protection to writes, including login/logout, with explicit trusted development origins. |
| Protections applied only to public routes | Apply shared security middleware before routing and authentication. Protected CORS preflights now work. |
| Spoofable rate-limit IPs and unbounded limiter map | Use the direct peer address, cap limiter storage and separately throttle expensive logins. |
| Concurrent password hashing and malformed PHC parameters | Bound concurrent derivations and validate algorithm version, resource parameters, salt and key sizes. |
| ResponseWriter race on timeout | Keep response writes on the serving goroutine and propagate request cancellation. Apply configured server socket timeouts. |
| SMTP header injection and unbounded network waits | Validate headers and recipients, encode subjects, require TLS and set network/cancellation deadlines. |
| Duplicate email sends | Atomically queue and claim each message; isolate worker-owned message copies. |
| Integration credentials stored directly | Encrypt new credential writes with AES-256-GCM using the external `CRM_ENCRYPTION_KEY`. Do not expose credentials or webhook signing secrets in JSON responses. |
| Static-file boundary | Serve the production SPA through an `os.Root` filesystem; keep unknown API paths out of SPA fallback. |

## Runtime and data fixes

- Correct SQLite URI construction: PRAGMAs are query parameters instead of part
  of a filename. Apply foreign keys and other connection settings to every
  pooled connection. Reuse an unambiguous legacy filename to preserve access to
  existing data; reject ambiguous legacy candidates.
- Validate PRAGMA values, preserve mandatory WAL mode, and synchronize database
  closure. Readiness now pings the database.
- Migration `00008_repair_fts_rowids.sql` rebuilds search indexes and fixes
  insert/update triggers to preserve entity rowids. Search excludes archives
  and quotes literal search terms.
- Migration `00009_soft_archive.sql` makes calendar, workflow, integration and
  webhook archives persistent. It also repairs the single owner's seeded
  pipeline workspace; new owner bootstrap assigns that pipeline correctly.
- Replace the worker-pool stub with bounded workers, queue admission control,
  cancellation, draining and counters. Close jobs before the database.
- Validate workflow delay types and duration bounds, honor cancellation and
  update execution counters atomically. Unsupported actions now fail explicitly
  rather than report a successful no-op.
- Enforce JSON limits even for a complete oversized JSON value or oversized
  trailing whitespace; reject multiple JSON values.
- Validate entity names, dates, calendar time ranges, numeric bounds, email
  recipients and cross-workspace relationships.
- Correct frontend API prefixes, request-header merging and error envelopes.
  Repair agent/key creation paths, agent selection, expiry serialization,
  response fields, scope controls and one-time key display.
- Return usable list envelopes and structured webhook events/integration
  configuration. Wire webhook tests and email-template update/delete routes.
- Correct calendar timezone conversion, task completion status and login return
  paths. Remove displayed demo credentials and connect the inbox route.
- Aggregate dashboard metrics in SQLite rather than a capped page. Remove
  fabricated activity records and the invented default sales-cycle value.
- Query real schema/migration/audit records, avoid nested connection starvation,
  handle empty aggregate results and prevent unsigned memory-delta underflow.
- Strictly parse environment overrides, honor runtime timeout/pool settings,
  validate production cookie/TLS settings and bind IPv6 addresses correctly.
- Match container toolchains to the project, include built frontend assets,
  repair build metadata/environment names, add the missing Nginx configuration
  and exclude sensitive/local files from the Docker context.
- Normalize Kubernetes resource names and use one SQLite writer pod with a
  recreate strategy. Remove incompatible horizontal autoscaling and unnecessary
  Kubernetes API permissions. Bind optional monitoring ports to loopback and
  require a configured Grafana password.

## Verification

Passed locally:

- `go test ./...`, including request-level login, CSRF, body limits, protected
  CRUD, MCP identity/scopes/revocation, schema and dashboard tests.
- Regression tests for per-connection SQLite settings, escaped paths, legacy
  data preservation, FTS updates, workspace isolation, archives, aggregates
  beyond 200 records, queue limits, email approval/claims, SSRF, response limits,
  password bounds, configuration parsing and credential encryption/tampering.
- `go vet ./...`.
- Frontend TypeScript compilation and production build (`npm.cmd run build`).
- Frontend lint exits successfully, with existing warnings remaining.

Additional checks and limits:

- `gosec` finished with 17 findings reviewed: 14 dynamic SQL queries using
  allowlisted identifiers, two dynamically configured cookie literals, and one
  SQL column-name constant mistaken for credentials. Dynamic SQL sorting/filtering is
  restricted to source-defined identifiers and bound values. Development cookie
  configuration and SQL column constants also produce scanner findings. A
  scanner exit code of zero is not claimed.
- `govulncheck` and `npm audit` could not contact their advisory databases because
  outbound network access is blocked. Dependency vulnerability status remains
  unverified.
- Race testing was attempted with CGO enabled, but `gcc` is unavailable. No
  race-detector pass is claimed.
- Docker/Kubernetes execution and an interactive browser run were not performed
  in this environment. The frontend bundle still has a size warning.

## Remaining feature gaps and deployment work

### Additional fixes completed in the requested 5–10 minute pass

- Implemented the password-change endpoint. It verifies the current password,
  updates the hash with a concurrency check, and revokes other sessions in one
  transaction while retaining the current session.
- Added an authenticated account endpoint and made frontend session checks use
  it, so dashboard calculation failures cannot masquerade as expired sessions.
- Added automatic audit records for successful protected REST writes. Request
  bodies, passwords, tokens and signing secrets are not copied into these logs.
- Replaced fabricated health checks, database connection counts and queue
  counters with actual database and worker-pool state.
- Added validated SMTP environment configuration: `CRM_SMTP_HOST`,
  `CRM_SMTP_PORT`, `CRM_SMTP_USERNAME`, `CRM_SMTP_PASSWORD`, `CRM_SMTP_FROM`, and
  `CRM_SMTP_IMPLICIT_TLS`. Port 465 defaults to implicit TLS; other ports require
  STARTTLS through the existing secure sender.
- Implemented atomic lead conversion into a contact, optional company and
  optional deal. Existing related IDs must belong to the workspace; creating
  entities requires their corresponding write scopes. Failed conversions roll
  back all changes, and repeated conversion cannot duplicate the records.

The complete Go suite and frontend production build passed again after these
changes. New tests cover password verification/session revocation, REST audit
records without secrets, SMTP parsing, and lead conversion relationships,
rollback, duplicate prevention and operation with a one-connection pool.

These pre-existing incomplete features are not represented as completed work:

- Integration synchronization and most workflow action adapters still require
  implementations. They now return an explicit unsupported-operation result.
- General profile/workspace settings persistence and some administration views
  remain incomplete. Password management and lead conversion are now implemented.
- Automatic auditing currently covers protected REST writes; MCP mutations and
  public authentication events still need dedicated audit instrumentation.
- SMTP credentials/providers and real external delivery must be configured and
  exercised before email sending can be considered operational.
- Previously stored plaintext integration credentials have not been rewritten.
  New writes require a 64-character hexadecimal `CRM_ENCRYPTION_KEY`; existing
  secrets need an explicit data migration or rotation before deployment.

The running application was not restarted, and production data was not migrated
by this review. A rebuilt executable is available at `bin/crm.exe`. Restarting a
rebuilt server applies migrations 8 and 9; preserve a database backup first.
Existing sessions require login again because session storage now uses hashes.
The supplied Nginx setup requires your certificates at `certs/fullchain.pem` and
`certs/privkey.pem`, and deployment placeholders must be replaced with real values.
