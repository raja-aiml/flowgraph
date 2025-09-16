# Production RAG with pgvector and OpenAI

This example demonstrates a production-ready RAG (Retrieval-Augmented Generation) implementation using:

- **PostgreSQL with pgvector extension** for vector storage and similarity search
- **OpenAI API** for embeddings and text generation
- **Connection pooling** and comprehensive error handling
- **Environment-based configuration** management
- **Structured logging** and monitoring
- **Docker support** for easy deployment
- **CI/CD ready** with proper testing and linting

## 🚀 Quick Start

### Option 1: Automated Setup (Recommended)
```bash
cd /Users/rajasoun/workspace/dev/flowgraph/examples/production-rag
./setup.sh
```

### Option 2: Manual Setup

1. **Install Dependencies**
```bash
go mod tidy
```

2. **Setup Environment**
```bash
cp .env.example .env
# Edit .env with your credentials
```

3. **Initialize Database**
```bash
go run setup/init_db.go
```

4. **Run Application**
```bash
go run main.go
```

### Option 3: Docker Compose (Easiest)
```bash
# Set your OpenAI API key
export OPENAI_API_KEY=your_key_here

# Start everything
docker-compose up -d
```

## 📋 Prerequisites

### Required
- **Go 1.21+**
- **PostgreSQL 14+** with pgvector extension
- **OpenAI API key**

### Installation Options

#### macOS (using Homebrew)
```bash
# Install PostgreSQL and pgvector
brew install postgresql pgvector

# Start PostgreSQL
brew services start postgresql

# Create database
createdb rag_production
```

#### Linux (Ubuntu/Debian)
```bash
# Install PostgreSQL
sudo apt update
sudo apt install postgresql postgresql-contrib

# Install pgvector
sudo apt install postgresql-14-pgvector
```

#### Docker (Cross-platform)
```bash
docker run --name postgres-vector \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=rag_production \
  -p 5432:5432 \
  -d ankane/pgvector
```

## ⚙️ Configuration

### Environment Variables

Create a `.env` file from the example:

```bash
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_NAME=rag_production
DB_USER=postgres
DB_PASSWORD=your_password_here
DB_SSL_MODE=disable
DB_MAX_CONNECTIONS=25
DB_MIN_CONNECTIONS=5

# OpenAI Configuration
OPENAI_API_KEY=your_openai_api_key_here
OPENAI_MODEL=gpt-4
OPENAI_EMBEDDING_MODEL=text-embedding-ada-002
OPENAI_MAX_TOKENS=1500
OPENAI_TEMPERATURE=0.7

# Vector Search Configuration
VECTOR_DIMENSIONS=1536
MAX_SEARCH_RESULTS=10
SIMILARITY_THRESHOLD=0.7

# Application Configuration
LOG_LEVEL=info
SERVER_PORT=8080
REQUEST_TIMEOUT=30s
```

## 🏗️ Architecture

### System Flow
```
Query Input
    ↓
[Query Processor] → Clean and validate input
    ↓
[OpenAI Embedder] → Convert text to 1536-dim vectors
    ↓
[pgvector Search] → Find similar documents using cosine similarity
    ↓
[Result Ranker] → Re-rank results based on relevance
    ↓
[OpenAI Generator] → Generate response with retrieved context
    ↓
Response Output
```

### Component Details

1. **Query Processor**: Input validation and preprocessing
2. **Embedder**: OpenAI text-embedding-ada-002 for vector generation
3. **Vector Database**: PostgreSQL + pgvector for efficient similarity search
4. **Search Engine**: Cosine similarity search with configurable thresholds
5. **Re-ranker**: Advanced ranking based on content quality and relevance
6. **Generator**: OpenAI GPT-4 for contextual response generation

## 🔧 Usage Examples

### Basic RAG Query
```go
// Initialize the system
ragRuntime := rag.NewEnhancedRAGRuntime()

// Query the system
result, err := ragRuntime.ExecuteRAGGraph(ctx, graph, "thread_1", map[string]interface{}{
    "query": "What is machine learning?",
})
```

### Advanced RAG with Re-ranking
```go
// Build advanced pipeline with re-ranking
config := rag.CustomSimpleRAGConfig(
    "advanced-rag",
    "Advanced RAG with Re-ranking",
    "query",
    nodes,
    edges,
)

graph, err := buildRAGGraph(config)
result, err := ragRuntime.ExecuteRAGGraph(ctx, graph, "advanced_thread", input)
```

## 🛠️ Development Commands

Use the included Makefile for common operations:

```bash
# Setup everything
make setup

# Build application
make build

# Run application
make run

# Run tests
make test

# Run with Docker
make docker-up

# View logs
make logs

# Clean build artifacts
make clean

# Format code
make format

# Run linters
make lint

# Generate documentation
make docs
```

