# TORQ Admin Guide

## Overview

This guide covers administration, configuration, and operational tasks for the TORQ system.

## System Requirements

### Minimum Requirements
- **CPU**: 2 cores
- **Memory**: 512 MB (1 GB recommended)
- **Storage**: 10 GB SSD
- **OS**: Linux (Ubuntu 22.04+, Debian 12+, Alpine 3.20+)

### Production Recommendations
- **CPU**: 4+ cores
- **Memory**: 2 GB+
- **Storage**: 50 GB+ NVMe SSD
- **Database**: SQLite with WAL mode on local SSD
- **Reverse Proxy**: nginx with TLS termination
- **Monitoring**: Prometheus + Grafana + Loki

## Installation

### Docker Compose (Recommended)

```bash
# Clone and configure
git clone https://github.com/yourorg/crm.git
cd crm

# Configure environment
cp .env.example .env
# Edit .env with your settings

# Start services
docker-compose up -d

# Verify
curl http://localhost:8080/health
```

### Kubernetes

```bash
# Apply manifests
kubectl apply -f k8s/

# Check status
kubectl get pods -n crm
kubectl logs -n crm -l app=crm
```

### Binary Deployment

```bash
# Download release
wget https://github.com/yourorg/crm/releases/download/v1.0.0/crm-linux-amd64.tar.gz
tar -xzf crm-linux-amd64.tar.gz

# Configure
export CRM_DATABASE_PATH=/var/lib/crm/crm.db
export CRM_HTTP_HOST=0.0.0.0
export CRM_HTTP_PORT=8080
# ... other config

# Run
./crm
```

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `CRM_DATABASE_PATH` | SQLite database file path | `/app/data/crm.db` | Yes |
| `CRM_DATABASE_MAX_OPEN_CONNS` | Max open DB connections | `10` | No |
| `CRM_DATABASE_MAX_IDLE_CONNS` | Max idle DB connections | `5` | No |
| `CRM_DATABASE_BUSY_TIMEOUT` | SQLite busy timeout | `5s` | No |
| `CRM_HTTP_HOST` | HTTP listen address | `0.0.0.0` | No |
| `CRM_HTTP_PORT` | HTTP listen port | `8080` | No |
| `CRM_HTTP_READ_TIMEOUT` | HTTP read timeout | `15s` | No |
| `CRM_HTTP_WRITE_TIMEOUT` | HTTP write timeout | `30s` | No |
| `CRM_HTTP_IDLE_TIMEOUT` | HTTP idle timeout | `60s` | No |
| `CRM_HTTP_SHUTDOWN_TIMEOUT` | Graceful shutdown timeout | `10s` | No |
| `CRM_LOG_FORMAT` | Log format (json/text) | `json` | No |
| `CRM_LOG_LEVEL` | Log level (debug/info/warn/error) | `info` | No |
| `CRM_SERVER_VERSION` | Version string | `dev` | No |
| `CRM_SERVER_BUILD_ID` | Build ID | `unknown` | No |
| `CRM_SERVER_GIT_SHA` | Git commit SHA | `unknown` | No |
| `CRM_SERVER_ENV` | Environment (development/production) | `development` | No |
| `CRM_JOBS_WORKERS` | Background job workers | `8` | No |
| `CRM_JOBS_QUEUE_SIZE` | Job queue size | `256` | No |

