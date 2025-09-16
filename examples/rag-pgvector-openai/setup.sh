#!/bin/bash

# Production RAG Setup Script
# This script sets up the production environment for RAG with pgvector and OpenAI

set -e

echo "🚀 Setting up Production RAG Environment..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if required tools are installed
check_requirements() {
    echo "Checking requirements..."
    
    # Check Go
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go 1.21 or later."
        exit 1
    fi
    print_status "Go is installed: $(go version)"
    
    # Check PostgreSQL
    if ! command -v psql &> /dev/null; then
        print_warning "psql command not found. Make sure PostgreSQL is installed and accessible."
    else
        print_status "PostgreSQL client is available"
    fi
    
    # Check Docker (optional)
    if command -v docker &> /dev/null; then
        print_status "Docker is available (optional)"
    else
        print_warning "Docker not found (optional for running PostgreSQL)"
    fi
}

# Install Go dependencies
install_dependencies() {
    echo "Installing Go dependencies..."
    
    if [ ! -f "go.mod" ]; then
        print_error "go.mod not found. Make sure you're in the production-rag directory."
        exit 1
    fi
    
    go mod tidy
    go mod download
    print_status "Go dependencies installed"
}

# Setup environment file
setup_env() {
    echo "Setting up environment configuration..."
    
    if [ ! -f ".env" ]; then
        if [ -f ".env.example" ]; then
            cp .env.example .env
            print_status "Created .env from .env.example"
            print_warning "Please edit .env file with your actual configuration values:"
            echo "  - Set OPENAI_API_KEY to your OpenAI API key"
            echo "  - Set DB_PASSWORD to your PostgreSQL password"
            echo "  - Adjust other settings as needed"
        else
            print_error ".env.example not found"
            exit 1
        fi
    else
        print_status ".env file already exists"
    fi
}

# Start PostgreSQL with Docker (optional)
start_postgres_docker() {
    read -p "Do you want to start PostgreSQL with Docker? (y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "Starting PostgreSQL with pgvector using Docker..."
        
        # Check if container already exists
        if docker ps -a | grep -q postgres-vector; then
            print_status "PostgreSQL container already exists"
            docker start postgres-vector || print_warning "Failed to start existing container"
        else
            # Run new container
            docker run --name postgres-vector \
                -e POSTGRES_PASSWORD=postgres \
                -e POSTGRES_DB=rag_production \
                -p 5432:5432 \
                -d ankane/pgvector
            
            print_status "PostgreSQL with pgvector started in Docker"
        fi
        
        # Wait for PostgreSQL to be ready
        echo "Waiting for PostgreSQL to be ready..."
        sleep 10
        
        # Update .env with Docker settings
        if grep -q "DB_PASSWORD=your_password_here" .env; then
            sed -i.bak 's/DB_PASSWORD=your_password_here/DB_PASSWORD=postgres/' .env
            print_status "Updated .env with Docker PostgreSQL password"
        fi
    fi
}

# Initialize database
init_database() {
    echo "Initializing database..."
    
    # Check if .env has required values
    if grep -q "DB_PASSWORD=your_password_here" .env || grep -q "OPENAI_API_KEY=your_openai_api_key_here" .env; then
        print_error "Please update .env file with your actual credentials before initializing database"
        exit 1
    fi
    
    # Run database initialization
    go run setup/init_db.go
    print_status "Database initialized"
}

# Build the application
build_app() {
    echo "Building application..."
    go build -o bin/production-rag main.go
    print_status "Application built successfully"
}

# Run tests
run_tests() {
    read -p "Do you want to run tests? (y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "Running tests..."
        go test ./... -v
        print_status "Tests completed"
    fi
}

# Main setup process
main() {
    echo "🔧 Production RAG Setup with pgvector and OpenAI"
    echo "================================================"
    
    check_requirements
    install_dependencies
    setup_env
    start_postgres_docker
    init_database
    build_app
    run_tests
    
    echo
    print_status "Setup completed successfully!"
    echo
    echo "📝 Next steps:"
    echo "1. Edit .env file with your OpenAI API key and database credentials"
    echo "2. Run the application: ./bin/production-rag"
    echo "3. Or run with go: go run main.go"
    echo
    echo "📚 Documentation:"
    echo "- README.md: Comprehensive setup and usage guide"
    echo "- .env.example: Configuration options"
    echo
}

# Run main function
main
