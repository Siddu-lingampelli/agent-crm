# TORQ API Documentation

## Overview

TORQ provides a RESTful API for programmatic access to all resources. The API follows JSON:API conventions with some simplifications.

- **Base URL**: `https://torq.yourdomain.com/api/v1`
- **Authentication**: Bearer token (session cookie for browser)
- **Content-Type**: `application/json`
- **Rate Limiting**: 100 req/s per IP (configurable)

## Authentication

### Session-based (Browser)

The web UI uses secure, HttpOnly cookies. No additional headers needed.

### Bearer Token (API)

```bash
# Login to get token
curl -X POST https://torq.yourdomain.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password123"}'

# Response
{"success": true, "token": "eyJhbGciOiJIUzI1NiIs...", "user": {...}}

# Use token
curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  https://torq.yourdomain.com/api/v1/companies
```

### API Keys (Service-to-Service)

Generate in Settings → API Keys:

```bash
curl -H "Authorization: Bearer sk_live_abc123..." \
  https://torq.yourdomain.com/api/v1/companies
```

## Common Patterns

### Pagination

All list endpoints support pagination:

```
GET /api/v1/contacts?limit=25&offset=0&sort=created_at&order=desc
```

Response:
```json
{
  "items": [...],
  "total": 150,
  "limit": 25,
  "offset": 0
}
```

### Filtering

Filter with query parameters:

```
GET /api/v1/contacts?email=john@example.com&status=active&company_id=abc123
```

### Sorting

```
GET /api/v1/deals?sort=value&order=desc
```

Valid sort fields vary by resource. Default: `created_at` desc.

### Search

Global search across all resources:

```
GET /api/v1/search?q=acme&limit=10
```

Advanced syntax:
- `field:value` - Exact field match
- `"exact phrase"` - Phrase search
- `term1 AND term2` - Both terms
- `term1 OR term2` - Either term
- `-excluded` - Exclude term

### Idempotency

For mutation endpoints, include `Idempotency-Key` header to safely retry:

```
POST /api/v1/contacts
Idempotency-Key: unique-key-123
Content-Type: application/json

{"first_name": "John", "last_name": "Doe", "email": "john@example.com"}
```

Keys expire after 24 hours (configurable).

## Resources

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/auth/login` | Login with email/password |
| POST | `/auth/logout` | Logout (revoke session) |
| GET | `/auth/me` | Get current user info |
| POST | `/auth/refresh` | Refresh access token |

#### Login

```bash
POST /api/v1/auth/login
Content-Type: application/json

{"email": "user@example.com", "password": "password123"}
```

Response:
```json
{
  "success": true,
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "usr_abc123",
    "email": "user@example.com",
    "name": "John Doe",
    "role": "admin",
    "workspace_id": "ws_xyz789"
  }
}
```

### Companies

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/companies` | List companies |
| POST | `/companies` | Create company |
| GET | `/companies/{id}` | Get company |
| PATCH | `/companies/{id}` | Update company |
| DELETE | `/companies/{id}` | Archive company |
| GET | `/companies/{id}/contacts` | List company contacts |
| GET | `/companies/{id}/deals` | List company deals |

#### Company Object

```json
{
  "id": "cmp_abc123",
  "workspace_id": "ws_xyz789",
  "name": "Acme Corporation",
  "domain": "acme.com",
  "industry": "Technology",
  "size_band": "51-200",
  "website": "https://acme.com",
  "phone": "+1-555-0100",
  "email": "contact@acme.com",
  "address": "123 Main St, San Francisco, CA 94105",
  "is_client": true,
  "status": "active",
  "owner_id": "usr_abc123",
  "tags": ["enterprise", "partner"],
  "custom_fields": {"account_tier": "premium"},
  "notes": "Key strategic partner",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-20T14:22:00Z"
}
```

#### Create Company

