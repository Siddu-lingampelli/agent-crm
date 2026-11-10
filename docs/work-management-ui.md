# Projects and Tasks redesign

Both pages share work navigation, compact headings, server-backed filters, white panels, orange actions, visible focus styles, and native dialogs. Summary cards label counts derived from the loaded records separately from the full matching total.

## Projects

- Cards and a table view show names, descriptions, linked companies, reported progress, status, priority, and delivery dates.
- Company names are resolved from actual records. Company search in the editor/filter loads matching choices rather than claiming the first lookup page contains every company.
- Create/edit validates dates and integer progress from 0 to 100. Empty optional fields can be cleared; existing custom fields are preserved.
- Project details link to `/tasks?project=...`, which filters the task API by that project. Existing dashboard/search `record` and `create` links still open editors.
- Selection and archive support partial failures: failed identifiers remain selected and can be retried. Archive descriptions do not claim an unimplemented undo action.
- Project priority filtering is now forwarded through the handler, service, and repository. Search includes name and description.

## Tasks

- List view includes completion, project, status, priority, due date, edit/archive, selection, and bulk actions. On narrow screens the table becomes readable stacked rows.
- Board view loads every matching page, groups tasks by actual status, and supports native drag/drop plus a keyboard-accessible status menu. New-task controls in a column preselect its status. More than 5,000 tasks asks the user to narrow the filters.
- The editor supports optional due dates, project lookup, status/priority, assignment and CRM reference details, and preserves custom fields.
- Status, priority, project, sort, search, due-today, and overdue controls reach the server. Due filters compare calendar date components against the device-local date, and exclude completed/cancelled tasks. Undated tasks remain supported and sort after dated tasks.
- Completion uses the existing endpoint. Status changes now set or clear the completion timestamp consistently, including reopening a completed task and creating a completed task.
- Bulk completion/archive use bounded batches and retain failed selections instead of reporting a whole batch as successful.

## Verification

- Production build, targeted lint, and all Go tests passed.
- New HTTP regressions cover project priority/description search, task priority/project/due filters, matching totals, injection-shaped queries, and completion timestamp transitions.
- Authenticated read-only checks passed against the updated local backend.
- Isolated browser fixtures verified project cards/table, company choices, validation, linked project/task routes, bulk archive partial failure/retry, task completion and bulk completion, overdue and priority filters, board moves/drop handling, column defaults, undated task creation, and existing record/create URL routes.
- Both pages passed desktop/mobile screenshot review and overflow checks at 320, 390, 768, 1024, and 1440 pixels. There were no browser runtime exceptions.
- Helper checks cover full multi-page board loading, partial batch outcomes, date display, and overdue classification. Browser mutations were intercepted and did not change user CRM records.

Frontend: http://localhost:5173/. Both development servers remain running.
