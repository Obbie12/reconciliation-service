.PHONY: build run test migrate-up migrate-down migrate-version clean deps setup help

# Build commands
build:
	go build -o reconciliation-service cmd/server/main.go

# Run the service
run: build
	./reconciliation-service

# Run tests
test:
	go test -v ./...

# Migration commands
migrate-up:
	./reconciliation-service -migrate=up

migrate-down:
	./reconciliation-service -migrate=down

migrate-version:
	./reconciliation-service -migrate=version

# Clean built binaries
clean:
	rm -f reconciliation-service

# Install dependencies
deps:
	go mod download
	go mod tidy

# Development setup
setup: deps build migrate-up

# Help
help:
	@echo "Available commands:"
	@echo "  make build          - Build the service"
	@echo "  make run           - Build and run the service"
	@echo "  make test          - Run tests"
	@echo "  make migrate-up    - Run all pending migrations"
	@echo "  make migrate-down  - Rollback all migrations"
	@echo "  make migrate-version - Show current migration version"
	@echo "  make clean         - Remove built binaries"
	@echo "  make deps          - Install dependencies"
	@echo "  make setup         - Initial setup (install deps, build, migrate)"

# Environment setup
env-setup:
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "Created .env file from .env.example"; \
		echo "Please update the .env file with your configuration"; \
	else \
		echo ".env file already exists"; \
	fi
