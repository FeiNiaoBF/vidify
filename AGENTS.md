# Repository Guidelines

## Project Structure & Module Organization
Vidify follows Go's standard layout: `main.go` wires the Cobra CLI from `cmd/cli`, while feature packages live under `internal` (audio, video, subtitle, whisper, ffmpeg, processor). Shared helpers belong in `pkg` (logger, validator, utils). Public HTTP contracts stay in `api/v1`, environment defaults in `configs/`, and long-form references in `docs/`. Place large models and bundled binaries in `models/` or `third_party/`; integration fixtures live in `test/testdata/`.

## Build, Test, and Development Commands
After cloning, run `go mod download` to sync dependencies. `make build` produces `vidify.exe`; `make run` or `go run .` is ideal for quick smoke checks. Format and vet with `gofmt -w ./` and `go vet ./...`; if available, `golangci-lint run ./...` mirrors our CI preset. Execute `go test ./...` for unit coverage and `go test -race ./...` before shipping concurrency-sensitive changes.

## Coding Style & Naming Conventions
Always commit code formatted by `gofmt`; avoid manual alignment. Packages use short, lower-case names; exported types and functions use PascalCase, private helpers camelCase. Keep CLI command verbs concise (`convert`, `serve`) with descriptive `Short` and `Long` text. Mirror YAML keys in structs without changing their snake_case names, and keep configuration overrides in versioned files under `configs/`.

## Testing Guidelines
Co-locate unit tests in `_test.go` files beside the code they exercise, using table-driven patterns for codec permutations. Integration flows belong in `test/integration/` and should stage media fixtures under `test/testdata/`. Generate coverage with `go test -coverprofile=coverage.out ./...` and review via `go tool cover -html=coverage.out`; attach findings to PRs that significantly alter pipelines.

## Commit & Pull Request Guidelines
Use &lt;type&gt;: &lt;imperative summary&gt; commit messages (e.g., `init: bootstrap CLI module`) and keep unsupported assets out of history. PRs must describe behaviour changes, list verification commands (`go test ./...`, `make build`), and mention config or asset updates. Link issues when applicable, add screenshots or sample API payloads for UX-affecting work, and flag migration steps in the checklist.

## Environment & Configuration Tips
Ensure FFmpeg is reachable on `PATH` and Whisper models reside under `models/`. Adjust defaults by copying `configs/config.yaml` and never commit machine-specific secrets. Temporary renders should target `output/` (ignored by Git); add new cache directories to `.gitignore` before pushing.

## Git Workflow & Branching
- Use `dev` as the long-lived integration branch; feature branches should fork from `dev` and merge back through reviewed pull requests.
- Keep `main` releasable at all times: only fast-forward from `dev` once smoke tests (`go test ./...`, `make build`) pass and release notes are prepared.
- Group commits by concern (e.g., scaffolding, docs, tooling) and follow the `<type>: <imperative>` convention already documented.
- Avoid committing large artefacts; prefer storing Whisper models under `models/` and referencing them via release assets or external storage.
- Tag releases from `main` (`git tag vX.Y.Z`) after validating binaries, and document deployment steps in PR descriptions.

## Whisper Integration (Windows CLI)
- Prefer using the official whisper.cpp Windows release zip. Keep the unpacked `whisper-cli.exe` (and accompanying DLLs) under `third_party/whisper.cpp/bin/` so it can be invoked from Go via `exec.Command`.
- Store Whisper models (e.g., `ggml-base.en.bin`) under `models/` and point configuration (`config.AppConfig.Tools.Whisper`) to that location for runtime discovery.
- `internal/tools/transcriber` should wrap `whisper-cli.exe` by:
  1. Converting source audio to 16 kHz mono WAV via FFmpeg if necessary.
  2. Executing `whisper-cli.exe -m <model> -f <wav> -osrt -otxt -of <temp_prefix>` and returning the generated subtitle path.
  3. Handling concurrency with a worker pool or mutex to avoid CLI saturation; surface useful stderr logs when the subprocess fails.
- Ensure `PATH` (or `PATHEXT`) includes the CLI directory at runtime, or resolve the binary explicitly from configuration before spawning the process.
- Document how to refresh the bundled CLI when upgrading whisper.cpp (download new release, replace binaries, validate with a sample transcription).

## Renderer Implementation
- The FFmpeg-backed renderer lives under `internal/tools/renderer` and must consume both the audio artifact and the `.srt` file produced by the transcriber.
- Normalize inputs first: ensure the subtitle path exists, the output directory is created, and temporary scratch space is available for intermediate assets (e.g., waveform visuals).
- Invoke FFmpeg via `exec.CommandContext`, passing `-i <audio>` and `-vf subtitles=<subtitle>` (or equivalent filter chain) alongside any user-specified `ExtraArgs`. Respect `ctx` for cancellation and limit stdout/stderr noise to what's helpful for troubleshooting.
- Bubble structured errors with stderr excerpts so the pipeline can label failures (`render stage: ...`). Log start/finish markers through the logger injected into the pipeline.
- Keep the renderer service configurable: allow overriding the FFmpeg binary path, subtitle styling options, and hardware acceleration flags via configuration so both CLI and future server modes reuse the same implementation.
