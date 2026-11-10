# Agent Console

The console at `/agent-console` uses the shared CRM palette and a tool-oriented layout: searchable library, arguments/schema editor, result inspector, raw JSON-RPC requests, and request history. It connects explicitly using the current workspace session; connecting does not execute a CRM write tool.

## Behavior

- Initializes the local stateless MCP endpoint, negotiates its protocol version, sends the initialized notification, and follows tool-catalog pagination.
- Shows actual server metadata, tool count, request durations, and outcomes.
- Distinguishes read tools and write actions using supplied annotations or the existing tool-name vocabulary.
- Checks object arguments, required fields, primitive types, enums, and basic bounds before sending. The server remains authoritative for complete schema and business validation.
- Reports HTTP, JSON-RPC, and tool-level failures; a tool reporting `isError` is displayed as failed.
- Keeps up to 100 request/response pairs in component memory. History is cleared when the page is unmounted and is never persisted to local storage.
- Supports copying results, formatting JSON, Ctrl/Cmd+Enter execution, tool filtering, catalog refresh, and inspecting prior requests.
- Uses response-ID checks, request timeouts, and cleanup on unmount. Expired sessions offer a sign-in link.

## Validation

The production frontend build and targeted lint pass. Rendering checks covered disconnected, selected-tool, pending, successful, and failed states; history; escaped output; and stylesheet coverage. Transport checks covered malformed arguments, JSON-RPC errors, tool errors, expired sessions, mismatched IDs, plain-text HTTP failures, and network failures.

A live session through the frontend proxy successfully initialized, discovered 77 tools, and executed the read-only `companies.list` tool. No CRM write tools were executed during verification. Browser visual and interaction checks remain unavailable because the browser preview bridge is not connected.