### Security Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `CRM_SECURITY_RATE_LIMIT_ENABLED` | Enable rate limiting | `true` |
| `CRM_SECURITY_RATE_LIMIT_RPS` | Requests per second | `100` |
| `CRM_SECURITY_RATE_LIMIT_BURST` | Burst allowance | `200` |
| `CRM_SECURITY_MAX_REQUEST_BODY` | Max request body (bytes) | `10485760` |
| `CRM_SECURITY_MAX_UPLOAD_SIZE` | Max upload size (bytes) | `52428800` |
| `CRM_SECURITY_CORS_ORIGINS` | Allowed CORS origins | `*` |
| `CRM_SECURITY_CORS_CREDENTIALS` | Allow credentials | `true` |
| `CRM_SECURITY_SSRF_BLOCK_PRIVATE` | Block private IPs | `true` |
| `CRM_SECURITY_REQUEST_TIMEOUT` | Request timeout | `30s` |
| `CRM_SECURITY_HTTP_CLIENT_TIMEOUT` | Outbound HTTP timeout | `30s` |
| `CRM_SECURITY_HTTP_CLIENT_MAX_RESPONSE` | Max response size | `10485760` |
| `CRM_SECURITY_IDEMPOTENCY_TTL` | Idempotency key TTL | `24h` |
| `CRM_SECURITY_AUDIT_MAX_SIZE` | Audit log max size | `1048576` |
| `CRM_SECURITY_AUDIT_MAX_ENTRIES` | Audit log max entries | `10000` |
| `CRM_SECURITY_SESSION_COOKIE_NAME` | Session cookie name | `crm_session` |
| `CRM_SECURITY_SESSION_COOKIE_SECURE` | Secure cookies | `true` |
| `CRM_SECURITY_SESSION_COOKIE_SAMESITE` | SameSite policy | `Lax` |
| `CRM_SECURITY_SESSION_MAX_AGE` | Session max age | `24h` |
| `CRM_SECURITY_CSRF_ENABLED` | Enable CSRF protection | `true` |
| `CRM_SECURITY_CSRF_HEADER` | CSRF header name | `X-CSRF-Token` |
| `CRM_SECURITY_CSRF_COOKIE` | CSRF cookie name | `csrf_token` |
| `CRM_SECURITY_CSRF_TOKEN_LENGTH` | CSRF token length | `32` |

### Owner Bootstrap

For first deployment, set these to auto-create the owner account:

| Variable | Description |
|----------|-------------|
| `CRM_OWNER_EMAIL` | Owner email address |
| `CRM_OWNER_PASSWORD` | Owner initial password |
| `CRM_ENCRYPTION_KEY` | 32-byte hex key for credential encryption |

Generate encryption key: `openssl rand -hex 32`

### Memory Configuration

```bash
# Set Go memory limit (matches container limit)
export GOMEMLIMIT=400MiB
```

## Database Management

### Migrations

Migrations run automatically on startup. To run manually:

```bash
# Using the binary
./crm migrate

# Check status
./crm migrate status
```

### Backup

```bash
# Manual backup (requires sqlite3)
sqlite3 /app/data/crm.db ".backup /app/backups/crm-$(date +%Y%m%d).db"

# Verify backup
sqlite3 /app/backups/crm-$(date +%Y%m%d).db "PRAGMA integrity_check;"
```

### Automated Backups

Configure cron for daily backups:

```bash
# /etc/cron.daily/crm-backup
#!/bin/bash
sqlite3 /app/data/crm.db ".backup /app/backups/crm-$(date +%Y%m%d).db"
find /app/backups -name "crm-*.db" -mtime +30 -delete
```

### Restore

```bash
# Stop application
systemctl stop crm

# Restore
cp /app/backups/crm-20240115.db /app/data/crm.db

# Verify
sqlite3 /app/data/crm.db "PRAGMA integrity_check;"

# Start application
systemctl start crm
```

### WAL Checkpoint

```bash
# Manual checkpoint
sqlite3 /app/data/crm.db "PRAGMA wal_checkpoint(TRUNCATE);"
```

## Monitoring & Observability

### Health Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Liveness probe - basic health |
| `GET /ready` | Readiness probe - DB, dependencies |
| `GET /metrics` | Prometheus metrics |

### Key Metrics

| Metric | Description | Alert Threshold |
|--------|-------------|-----------------|
| `crm_http_requests_total` | Total HTTP requests | - |
| `crm_http_request_duration_seconds` | Request latency histogram | P99 > 1s |
| `crm_memory_rss_bytes` | RSS memory usage | > 400MB |
| `crm_goroutines_total` | Goroutine count | > 10000 |
| `crm_database_open_connections` | Open DB connections | > 8 |
| `crm_jobs_pending` | Pending background jobs | > 100 |
| `crm_rate_limit_exceeded_total` | Rate limit hits | > 10/s |
| `crm_ssrf_blocked_total` | SSRF attempts blocked | > 0 |
| `crm_auth_failed_total` | Failed login attempts | > 5/s |

