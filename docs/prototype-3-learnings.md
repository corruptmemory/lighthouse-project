# Prototype-3 Learnings

## What Prototype-3 Proves

Prototype-3 merges the CRUD order management from a standalone UI prototype
with the AI chat + widget system from prototype-2 into a single two-page
application. The thesis: traditional CRUD screens and AI-powered analytics
coexist naturally in the same binary, sharing the same database.

## Architecture: Two Pages, One App

```
GET /           → Orders page (master-detail, htmx CRUD)
GET /dashboard  → Dashboard page (AI chat + widget panel)
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

## What's Still Rough

- Widget iframe height is fixed; should auto-resize based on content
- No error recovery if the AI generates invalid SQL in a widget
- Chat history is in-memory per session; lost on server restart
- The sidebar is duplicated in two templates (layout.html, dashboard.html)
  rather than shared via template composition — acceptable for prototyping

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
- `handlers.go` — CRUD handlers + chat/widget handlers + product search
- `templates.go` — Go html/template loading and rendering
- `systemprompt.go` — Dynamic system prompt with CRUD context
- `build.sh` — Single build entry point
- `templates/layout.html` — Orders page (master-detail)
- `templates/dashboard.html` — Dashboard page (chat + widgets)
- `templates/order-form.html` — Order form with product autocomplete
- `templates/order-detail.html` — Read-only order detail view
- `templates/order-table.html` — Order table rows (htmx partial)
- `static/app.js` — Chat/SSE/widget JS + autocomplete component
- `static/app.css` — Full stylesheet (CRUD + chat + widgets + autocomplete)
