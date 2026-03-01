# Retrospective: What a Complete Spec Would Have Looked Like

**Session date:** 2026-03-01
**What happened:** Two phases — (1) implement session recovery UI for durable
chat history, (2) design discussion about structured agent tools replacing raw
SQL.

This retrospective illustrates what you could have handed me at the start of
the session to get autonomous execution with no back-and-forth. The goal is
to help you build the skill of writing specs that decompose cleanly into
discrete, executable tasks.

---

## Phase 1: Session Recovery UI (implementation)

The plan you gave me was already excellent — it was the output of a planning
session and I executed it with minimal questions. Here's what made it work,
and what a slightly tighter version looks like:

### What the spec got right

- **Files to modify** — explicitly listed, with function signatures
- **Data types** — Go structs defined with JSON tags
- **Endpoint contracts** — HTTP method, path, request/response shapes
- **JS behavior** — step-by-step flow (fetch → check → render → attach events)
- **Verification criteria** — numbered test scenarios

### A "zero-questions" version

```markdown
# Session Recovery UI

## Task 1: Backend — chats.go
Add these types and functions:

    type ChatSessionInfo struct { ... }     // as specified
    type ChatExchangeSummary struct { ... } // as specified
    func listChatSessions(dir string) ([]ChatSessionInfo, error)
    func loadChatSummaries(dir, sessionID string) ([]ChatExchangeSummary, error)

listChatSessions: read subdirs of dir, count .md files per session,
read first file's ## Prompt for preview, sort by dir mod time descending.

loadChatSummaries: read all .md files in session dir (sorted), extract
prompt text and last assistant text message (skip tool_use blocks).

## Task 2: Backend — handlers.go + main.go
Add chatsDir string field to Handlers struct (from opts.ChatsDir).
Two new handlers:
  GET /api/chat/sessions        → HandleListSessions  → JSON []ChatSessionInfo
  GET /api/chat/sessions/{id}/history → HandleSessionHistory → JSON []ChatExchangeSummary
Wire routes after existing POST /api/chat.

## Task 3: Frontend — app.js
After chatInput.focus(), call checkForSessions():
  - Fetch GET /api/chat/sessions
  - If empty, do nothing
  - If sessions exist, replace .message.assistant welcome div with:
    - Recovery card per session: truncated first_prompt (80 chars),
      exchange count, relative timestamp, "Continue" button
    - "New Conversation" button below all cards
  - Continue: fetch /api/chat/sessions/{id}/history, set sessionId,
    clear chat, render exchanges via appendMessage + renderMarkdown
  - New Conversation: remove cards, restore original welcome HTML

## Task 4: Frontend — app.css
Recovery card styles: .session-recovery, .session-recovery-card,
.session-recovery-preview, .session-recovery-meta, .session-recovery-actions.
Match existing chat bubble aesthetic (bg-card, border, radius).

## Task 5: Build + browser test
./build.sh must pass.
Start server with existing chats/ dir, verify in browser:
  - Recovery card appears with correct preview
  - Continue loads history (text only, no tool blocks)
  - New Conversation restores welcome message
  - Page reload re-shows recovery card
```

### Why this works for autonomous execution

1. **Each task is independent enough to verify in isolation.** Task 1 is pure
   Go with no HTTP. Task 2 is thin HTTP wiring. Task 3 is pure JS. Task 4 is
   CSS. Task 5 is integration.

2. **No ambiguous decisions left.** The spec says "truncated to 80 chars",
   "sort by mod time descending", "skip tool_use blocks". These are the
   micro-decisions that cause back-and-forth when left open.

3. **The verification step is concrete.** "Recovery card appears" is testable.
   "Works correctly" is not.

---

## Phase 2: Structured Agent Tools (design discussion)

This phase was inherently collaborative — you were thinking through a new
design space. A spec wouldn't have helped here because the decisions hadn't
been made yet. But the *output* of our discussion can now become a spec for
the next session.

### What a spec for implementing `order_search` would look like

