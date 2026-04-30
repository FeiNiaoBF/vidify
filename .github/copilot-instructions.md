# Vidify - Copilot Instructions

## Project Overview

Vidify is a local-first CLI/GUI tool that converts audio files into MP4 videos with burned-in subtitles. It uses Whisper.cpp for speech-to-text and FFmpeg for video synthesis. Written in Go 1.26.

## Architecture

Hexagonal Architecture (Ports & Adapters) + Modular Monolith with strict dependency inversion:

```
cmd/cli/                   → Entry point (minimal: parse flags, call adapter/cli)
internal/adapter/cli/      → Presentation layer (Cobra commands, calls di.Build)
internal/di/               → Composition root (wire.go — only place that imports both app and infra)
internal/application/      → Application layer (Service, use cases — no infra imports)
internal/domain/           → Domain layer (models + port interfaces, zero external deps)
internal/domain/port/      → Port interfaces (Processor, Transcriber, VideoMerger, ProgressReporter, Logger)
internal/infrastructure/   → Infrastructure layer (Whisper, FFmpeg, Config, Logger, Progress, AppDir adapters)
internal/infrastructure/appdir/ → App data directory authority — single source for all ~/.vidify/* paths
internal/common/           → Shared utilities (errors, helpers — no domain or infra imports)
```

**Key principles:**
- Domain defines port interfaces; Infrastructure implements them. Dependencies point inward.
- `internal/adapter/cli/` imports only `di` and `port` — never individual infra packages.
- `internal/di/wire.go` is the **only** composition root; all wiring is concentrated there.
- Application layer (`application/`) uses `port.Logger` — never imports `log/slog` directly.
- Port interfaces live in `internal/domain/port/` (package `port`), not scattered across packages.

## Tech Stack

| Component       | Library                              |
|-----------------|--------------------------------------|
| CLI framework   | `github.com/spf13/cobra`             |
| Config          | `github.com/spf13/viper`             |
| Logging         | `log/slog` (stdlib)                  |
| Log rotation    | `gopkg.in/natefinch/lumberjack.v2`   |
| Progress bar    | `github.com/vbauerster/mpb/v8`       |
| Task ID         | `github.com/google/uuid`             |
| Error wrapping  | `github.com/pkg/errors`              |
| External tools  | `os/exec` (Whisper.cpp, FFmpeg)      |

## Go Coding Conventions

- **Go version:** 1.26, use all modern features available
- **Style:** Follow [Google Go Style Guide](https://google.github.io/styleguide/go/)
- **Comments:** Focus on **"why"**, not "what". Every exported symbol needs a doc comment.
  - Package comments: one sentence describing the package purpose
  - Function comments: explain behavior, not implementation
  - Inline comments: explain non-obvious decisions, trade-offs, workarounds
- **Error handling:** Wrap with context using `pkg/errors`. Use `errors.Is`/`errors.As` for checking.
- **Naming:** No stutter (`domain.Audio`, not `domain.AudioModel`). Short, clear names.
- **Interfaces:** Define where consumed, not where implemented. Keep small (1-3 methods).
- **Constructors:** `New*` functions. Validate all required dependencies. Return concrete types.
- **Context:** Always pass `context.Context` as first parameter in functions that do I/O.

## Patterns to Follow

- **Functional options** for complex constructors (e.g., `WithModel()`, `WithLanguage()`)
- **Dependency injection** via constructor parameters
- **Sentinel errors** for domain-level errors (`ErrNotFound`, `ErrUnsupportedFormat`)
- **Table-driven tests** for all unit tests
- **`internal/`** for all non-public packages

## File Organization

- One primary type per file (e.g., `audio.go` defines `Audio`)
- Test files next to source: `audio_test.go`
- Keep `main.go` minimal — parse flags, wire dependencies, call `app.Run()`
- Config files in `configs/`

## CLI Design

- Entry point: `cmd/cli/main.go`
- Root command: `vidify`
- Processing flags: `-i/--input`, `-m/--model`, `-l/--language`, `--translate`, `--bg-image`, `-o/--output`
- Logging flags: `--log-dir`, `--log-level`, `--log-to-file`, `--log-max-backups`, `--log-max-age`
- Shortcut: `-v/--verbose` sets `--log-level debug`
- Viper binds to config file (`~/.vidify/config.yaml`) under `log:` block and environment variables (`VIDIFY_*`)
- Config file search order: `--config` flag → `~/.vidify/config.yaml` → `~/.vidify.yaml` (legacy) → `./.vidify.yaml`
- `vidify init` creates `~/.vidify/` directory tree + writes `config.yaml`

## External Tool Calls

- Whisper.cpp and FFmpeg are called via `os/exec`
- Always use `context.Context` for cancellation and timeouts
- Capture both stdout and stderr
- Parse output for progress reporting where possible

## Things to Avoid

- Do NOT use `init()` functions
- Do NOT use global mutable state
- Do NOT import `internal/domain` from `internal/infrastructure` (infra implements domain interfaces, not the reverse)
- Do NOT panic in library code; return errors
- Do NOT use `log` package; use `log/slog`
- Do NOT add features beyond what the PRD specifies
