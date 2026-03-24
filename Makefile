.PHONY: all build test test-cover lint clean

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary names
MAIN_CMD=.
BIN_DIR=./bin

# Coverage
COVERAGE_DIR=./coverage
COVERAGE_FILE=$(COVERAGE_DIR)/coverage.out

all: lint test build

build:
	@echo "Building..."
	$(GOBUILD) ./...

test:
	@echo "Running tests..."
	$(GOTEST) -v -race ./...

test-cover:
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report generated: $(COVERAGE_DIR)/coverage.html"

lint:
	@echo "Running linters..."
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run ./...

clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BIN_DIR)
	rm -rf $(COVERAGE_DIR)

deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

# Integration tests (requires WEIXIN_TEST_TOKEN env)
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -race -tags=integration ./...

# Generate mocks
mocks:
	@echo "Generating mocks..."
	@which mockgen > /dev/null || go install github.com/golang/mock/mockgen@latest
	mockgen -source=plugin/plugin.go -destination=plugin/mock_plugin.go -package=plugin

# CI targets
ci: deps lint test-cover

# Development
dev: deps fmt vet test