### Prometheus Alerts

See `monitoring/alerts.yml` for complete alerting rules.

### Grafana Dashboards

Import dashboards from `monitoring/grafana/dashboards/`:
- TORQ Overview
- TORQ Performance
- TORQ Database
- TORQ Security
- CRM Jobs

### Log Aggregation

Logs are JSON formatted. Ship to Loki/Grafana:

```yaml
# promtail-config.yml
clients:
  - url: http://loki:3100/loki/api/v1/push
positions:
  filename: /tmp/positions.yaml
scrape_configs:
  - job_name: crm
    static_configs:
      - targets: [localhost]
        labels:
          job: crm
          __path__: /app/logs/*.log
```

## Security Operations

### TLS Configuration

For production, terminate TLS at nginx:

```nginx
# nginx.conf
server {
    listen 443 ssl http2;
    server_name torq.yourdomain.com;
    
    ssl_certificate /etc/nginx/certs/torq.crt;
    ssl_certificate_key /etc/nginx/certs/torq.key;
    
    # Security headers
    add_header X-Content-Type-Options nosniff;
    add_header X-Frame-Options DENY;
    add_header Content-Security-Policy "default-src 'self'; ...";
    
    location / {
        proxy_pass http://torq:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Certificate Management

Use cert-manager with Let's Encrypt:

```yaml
# Certificate resource
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: torq-tls
  namespace: torq
spec:
  secretName: torq-tls
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
    - torq.yourdomain.com
```

### Rate Limiting

Configure at nginx level for additional protection:

```nginx
limit_req_zone $binary_remote_addr zone=torq:10m rate=100r/s;

location / {
    limit_req zone=torq burst=200 nodelay;
    limit_req_status 429;
    ...
}
```

### Audit Log Review

```bash
# View recent audit logs
curl -H "Authorization: Bearer $TOKEN" \
  https://crm.yourdomain.com/api/v1/admin/audit-logs?limit=50

# Search for specific actions
curl -H "Authorization: Bearer $TOKEN" \
  "https://crm.yourdomain.com/api/v1/admin/audit-logs?action=delete&start_from=2024-01-01"
```

### Session Management

```bash
# List active sessions
curl -H "Authorization: Bearer $TOKEN" \
  https://crm.yourdomain.com/api/v1/admin/sessions

# Revoke session
curl -X DELETE -H "Authorization: Bearer $TOKEN" \
  https://crm.yourdomain.com/api/v1/admin/sessions/{session_id}
```

## Performance Tuning

### Database

```bash
# Increase cache size (in pages, ~4KB each)
# Add to DSN: _pragma=cache_size(-32768)  # 128MB

# Enable mmap
# Add to DSN: _pragma=mmap_size(268435456)  # 256MB

# Tune synchronous mode (NORMAL for performance, FULL for safety)
# Add to DSN: _pragma=synchronous(NORMAL)
```

### Connection Pool

```bash
CRM_DATABASE_MAX_OPEN_CONNS=20
CRM_DATABASE_MAX_IDLE_CONNS=10
CRM_DATABASE_CONN_MAX_LIFETIME=1h
```

### Memory

```bash
# Set GOMEMLIMIT to 80% of container limit
# Container: 512MiB -> GOMEMLIMIT=400MiB
GOMEMLIMIT=400MiB
```

### Job Workers

```bash
# Scale based on workload
CRM_JOBS_WORKERS=16
CRM_JOBS_QUEUE_SIZE=512
```

## Upgrades

### Rolling Upgrade (Kubernetes)

```bash
# Update image
kubectl set image deployment/crm crm=crm:v1.1.0 -n crm

# Monitor rollout
kubectl rollout status deployment/crm -n crm

# Rollback if needed
kubectl rollout undo deployment/crm -n crm
```

### Binary Upgrade

```bash
# Stop service
systemctl stop crm

