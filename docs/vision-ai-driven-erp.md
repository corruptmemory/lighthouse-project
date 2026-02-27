### Vision: AI-Driven ERP — Eliminating Complexity as a Product

*Captured 2026-02-26. This emerged from a design conversation exploring what
happens when you put an AI agent at the center of an ERP instead of the
traditional query/report/dashboard configuration stack.*

---

#### The Core Idea

Replace the entire ERP "customization layer" — query builders, report
designers, dashboard configurators, saved searches, role-based view
permissions — with a conversational AI agent that has direct access to the
database and can generate visualizations on the fly.

The traditional ERP interaction:

```
Business person → describes need → super user → translates to ERP query
language → builds report → iterates 3 times → delivers
```

The proposed interaction:

```
Business person → describes need → AI agent builds it live →
iterates in real-time conversation → done
```

The agent speaks both languages natively: English and SQL. It doesn't need a
"Saved Search builder" or a "JQL-like language" because it *is* the universal
adapter between business intent and data retrieval.

---

#### Why This Works

**The AI eliminates three layers simultaneously:**

1. **The query abstraction layer.** ERPs ship proprietary query languages
   (NetSuite SuiteQL, Salesforce SOQL, SAP CDS Views) because they can't
   expose raw SQL to business users. An AI speaks SQL fluently and English
   fluently — the translation layer becomes the model itself.

2. **The "super user" bottleneck.** In every mid-market company running an
   ERP, 1-3 people become the de facto system administrators. They're usually
   operations or finance people who learned the report builder and became
   indispensable single points of failure. Their tribal knowledge of
   configurations, custom fields, and saved searches is undocumented and
   fragile. The AI replaces this bottleneck by making everyone capable of
   getting what they need directly.

3. **The fixed widget/chart vocabulary.** Traditional ERPs constrain you to
   the visualization types the vendor shipped. An AI generating JavaScript
   on the fly can produce *any* visualization — the rendering surface is
   unconstrained.

---

#### Architecture Sketch

The "ERP" becomes a much smaller thing:

- **PostgreSQL database** — the domain schema, the source of truth
- **Go back-end** — serves the MCP capabilities (query execution, widget
  persistence, user auth), handles integrations via exec bridge pattern
- **Minimal shell UI** — sidebar navigation, layout grid, chat panel
- **AI agent** — connected via MCP to a constrained capability surface,
  does everything else

The shell is dumb. The database is the source of truth. The AI is the
interface. The widgets it produces are persisted artifacts that become
the dashboards.

**The agent's MCP capability surface:**

- Read access to the database (execute queries)
- Write access to a rendering surface (generate JS/HTML for widgets)
- Knowledge of the domain schema (table definitions, column semantics,
  relationships)
- Potentially: write access to data (create orders, update inventory)
  with appropriate guardrails
- Domain context: business rules, brand definitions, threshold values

**Analogy: AI-powered Jupyter notebook.** In Jupyter, a human writes code
against data and sees results inline. This is the same loop but with the AI
writing code based on natural language, and results rendered as proper
widgets. The "notebook" is the conversation. The "kernel" is the
MCP-connected agent. The "cells" are the widgets it produces.

---

#### Hard Problems

##### 1. Trust and Correctness

When a super user builds a report in NetSuite, it's wrong in predictable
ways — wrong filter, wrong date range. When an AI generates SQL, it can be
wrong in *surprising* ways — a subtle JOIN that silently drops rows, an
aggregation that double-counts. The user may not notice because the output
*looks* right.

