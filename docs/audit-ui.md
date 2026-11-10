# Audit log UI

Activity and Audit Log previously rendered the same AuditPage and the same `/api/v1/admin/audit` records. The workspace now has one Audit Log destination. `/activity` redirects to `/audit`, preserving query parameters and the hash. The header shortcut and page search use the canonical route; searching “activity” still finds Audit Log.

The page uses the workspace black, orange, and white design with a grouped UTC timeline and a compact table. It supports server-backed metadata search, action/resource/actor filters, exact actor and resource IDs, inclusive UTC calendar dates, sorting, pagination, refresh, loading/error/empty states, and page CSV export. Filter and view state stays in the URL.

The native event dialog shows recorded metadata, safe text rendering of values, raw JSON, copyable IDs, shareable event links, actor/record filters, and CRM record links where supported. Missing values are labeled accurately. HTTP methods remain HTTP methods: POST does not always mean record creation, and DELETE does not always mean permanent deletion. The existing middleware records successful REST writes and does not capture request bodies or before/after snapshots. This redesign does not expand logging coverage.

CSV includes event IDs and the current page's metadata, with spreadsheet formula prefixes neutralized. It does not claim to export the complete history.

## Timestamp compatibility

Existing Go-driver timestamps such as `2026-11-10 13:30:40.2445691 +0000 UTC` were not recognized by the shared repository reader or SQLite's native date function. Readers now accept the legacy format, and audit queries normalize UTC driver strings while sorting/filtering. New audit entries use UTC RFC3339Nano. Existing history is not rewritten. The API retains `start_to` compatibility and adds exclusive `end_before` for inclusive end-day selection.

Search and date filters are bound SQL parameters. The authenticated workspace boundary continues to apply to both records and totals.

## Validation

- Frontend production build and targeted lint passed.
- `go test ./...` passed, including mixed timestamp parsing, search fields, SQL parameter binding, workspace isolation, UTC offset ordering/ranges, filtered totals, and unchanged legacy history.
- Authenticated reads through the running frontend verified valid dates, no-match search, and the new date parameters.
- Isolated browser fixtures verified redirects, timeline/table, search and filters, pagination, page exports, inspector values and missing fields, clipboard/share links, off-page event lookup, invalid date ranges, loading failures/retry, and Escape dismissal.
- Desktop and mobile screenshots were inspected. Both views fit widths 320, 390, 768, 1024, and 1440 without page overflow; wide tables scroll internally.
- Browser helper checks covered invalid calendar dates, year-end UTC boundaries, unavailable dates, empty values, and CSV formula escaping.

Browser fixtures intercepted only audit reads and did not add or modify CRM history. Temporary verification sessions were logged out. Frontend and updated backend remain running.
