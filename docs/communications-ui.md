# Calendar and Inbox redesign

The Calendar and Inbox pages use the workspace's minimal black, orange, and white controls, white panels, native accessible dialogs, visible focus styles, and responsive layouts.

## Calendar

- Month, week, day, and agenda views are functional. Previous/next navigation follows the selected view; Today resets the date.
- Month cells select a day, expose overflow counts, and open event details. Arrow keys and Home/End navigate the date buttons. A selected-day panel lists all events on that day.
- Visible ranges load all matching pages (200 records per request) rather than silently showing the first page. More than 5,000 events produces an error asking the user to narrow the range.
- Search, status, local time-zone labels, event creation/editing, meeting links, and archive confirmation use the existing API. Attendee identifiers and CRM references remain available under advanced details; references are fixed on edit because the update endpoint does not change them.
- All-day dates use local midnight boundaries with an exclusive end. Multi-day and overnight events appear on every overlapping day. Legacy all-day records with timed ends remain editable.
- The list endpoint adds `status`, `range_start`, and `range_end`. Range filters compare actual instants using SQLite julianday, including offset timestamps, and preserve the authenticated workspace boundary.

## Inbox

- The existing backend stores outgoing mail. Folder labels and the page description reflect drafts, queued, sending, sent, failed, and templates; the redesign does not invent incoming mail or unread state.
- A message list and reading pane show recipients, delivery state, timestamps, errors, and content. Narrow screens switch between the list and reader with a Messages back control.
- Compose saves a draft. Draft edit/delete, approval, queued-message delivery, and retrying eligible failed messages use supported endpoints. Approval, delivery, retry, and deletion have explicit confirmation with the affected scope. Delivery notices say queued rather than claiming successful delivery.
- Templates can be created, edited, deleted, and used to populate a draft. The old template edit placeholder is replaced with the existing update endpoint.
- Email search now reaches the repository and matches subject, recipient, and plain text body. Status, related-record filters, sort, counts, and pagination remain server-side.
- Plain text is rendered as text. Optional HTML uses a sandboxed iframe with a restrictive content policy; scripts and remote resources are blocked. Header fields and body text are not inserted directly into the app document.
- Sending continues to depend on the server's configured email provider. No SMTP settings were changed.

## Verification

Production frontend build, targeted lint, and `go test ./...` passed. HTTP regression cases cover calendar overlap at range boundaries, timezone offsets, status filters, email search/counts, and injection-shaped searches. Authenticated read-only requests against the running backend passed.

Desktop and mobile screenshots were reviewed. Browser fixtures verified all calendar views, multi-day events, status filters, all-day create/edit, compose validation and saving, approval/send confirmations, template editing/use, sandboxed HTML, search, and mobile list/reader navigation. Both pages passed page-overflow checks at 320, 390, 768, 1024, and 1440 pixels. Helper checks cover more than one page of calendar events.

Verification writes were intercepted in an isolated browser. No actual emails were sent or user CRM records created, edited, or archived during those checks. Temporary verification sessions were logged out.
