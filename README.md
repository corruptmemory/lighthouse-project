# Lighthouse Back-End System

Commerce back-end for two business units of the Leuchtturm group:

- **Leuchtturm1917** — premium stationery (notebooks, journals, writing instruments)
- **Lighthouse Publications** — numismatic and philatelic collectibles (coins, stamps, optical equipment), including the **KABE** brand

## Status

Early design phase. The technical specification and architecture principles
are defined; domain validation with a subject matter expert is in progress.
No application code exists yet.

## Project Documents

All design documents live in `docs/`:

| Document | Purpose |
|---|---|
| [Technical Specification](docs/technical-specifications.md) | Domain model, schema, product taxonomy, business rules |
| [Architecture Principles](docs/architecture-principles.md) | Binding constraints on how the system is built |
| [Domain Expert Questions](docs/domain-expert-questions.md) | Validation questions for the business (answers pending) |
| [ERP Landscape Research](docs/research-erp-landscape.md) | Survey of off-the-shelf commerce/ERP solutions |
| [Go Integration Research](docs/research-go-integrations.md) | Go ecosystem maturity for each integration target |

## Technology Stack

- **Language:** Go
- **Router:** [chi](https://github.com/go-chi/chi/v5)
- **Templates:** [templ](https://github.com/a-h/templ)
- **Frontend interactivity:** [htmx](https://htmx.org/) (vendored, no npm/node)
- **Database:** PostgreSQL
- **Config:** TOML via [BurntSushi/toml](https://github.com/BurntSushi/toml)
- **Build:** `./build.sh` (single entry point for all build/test operations)

## Architecture Overview

The system deploys as a **single coherent bundle**: one Go binary plus any
exec bridge wrapper binaries for proprietary protocol integration (SAP RFC,
SOAP, etc.). All components live in this repository, are versioned together,
and are built and tested together. There is no internal API versioning.

For external integrations:
- Systems with REST/GraphQL/OData APIs (Stripe, Shopify, SAP Cloud) are
  called directly from Go via `net/http`.
- Systems requiring proprietary protocols use **exec bridge wrappers** —
  standalone binaries that communicate over stdin/stdout using JSON, avoiding
  CGo and separate service deployments.

See [Architecture Principles](docs/architecture-principles.md) for the full
set of constraints.

## Building

```sh
./build.sh
```

No code exists yet, but when it does, `build.sh` will be the single entry
point. Never invoke `go build`, `go test`, or `templ generate` directly.
