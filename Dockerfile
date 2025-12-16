# Multi-stage Dockerfile for LLM Verifier
# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o llm-verifier cmd/main.go

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create a non-root user
RUN adduser -D -s /bin/sh llm-verifier

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/llm-verifier .

# Create directory for database and exports
RUN mkdir -p /app/data /app/exports

# Change ownership to non-root user
RUN chown -R llm-verifier:llm-verifier /app

# Switch to non-root user
USER llm-verifier

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Set default database path
ENV LLM_DB_PATH=/app/data/llm-verifier.db

# Run the application
CMD ["./llm-verifier", "server", "--port", "8080"]