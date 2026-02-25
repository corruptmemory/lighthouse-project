### Architecture Principles: Lighthouse Back-End System

These are load-bearing decisions that shape how the system is built. They are
not suggestions — they are constraints. Code that violates these principles
is wrong regardless of whether it "works."

---

#### 1. Single Deployment Bundle

The system is deployed as a single coherent bundle: one primary Go binary
plus any exec bridge wrapper binaries. All components live in one repository,
are versioned together, and are built and tested together by `./build.sh`.

**What this means:**
- There is no API versioning between internal components. The JSON message
  formats used by exec bridge wrappers are defined once, in Go types, and
  the wrappers are tested against those types in the same build.
- "Does the API match?" is answered by "does the build pass?" — not by
  version negotiation, compatibility matrices, or runtime discovery.
- Integration tests are just regular tests. There is no separate "integration
  environment" needed to verify that components work together.

**What this prohibits:**
- Separately versioned internal services.
- API deprecation cycles between components that ship together.
- Independent deployment of subsystems (they deploy as a unit, always).

**Why:** API evolution between separately deployed services is an enormous
time sink that produces no customer value. It exists to solve a coordination
problem between teams that can't release together. We can release together,
so we don't have that problem, and we refuse to import it.

---

#### 2. CLI-Testable Components

Every exec bridge wrapper must be runnable from the command line. Pipe JSON
in, get JSON out. No deployment ceremony, no environment setup beyond the
binary and its config.

```sh
echo '{"method":"...","params":{...}}' | ./bin/some-bridge
```

**What this means:**
- Wrapper binaries are self-contained. They read config from a TOML file
  (path passed as a CLI flag) and communicate over stdin/stdout.
- Developers can inspect behavior firsthand, in seconds, with no tooling
  beyond a terminal.
- Debugging an integration issue starts with running the wrapper by hand,
  not by reading logs from a running system.

**What this prohibits:**
- Wrappers that require a running Go server to function.
- Wrappers that depend on environment variables for configuration (use CLI
  flags and TOML config files).
- Wrappers that write operational output to anything other than stdout (use
  stderr for logs/diagnostics).

---

#### 3. Exec Bridge over CGo

When integrating with systems that require proprietary protocols or non-Go
SDKs (SAP RFC, SOAP, EDI), use standalone wrapper executables communicating
over stdin/stdout — never CGo.

**What this means:**
- The Go binary remains pure Go: fast builds, easy cross-compilation, no
  C toolchain dependency.
- The proprietary SDK dependency (e.g., SAP NW RFC C library, Java JCo) is
  quarantined to the wrapper binary's build process.
- Two modes are supported (same JSON format):
    - **One-shot:** exec per request, suitable for low-frequency calls.
    - **Persistent (LSP-style):** long-running process with multiplexed
      JSON-RPC over stdin/stdout, suitable for higher throughput or runtimes
      with expensive startup (JVM).

**What this prohibits:**
- `import "C"` in any Go source file.
- CGo-dependent Go libraries for SAP, SOAP, or similar protocols.
- Sidecar services that communicate over TCP/HTTP (use stdin/stdout).

---

#### 4. Go for HTTP, Wrappers for Everything Else

The Go service talks directly to any system that exposes a clean REST,
GraphQL, or OData API — no middleware, no wrappers. This includes Stripe,
Shopify, commercetools, SAP S/4HANA Cloud (OData), and similar.

Wrappers are only for systems that require proprietary binary protocols,
SOAP with complex WS-Security, or vendor-specific SDKs that don't exist
in Go.

**The test:** Can you talk to it with `net/http`? If yes, Go talks directly.
If no, build a wrapper.

---

#### 5. Configuration via CLI Flags and TOML

All runtime configuration is provided through CLI flags with sensible
defaults. Complex or multi-value configuration uses TOML files, but the
path to the TOML file is itself a CLI flag.

**What this prohibits:**
- Environment variables as the primary configuration mechanism.
- "Magic" config file paths that are discovered by convention rather than
  specified explicitly.

**Why:** There is no tooling to ask an arbitrary program "what environment
variables do you look at?" Any CLI-driven program can tell you what it
accepts via `--help`.
