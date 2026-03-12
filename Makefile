.PHONY: build install clean test run-example

# Build the docksmith binary
build:
	go build -o docksmith .

# Install docksmith to GOPATH/bin
install:
	go install

# Clean build artifacts
clean:
	rm -f docksmith
	go clean

# Run tests
test:
	go test ./...

# Build and run simple example
run-example: build
	./docksmith build -t simple:latest examples/simple-app
	./docksmith images
	./docksmith run simple:latest

# Download dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	go vet ./...
