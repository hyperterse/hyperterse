.PHONY: help generate build run setup

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

generate: ## Generate protobuf code using protoc
	@./scripts/generate-proto.sh

build: generate ## Build the project
	@./scripts/build.sh

run: build ## Build and run the server (requires CONFIG_FILE env var or -file flag)
	@if [ -z "$(CONFIG_FILE)" ]; then \
		echo "Usage: make run CONFIG_FILE=path/to/config.terse"; \
		echo "   or: ./hyperterse -file path/to/config.terse"; \
	else \
		./hyperterse -file $(CONFIG_FILE); \
	fi

setup: ## Complete setup: install dependencies and generate code
	@echo "Running setup script..."
	@./scripts/setup.sh
	@echo ""
	@echo "✓ Setup complete! Run 'make build' to build the project."

