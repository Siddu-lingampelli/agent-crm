# Workflows and integrations UI

The `/workflows` and `/integrations` pages share a compact automation navigation, black text, orange controls, white panels, consistent forms, visible focus styles, and responsive layouts. All record counts and execution statuses come from the API. Empty, loading, and failed requests have separate states.

## Workflows

- Search, status filters, sorting, server pagination, and a selectable workflow library.
- A detail panel shows the stored trigger, ordered steps, run counters, and last/next run times.
- Create/edit uses one dialog with inline step configuration, add/remove, and keyboard-accessible ordering controls. Configuration must be a JSON object. Empty active workflows are rejected in the editor.
- Run confirmation accepts trigger data, then opens actual execution history. History supports a workflow filter, pagination, refresh, progress, errors, and trigger details.
- Archive confirmation preserves recorded execution history.

## Integrations

- Saved connections appear as cards with provider, reported connection status, enabled state, last sync, and last error.
- Search matches names and providers across the server-side result set. Provider/status filters and pagination use the same count query.
- The provider is fixed when editing because the update endpoint does not change it. Configuration accepts a JSON object rather than silently accepting arrays or scalars.
- Enable/disable controls use the update endpoint and wait for the server response. Credentials have a separate replacement dialog and endpoint; stored credentials are never displayed.
- The previous unused connection-test/sync-log controls had no backing endpoints. The new page presents saved configurations and links to webhooks and the agent console.

## Existing server capabilities

Provider sync adapters and most workflow step handlers are not configured in the existing backend. Delay steps execute; unsupported step handlers produce recorded failures. The UI identifies those limits and does not claim a saved integration has successfully connected or synced. Trigger definitions remain stored configuration; this redesign does not implement automatic scheduling or event dispatch.

## Verification

- Production frontend build and targeted lint passed.
- `go test ./...` passed. New HTTP regressions cover integration search by name/provider, matching totals, combined filters, and a SQL-injection-shaped search string.
- Authenticated read-only requests against the running backend passed for workflows, execution history, integrations, and a no-match integration search.
- Desktop/mobile screenshots reviewed for both pages and their dialogs. Browser checks found no page-level overflow at 320, 390, 768, 1024, and 1440 pixels.
- Isolated browser fixtures verified workflow validation, creation, ordered step payloads, run confirmation, history/details, integration create/edit, JSON validation, toggles, credential replacement, archive, search, empty states, native dialog focus, and Escape handling. These writes were intercepted in the verification browser and did not alter real CRM records.
- No browser runtime exceptions were observed. Both local development servers remain running.
