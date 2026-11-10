# Administration UI

The administration overview is available at `/admin` (`/administration` redirects there). The existing sidebar Administration group now includes Overview. Shared navigation connects the overview, access/agents, webhooks, audit log, data model, system health, and settings.

## Views

- **Overview:** live agent, API-key, and webhook totals; service health; process metrics; configuration links; and recent audit records.
- **Access and webhooks:** existing create/edit/revoke/configuration workflows remain, with the shared administration heading and styling. `/agents?tab=agents` opens the agent list.
- **Audit:** uses the server's actual actor, resource, timestamp, and value fields. Action/resource/actor/date filters use supported parameters. Export downloads only the displayed page and neutralizes spreadsheet formula prefixes. Event details are read-only.
- **Data model:** searchable, paginated table catalog; column and index inspection; JSON metadata copy; and applied migration history. It does not reconstruct incomplete SQL or invent checksum, index uniqueness, or row-size values.
- **System:** reports only CRM backend process RAM (current resident memory, peak resident memory, and process ID), with heap/runtime memory shown separately. It also shows connection counts, uptime, goroutines, health checks, and real queue counters. CPU, disk, and network collectors remain outside this view. Auto-refresh runs every 30 seconds while the tab is visible.
- **Settings:** uses the working identity and password-change endpoints, with links to integration, webhook, and access configuration. Previous profile/preferences/SMTP/custom-code/delete-workspace controls had no backing endpoints and were replaced with supported account/security controls. No account or security settings were changed during verification.

The audit handler now forwards validated repository sort/order parameters and bound actor/resource IDs. The repository continues to enforce the authenticated workspace boundary.

## Validation

- Frontend build and lint completed successfully; the new administration files pass targeted lint without warnings.
- All seven pages passed server-rendering checks for navigation, one page heading, loading states, and absence of invalid dates or undefined values.
- Helper checks cover zero/missing metrics, UTC timestamps, degraded HTTP 503 health responses, and CSV formula escaping.
- `go test ./...` passed. HTTP regressions cover audit ordering, actor/resource filters, and workspace isolation using isolated test data.
- Authenticated requests through the running frontend proxy passed for identity, health, metrics, queues, agents, API keys, webhooks, audit, tables, and migrations.

The local frontend and updated backend remain running. Browser visual and interaction review remains outstanding because the browser preview bridge is unavailable.

## Application RAM measurements

`GET /api/v1/admin/metrics` includes `process_memory`: process ID, current resident bytes, peak resident bytes, and sample timestamp. Whole-machine RAM is no longer collected or returned. Windows uses [GetProcessMemoryInfo](https://learn.microsoft.com/en-us/windows/win32/api/psapi/nf-psapi-getprocessmemoryinfo) for the current process's working set; Linux reads VmRSS and VmHWM from /proc/self/status. Unsupported or unavailable collectors return null and the UI displays an unavailable state.

This measures the CRM backend process only. Browser tabs and development tooling are separate processes. Existing memory_usage and memory_total remain Go heap allocation and Go runtime memory obtained from the OS; they overlap process resident memory and must not be added to it. Automatic refresh runs every 30 seconds while the tab is visible.

## Consolidated audit history

Activity and Audit Log were duplicate routes for the same component and records. Audit Log is now the single destination, with `/activity` redirecting there. See [Audit UI](audit-ui.md) for the timeline/table redesign, timestamp compatibility, supported filters, and completed browser verification.
