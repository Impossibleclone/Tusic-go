# Variables
BINARY_NAME=tusic
MAIN_PATH=cmd/tusic/main.go

# Default target when you just run 'make'
.DEFAULT_GOAL := help

.PHONY: build run clean tidy fmt help

build: ## Compile the highly optimized standalone binary
	@echo "Building $(BINARY_NAME)..."
	go build -ldflags="-s -w" -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "Done! Run with ./$(BINARY_NAME)"

run: ## Run the application directly (for development)
	@echo "Running $(BINARY_NAME)..."
	go run $(MAIN_PATH)

clean: ## Remove the compiled binary
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME)
	@echo "Clean complete."

tidy: ## Format code and download missing dependencies
	@echo "Tidying modules..."
	go mod tidy
	go fmt ./...

install: build ## Build and install the binary to your Go bin directory
	@echo "Installing $(BINARY_NAME) to $(GOPATH)/bin..."
	go install -ldflags="-s -w" $(MAIN_PATH)

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-10s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
