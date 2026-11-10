# CRM design continuation

Companies, Contacts, Leads, and Deals now share the workspace's warm surfaces, orange actions, compact headings, and sales navigation. Shared components live in `web/src/CrmUI.tsx`, scoped styles in `web/src/Crm.css`, and currency/export helpers in `web/src/crm-data.ts`.

## Views and behavior

- Companies and Contacts have tables and redesigned cards with keyboard-accessible names, selection, edit/archive actions, and direct contact/website links. Website links accept HTTP/HTTPS URLs. An unsupported company industry dropdown was removed; the existing server-backed search still searches industry.
- Each page labels the matching total separately from page-level summaries. Deals group financial values by currency, retain cents, and show weighted values using probability. Lead follow-up counts use the local calendar date and exclude won/lost leads.
- All views, including cards and boards, have working pagination and page-size controls. Boards explicitly describe their page-level scope. Deal stage names include the pipeline when multiple pipelines exist.
- Search resets pagination, lead source/priority filters reach their existing backend parameters, and late search responses cannot overwrite newer results. Loading, empty, and failed requests have separate states with retry controls.
- Export page downloads the current loaded records as CSV and escapes embedded quotes and formula prefixes. Selection is cleared when filters or pagination change. Bulk archive uses bounded batches, reports successes/failures separately, and retains failed selections for retry. Archive confirmations no longer promise an unavailable undo action.
- Editors use native dialogs with accessible titles, browser focus handling, Escape dismissal, and close buttons disabled while saving. Existing record/create URL entry points remain. Board creation resets old form values and preselects the requested status or pipeline/stage. Board moves also update the deal's pipeline consistently.
- The existing lead-conversion endpoint is now reachable from the lead editor, with named pipeline/stage choices and input validation replacing raw identifier fields. Conversion was not submitted against user records during verification.

## Verification

- TypeScript and production build pass.
- Targeted lint reports no errors; React effect/memoization warnings remain in the existing page loading/state patterns.
- Server-rendering checks pass for all four page headings, active sales navigation, and loading states. Shared-component checks cover cards, unsafe website URLs, pagination bounds, and error/retry states.
- Helper checks pass for currency precision, mixed-currency grouping, CSV quote/formula escaping, and lead source/priority query forwarding.
- Both local servers respond successfully: frontend at `http://localhost:5173`, backend at `http://localhost:8080`.
- Browser visual and interaction verification remains outstanding because the browser access host is unavailable. No user CRM records were changed during verification.
