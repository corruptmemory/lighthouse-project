# Prototype 2 — Learnings & Notes

## What This Prototype Proves

The AI agent is the **author**; the Go server is the **runtime**. These are
cleanly separated concerns:

- The agent creates widget markdown files (expensive, creative, one-time)
- The server parses markdown, re-executes SQL, injects fresh data, serves HTML
  (cheap, mechanical, every page load)

No AI is involved at render time. Widgets are static artifacts that produce
live data.

## What Worked

### Markdown-as-artifact

Widget markdown files (`.widget.md`) with separate `## SQL` and `## Widget`
fenced code blocks are the core innovation. Benefits realized in practice:

- **Human-readable**: You can open a widget file, read the SQL, understand
  what it does, and edit it by hand. We verified this — editing the SQL in
  the markdown file and reloading produces updated results.
- **Git-diffable**: The files are plain text. Version control works naturally.
- **Self-documenting**: The `## Context` section explains *why* the widget
  exists. The AI-generated artifact IS the documentation.
- **AI-native**: The agent produces markdown naturally. No fighting the model
  into a JSON schema.

### Data injection via `window.WIDGET_DATA`

The Go server injects `<script>window.WIDGET_DATA = {...};</script>` before
the `<body>` tag. The widget HTML reads this instead of hardcoding values.
This means:

- Widgets show **fresh data on every load** — SQL is re-executed at render time
- The same widget HTML works regardless of when or how often data changes
- The pattern is simple enough that the AI gets it right consistently

### Inline SQL display with syntax highlighting

SQL queries appear inline in chat inside collapsible expanders. When expanded,
you see syntax-highlighted SQL (keywords in blue, strings in green) plus a
compact data table with the results. This directly addresses the "trust and
confidence" problem from the vision doc.

Key UX learning: the first implementation showed SQL **twice** — once in the
raw expander and once in a separate highlighted block. User feedback: "too
much duplication." The fix was to upgrade the expander's body in-place with
the rich SQL display. One location, collapsed by default, click to inspect.

### Widget persistence across restarts

Stopped the server, restarted it — widgets loaded from disk automatically.
The `loadSavedWidgets()` function on the frontend fetches
`GET /api/widgets` and populates the sidebar. This was the whole point of
prototype-2 and it works cleanly.

**Gotcha — SQLite `:memory:` and connection pools:** The default DB path is
`:memory:`, which creates a private database per connection. Go's
`database/sql` manages a pool, so `InitDB` seeds tables on one connection but
later HTTP requests may get a different connection with an empty DB. Fix:
`db.SetMaxOpenConns(1)` forces all operations through one connection. This is
the standard fix for in-memory SQLite with Go and is fine for a single-user
prototype. For production, use an on-disk DB or `file::memory:?cache=shared`.

### Agent self-correction on SQL errors

The agent attempted a query referencing a column that didn't exist
(`quantity`), got an error, examined the schema, and retried with the correct
column (`stock_on_hand`). This error-recovery loop was already working in
prototype-1 but continues to work well with the new tool definitions.

### Dashboard composition

`create_dashboard` writes a `.dashboard.md` file referencing widget slugs.
The server renders an HTML page with a CSS grid of iframes pointing to
`/api/widgets/{slug}`. Each iframe independently loads its widget with fresh
data. Simple, composable, works.

## Architecture Decisions Made

### Separate SQL execution paths

- `executeSQL()` returns pipe-delimited text — what the AI agent reads
- `executeSQLStructured()` returns `QueryResult{Columns, Rows}` — feeds
  SSE events and data injection

The agent doesn't need structured data; it needs readable text. The browser
and widget renderer need structured data. Keeping these separate avoids
complicating either path.

### Slug-based routing

`/api/widgets/stock-levels-by-brand` instead of `/api/widgets/w3`. URLs are
human-readable, stable across restarts, and correspond directly to filenames
on disk (`stock-levels-by-brand.widget.md`).

### Iframes as security boundaries

Widget HTML runs in `<iframe sandbox="allow-scripts">`. The iframe has no
access to the parent page's DOM, cookies, or network context. This is the
right security model for AI-generated content — treat it as untrusted by
default. The iframe went from a layout hack (2005) to a security primitive
(2025).

## Discussions for Future Prototypes

### Auth context in widget iframes

When users and roles arrive, widgets may need to know who's viewing them
(e.g., to filter data by department). The recommended approach: expand the
data injection pattern. The server already injects `window.WIDGET_DATA`; it
can also inject `window.WIDGET_CONTEXT = {user, role, permissions}`. This
keeps the iframe sandboxed (no cookie access needed) while providing
user-specific context.

### Widget live updates

Current behavior: widgets show data as of page load. Future options discussed
(in order of complexity):