**This is the same class of problem that exists today** ("incorrect data
presented confidently"), just with a different failure mechanism. The AI is
actually *better* at explaining what it did than a saved search config
screen is.

Potential mitigations:
- Show the generated query alongside the results (transparency)
- Run sanity checks ("this returned 847 rows — does that match?")
- Audit trail of every query the agent executed
- Review workflow for queries that will be saved as persistent widgets
- Test against known-good reference data

**Status: No complete answer yet.** This is the hardest problem and needs
more thought.

##### 2. Reproducibility and Persistence

"Show me the report Sarah built last week" is trivial in a traditional
ERP — it's a saved search with a name. In a conversational system, you need
a way to persist what the AI built. The conversation produces an artifact
(a widget definition: SQL + JS + layout config), and that artifact needs
to be:

- **Saveable** — stored as a first-class object, not buried in chat history
- **Shareable** — other users can see and use it
- **Re-runnable** — executes the same query without replaying the conversation
- **Editable** — can be refined later, either via conversation or directly
- **Versionable** — changes tracked, previous versions recoverable

Without this, you replace the super user's tribal knowledge with a different
form of tribal knowledge: "the conversation I had with the AI last Tuesday."

**Status: Some ideas forming.** Widgets as persisted artifacts (SQL + JS +
metadata) stored in the database, versioned, and surfaced in the UI
independently of the conversations that created them.

##### 3. Security and Access Control

An agent with raw SQL access is either read-only (safe but limited) or
read-write (powerful but dangerous). The MCP capability surface needs to be
designed with security in mind from day one.

Considerations:
- Row-level security (user A can only see their region's data)
- Operation-level permissions (read vs write vs DDL)
- Query sandboxing (prevent resource-exhaustive queries, `DROP TABLE`, etc.)
- Audit logging of every query executed
- Approval workflows for write operations

**Status: Not yet addressed.** Acknowledged as load-bearing; will need
careful design before any write access is enabled.

---

#### Strategic Context: The SaaS Disruption Angle

The business model of enterprise software (SAP, Oracle, NetSuite, Salesforce)
rests on **complexity as a product**. Revenue comes from:

1. Licensing the software
2. Paying consultants to configure it
3. Paying super users to maintain configurations
4. Paying for upgrades that break configurations
5. Paying consultants again

The revenue isn't in solving the business problem — it's in managing the
complexity of the solution.

An AI agent that eliminates the configuration/query/report layers removes
the thing people are actually paying for. This isn't an incremental
improvement; it's removing the revenue-generating complexity itself.

**What remains defensible** (not replaceable by AI):
- The domain model (a schema that correctly represents *this* business)
- The data itself (actual orders, inventory, subscriptions, history)
- Trust relationships (this system is allowed to see our financials)
- Integration plumbing (EDI connections, carrier rate negotiations, SAP RFC)
- Transaction guarantees, compliance audit trails, data integrity

**What becomes commodity** (replaceable by AI):
- Query builders and report designers
- Dashboard configurators
- Role-based view permissions (partially — the *enforcement* is still needed,
  but the *configuration UI* is replaced by conversation)
- Workflow builders for simple automations
- Data export/transformation tools

Reference: "The SaaSpocalypse" — Forbes, Feb 2026
(https://www.forbes.com/sites/donmuir/2026/02/04/300-billion-evaporated-the-saaspocalypse-has-begun/)

**The AI itself is not defensible.** Once the pattern is demonstrated, anyone
with API access to a capable model can replicate it. The moat is in the
domain model, the data, and the trust relationships — not in the AI layer.

---

#### What Layers of an ERP Are Genuine vs. Complexity Theater?

| Layer | Genuine Value | Complexity Theater |
|---|---|---|
| Data integrity / transactions | ✓ | |
| Compliance / audit trails | ✓ | |
| Integration plumbing (SAP, EDI, carriers) | ✓ | |
| Domain schema design | ✓ | |
| Query builders / report designers | | ✓ |
| Dashboard configurators | | ✓ |
| Proprietary query languages | | ✓ |
| Super user configuration knowledge | | ✓ |
| Workflow visual designers | | ✓ (for simple flows) |
| Role-based view *configuration* | | ✓ |
| Role-based view *enforcement* | ✓ | |

The genuine-value layers still need to be built properly. The
complexity-theater layers are exactly where an AI agent collapses the value
chain.

---

#### Open Questions

1. How does the trust/correctness problem get solved at a level that makes
   business users confident in the output? Showing the SQL helps technical
   users but not the operations manager.
2. What's the right persistence model for AI-generated widgets? How do they
   relate to the conversations that created them?
3. How do you handle the transition from "AI builds it live" to "this is now
   a production dashboard that 10 people rely on every morning"?
4. Where exactly does the security boundary sit in the MCP capability
   surface? What's the minimum viable access control?
5. Can the AI agent also handle *write* operations (create order, adjust
   inventory) safely, or should that remain in traditional coded workflows?
6. How does this interact with the exec bridge architecture? Can the AI
   agent invoke bridge wrappers to pull data from external systems (SAP,
   Shopify) as part of answering a question?
