# Schema Evolution — Design Discussion

*Captured 2026-02-28. This is an in-progress design conversation, not a
finished spec. Jim has answers to the open questions below — to be worked
through point by point in a follow-up session.*

---

## Core Principles (Jim's Position)

### 1. Schema-as-code, not schema-as-migration-history

The schema is a **data structure in Go code** — a declarative constant that
says "this is what the database looks like." Not a sequence of migration
files that must be replayed to discover the current state.

Multiple versions of this data structure can be embedded in the binary as
code. The system knows what version it's running against and what version
it needs to be.

### 2. No ORMs

Full stop. But steal the one good idea from ORMs: encode the schema in code
so there's a single source of truth. Not for generating accessors — for
knowing what the database *is* and being able to reason about it
programmatically.

### 3. The AI queries the data structure, not the database

The agent's system prompt context is derived from the schema data structure,
not from interrogating the live database. This means schema changes
automatically propagate to the agent's understanding. No drift between what
the code thinks the schema is and what the agent thinks it is.

### 4. Blue-green database migration

When a migration is needed:

- Old database (ABC) stays untouched and operational
- New database (DEF) is created with the new schema
- Data is migrated from ABC → DEF
- The process switches to DEF automatically
- The starting configuration is updated to point at DEF

If DEF fails to come up correctly, you're still running against ABC. No
destructive rollback needed because you never modified ABC.

### 5. CSV as interchange, not pg_dump

All actual tables are dumped as CSV files. CSV is dumb, inspectable, and
tool-agnostic — `head`, `wc -l`, `diff` all work. No version mismatches,
no binary format incompatibilities, no partial restore failures leaving the
database indeterminate.

### 6. Load without constraints, overlay after

New database is created with the schema but **no constraints or indexes**.
Data is loaded (directly or through temp tables for transformations). Then
constraints and indexes are added incrementally. This is how DBAs do bulk
loads — migration tools just never bother.

### 7. Derived diff logic

The diff between old and new schema versions is computed by the tool, not
hand-written by developers. Humans specify *what* transformations are needed
(e.g., "split `full_name` into `first_name` + `last_name`"). The tool
handles *how* (temp tables, load ordering, constraint sequencing).

### 8. Database-specific, not generic

> "You picked a DB technology for a reason, once chosen you're ALL IN. No
> hedging."

No half-assed database-agnostic abstraction layers. If you chose Postgres,
you use Postgres features fully — and the schema data structure, migration
tool, and diff engine are Postgres-aware. This extends to stored procs,
functions, and triggers (design for those coming later).

---

## Open Design Questions (Claude's Analysis)

These were raised during the initial discussion. Jim has at least partial
answers to all of them — to be worked through in a follow-up session.

### Q1: The database name swap mechanics (ABC → DEF)

Blue-green for databases. Specific questions:

- How does the running process discover DEF is ready? Config file update?
  Meta-database pointer? Process restart?
- In-flight requests during the swap — clean handoff point needed.
- Connection pool draining — can't just change the DB name; need to drain
  old pool, establish new one.

The simplest version (offline migration, config update, process restart)
is already vastly better than ALTER TABLE on a live database.

### Q2: Lineage annotations for the diff engine

The diff engine needs to distinguish "column `address` was renamed to
`street_address`" from "column `address` was removed and column
`street_address` was added." Without lineage, every rename looks like a
destructive add/drop and data is lost.

Proposed: `TableDef` and `ColumnDef` carry lineage fields:
```
Lineage: "was previously X"
Lineage: "was previously X with type Y"
```

### Q3: Transform functions for inline data changes

When a column changes type or gets split/merged, the human must specify the
conversion logic. Everything else (temp table creation, load sequencing,
constraint ordering) is derived. Where do these transform functions live?
In the schema data structure? As companion code next to the version?

### Q4: Version chain — how many schemas in the binary?

Does the binary carry every historical schema, or just current + previous?

- If only N-1 → N: two versions suffice
- If you need N-3 → N: need intermediate versions or the diff engine must
  handle multi-step jumps

### Q5: Direct vs indirect load path

For tables where the schema is compatible (e.g., only added a column with
a default), loading through temp tables adds overhead for no benefit. The
diff engine should detect this and choose:

- **Direct load**: table schema is compatible, no transforms needed
- **Indirect via temp table**: column transformations required

### Q6: Stored procs, functions, triggers

Flagged as "coming later." These are behavioral, not structural. The diff
logic for behavioral schema objects is different — you can't meaningfully
diff two function bodies to derive a migration; you just replace. The data
structure representation needs to account for this eventually.

Jim has answers on this. Design TBD.

---

## Rough Data Structure Shape

```go
type SchemaVersion struct {
    Version     int
    Tables      []TableDef
}

type TableDef struct {
    Name        string
    Columns     []ColumnDef
    Constraints []ConstraintDef
    Indexes     []IndexDef
    Lineage     *TableLineage  // "was previously called X"
}

type ColumnDef struct {
    Name     string
    Type     ColumnType  // typed, not string
    Nullable bool
    Default  *string
    Lineage  *ColumnLineage  // "was previously X with type Y"
}
```

The diff engine compares two `SchemaVersion` values and produces a
`MigrationPlan` — an ordered list of steps (dump table, create table,
load direct, load via temp with transform, add constraint, add index).
The plan is inspectable, serializable, and executable.

---

## Why This Is Better Than the Status Quo

Traditional migration pain (all self-induced):

1. Schema definitions live in migration files — history as source of truth
2. Up/down migrations are manually written and poorly tested
3. Expensive operations run inefficiently (bulk updates with indexes active)
4. Humans hate writing migrations, hate testing them more
5. Failures during migration = extended outages, all-nighters
6. Rollback is often as dangerous as the migration itself

This design eliminates:

- Manual migration writing (diffs are derived)
- Destructive in-place modification (blue-green swap)
- Inefficient bulk operations (load without constraints)
- Risky rollbacks (old database is untouched)
- Schema drift between code and database (single source of truth)
- Schema drift between agent and database (agent reads the data structure)
