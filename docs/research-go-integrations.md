### Research: Go Integration with Commerce/ERP Platforms

*Compiled 2026-02-25. Assessment of Go ecosystem maturity for integrating
with the systems a company like Lighthouse might use.*

---

#### Integration Maturity Summary

| Integration Need | Go Ecosystem | Notes |
|---|---|---|
| REST/JSON APIs | Excellent (native) | `net/http` is first-class |
| GraphQL APIs | Good | `gqlgenc` for type-safe codegen |
| OAuth2 | Excellent | `golang.org/x/oauth2` |
| OData (SAP) | Fair | Small generic libraries, no SAP-specific |
| SOAP/WS-Security | Poor | Basic libraries, no WSDL codegen |
| SAP RFC/BAPI | Poor | Archived official lib, CGo required |
| EDI/AS2 | Absent | Requires middleware |
| Webhook HMAC verification | Good | Trivial to implement |

---

#### Platform-Specific Details

##### Stripe (Payment Processing)

**Official Go SDK: `stripe/stripe-go`** — the gold standard. Auto-generated
from Stripe's OpenAPI spec, always complete. Full subscription/recurring
billing support. Webhook signature verification built in. Currently v82+.

No trouble spots. This is the recommended payment processor for a Go back-end.

##### Shopify

**No official Go SDK.** Two community options:

- `bold-commerce/go-shopify` (v3) — Most established. Primarily REST-focused.
  Has webhook HMAC verification. **But Shopify is deprecating REST in favor of
  GraphQL** (mandatory for new apps since April 2025).
- `r0busta/go-shopify-graphql` (v5+) — GraphQL-focused. Supports bulk
  operations. More aligned with Shopify's direction.

**Recommendation:** Use `gqlgenc` to generate a typed GraphQL client from
Shopify's schema for future-proof integration.

**Gotcha:** Shopify is dropping permanent access tokens (January 2026),
requiring migration to client credential grants.

##### PayPal

**No official Go SDK.** Community library `plutov/paypal` (v4) covers core
operations. For missing endpoints, use `NewRequest` + `SendWithAuth` helpers.

##### Adyen

**Official Go SDK: `Adyen/adyen-go-api-library` (v9).** Covers Checkout,
Payments, and Recurring APIs.

##### commercetools

**Community Go SDK: `labd/commercetools-go-sdk`** — auto-generated from
commercetools' API specs. Near-complete coverage. Originally built for the
Terraform provider. **No backward-compatibility guarantees between releases** —
pin your dependency.

##### Medusa

**No Go SDK.** Pure REST API, easily consumed with `net/http`. Auth via
`x-publishable-api-key` header or JWT. Medusa itself is Node.js — plugins
must be TypeScript.

##### Saleor

**No Go SDK.** GraphQL-only API. Use `gqlgenc` to generate typed client from
Saleor's schema.

##### SAP S/4HANA Cloud (OData)

OData is REST with conventions — consumable via `net/http`. Generic OData
libraries exist (`Uffe-Code/go-odata`) but are not SAP-specific. Expect to
hand-roll some SAP OData quirks (batch request formatting, `__next`
pagination).

Auth: standard OAuth2 client credentials via SAP BTP destination service.

##### SAP RFC/BAPI (On-Premise)

**Avoid if possible.** Requires proprietary NW RFC C library + CGo wrapper.
SAP's own Go connector (`SAP/gorfc`) is archived. Community fork
(`SimonWaldherr/saprfc`) exists but is thin.

---

#### Architecture: Exec Bridge Pattern (Preferred over CGo/Sidecar)

For integrations requiring proprietary protocols (SAP RFC, SOAP, EDI), use
standalone wrapper executables instead of CGo or sidecar services. The bridge
supports two operating modes that share the same JSON message format.

**Advantages over CGo:**
- Go binary stays pure Go (fast builds, easy cross-compilation)
- C library dependency is quarantined to the wrapper binary's build
- No viral build complexity

**Advantages over sidecar services:**
- No network surface (no ports, no TLS, no health checks)
- No API drift (wrapper and caller versioned together in same repo)
- Single deployment unit (multiple binaries, one repo)
- Testable in isolation (pipe JSON in, inspect JSON out)

##### Mode 1: One-Shot (default)

