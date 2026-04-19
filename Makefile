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

.PHONY: build run test clean tidy lint