1. **Polling** — widget JS calls `fetch('/api/widgets/slug/data')` on a timer
2. **SSE per widget** — server pushes when underlying data changes
3. **Database-driven push** — Postgres `LISTEN/NOTIFY` triggers SSE events

For the prototype stage, polling is sufficient. SSE becomes worthwhile when
multiple users view the same dashboard.

### Widget embeddability

Widgets are self-contained HTML documents. Future distribution options:

- **Iframe embed snippets** — `<iframe src="https://erp.example/api/widgets/slug">`
- **OEmbed protocol** — paste a widget URL into Slack/Notion, get an inline preview
- **Static HTML export** — snapshot data at a point in time, serve without the server
- **PDF generation** — headless browser renders widget to PDF for reports

The markdown-as-artifact pattern makes all of these straightforward because
the widget format is well-defined and parse-able.

## What Needs Work

### Trust layer is still shallow

Inline SQL display is a start, but the full SQL visualizer from
`prototype-2-idea.md` — hover over table names to see sample data, hover
over JOINs to see intermediate results — hasn't been built yet. That's the
real trust unlock for non-technical users.

### Widget editing / re-prompting

Users can't yet say "update the stock chart to also show reorder points."
The agent would need to load the existing widget markdown, understand its
current state, and produce a modified version. The markdown format makes
this feasible (feed the old markdown to the agent as context), but the
workflow isn't implemented.

### No widget de-duplication

If you ask for "stock levels by brand" twice, you get two widgets. The
system prompt tells the agent to check `list_widgets` first, and it
sometimes does, but there's no enforcement. Could add slug collision
detection in `create_widget`.

### Session state still in-memory

Conversation history dies on restart. Widget persistence was the priority;
session persistence is deferred.

### Widget creation should be user-driven, not AI-driven

The system prompt says "proactively offer visualizations" but the agent
is unpredictable about it. For example, "Which products are we at risk of
running out of?" got a thorough text answer with inline SQL + data table
but no widget — even though a low-stock visualization would be useful.

**The better model**: Don't try to make the AI guess. The agent's job is to
answer the question well (text + inline SQL + data). The *user* decides
whether to promote that answer to a persistent widget. Two paths:

1. **Explicit intent**: "Create a widget that shows low-stock products" —
   predictable, always produces a widget.
2. **Post-hoc promotion**: After seeing an answer, a "Make a widget from
   this" button captures the context (question + working SQL + data shape)
   and feeds it back to the agent to produce a `.widget.md`.

Benefits:
- Predictable behavior — users learn the vocabulary ("create a widget...")
- No unwanted widget clutter from overeager AI
- De-duplication solved naturally (user-initiated = intentional)
- The text+SQL answer is often exactly what's needed; no widget required

The system prompt should shift from "proactively offer visualizations" to
"always answer thoroughly with data; create widgets only when explicitly
asked."

## Technical Debt

- `interface{}` should be `any` throughout (Go 1.18+ style)
- No favicon (404 on every page load)
- Markdown rendering in app.js is minimal — no tables, no nested lists
- Widget iframe auto-resize is fragile (sandbox blocks `contentDocument` access)
- No error page when a widget's SQL fails at render time (just a 500)
- Dashboard layout is fixed 2-column grid — no user-customizable layout

## File Inventory

```
prototype-2/
  build.sh              # go build -o bin/prototype-2 .
  main.go               # +--widgets-dir flag, new routes
  agent.go              # 4 tools: execute_sql, create_widget, list_widgets, create_dashboard
  tools.go              # executeSQL + executeSQLStructured, widget/dashboard input structs
  widgets.go            # ParsedWidget, parse/write/load/list/render functions
  dashboards.go         # ParsedDashboard, parse/write/load/render functions
  db.go                 # SQLite schema DDL + seed data + SetMaxOpenConns(1) for :memory:
  handlers.go           # +HandleGetWidgetBySlug, +HandleDashboard, +HandleListWidgets
  session.go            # In-memory session store (unchanged from p1)
  systemprompt.go       # Rewritten — window.WIDGET_DATA pattern, 4 tool descriptions
  static/
    index.html          # +Dashboards nav, widget panel loads saved widgets
    app.js              # +sql_result handler, +highlightSQL, +buildResultTable, +loadSavedWidgets
    app.css             # +sql-result-table styles, +sql keyword highlighting, +dashboard-link
  widgets/              # Runtime directory — .widget.md and .dashboard.md files (gitignored)
```

## Dependencies

Same as prototype-1, minus unused `invopop/jsonschema`:

- `anthropic-sdk-go` — Anthropic API client
- `chi/v5` — HTTP router
- `go-flags` — CLI argument parsing
- `modernc.org/sqlite` — Pure Go SQLite (no CGo)
