# Prototype-3 Learnings

## What Prototype-3 Proves

Prototype-3 merges the CRUD order management from a standalone UI prototype
with the AI chat + widget system from prototype-2 into a single two-page
application. The thesis: traditional CRUD screens and AI-powered analytics
coexist naturally in the same binary, sharing the same database.

## Architecture: Two Pages, One App

```
GET /           → Dashboard page (AI chat + widget panel)
GET /orders     → Orders page (master-detail, htmx CRUD)
```

Both pages share:
- Same Go binary, same `*sql.DB` (in-memory SQLite for prototyping)
- Same sidebar navigation (active state changes per page)
- Same CSS design tokens (warm neutrals, brand colors)

This is simpler than a SPA. Each page is a full HTML document with its own
template. No client-side routing, no state management library, no build step.

## Key Findings

### Shared database is the killer feature

Create an order on the Orders page, navigate to Dashboard, ask the AI "show
me the most recent order" — it finds it immediately. No sync, no webhooks,
no eventual consistency. Just one database. This validates the core
architectural bet: the AI agent and the CRUD UI are views onto the same data.

### Product autocomplete over dropdowns

Real catalogs have too many SKUs for a `<select>` dropdown. The autocomplete
component (vanilla JS, no library) provides:
- Debounced search against `/api/products?search=`
- Arrow key navigation, Enter to select, Escape to close
- Hidden `<input>` for product_id, visible text input for UX
- Works with htmx dynamic form loading via `htmx:afterSettle` listener

This pattern generalizes to any entity picker (customers, addresses, etc.).

### AI-generated JS needs server-side guardrails

The AI agent generates widget HTML including `<script>` blocks. A common
LLM antipattern: writing `return` at the top level of a script (as if inside
a function). Browsers reject this as "Illegal return statement".

Fix: `wrapScriptsInIIFE()` in `widgets.go` wraps every inline `<script>`
body in `(function(){ ... })();` before serving. The AI doesn't need to know
about it. All current and future widgets automatically benefit.

This is the right layer for the fix — the server is the runtime, the AI is
the author. The runtime should be tolerant of the author's quirks.

### Widget persistence across restarts

Widgets are `.widget.md` files on disk. The dashboard JS calls
`/api/widgets` on load and renders any saved widgets. After a server restart
(fresh in-memory DB with seed data), the widget SQL re-executes against the
new database and renders current results. The widget *definition* is stable;
the *data* is always live.

### SSE streaming with POST

The chat uses `fetch()` with `ReadableStream` rather than `EventSource`
because the endpoint is POST (sends message body). This is the same approach
as prototype-2, confirmed to work well across the two-page architecture.

### AI Filter: one-shot agent for natural language queries

The AI Filter feature adds a "Fields / AI" mode toggle to the order filter bar.
In AI mode, users type natural language like "Processing orders with items that
overlap Staples' large Confirmed orders" and the agent finds matching rows.

Architecture: a one-shot agent loop (`runFilterAgentLoop`) with just two tools:
`execute_sql` (read-only exploration) and `return_order_ids` (structured stop
signal). The stop-signal pattern avoids parsing free-text — the agent calls
`return_order_ids({order_ids: [...], explanation: "..."})` and the loop exits
immediately. The explanation is delivered to the UI via an `HX-Trigger` response
header (htmx sideband pattern), not in the HTML body.

Key learnings:
- Agent iteration limit needs headroom: 15, not 10. Complex cross-table queries
  use 10+ SQL explorations before the agent is confident.
- The `from:find input` htmx trigger pattern doesn't match a `<div>` — clearing
  the AI filter must use `htmx.trigger(body, 'refreshTable')`.

### AI Bulk Operations: propose-review-execute with transactions

