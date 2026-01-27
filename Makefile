.PHONY: build run test clean coverage

# Build the application
build:
	go build -o avweather_cache

# Run the application
run: build
	./avweather_cache

# Run tests
test:
	go test -v ./...

# Run tests with coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
clean:
	rm -f avweather_cache coverage.out coverage.html

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Run the application with race detector
run-race: build
	go run -race .
