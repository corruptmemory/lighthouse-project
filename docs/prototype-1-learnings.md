# Prototype 1 — Learnings & Notes for Prototype 2

## What Worked

### The core loop is sound
Browser → POST /api/chat → SSE → agent loop → Anthropic API → tool dispatch
→ results fed back → next API call → final response + widgets. The
non-streaming approach (complete `Messages.New()` calls, not streaming deltas)
keeps the code dead simple while still giving the user real-time progress via
SSE events between API round-trips.

### Agent iterates naturally
For the shipping orders question, the agent made 4 SQL queries before rendering
a widget — exploratory queries to understand the data shape, then targeted
queries for the visualization. This is exactly the behavior that makes "AI
replaces query builders" work: the AI explores the schema iteratively rather
than needing a perfect query on the first try.

### Domain context in system prompt works well
Embedding the DDL schema + domain knowledge (brand names, SKU conventions,
business rules, design tokens) in the system prompt produces accurate,
contextual responses. The agent correctly uses QUADRUM, VOLTERRA, Drehgriffel
with proper casing.

### Widget rendering via sandboxed iframes
Agent-generated HTML/CSS/JS runs safely in `<iframe sandbox="allow-scripts">`
with no access to the parent page. The widgets look good using inline CSS with
the design tokens provided in the system prompt.

## What Needs Work (Prototype 2 Ideas)

### 1. SQL as a Trust Touchpoint

The collapsible `⚡ SQL Query` sections work but are hidden by default. For
trust, SQL should be more prominent:

- Show SQL inline (not collapsed) with syntax highlighting
- Let the user **edit and re-run** SQL — bridges "just works" users and power
  users who want to verify
