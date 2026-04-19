# ADR-001: Hexagonal Architecture + Modular Monolith

**Status:** Accepted
**Date:** 2025-07

---

## Context

Vidify is a local-first CLI tool (with a potential GUI future) that pipes audio
through two external processes — Whisper.cpp (speech-to-text) and FFmpeg
(video synthesis). The project is maintained by a single developer who needs:

- A codebase that is easy to test without spawning real FFmpeg/Whisper processes.
- A structure that can evolve from CLI to a Wails-based GUI without architectural
  surgery.
- Clear boundaries so infrastructure details (binary paths, progress bars) never
  bleed into business logic.

Several architectural options were considered:

| Option | Pros | Cons |
|--------|------|------|
| Flat package structure | Simple to start | No boundaries; hard to test infra |
| Layered (traditional) | Familiar | Top-down imports allow accidental coupling |
| Microservices | High isolation | Massive over-engineering for a local CLI tool |
| **Hexagonal + Modular Monolith** | Clean boundaries, testable, single binary | Slightly more up-front structure |

---

## Decision

Adopt **Hexagonal Architecture (Ports & Adapters)** within a **single Go module
(Modular Monolith)**.

### Layer map

```
cmd/cli/                   → Entry point only; delegates to adapter/cli
internal/adapter/cli/      → Driving adapter: translates Cobra commands → port.Processor calls
internal/di/               → Single composition root; wires everything together
internal/application/      → Service + use cases; depends only on domain and port
internal/domain/           → Models (Audio, Subtitle, Video, Config) + port interfaces
internal/domain/port/      → All port interfaces (Processor, Transcriber, VideoMerger,
                             ProgressReporter, Logger); package name: `port`
internal/infrastructure/   → Driven adapters: whisper, ffmpeg, config, logger, progress
internal/common/           → Shared, domain-agnostic utilities
```

### Dependency rules (enforced by package boundaries)

1. **Domain** (`domain/`, `domain/port/`) has zero external library imports.
2. **Application** (`application/`) imports `domain` and `domain/port` only.
   It never imports `log/slog`, `os/exec`, or any `infrastructure/` package.
3. **Infrastructure** (`infrastructure/`) implements port interfaces.
   It may import `domain` and `domain/port` but never `application/`.
4. **DI** (`di/`) is the *only* package that imports both `application/` and
   `infrastructure/`. This deliberate bottleneck makes the coupling visible and
   auditable in a single file (`wire.go`).
5. **Adapter** (`adapter/cli/`) imports `di` and `domain/port` only.
   It never reaches into individual infra packages.

### Why `port.Logger` instead of passing `*slog.Logger`

Passing `*slog.Logger` directly through application and use-case constructors
would make `log/slog` a transitive import of the application layer, coupling
it to a specific logging backend. The `port.Logger` interface (Debug/Info/Warn/
Error/With) decouples application code from the backend without adding any
meaningful overhead — the interface has only five methods, all of which map
directly to slog calls in the concrete adapter.

### Why manual DI instead of `wire` or `fx`

`wire` (code generation) and `fx` (reflection-based) add build complexity that
is not justified for a project with fewer than ten injectable dependencies. The
`di/wire.go` file achieves the same goal — a single, readable composition root —
with plain Go. If the object graph grows significantly, adopting `wire` is a
straightforward migration.

---

## Consequences

**Positive:**
- Every use case can be unit-tested with mock implementations of port interfaces,
  no real binaries needed.
- Adding a GUI adapter (Wails) requires implementing a new adapter that calls
  `di.Build()` — no changes to application or domain code.
- The composition root (`di/wire.go`) is the single place to audit all
  dependency wiring.

**Negative / Trade-offs:**
- More files and packages than a flat structure requires.
- Contributors unfamiliar with hexagonal architecture need a brief orientation
  (see `docs/architecture.md`).
- `internal/domain/port/` importing `internal/domain` for model types is an
  intra-domain import, not a cycle, but can look unusual at first glance.