Start the wrapper binary per request. Suitable for low-frequency, high-value
calls (e.g., nightly inventory sync, posting financial transactions).

```
Go service
  → exec's wrapper binary
  → writes JSON request to stdin, closes stdin
  → wrapper: JSON → proprietary protocol → remote call → JSON
  → wrapper writes JSON response to stdout, exits
  → Go service reads response
```

Non-zero exit code = error. Stderr captured for error messages.

##### Mode 2: Persistent (LSP-style)

Start the wrapper binary once, keep it running, multiplex concurrent requests
over stdin/stdout using request IDs. Suitable when call frequency or startup
cost (JVM ~1-2s) makes one-shot impractical.

```
Go service                              Wrapper process (long-running)
  → writes {"id":"a1","method":"...",     → reads stdin line by line
     "params":{...}}\n to stdin           → dispatches by method
  → writes {"id":"b2","method":"...",     → calls remote system
     "params":{...}}\n to stdin           → writes {"id":"a1","result":{...}}\n
  ← reads stdout, dispatches by id       → writes {"id":"b2","result":{...}}\n
```

This is the same pattern as JSON-RPC 2.0 / LSP / DAP. Each message is a
single newline-delimited JSON object. The Go side maintains a
`map[string]chan Response` keyed by request ID; a single goroutine reads
stdout and dispatches responses to waiting callers.

Errors use the same envelope: `{"id":"a1","error":{"code":...,"message":...}}`.

The wrapper process writes logs/diagnostics to stderr (not stdout).

##### Shared Design Principles

Both modes use the same JSON message format. A wrapper that supports mode 2
also works in mode 1 (single request, then EOF on stdin triggers clean
shutdown). This means:

- Design the message format once
- Build the wrapper with persistent mode from the start
- The Go convenience library chooses the mode based on configuration

##### Go Convenience Library

```go
// execbridge provides typed subprocess calls using JSON over stdin/stdout.
// One-shot mode: starts and stops the binary per call.
// Persistent mode: holds a running process, multiplexes via request IDs.

// One-shot call.
func Call[Req, Resp any](ctx context.Context, binary string, req Req) (Resp, error)

// Persistent bridge. Safe for concurrent use.
type Bridge struct { /* holds cmd, stdin/stdout pipes, pending map */ }
func NewBridge(ctx context.Context, binary string) (*Bridge, error)
func (b *Bridge) Call[Req, Resp any](ctx context.Context, method string, req Req) (Resp, error)
func (b *Bridge) Close() error
```

##### Startup Time by Runtime

| Runtime | Cold Start | Use Case |
|---|---|---|
| Python (pyrfc) | ~100ms | SAP RFC if avoiding JVM |
| Node (node-rfc) | ~100ms | SAP RFC, SOAP |
| JVM (JCo for SAP) | ~1-2s | SAP RFC (use persistent mode) |
| GraalVM native-image | ~50ms | JVM wrapper compiled to native |

For JVM-based wrappers, persistent mode is strongly preferred to avoid
paying the cold start on every call.

---

#### iPaaS: When It Makes Sense

An iPaaS (MuleSoft, Boomi, n8n) is justified when:
- You have many heterogeneous enterprise systems to connect
- Those systems use proprietary protocols (EDI/AS2, SAP RFC)
- You don't want to build and maintain wrapper executables yourself

An iPaaS is overkill when:
- All your integrations are REST/GraphQL (Shopify, Stripe, commercetools)
- Go can talk HTTP natively with no friction

**The hybrid:** Use the exec bridge pattern for 1-2 proprietary integrations
(e.g., SAP), use Go's native HTTP for everything else, and only bring in an
iPaaS if the number of proprietary integrations grows beyond what's reasonable
to maintain as wrapper executables.

---

#### Bottom Line

For a Go back-end at this scale:

- **Stripe + Shopify GraphQL** is the easiest integration path.
- **SAP OData** (cloud) is workable via `net/http` + OAuth2.
- **SAP RFC** (on-premise) → use exec bridge with Python/Node/Java wrapper.
- **SOAP** → use exec bridge with Java/Python wrapper.
- **Avoid CGo** for SAP/SOAP integration — the exec bridge pattern is simpler,
  more maintainable, and keeps the Go build clean.