# Backup database
cp /app/data/crm.db /app/backups/crm-pre-upgrade-$(date +%Y%m%d).db

# Replace binary
cp crm-new /usr/local/bin/crm

# Start service
systemctl start crm

# Verify
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

### Database Migrations

Migrations run automatically on startup. For zero-downtime:

1. Deploy new version alongside old (blue-green)
2. Run migrations against copy of database
3. Verify
4. Switch traffic

## Troubleshooting

### Application Won't Start

```bash
# Check logs
journalctl -u crm -f

# Common issues:
# - Database locked: check for stale processes
# - Port in use: check CRM_HTTP_PORT
# - Permissions: data dir must be writable by crm user
# - Config: validate all required env vars
```

### Database Issues

```bash
# Check integrity
sqlite3 /app/data/crm.db "PRAGMA integrity_check;"

# Check WAL mode
sqlite3 /app/data/crm.db "PRAGMA journal_mode;"

# Check locks
sqlite3 /app/data/crm.db "PRAGMA lock_status;"

# Rebuild if corrupted
# 1. Stop app
# 2. Restore from backup
# 3. Start app
```

### High Memory Usage

```bash
# Check memory stats
curl http://localhost:8080/api/v1/admin/memory

# Force GC
curl -X POST http://localhost:8080/api/v1/admin/gc

# Check for leaks
# - Review goroutine count
# - Check for unclosed connections
# - Verify job queue not growing unbounded
```

### Slow Queries

```bash
# Enable query logging temporarily
# Add to DSN: _pragma=trace(1)

# Check slow queries in logs
grep "slow_query" /app/logs/crm.log

# Add indexes if needed (see db/migrations/00007_performance_indexes.sql)
```

### Rate Limiting Issues

```bash
# Check rate limit hits
curl http://localhost:8080/metrics | grep rate_limit

# Adjust limits
CRM_SECURITY_RATE_LIMIT_RPS=200
CRM_SECURITY_RATE_LIMIT_BURST=400
```

## Disaster Recovery

### Recovery Point Objective (RPO)

- **Target**: < 1 hour
- **Method**: Hourly automated backups to object storage

### Recovery Time Objective (RTO)

- **Target**: < 30 minutes
- **Method**: Documented restore procedure, tested monthly

### DR Checklist

- [ ] Backup storage accessible from DR site
- [ ] Restore procedure documented and tested
- [ ] DNS failover configured
- [ ] TLS certificates available at DR site
- [ ] Configuration replicated to DR site
- [ ] Team trained on DR procedure

## Maintenance Windows

### Scheduled Maintenance

- **Weekly**: Sunday 02:00-04:00 UTC
- **Tasks**: OS updates, backup verification, log rotation

### Emergency Maintenance

- **Trigger**: Security patches, critical bugs
- **Process**: 
  1. Assess impact
  2. Notify stakeholders
  3. Execute fix
  4. Verify
  5. Post-incident review

## Compliance

### Data Retention

Configure in Settings → Data Retention:
- **Audit logs**: 7 years (configurable)
- **Deleted records**: 30 days soft delete
- **Exported data**: Per request

### Access Control

- **RBAC**: Role-based access control
- **Audit trail**: All admin actions logged
- **Session management**: Configurable timeouts
- **API keys**: Scoped, rotatable, auditable

### Encryption

- **At rest**: SQLite encryption (if compiled with SQLCipher)
- **In transit**: TLS 1.2+ everywhere
- **Credentials**: Argon2id password hashing
- **Integration secrets**: AES-256-GCM with per-workspace keys

## Support Contacts

| Issue | Contact | SLA |
|-------|---------|-----|
| Critical outage | oncall@yourdomain.com | 15 min |
| Security incident | security@yourdomain.com | 1 hour |
| General support | support@yourdomain.com | 4 hours |
| Feature requests | product@yourdomain.com | Next release |

---

*For user documentation, see [User Guide](user-guide.md). For API reference, see [API Documentation](api-docs.md). For developer setup, see [Developer Guide](developer-guide.md).*