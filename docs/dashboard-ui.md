# CRM dashboard UI

The authenticated workspace follows the Zapier-inspired reference in `DESIGN.md`: warm white surfaces, dark sans-serif text, quiet borders, and orange primary actions. The public homepage retains its existing design.

## Implemented

- Grouped workspace navigation, active routes, create menu, account controls, and responsive mobile navigation.
- Debounced CRM record search and page navigation with Ctrl/Cmd+K.
- Live dashboard metrics, default-pipeline distribution, sortable recent deals, follow-up queue, and sales performance.
- Task completion with pending, failure, and success feedback; dashboard refresh retains previously loaded data if an individual request fails.
- Shared record-page headers, search/filter controls, selection actions, tables, forms, modals, and Kanban styling. Boards support pointer drag/drop and a keyboard-accessible move menu.
- Creation and record links open the existing editors for companies, contacts, leads, deals, tasks, and projects.
- Deal forms use named pipeline/stage choices from authenticated, workspace-scoped `GET /api/v1/pipelines`. The API requires `deals:read`; stages belong to the selected pipeline.
- Route code splitting and recoverable route-loading failures.

## Data boundaries

The dashboard uses real API responses; no sample data is seeded. Recent deals are a server-sorted six-record page. Follow-ups are selected from the 200 most recently updated tasks; the panel identifies them as recent tasks and links to the complete task list. Pipeline and performance values use the existing dashboard API's USD presentation. There is no historical chart or invented comparison percentage.

## Validation

- `npm.cmd --prefix web run build`: passed; route splitting removed the oversized-chunk warning.
- `npm.cmd --prefix web run lint`: passed with existing repository warnings; new dashboard, shell, routing hook, and data utilities have no targeted lint warnings.
- `go test ./...`: passed, including pipeline authentication, method, workspace-isolation, and scope checks.
- Server-rendering checks cover populated, empty, loading, and partial-failure dashboards; navigation; record links; task controls; table/board actions; currency/date helpers; and stylesheet coverage.
- The running frontend proxy was checked with authenticated requests for identity, metrics, deals, tasks, pipelines, and search.

The browser/native preview bridge was unavailable during this run, so desktop/mobile visual inspection and browser interaction checks remain outstanding. Build and render checks do not replace those checks or a deployment review.
