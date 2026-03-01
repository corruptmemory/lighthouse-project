# Shared AI Memory — Lighthouse Project

This file is committed to the repo and shared across machines. Claude Code
on each machine reads this for accumulated project knowledge. Machine-local
notes stay in `~/.claude/projects/.../memory/MEMORY.md`.

Last updated: 2026-03-01

## Project Status

- Early design phase — no production application code yet
- Prototype-1: AI agent + SQLite + SSE chat (2 tools: execute_sql, render_html)
- Prototype-2: persistent widgets as `.widget.md`, dashboards, data injection
- Prototype-3: two-page app — CRUD order management + AI dashboard with widgets
- Domain expert questions drafted but unanswered
- Implementation plan not yet created

## Branches

- `master` — base docs + prototype-1
- `jim/experiments` — adds prototype-2, prototype-3, updated learnings

## Key Docs

- `docs/architecture-principles.md` — 5 binding constraints
- `docs/technical-specifications.md` — schema, taxonomy, shipping rules, 6 modules
- `docs/vision-ai-driven-erp.md` — AI replaces ERP query/report/dashboard layer
- `docs/prototype-1-learnings.md` — core loop, Power BI case study, embedded Postgres idea
- `docs/prototype-2-learnings.md` — markdown-as-artifact, data injection, widget creation UX
- `docs/prototype-3-learnings.md` — two-page architecture, product autocomplete, IIFE fix
- `docs/research-erp-landscape.md` — SAP/NetSuite/Odoo/Shopify comparisons
- `docs/research-go-integrations.md` — Go SDK maturity for Stripe, Shopify, SAP
- `docs/domain-expert-questions.md` — awaiting business stakeholder answers
- `docs/schema-evolution-design.md` — blue-green DB migration, schema-as-code (in progress)
- `prototype-2-idea.md` — SQL visualizer for trust, dashboard vs widget creation UX

## Prototype-2 Architecture

- 4 tools: execute_sql, create_widget, list_widgets, create_dashboard
- Widgets: `.widget.md` files (markdown with ## SQL + ## Widget code blocks)
- Data injection: server re-executes SQL at render time, injects `window.WIDGET_DATA`
- Dashboards: `.dashboard.md` files referencing widget slugs, rendered as iframe grids
- Slug-based routing: `/api/widgets/{slug}`, `/api/dashboards/{slug}`
- Iframes as security boundary: `sandbox="allow-scripts"`
- Two SQL execution paths: pipe-delimited (for agent) vs structured (for SSE/widgets)
- `db.SetMaxOpenConns(1)` required for `:memory:` SQLite (pool creates separate DBs per connection)

## Prototype-3 Architecture

- Two pages, one binary: Dashboard (AI chat) at `/` and Orders (CRUD) at `/orders`
- Shared sidebar partial (`sidebar.html`), active nav state, shared `*sql.DB`
- Orders page: htmx master-detail, filter/sort/CRUD, product autocomplete
- Dashboard page: SSE streaming chat, SQL tool use, widget creation
- Product autocomplete: vanilla JS, `/api/products?search=`, debounced, arrow keys
- IIFE wrapping: `wrapScriptsInIIFE` in widgets.go fixes AI-generated top-level `return`
- Widget persistence: `.widget.md` on disk, auto-loaded on dashboard page load
- Ported from p2: session.go, tools.go, widgets.go, dashboards.go, agent.go
- AI Filter on Orders page: one-shot agent filters orders by natural language
  - `POST /api/orders/ai-filter` — sync request, returns HTML partial
  - Two tools: `execute_sql` + `return_order_ids` (structured stop signal)
  - Explanation via `HX-Trigger` header (htmx sideband pattern)
  - Fields/AI mode dropdown toggles between standard filters and AI input
  - Fixture data: 3 Staples B2B "Confirmed" orders with item overlaps
- AI Bulk Operations: propose-review-execute modal with transactional DML
  - `POST /api/orders/ai-bulk-ops` — agent proposes actions as JSON
  - `POST /api/orders/ai-bulk-ops/execute` — executes selected actions in one tx
  - Two tools: `execute_sql` + `propose_actions` (structured stop signal)
  - All-or-nothing: `db.BeginTx` + `defer tx.Rollback()`, commit only if all succeed
  - `executeDML()` whitelists UPDATE/INSERT/DELETE, blocks DDL keywords
  - Works end-to-end, needs UX polish (next iteration)
- Durable chat history: exchanges persisted to `chats/{session-id}/*.md`
  - `writeChatExchange()` writes numbered markdown files (## Prompt + ## Messages JSON)
  - `SessionStore.GetOrCreate()` restores message history from disk on reconnect
  - `SessionStore.SaveExchange()` persists new messages after each agent loop
  - Session recovery UI: JS fetches `/api/chat/sessions` on page load, shows
    recovery card(s) with prompt preview + exchange count + relative timestamp
  - "Continue" fetches `/api/chat/sessions/{id}/history`, restores visible chat
    (clean text only, no tool-use blocks), sets JS `sessionId` for routing
  - "New Conversation" dismisses card, restores welcome message

## Design Insights (confirmed across sessions)

- Widget creation should be user-driven, not AI-initiated. The agent answers
  questions with text + inline SQL + data. The user explicitly promotes
  answers to persistent widgets when they want to.
- Markdown-as-artifact is the right format. Human-readable, git-diffable,
  self-documenting, AI-native.
- The AI is the author; the Go server is the runtime. Cleanly separated.
- SQL transparency is the cheapest trust mitigation — show queries inline,
  let users inspect. The full SQL visualizer (hover to see intermediate
  results) is the deeper trust unlock, not yet built.
- Two-page architecture beats cramming everything into one page. Each page
  gets full width for its job. Shared database means zero sync overhead.
- AI-generated JS needs guardrails. Wrapping `<script>` in IIFE is a
  server-side defense against common LLM code generation antipatterns.
- One-shot agent pattern (no SSE, no session) is simpler for "query → result"
  UX. The `return_order_ids` tool gives a clean structured exit signal vs
  parsing free-text. Agent iteration limit needs headroom (15, not 10) —
  complex cross-table queries consume 10+ exploration steps.
- Propose-review-execute pattern for AI bulk operations: the agent proposes
  structured actions, human reviews via checkboxes, execution happens in one
  DB transaction. `defer tx.Rollback()` + conditional `tx.Commit()` is the
  cleanest Go pattern for all-or-nothing semantics.
- Each prototype builds on the prior one (cumulative feature set).

## Schema Evolution (moved to separate repo)

The DB migration / schema evolution work now lives in `../db-space`. The
design discussion in `docs/schema-evolution-design.md` is kept for historical
context but is no longer maintained here. The key insight that still applies:
the agent's system prompt contains the DDL, so schema changes in db-space
must propagate to agent context in this project.

## Architecture Decisions

- Exec bridge pattern for proprietary protocols (JSON over stdin/stdout)
- Go for all HTTP integrations (Stripe, Shopify GraphQL, SAP OData)
- PostgreSQL for production; SQLite for prototyping
- Embedded Postgres (`fergusstrange/embedded-postgres`) planned for demos/tests
- No CGo ever
- Model is currently claude-sonnet-4; should become a `--model` CLI flag

## Stack

- Go, chi/v5, go-flags, modernc.org/sqlite, anthropic-sdk-go
- TOML config (production), CLI flags
- templ templates + htmx (production); static HTML/JS (prototypes)
