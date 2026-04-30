TARGET  = vidify
CLI_ENTRY = ./cmd/cli
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
DIST    = dist

# ---------------------------------------------------------------------------
# Development targets
# ---------------------------------------------------------------------------

# Build the CLI binary for the current platform.
build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(TARGET)$(if $(filter windows,$(GOOS)),.exe) $(CLI_ENTRY)

# Run the CLI directly without building a binary first.
run:
	go run $(CLI_ENTRY) $(ARGS)

# Run all tests with race detection enabled.
test:
	go test -race -count=1 ./...

# Remove build artifacts and dist packages.
clean:
	rm -f $(TARGET).exe $(TARGET)
	rm -rf $(DIST)

# Tidy module dependencies and verify they are correct.
tidy:
	go mod tidy
	go mod verify

# Run go vet and check for common issues.
lint:
	go vet ./...

# Format all Go source files.
fmt:
	gofmt -w ./...

# Run vidify init to set up ~/.vidify/ app directories and default config.
init:
	go run $(CLI_ENTRY) init

# Run tests with race detection and generate an HTML coverage report.
cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# ---------------------------------------------------------------------------
# Release packaging targets
#
# Each target cross-compiles vidify and creates a dist/<platform>/ tree:
#
#   dist/<platform>/
#   ├── vidify(.exe)
#   ├── runtime-tools/
#   │   ├── whisper/<platform>/   ← populated by CI (download step)
#   │   └── ffmpeg/<platform>/    ← populated by CI (download step)
#   └── THIRD_PARTY_LICENSES.md
#
# The runtime-tools directories are created here as empty placeholders.
# The GitHub Actions release workflow fills them with the actual pre-built
# binaries before zipping the final artifact.
# ---------------------------------------------------------------------------

# Build release layout for all supported platforms.
release-all: release-linux-x64 release-linux-arm64 release-windows-x64 release-darwin-x64 release-darwin-arm64

release-linux-x64:
	$(MAKE) _release GOOS=linux   GOARCH=amd64 PLATFORM=linux-x64   EXT=

release-linux-arm64:
	$(MAKE) _release GOOS=linux   GOARCH=arm64 PLATFORM=linux-arm64  EXT=

release-windows-x64:
	$(MAKE) _release GOOS=windows GOARCH=amd64 PLATFORM=windows-x64  EXT=.exe

release-darwin-x64:
	$(MAKE) _release GOOS=darwin  GOARCH=amd64 PLATFORM=darwin-x64   EXT=

release-darwin-arm64:
	$(MAKE) _release GOOS=darwin  GOARCH=arm64 PLATFORM=darwin-arm64  EXT=

# Internal target — not called directly.
# Requires GOOS, GOARCH, PLATFORM, EXT to be set by the caller.
_release:
	@echo "→ Building $(PLATFORM)…"
	@mkdir -p $(DIST)/$(PLATFORM)/runtime-tools/whisper/$(PLATFORM)
	@mkdir -p $(DIST)/$(PLATFORM)/runtime-tools/ffmpeg/$(PLATFORM)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build \
	  -ldflags "-X main.version=$(VERSION)" \
	  -o $(DIST)/$(PLATFORM)/$(TARGET)$(EXT) \
	  $(CLI_ENTRY)
	@cp THIRD_PARTY_LICENSES.md $(DIST)/$(PLATFORM)/

# Pack a single platform's dist layout into a distributable archive.
# Usage: make pack PLATFORM=linux-x64
# Produces dist/vidify-<VERSION>-<PLATFORM>.tar.gz  (or .zip on Windows)
pack:
	@[ -n "$(PLATFORM)" ] || (echo "ERROR: set PLATFORM=<value>"; exit 1)
	@if echo "$(PLATFORM)" | grep -q "windows"; then \
	  cd $(DIST) && zip -r vidify-$(VERSION)-$(PLATFORM).zip $(PLATFORM)/; \
	else \
	  cd $(DIST) && tar -czf vidify-$(VERSION)-$(PLATFORM).tar.gz $(PLATFORM)/; \
	fi
	@echo "→ Packed dist/vidify-$(VERSION)-$(PLATFORM)"

.PHONY: build run test clean tidy lint fmt init cover \
        release-all release-linux-x64 release-linux-arm64 \
        release-windows-x64 release-darwin-x64 release-darwin-arm64 \
        _release pack
