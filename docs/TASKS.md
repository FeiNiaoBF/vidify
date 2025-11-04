# Project Tasks

## Done
- [x] Bootstrap Cobra CLI commands (`convert`, `batch`, `version`) and surface them through `cmd/vidify/main.go`.
- [x] Wire `internal/tools/transcriber` to invoke `whisper-cli` and emit `.srt` subtitles into the output pipeline.

## In Progress
- [ ] Implement FFmpeg-backed renderer service to turn audio + subtitles into captioned video files.
  - [ ] Accept audio + subtitle inputs and resolve temp workspace layout.
  - [ ] Spawn FFmpeg with configurable args to mux subtitles into the output container.
  - [ ] Bubble meaningful logs/errors back through `internal/pipeline`.
- [ ] Add integration test covering end-to-end CLI flow with bundled sample audio/model artifacts.
- [ ] Expose configuration knobs for whisper threading/performance tuning via `runtime.env`.

## Backlog
- [ ] Activate the Web API surface in `cmd/server` so HTTP requests reuse the CLI pipeline.
- [ ] Support additional subtitle formats (SRT/VTT) and configurable styling presets.
- [ ] Wire cloud storage targets (S3/OSS) for rendered outputs.
- [ ] Stream task progress over WebSocket once the server mode ships.
- [ ] Prototype a lightweight preview UI for local testing sessions.
