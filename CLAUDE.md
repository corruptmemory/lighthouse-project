# Lighthouse Back-End System

## What This Is

Commerce back-end serving two business units: Leuchtturm1917 (premium
stationery) and Lighthouse Publications (numismatic/philatelic collectibles,
including the KABE brand). Early design phase — no application code yet.

Read `docs/` before making architectural decisions. The architecture
principles in `docs/architecture-principles.md` are binding constraints.

## Build & Test

```sh
./build.sh           # Single entry point for everything
```

Never invoke `go build`, `go test`, or `templ generate` directly.

## Stack

- Go, chi router, templ templates, htmx (vendored JS, no npm/node)
- PostgreSQL
- TOML config via BurntSushi/toml

## Architecture Constraints

These are non-negotiable. See `docs/architecture-principles.md` for rationale.

1. **Single deployment bundle.** All components (Go binary + exec bridge
   wrappers) live in this repo, version together, deploy together. No
   separately versioned internal services. No internal API versioning.

2. **CLI-testable components.** Every exec bridge wrapper is runnable from the
   command line: pipe JSON in, get JSON out. No deployment ceremony.

3. **No CGo.** Use exec bridge wrappers for proprietary protocols (SAP RFC,
   SOAP, EDI). The Go binary stays pure Go.

4. **Go for HTTP, wrappers for everything else.** If you can talk to it with
   `net/http`, Go talks directly. If not, build a wrapper.

5. **CLI flags and TOML for config.** No environment variables. Config file
   path is itself a CLI flag. All programs support `--help`.

## Prototypes

Each prototype builds on the prior one unless otherwise instructed. Prototype-4
has all features of prototype-3, prototype-3 has all features of prototype-2,
etc.

## Code Style

- Format Go with `gofmt`
- Tests: straightforward table-driven or sequential, using `t.TempDir()` for
  isolation
- No over-abstraction — three similar lines is better than a premature helper
- Strict typing: use Go structs, typed constants, enums via `iota` for domain
  values. No untyped string constants for things like brand names, capsule
  systems, or collectible types.
- Trademark fidelity: QUADRUM, QUICKSLAB, VOLTERRA, Drehgriffel, INTERCEPT®
  must maintain exact casing and spelling in all code, comments, and output.

## Playwright Testing

When running a web browser interactively via Playwright, always close tabs that
you are not directly controlling so the user can clearly see your interaction
with the application.

## AI Memory

Read `.claude/shared-memory.md` at session start for accumulated project
knowledge (status, architecture decisions, design insights). That file is
shared across machines via git. Machine-local notes live in the auto-memory
directory (`~/.claude/projects/.../memory/`).

## Project Status

- [x] Technical specification (cleaned up, Go-oriented)
- [x] Architecture principles documented
- [x] ERP landscape research
- [x] Go integration research
- [ ] Domain expert validation (questions drafted, answers pending)
- [ ] Implementation plan
- [ ] Application code