```bash
POST /api/v1/companies
Content-Type: application/json

{
  "name": "Acme Corporation",
  "domain": "acme.com",
  "industry": "Technology",
  "size_band": "51-200",
  "website": "https://acme.com",
  "phone": "+1-555-0100",
  "email": "contact@acme.com",
  "address": "123 Main St, San Francisco, CA 94105",
  "is_client": true,
  "tags": ["enterprise"],
  "custom_fields": {"account_tier": "premium"}
}
```

### Contacts

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/contacts` | List contacts |
| POST | `/contacts` | Create contact |
| GET | `/contacts/{id}` | Get contact |
| PATCH | `/contacts/{id}` | Update contact |
| DELETE | `/contacts/{id}` | Archive contact |
| GET | `/contacts/{id}/activities` | List contact activities |
| GET | `/contacts/{id}/deals` | List contact deals |
| GET | `/contacts/{id}/tasks` | List contact tasks |

#### Contact Object

```json
{
  "id": "cnt_abc123",
  "workspace_id": "ws_xyz789",
  "first_name": "John",
  "last_name": "Doe",
  "email": "john@acme.com",
  "phone": "+1-555-0101",
  "job_title": "CTO",
  "company_id": "cmp_abc123",
  "owner_id": "usr_abc123",
  "status": "active",
  "tags": ["decision-maker", "technical"],
  "custom_fields": {"linkedin": "linkedin.com/in/johndoe"},
  "notes": "Met at conference 2024",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-20T14:22:00Z"
}
```

### Deals

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/deals` | List deals |
| POST | `/deals` | Create deal |
| GET | `/deals/{id}` | Get deal |
| PATCH | `/deals/{id}` | Update deal |
| DELETE | `/deals/{id}` | Archive deal |
| POST | `/deals/{id}/move` | Move deal to stage |
| GET | `/deals/{id}/activities` | List deal activities |
| GET | `/deals/{id}/tasks` | List deal tasks |

#### Deal Object

```json
{
  "id": "del_abc123",
  "workspace_id": "ws_xyz789",
  "name": "Acme - Enterprise License",
  "description": "50-seat annual license",
  "value": 25000,
  "currency": "USD",
  "stage": "proposal",
  "probability": 60,
  "expected_close_date": "2024-03-31",
  "actual_close_date": null,
  "company_id": "cmp_abc123",
  "contact_id": "cnt_abc123",
  "owner_id": "usr_abc123",
  "tags": ["renewal", "expansion"],
  "custom_fields": {"competitor": "competitor_x"},
  "notes": "Budget approved, legal review pending",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-20T14:22:00Z"
}
```

#### Move Deal Stage

```bash
POST /api/v1/deals/del_abc123/move
Content-Type: application/json

{"stage": "negotiation", "probability": 80}
```

### Tasks

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/tasks` | List tasks |
| POST | `/tasks` | Create task |
| GET | `/tasks/{id}` | Get task |
| PATCH | `/tasks/{id}` | Update task |
| DELETE | `/tasks/{id}` | Delete task |
| POST | `/tasks/{id}/complete` | Mark complete |
| GET | `/tasks/{id}/subtasks` | List subtasks |

#### Task Object

```json
{
  "id": "tsk_abc123",
  "workspace_id": "ws_xyz789",
  "title": "Follow up with Acme CTO",
  "description": "Send proposal and schedule demo",
  "status": "pending",
  "priority": "high",
  "due_date": "2024-01-25T17:00:00Z",
  "completed_at": null,
  "assignee_id": "usr_abc123",
  "contact_id": "cnt_abc123",
  "company_id": "cmp_abc123",
  "deal_id": "del_abc123",
  "project_id": null,
  "recurrence": null,
  "tags": ["follow-up"],
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-20T14:22:00Z"
}
```

### Projects

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/projects` | List projects |
| POST | `/projects` | Create project |
| GET | `/projects/{id}` | Get project |
| PATCH | `/projects/{id}` | Update project |
| DELETE | `/projects/{id}` | Archive project |
| GET | `/projects/{id}/tasks` | List project tasks |
| GET | `/projects/{id}/members` | List project members |
| POST | `/projects/{id}/members` | Add project member |

