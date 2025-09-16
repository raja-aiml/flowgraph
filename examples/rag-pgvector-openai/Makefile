.PHONY: help setup build run test clean docker-up docker-down deps

# Default target
help: ## Show this help message
	@echo "Production RAG with pgvector and OpenAI"
	@echo "Available commands:"
	@awk 'BEGIN {FS = ":.*##"}; /^[a-zA-Z_-]+:.*?##/ { printf "  %-15s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

setup: ## Run complete setup (dependencies, env, database)
	@echo "🚀 Running complete setup..."
	./setup.sh

deps: ## Install Go dependencies
	@echo "📦 Installing dependencies..."
	go mod tidy
	go mod download

build: ## Build the application
	@echo "🔨 Building application..."
	go build -o bin/production-rag main.go

run: build ## Run the application
	@echo "🏃 Running production RAG..."
	./bin/production-rag

test: ## Run tests
	@echo "🧪 Running tests..."
	go test ./... -v

test-coverage: ## Run tests with coverage
	@echo "📊 Running tests with coverage..."
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

init-db: ## Initialize database only
	@echo "🗄️  Initializing database..."
	go run setup/init_db.go

clean: ## Clean build artifacts
	@echo "🧹 Cleaning up..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -f *.log

docker-up: ## Start services with Docker Compose
	@echo "🐳 Starting services with Docker Compose..."
	docker-compose up -d

docker-down: ## Stop Docker Compose services
	@echo "🐳 Stopping Docker Compose services..."
	docker-compose down

docker-logs: ## View Docker Compose logs
	docker-compose logs -f

docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	docker build -t production-rag:latest .

lint: ## Run linters
	@echo "🔍 Running linters..."
	golangci-lint run

format: ## Format code
	@echo "✨ Formatting code..."
	go fmt ./...
	goimports -w .

dev: ## Development mode with hot reload
	@echo "🔄 Starting development mode..."
	air

benchmark: ## Run benchmarks
	@echo "⚡ Running benchmarks..."
	go test ./... -bench=. -benchmem

# Database operations
db-reset: ## Reset database (DROP and CREATE)
	@echo "🔄 Resetting database..."
	@read -p "Are you sure you want to reset the database? (y/N): " confirm && [ "$$confirm" = "y" ]
	go run setup/reset_db.go

db-migrate: ## Run database migrations
	@echo "📈 Running database migrations..."
	go run setup/migrate.go

# Environment management  
env-check: ## Check environment configuration
	@echo "🔧 Checking environment..."
	@go run -c 'package main; import ("production-rag/internal/config"); func main() { cfg, err := config.LoadConfig(); if err != nil { panic(err) }; println("✅ Configuration is valid") }'

# Production deployment helpers
deploy-staging: ## Deploy to staging
	@echo "🚀 Deploying to staging..."
	# Add your staging deployment commands here

deploy-prod: ## Deploy to production  
	@echo "🚀 Deploying to production..."
	# Add your production deployment commands here
	@read -p "Are you sure you want to deploy to production? (y/N): " confirm && [ "$$confirm" = "y" ]

# Monitoring and health checks
health-check: ## Run health check
	@echo "🏥 Running health check..."
	curl -f http://localhost:8080/health || echo "Health check failed"

logs: ## View application logs
	@echo "📋 Viewing logs..."
	tail -f *.log

# Documentation
docs: ## Generate documentation
	@echo "📚 Generating documentation..."
	godoc -http=:6060
	@echo "Documentation available at http://localhost:6060"

install: ## Install the application globally
	@echo "📦 Installing application..."
	go install .

uninstall: ## Uninstall the application
	@echo "🗑️  Uninstalling application..."
	rm -f $(GOPATH)/bin/production-rag
