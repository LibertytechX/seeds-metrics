.PHONY: help build run test clean docker-build docker-up docker-down docker-logs db-migrate db-reset

# Default target
help:
	@echo "Available commands:"
	@echo "  make build        - Build the Go binary"
	@echo "  make run          - Run the application locally"
	@echo "  make test         - Run tests"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-up    - Start all Docker services"
	@echo "  make docker-down  - Stop all Docker services"
	@echo "  make docker-logs  - View Docker logs"
	@echo "  make db-migrate   - Run database migrations"
	@echo "  make db-reset     - Reset database (⚠️  deletes all data)"

# Build the Go binary
build:
	@echo "🔨 Building Go binary..."
	go build -o bin/analytics-api cmd/api/main.go
	@echo "✅ Build complete: bin/analytics-api"

# Run the application locally
run:
	@echo "🚀 Starting application..."
	go run cmd/api/main.go

# Run tests
test:
	@echo "🧪 Running tests..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf bin/
	@echo "✅ Clean complete"

# Build Docker image
docker-build:
	@echo "🐳 Building Docker image..."
	docker-compose build
	@echo "✅ Docker image built"

# Start all Docker services
docker-up:
	@echo "🚀 Starting Docker services..."
	docker-compose up -d
	@echo "✅ Services started"
	@echo ""
	@echo "📡 API: http://localhost:8080"
	@echo "🗄️  PostgreSQL: localhost:5432"
	@echo "🔴 Redis: localhost:6379"
	@echo ""
	@echo "Run 'make docker-logs' to view logs"

# Stop all Docker services
docker-down:
	@echo "🛑 Stopping Docker services..."
	docker-compose down
	@echo "✅ Services stopped"

# View Docker logs
docker-logs:
	docker-compose logs -f

# Run database migrations
db-migrate:
	@echo "📊 Running database migrations..."
	docker exec -it analytics-postgres psql -U analytics_user -d analytics_db -f /docker-entrypoint-initdb.d/001_initial_schema.sql
	@echo "✅ Migrations complete"

# Reset database (⚠️ deletes all data)
db-reset:
	@echo "⚠️  WARNING: This will delete all data!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker-compose down -v; \
		docker-compose up -d postgres; \
		sleep 5; \
		make db-migrate; \
		echo "✅ Database reset complete"; \
	else \
		echo "❌ Cancelled"; \
	fi

# Install Go dependencies
deps:
	@echo "📦 Installing Go dependencies..."
	go mod download
	@echo "✅ Dependencies installed"

# Format code
fmt:
	@echo "🎨 Formatting code..."
	go fmt ./...
	@echo "✅ Code formatted"

# Lint code
lint:
	@echo "🔍 Linting code..."
	golangci-lint run
	@echo "✅ Linting complete"

# Create .env file from example
env:
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "✅ Created .env file from .env.example"; \
	else \
		echo "⚠️  .env file already exists"; \
	fi