### Calendar

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/calendar/events` | List events |
| POST | `/calendar/events` | Create event |
| GET | `/calendar/events/{id}` | Get event |
| PATCH | `/calendar/events/{id}` | Update event |
| DELETE | `/calendar/events/{id}` | Delete event |
| GET | `/calendar/availability` | Check availability |

### Search

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/search` | Global search |

#### Search Parameters

| Parameter | Type | Description |
|-----------|------|-------------|
| `q` | string | Search query |
| `types` | string | Comma-separated types (contacts,companies,deals,tasks,projects,emails) |
| `workspace` | string | Workspace ID (default: current) |
| `limit` | int | Max results (default: 10, max: 50) |

#### Search Response

```json
{
  "results": [
    {
      "type": "contact",
      "id": "cnt_abc123",
      "title": "John Doe",
      "subtitle": "CTO at Acme Corporation",
      "url": "/contacts/cnt_abc123",
      "score": 0.95
    },
    {
      "type": "deal",
      "id": "del_abc123",
      "title": "Acme - Enterprise License",
      "subtitle": "$25,000 • Proposal",
      "url": "/deals/del_abc123",
      "score": 0.87
    }
  ],
  "total": 2,
  "took_ms": 15
}
```

### Email

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/emails` | List emails |
| GET | `/emails/{id}` | Get email |
| POST | `/emails` | Send email |
| POST | `/emails/{id}/reply` | Reply to email |
| POST | `/emails/{id}/forward` | Forward email |

### Webhooks

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/webhooks` | List webhooks |
| POST | `/webhooks` | Create webhook |
| GET | `/webhooks/{id}` | Get webhook |
| PATCH | `/webhooks/{id}` | Update webhook |
| DELETE | `/webhooks/{id}` | Delete webhook |
| POST | `/webhooks/{id}/test` | Send test payload |

#### Webhook Events

- `contact.created` / `contact.updated` / `contact.deleted`
- `company.created` / `company.updated` / `company.deleted`
- `deal.created` / `deal.updated` / `deal.deleted` / `deal.stage_changed`
- `task.created` / `task.updated` / `task.completed` / `task.deleted`
- `project.created` / `project.updated` / `project.deleted`
- `email.received` / `email.sent`

#### Webhook Payload

```json
{
  "event": "deal.stage_changed",
  "timestamp": "2024-01-20T14:22:00Z",
  "workspace_id": "ws_xyz789",
  "data": {
    "id": "del_abc123",
    "name": "Acme - Enterprise License",
    "old_stage": "proposal",
    "new_stage": "negotiation",
    "value": 25000
  }
}
```

### Admin (Requires Admin Role)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/admin/health` | System health |
| GET | `/admin/metrics` | System metrics |
| GET | `/admin/memory` | Memory status |
| POST | `/admin/gc` | Force garbage collection |
| GET | `/admin/audit-logs` | Audit logs |
| GET | `/admin/sessions` | Active sessions |
| DELETE | `/admin/sessions/{id}` | Revoke session |
| GET | `/admin/schema` | Database schema |
| GET | `/admin/jobs` | Job queue status |
| POST | `/admin/jobs/{id}/retry` | Retry failed job |

## Error Responses

### Standard Error Format

```json
{
  "error": {
    "code": "validation_error",
    "message": "Invalid input",
    "details": [
      {"field": "email", "message": "Invalid email format"}
    ]
  }
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `unauthorized` | 401 | Invalid/missing authentication |
| `forbidden` | 403 | Insufficient permissions |
| `not_found` | 404 | Resource not found |
| `validation_error` | 400 | Invalid request data |
| `conflict` | 409 | Resource conflict (e.g., duplicate email) |
| `rate_limited` | 429 | Too many requests |
| `internal_error` | 500 | Server error |
| `service_unavailable` | 503 | Service temporarily unavailable |

## Rate Limiting

Headers returned on all responses:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1705764000
Retry-After: 60  (on 429 response)
```

