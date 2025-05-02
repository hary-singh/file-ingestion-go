.PHONY: build test clean dev docker test-local check-prereqs stop start restart logs rebuild

# Variables
BINARY_NAME=validator
BINARY_PATH=bin/$(BINARY_NAME)
MAX_WAIT=60

# Build and test
build:
	go build -o $(BINARY_PATH) ./cmd/function

test:
	go test -v -race ./...

clean:
	rm -rf bin/
	docker-compose down -v

# Docker commands
docker:
	docker build --no-cache -t validator-function .

# Service management
start: check-prereqs
	@echo "Starting services..."
	docker-compose up -d
	@echo "Waiting for services to be healthy..."
	@for i in $$(seq 1 $(MAX_WAIT)); do \
		if docker-compose ps | grep -q "healthy"; then \
			echo "Services are healthy"; \
			exit 0; \
		fi; \
		echo "Waiting for services... ($$i/$(MAX_WAIT))"; \
		sleep 2; \
	done; \
	echo "Services failed to become healthy within $(MAX_WAIT) seconds"; \
	exit 1

stop:
	@echo "Stopping services..."
	docker-compose down

restart: stop start

logs:
	docker-compose logs -f

# Rebuild specific service
rebuild:
	docker-compose build --no-cache validator
	docker-compose up -d validator

# Testing helpers
test-local: clean
	@echo "Building and starting services..."
	docker-compose build validator
	docker-compose up -d
	@echo "Running local tests..."
	@chmod +x ./test-files/test-local.sh
	./test-files/test-local.sh

# Prerequisites check
check-prereqs:
	@echo "Checking prerequisites..."
	@command -v az >/dev/null 2>&1 || { echo "Error: Azure CLI is required"; exit 1; }
	@command -v docker-compose >/dev/null 2>&1 || { echo "Error: Docker Compose is required"; exit 1; }
	@command -v curl >/dev/null 2>&1 || { echo "Error: curl is required"; exit 1; }

# Development workflow
dev: build start