The bulk operations modal completes the filter-then-act workflow. Users type a
natural language request (e.g., "Divert overlapping items from Processing orders
to Staples orders"), an AI agent proposes concrete DML actions, the user reviews
via checkboxes, and executes in a single database transaction.

Two endpoints:
- `POST /api/orders/ai-bulk-ops` — agent explores DB, returns proposed actions as JSON
- `POST /api/orders/ai-bulk-ops/execute` — runs selected actions in one transaction

The agent uses the same one-shot pattern as the filter agent, with `execute_sql`
for exploration and `propose_actions` as the structured stop signal. Each
proposed action has a human-readable description and one SQL statement.

Transaction semantics: all-or-nothing via `db.BeginTx`. If action 3 of 5 fails,
the entire transaction rolls back, and the results UI shows which action failed
(red), which succeeded before it (green, but rolled back), and which were
skipped (grey). `defer tx.Rollback()` handles the rollback automatically unless
`tx.Commit()` is called first.

DML safety: `executeDML()` whitelists UPDATE/INSERT/DELETE and blocks DDL
keywords (DROP, ALTER, CREATE, ATTACH, DETACH). The human-in-the-loop checkbox
selection is the primary trust boundary.

Current state: works end-to-end but operates at a "brutally raw" level — the
agent generates valid SQL with correct descriptions, transactions commit/rollback
properly, but the UX needs polish (action grouping, undo, progress feedback,
partial selection defaults, etc.).

### Durable chat history with session recovery

Chat exchanges are persisted to `chats/{session-id}/*.md` files — same
markdown-as-artifact pattern as widgets. Each exchange file has a `## Prompt`
section and a `## Messages` JSON block containing the full Anthropic message
history for that turn.

On page load, JS fetches `/api/chat/sessions`. If previous sessions exist, a
recovery card replaces the welcome message, showing the first prompt as a
preview, the exchange count, and a relative timestamp. "Continue" fetches
`/api/chat/sessions/{id}/history`, restores the visible chat (clean text
responses only — tool-use blocks are filtered out), and sets the JS `sessionId`
so subsequent messages route to the correct server-side session with full AI
context. "New Conversation" dismisses the card and starts fresh.

Key design choices:
- The server extracts prompt + last assistant text per exchange for the UI
  summary, using a marshal/unmarshal round-trip to handle the Anthropic SDK's
  union types for message content blocks.
- The recovery card is built entirely with safe DOM methods (`createElement`,
  `textContent`) — no `innerHTML` with user data.
- Session restoration is two-phase: the JS shows visible history immediately
  (text only), while `SessionStore.GetOrCreate()` loads the full message
  history (including tool_use blocks) server-side for AI context.

### Sidebar template extraction

The sidebar was duplicated across `layout.html` and `dashboard.html`. Now
extracted into `sidebar.html` as a shared Go template partial. Both pages
use `{{ template "sidebar" . }}` with a `.Page` field for active nav state.

## What's Still Rough

- Widget iframe height is fixed; should auto-resize based on content
- No error recovery if the AI generates invalid SQL in a widget
- Bulk operations modal needs UX refinement: action grouping, progress
  feedback, better default selection, undo capability

## Files

### Ported from prototype-2 (verbatim)
- `session.go` — Session + SessionStore
- `tools.go` — SQL execution, tool input types
- `widgets.go` — Widget markdown parse/write/load/render (+ IIFE fix)
- `dashboards.go` — Dashboard markdown parse/write/render
- `agent.go` — Anthropic agent loop, 4 tool definitions

### New for prototype-3
- `db.go` — SQLite schema + seed data (orders, products, subscriptions)
- `main.go` — chi router, Options struct, handler wiring
- `handlers.go` — CRUD handlers + chat/widget handlers + product search + session recovery
- `templates.go` — Go html/template loading and rendering
- `systemprompt.go` — Dynamic system prompt with CRUD context
- `chats.go` — Chat persistence: write/load/list exchanges, session summaries
- `build.sh` — Single build entry point
- `templates/layout.html` — Orders page (master-detail)
- `templates/dashboard.html` — Dashboard page (chat + widgets)
- `templates/sidebar.html` — Shared sidebar partial
- `templates/order-form.html` — Order form with product autocomplete
- `templates/order-detail.html` — Read-only order detail view
- `templates/order-table.html` — Order table rows (htmx partial)
- `static/app.js` — Chat/SSE/widget JS + autocomplete + session recovery
- `static/app.css` — Full stylesheet (CRUD + chat + widgets + autocomplete + recovery cards)
