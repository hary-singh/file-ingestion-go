.PHONY: all build test clean dev docker test-small test-large test-all check-prereqs stop start restart logs rebuild

# Default target
all: build

# Check prerequisites
check-prereqs:
	@command -v docker >/dev/null 2>&1 || { echo "docker is required but not installed. Aborting." >&2; exit 1; }
	@command -v docker-compose >/dev/null 2>&1 || { echo "docker-compose is required but not installed. Aborting." >&2; exit 1; }
	@command -v az >/dev/null 2>&1 || { echo "azure-cli is required but not installed. Aborting." >&2; exit 1; }

# Build the application
build: check-prereqs
	@echo "Building validator function..."
	@docker-compose build validator

# Clean up resources
clean: stop
	@echo "Cleaning up..."
	@docker-compose down -v

# Development environment
dev: start
	@echo "Starting development environment..."
	@docker-compose logs -f

# Container management
start: check-prereqs
	@echo "Starting services..."
	@docker-compose up -d

stop:
	@echo "Stopping services..."
	@docker-compose down

restart: stop start

# Testing
test-small: start
	@echo "Running small file test..."
	@chmod +x test-files/test-small.sh
	@./test-files/test-small.sh

test-large: start
	@echo "Running large file test..."
	@chmod +x test-files/test-large.sh
	@./test-files/test-large.sh

test-all: clean start
	@echo "Running all tests..."
	@make test-small
	@make test-large

# Utility commands
logs:
	@docker-compose logs -f

rebuild: clean build start