- This is the cheapest mitigation for the trust problem (vision doc hard
  problem #1): users who care can read the SQL, users who don't can ignore it

### 2. Widgets as Markdown Documents

Instead of storing widget HTML as blobs in a `sync.Map`, persist widgets as
markdown documents:

```markdown
# Revenue by Brand — February 2026

Generated: 2026-02-26 14:32
Session: abc123

## Context
User asked: "Show me revenue breakdown by brand"
The agent queried order totals grouped by brand_mix.

## SQL
​```sql
SELECT brand_mix, SUM(total) as revenue, COUNT(*) as orders
FROM orders WHERE order_date >= '2026-02-01'
GROUP BY brand_mix ORDER BY revenue DESC
​```

## Widget
​```js
// rendering code here
​```
```

Benefits:
- **Human-readable** — a person can read the markdown, understand what the
  widget does, spot errors in the SQL, and edit it
- **Versionable** — git diff on markdown is readable
- **Debuggable** — the context section explains *why* the widget exists
- **AI-native** — LLMs produce markdown naturally; you work with the grain of
  how the AI generates output rather than fighting it into a widget-config
  JSON schema
- **Editor-friendly** — a "widget editor" is really just a markdown editor with
  a live preview pane

The AI-generated artifact *is* the documentation.

### 3. Model Selection Trade-offs

Currently wired to `claude-sonnet-4-20250514`. Options:

| Model | Speed | SQL Accuracy | Cost | Notes |
|-------|-------|-------------|------|-------|
| Opus 4.6 | Slowest | Highest | Highest | Best for complex multi-table JOINs, aggregate queries |
| Sonnet 4 | Medium | Good | Medium | Current choice — good balance for a prototype |
| Haiku 4.5 | Fastest | Adequate? | Lowest | Untested — risk of more SQL errors |

Trade-off: faster/cheaper models may mess up SQL more regularly, but the cost of
a wrong business decision from bad SQL could far exceed the cost savings. For a
prototype, Sonnet is a reasonable middle ground. Consider:

- Making the model a `--model` CLI flag so it's easy to experiment
- Using a smarter model for the first query (schema exploration) and a faster
  model for follow-ups
- Letting the user choose: "fast mode" vs "accurate mode"

### 4. Database Migrations (Flag Planted)

The utter bane of sysadmin/DBAs. Ideas deferred to much later in the
development plan, but the pain point is acknowledged: schema evolution over time
needs to be handled gracefully. This is especially interesting in an AI-driven
system where the agent's system prompt contains the DDL — schema changes need to
propagate to the agent's context, and historical widgets may reference
columns/tables that no longer exist.

## Real-World Comparison: Power BI at Brighthouse Financial

During prototype-1 development, we examined Power BI dashboards used by
Brighthouse Financial for HPC compute cost management. These dashboards track
actuarial/financial model runs (Shield Market Model, World Market Modeling)
across compute clusters, showing cost, core-hour utilization, and efficiency.

This is a textbook example of every problem the vision doc identifies.

### What the Power BI dashboard shows

- **KPI cards**: Request Core Hours (5.91M), Hedge Core Hours (1.77M), Unused
  (4.14M), Efficiency (29.9%)
- **Total Cost**: $2.54M across 2K projects
- **Visualizations**: Cost Trend by Cluster (line chart), Cost by Cluster
  (donut chart)
- **Data table**: Business Name, Owner, Project Name, Allocation, Cluster,
  Total with Infra Cost, Request/Usage Core Hours, Efficiency
- **Filters**: Business Process checkboxes, date range picker

Filtering to a specific business process changes efficiency from 29.9% to 91.2%
— some processes are massively over-provisioning compute.

### How the five vision doc problems manifest here

**1. The customization layer exists.** Someone — a Power BI developer, a data
engineer, a consulting team — built this dashboard. That required understanding
the data model, writing DAX queries (Power BI's proprietary query language),
designing the layout, setting up the data pipeline, and maintaining it as
requirements change. This is the entire "customization layer" the vision doc
says AI can replace.

**2. The super-user bottleneck is visible.** If someone on a call says "Can we
also see cost per core-hour broken out by model variant?" — nobody can answer
that live. Someone files a request, a Power BI person modifies the DAX, updates
the dashboard, and maybe next week you see the answer. With the prototype-1
approach, you type the question and get the answer during the meeting.

**3. The fixed widget vocabulary.** This dashboard has exactly 4 KPI cards,
1 line chart, 1 donut chart, 1 data table. Those were chosen at design time.
What if the right visualization is a Sankey diagram of cost flow between
clusters? A heatmap of efficiency by week and model? A waterfall chart showing
where the 70% waste is coming from? Power BI can do those, but someone has to
build them. The AI generates whatever fits the question.

**4. The trust problem is real.** Look at those numbers: $2.5M total cost,
29.9% efficiency. People make real decisions based on these. If an AI generated
that 29.9%, would they trust it? This is exactly hard problem #1 from the
vision doc. The "show the SQL" approach is the right first step — a finance
person can audit `SELECT SUM(request - usage) / SUM(request) FROM ...` and
verify the math.

**5. Proactive insight is the unlock.** 29.9% efficiency across all projects vs
91.2% filtered down — that gap tells a story. An AI agent could surface this
proactively: "I notice your Shield Market Model (GMMA) projects are running at
15% efficiency — they're requesting 4x the core hours they use. Want me to show
you the worst offenders?" No fixed dashboard does this.

### The gap between prototype-1 and replacing this dashboard

The core loop already works (ask question → SQL → visualization). What's
missing:

1. **Real database connection** — PostgreSQL or a data warehouse instead of
   SQLite seed data
2. **Trust/verification layer** — SQL display, edit-and-rerun, audit trail
3. **Widget persistence** — the markdown document approach, so dashboards
   survive across sessions
4. **Schema awareness** — the agent needs the DDL in its system prompt, which
   means schema changes must propagate (the migration problem)

### The strategic observation

Power BI is a ~$10B/year Microsoft business built on the premise that business
users need a specialized tool + specialized people to answer questions about
their own data. The AI-driven ERP thesis says that premise is becoming false.
Prototype-1 — an agent that takes natural language, writes SQL, and generates
visualizations — is a direct challenge to that premise. The Brighthouse
dashboard is what "the old way" looks like: rigid, requires specialists to
modify, and can only answer questions it was pre-designed to answer.

## Technical Debt

- Session and widget storage is in-memory — lost on restart
- No error retry on transient API failures
- `invopop/jsonschema` dependency is unused (tool schemas are inline maps)
- The `interface{}` types should be `any` (Go 1.18+ style)
- No favicon (404 on every page load)
- Markdown rendering in app.js is minimal — no tables, no nested lists

## File Inventory

```
prototype-1/
  build.sh              # go build -o bin/prototype-1 .
  main.go               # CLI flags, chi router, server startup
  agent.go              # Anthropic agent loop (non-streaming)
  tools.go              # execute_sql + render_html tool executors
  db.go                 # SQLite schema DDL + seed data
  handlers.go           # POST /api/chat (SSE), GET /api/widgets/{id}
  session.go            # In-memory session store
  systemprompt.go       # System prompt with DDL + domain context
  static/
    index.html          # Two-panel layout (sidebar + chat + widgets)
    app.js              # SSE parsing, chat rendering, widget iframes
    app.css             # Design tokens from mockup
```

## Dependencies

- `anthropic-sdk-go` v1.26.0 — Anthropic API client
- `chi/v5` — HTTP router
- `go-flags` — CLI argument parsing
- `modernc.org/sqlite` — Pure Go SQLite (no CGo, per architecture principles)
- `invopop/jsonschema` — Unused, can remove
