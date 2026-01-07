.PHONY: help generate build run lint format clean setup

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

generate: ## Generate protobuf code using protoc
	@echo "Generating protobuf files with protoc..."
	@which protoc > /dev/null || (echo "Error: protoc not found. Run 'make setup' or './scripts/setup.sh' first" && exit 1)
	@mkdir -p core/pb
	@export PATH="$$(go env GOPATH)/bin:$$PATH" && \
	protoc \
		-I. \
		--go_out=core/pb \
		--go_opt=paths=source_relative \
		proto/connectors.proto \
		proto/primitives.proto \
		proto/hyperterse.proto \
		proto/runtime.proto && \
	echo "Moving generated files to correct location..." && \
	if [ -f "core/pb/proto/hyperterse.pb.go" ]; then \
		mv core/pb/proto/hyperterse.pb.go core/pb/ 2>/dev/null || true; \
		mv core/pb/proto/connectors.pb.go core/pb/ 2>/dev/null || true; \
		mv core/pb/proto/primitives.pb.go core/pb/ 2>/dev/null || true; \
		mv core/pb/proto/runtime.pb.go core/pb/ 2>/dev/null || true; \
		rmdir core/pb/proto 2>/dev/null || true; \
	fi && \
	if [ -d "core/pb/hyperterse" ]; then \
		mv core/pb/hyperterse/hyperterse.pb.go core/pb/ 2>/dev/null || true; \
		mv core/pb/hyperterse/connectors.pb.go core/pb/ 2>/dev/null || true; \
		mv core/pb/hyperterse/primitives.pb.go core/pb/ 2>/dev/null || true; \
		rmdir core/pb/hyperterse 2>/dev/null || true; \
	fi && \
	if [ -f "core/pb/runtime.pb.go" ]; then \
		mkdir -p core/pb/runtime && \
		mv core/pb/runtime.pb.go core/pb/runtime/ 2>/dev/null || true; \
	fi && \
	echo "Generating types..." && \
	mkdir -p core/types && \
	go run scripts/generate_types/script.go proto/connectors.proto proto/primitives.proto && \
	echo "✓ Protobuf generation complete"

build: generate ## Build the project
	@echo "Building hyperterse..."
	go build -mod=mod -o hyperterse
	@echo "✓ Build complete"

run: build ## Build and run the server (requires CONFIG_FILE env var or -file flag)
	@if [ -z "$(CONFIG_FILE)" ]; then \
		echo "Usage: make run CONFIG_FILE=path/to/config.yaml"; \
		echo "   or: ./hyperterse -file path/to/config.yaml"; \
	else \
		./hyperterse -file $(CONFIG_FILE); \
	fi

lint: ## Lint proto files (requires buf CLI - optional)
	@echo "Linting proto files..."
	@if command -v buf > /dev/null; then \
		cd proto && buf lint; \
	else \
		echo "⚠️  buf CLI not found. Install it for linting: https://buf.build/docs/installation"; \
	fi

format: ## Format proto files (requires buf CLI - optional)
	@echo "Formatting proto files..."
	@if command -v buf > /dev/null; then \
		cd proto && buf format -w; \
	else \
		echo "⚠️  buf CLI not found. Install it for formatting: https://buf.build/docs/installation"; \
	fi

clean: ## Clean generated files and binaries
	@echo "Cleaning generated files..."
	rm -rf core/pb/*.pb.go core/pb/runtime
	rm -f hyperterse
	@echo "✓ Clean complete"

setup: ## Complete setup: install dependencies and generate code
	@echo "Running setup script..."
	@./scripts/setup.sh
	@echo ""
	@echo "✓ Setup complete! Run 'make build' to build the project."