```markdown
# Implement order_search tool

## Context
Replace raw execute_sql with a structured query tool for the AI agent.
The agent sends structured JSON params, Go compiles them to SQL.
See new-tools-for-the-agent.md for the full param spec.

## Task 1: Query builder — prototype-3/order_query.go (new file)
func buildOrderSearchSQL(params OrderSearchParams) (string, []any, error)

OrderSearchParams struct mirrors the tool spec:
  - OrderNumber: string or []string (single or IN query)
  - CustomerName, CustomerEmail, BrandMix, ShippingCarrier,
    ShippingMethod: string (GLOB matching)
  - Status: []string (IN query)
  - OrderDate, CreatedAt: string (expression parser: =, !=, >, <,
    >=, <=, between(), [])
  - ContainsProduct: struct { Brand, SKU: string or []string;
    Name: string (GLOB) }
  - Sort: string (default "order_date desc", whitelist columns)
  - Limit: int (default 50, max 200)
  - Offset: int (default 0)

The function returns a complete SQL query string with ? placeholders
and the corresponding args slice. Use json_group_array/json_object
for the order_items column.

## Task 2: Expression parser — prototype-3/expr.go (new file)
func parseDateExpr(expr string) (sqlFragment string, args []any, err error)

Parses the date expression mini-language:
  "= 2026-02-01"              → "= ?", [2026-02-01]
  ">= 2026-02-01"             → ">= ?", [2026-02-01]
  "between(2026-02-01,2026-02-10)" → "BETWEEN ? AND ?", [..,..]
  "[2026-02-01,2026-02-10]"   → "IN (?, ?)", [..,..]

## Task 3: Tool definition — update agent.go
Add order_search to the tool list with JSON schema matching the params.
Keep execute_sql available (needed for analytics until order_stats exists).

## Task 4: Tool handler — update tools.go
When agent calls order_search, parse JSON input into OrderSearchParams,
call buildOrderSearchSQL, execute query, return formatted results.

## Task 5: Tests — prototype-3/order_query_test.go
Table-driven tests for buildOrderSearchSQL:
  - Empty params → all orders
  - Single order_number → WHERE order_number = ?
  - Status array → WHERE status IN (?, ?)
  - order_date between → WHERE order_date BETWEEN ? AND ?
  - contains_product with brand → EXISTS subquery
  - Sort + limit → ORDER BY ... LIMIT ...
  - Invalid sort column → falls back to order_date desc

## Task 6: Integration test
Start server, send chat message "Show me Processing orders containing
KABE products", verify agent uses order_search (not execute_sql),
verify correct results returned.
```

---

## The Pattern

Looking across both phases, the spec pattern that enables autonomous
execution is:

1. **State the design intent.** Not just *what* but *why* — the
   overarching idea that connects all the tasks. Three sentences that
   give a fresh reader (or a fresh session) the north star for resolving
   ambiguous choices. Without this, tasks may be executed correctly in
   isolation but fail to cohere. Example:

   > The current agent writes raw SQL via execute_sql. This is slow,
   > fragile, and opaque. order_search replaces the query half: agent
   > fills structured params, Go compiles to SQL. The agent becomes a
   > translator, Go becomes the compiler.

2. **List the files** — modified or new — with function signatures.
3. **Define data shapes** — structs, JSON schemas, API contracts.
4. **Specify behavior at decision points.** Defaults, maximums, fallbacks,
   edge cases. Every "what should happen when..." needs an answer.
5. **Concrete verification.** Not "test it works" but specific scenarios
   with expected outcomes.
6. **Separate implementation tasks from design tasks.** If a decision
   hasn't been made, that's a discussion, not a task. Make the decision
   first, then write the task.

The key insight: **the spec should contain no questions.** Every question
is a round-trip. If writing the spec surfaces questions you can't answer,
those are the things to discuss first — then update the spec and hand it
off for execution.

The second insight: **make the connecting thread explicit.** Task
breakdowns have an implicit overarching idea — the *why* behind all the
*whats*. If you don't state it, you're relying on the executor to infer
it. They might get it right, or they might execute each task correctly
but make choices that don't cohere. A short "Design Intent" section at
the top costs almost nothing to write and eliminates this risk.
