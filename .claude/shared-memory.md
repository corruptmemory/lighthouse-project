# Shared AI Memory — Lighthouse Project

This file is committed to the repo and shared across machines. Claude Code
on each machine reads this for accumulated project knowledge. Machine-local
notes stay in `~/.claude/projects/.../memory/MEMORY.md`.

Last updated: 2026-02-28

## Project Status

- Early design phase — no production application code yet
- Prototype-1: AI agent + SQLite + SSE chat (2 tools: execute_sql, render_html)
- Prototype-2: persistent widgets as `.widget.md`, dashboards, data injection
- Domain expert questions drafted but unanswered
- Implementation plan not yet created

## Branches

- `master` — base docs + prototype-1
- `jim/experiments` — adds prototype-2, updated learnings, prototype-2-idea.md

## Key Docs

- `docs/architecture-principles.md` — 5 binding constraints
- `docs/technical-specifications.md` — schema, taxonomy, shipping rules, 6 modules
- `docs/vision-ai-driven-erp.md` — AI replaces ERP query/report/dashboard layer
- `docs/prototype-1-learnings.md` — core loop, Power BI case study, embedded Postgres idea
- `docs/prototype-2-learnings.md` — markdown-as-artifact, data injection, widget creation UX
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
