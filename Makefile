TARGET = vidify
CLI_ENTRY = ./cmd/cli

# Build the CLI binary from the cmd/cli entry point.
build:
	go build -o $(TARGET).exe $(CLI_ENTRY)

# Run the CLI directly without building a binary first.
run:
	go run $(CLI_ENTRY) $(ARGS)

# Run all tests with race detection enabled.
test:
	go test -race -count=1 ./...

# Remove build artifacts.
clean:
	rm -f $(TARGET).exe $(TARGET)

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

.PHONY: build run test clean tidy lint fmt init cover