## Versioning

API version in URL path: `/api/v1/`

Breaking changes will result in new version: `/api/v2/`

## Webhooks Security

Verify webhook signatures:

```python
import hmac
import hashlib

def verify_webhook(payload, signature, secret):
    expected = hmac.new(
        secret.encode(),
        payload.encode(),
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(expected, signature)
```

## SDKs & Examples

### cURL

```bash
# List companies
curl -H "Authorization: Bearer $TOKEN" \
  "https://torq.yourdomain.com/api/v1/companies?limit=10"

# Create contact
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: unique-123" \
  -d '{"first_name":"Jane","last_name":"Smith","email":"jane@example.com"}' \
  https://torq.yourdomain.com/api/v1/contacts
```

### Python

```python
import requests

class TORQClient:
    def __init__(self, base_url, token):
        self.base_url = base_url.rstrip('/')
        self.headers = {"Authorization": f"Bearer {token}"}
    
    def list_companies(self, limit=25, offset=0):
        resp = requests.get(
            f"{self.base_url}/api/v1/companies",
            params={"limit": limit, "offset": offset},
            headers=self.headers
        )
        resp.raise_for_status()
        return resp.json()
    
    def create_contact(self, data, idempotency_key=None):
        headers = self.headers.copy()
        if idempotency_key:
            headers["Idempotency-Key"] = idempotency_key
        resp = requests.post(
            f"{self.base_url}/api/v1/contacts",
            json=data,
            headers=headers
        )
        resp.raise_for_status()
        return resp.json()

client = TORQClient("https://torq.yourdomain.com", "your-token")
companies = client.list_companies()
contact = client.create_contact({
    "first_name": "Jane",
    "last_name": "Smith",
    "email": "jane@example.com"
})
```

### JavaScript/TypeScript

```typescript
interface TORQClientOptions {
  baseUrl: string;
  token: string;
}

class TORQClient {
  private baseUrl: string;
  private headers: HeadersInit;

  constructor({ baseUrl, token }: TORQClientOptions) {
    this.baseUrl = baseUrl.replace(/\/$/, '');
    this.headers = { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' };
  }

  async listCompanies(limit = 25, offset = 0) {
    const resp = await fetch(`${this.baseUrl}/api/v1/companies?limit=${limit}&offset=${offset}`, {
      headers: this.headers
    });
    if (!resp.ok) throw new Error(await resp.text());
    return resp.json();
  }

  async createContact(data: object, idempotencyKey?: string) {
    const headers = new Headers(this.headers);
    if (idempotencyKey) headers.set('Idempotency-Key', idempotencyKey);
    const resp = await fetch(`${this.baseUrl}/api/v1/contacts`, {
      method: 'POST',
      headers,
      body: JSON.stringify(data)
    });
    if (!resp.ok) throw new Error(await resp.text());
    return resp.json();
  }
}

const client = new TORQClient({ baseUrl: 'https://torq.yourdomain.com', token: 'your-token' });
const companies = await client.listCompanies();
```

## OpenAPI Specification

Full OpenAPI 3.0 spec available at:
- `https://torq.yourdomain.com/api/openapi.yaml`
- `https://torq.yourdomain.com/api/openapi.json`

## Changelog

| Version | Date | Changes |
|---------|------|---------|
| v1.0 | 2024-01-15 | Initial release |
| v1.1 | 2024-03-01 | Added webhook retry, search improvements |

---

*For user documentation, see [User Guide](user-guide.md). For admin documentation, see [Admin Guide](admin-guide.md). For developer setup, see [Developer Guide](developer-guide.md).*