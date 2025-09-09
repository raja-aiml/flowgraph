# FlowGraph Dockerfile - Multi-stage build for optimal size and security

# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
RUN make build

# Final stage - minimal runtime image
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -g 1001 -S flowgraph \
    && adduser -S flowgraph -u 1001 -G flowgraph

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary from builder stage
COPY --from=builder /app/bin/flowgraph /usr/local/bin/flowgraph

# Create necessary directories
RUN mkdir -p /app/data /app/logs \
    && chown -R flowgraph:flowgraph /app

# Switch to non-root user
USER flowgraph

# Set working directory
WORKDIR /app

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD flowgraph health || exit 1

# Expose port
EXPOSE 8080

# Default command
ENTRYPOINT ["flowgraph"]
CMD ["server", "--config", "/app/config.yaml"]

# Labels
LABEL maintainer="FlowGraph Team"
LABEL version="1.0.0"
LABEL description="FlowGraph - High-performance Go implementation of graph-based workflows"