## 🗄️ Database Schema

The system automatically creates the following schema:

```sql
-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Documents table with vector embeddings
CREATE TABLE documents (
    id TEXT PRIMARY KEY,
    content TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    embedding vector(1536),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX documents_embedding_idx ON documents 
    USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
    
CREATE INDEX documents_metadata_idx ON documents USING GIN (metadata);
```

## 📊 Performance Features

### Vector Search Optimization
- **IVFFlat index** for fast approximate nearest neighbor search
- **Configurable similarity thresholds** to balance accuracy and performance
- **Connection pooling** for database efficiency
- **Batch embedding** for multiple documents

### Memory Management
- **Streaming processing** for large documents
- **Result pagination** to handle large result sets
- **Configurable token limits** to control costs
- **Connection pool sizing** based on workload

### Error Handling
- **Retry mechanisms** for API calls
- **Circuit breakers** for external services
- **Graceful degradation** when services are unavailable
- **Comprehensive logging** for debugging

## 🧪 Testing

### Run All Tests
```bash
make test
```

### Test Coverage
```bash
make test-coverage
open coverage.html
```

### Integration Tests
```bash
# Start test database
docker-compose -f docker-compose.test.yml up -d

# Run integration tests
go test -tags=integration ./...
```

### Load Testing
```bash
# Benchmark vector operations
make benchmark

# Load test the API endpoints
go test -bench=BenchmarkRAGPipeline ./...
```

## 🚀 Deployment

### Docker Deployment
```bash
# Build image
docker build -t production-rag:latest .

# Run with docker-compose
docker-compose up -d

# Check health
curl http://localhost:8080/health
```

### Production Deployment
```bash
# Deploy to staging
make deploy-staging

# Deploy to production (requires confirmation)
make deploy-prod
```

### Environment-Specific Configurations

Create environment-specific `.env` files:
- `.env.development` - Development settings
- `.env.staging` - Staging environment
- `.env.production` - Production environment

## 🔍 Monitoring and Observability

### Health Checks
```bash
# Application health
make health-check

# Database health
curl http://localhost:8080/health/db

# OpenAI API health  
curl http://localhost:8080/health/openai
```

### Logging
```bash
# View application logs
make logs

# Filter by log level
grep "ERROR" *.log

# Monitor in real-time
tail -f production-rag.log
```

### Metrics
The application exposes metrics for:
- Query processing time
- Embedding generation latency
- Vector search performance
- Database connection pool usage
- OpenAI API usage and costs

## 🔐 Security Considerations

### API Key Management
- Store API keys in environment variables or secure vaults
- Use different keys for different environments
- Implement key rotation procedures
- Monitor API usage for anomalies

### Database Security
- Use SSL/TLS for database connections
- Implement connection string encryption
- Regular security patches for PostgreSQL
- Backup and disaster recovery procedures

### Application Security
- Input validation and sanitization
- Rate limiting for API endpoints
- Authentication and authorization
- Audit logging for sensitive operations

## 📈 Scaling Considerations

### Horizontal Scaling
- **Stateless application** design enables easy horizontal scaling
- **Load balancer** configuration for multiple instances
- **Database read replicas** for read-heavy workloads
- **Caching layer** (Redis) for frequently accessed data

### Vertical Scaling
- **Connection pool tuning** based on database capacity
- **Memory optimization** for large embedding datasets
- **CPU optimization** for vector operations
- **Storage optimization** for database performance

## 🔧 Troubleshooting

### Common Issues

#### Database Connection Issues
```bash
# Check PostgreSQL status
pg_isready -h localhost -p 5432

# Test connection manually
psql -h localhost -p 5432 -U postgres -d rag_production

# Check pgvector extension
SELECT * FROM pg_extension WHERE extname = 'vector';
```

#### OpenAI API Issues
```bash
# Test API connectivity
curl -H "Authorization: Bearer $OPENAI_API_KEY" \
     https://api.openai.com/v1/models

# Check rate limits in logs
grep "rate_limit" production-rag.log
```

#### Performance Issues
```bash
# Check database performance
EXPLAIN ANALYZE SELECT * FROM documents 
ORDER BY embedding <=> '[your_vector]' LIMIT 10;

# Monitor connection pool
grep "connection_pool" production-rag.log
```

## 📚 Additional Resources

### Documentation
- [pgvector Documentation](https://github.com/pgvector/pgvector)
- [OpenAI API Documentation](https://platform.openai.com/docs)
- [FlowGraph Framework](../../README.md)

### Community
- [Issues and Bug Reports](https://github.com/your-org/flowgraph/issues)
- [Discussions](https://github.com/your-org/flowgraph/discussions)
- [Contributing Guide](../../CONTRIBUTING.md)

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](../../LICENSE) file for details.
