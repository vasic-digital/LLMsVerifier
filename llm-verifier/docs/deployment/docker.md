# Docker Deployment Guide

This guide covers deploying the LLM Verifier application using Docker and Docker Compose.

## Prerequisites

- Docker Engine 20.10+
- Docker Compose 2.0+
- At least 4GB RAM available
- At least 10GB disk space

## Quick Start

1. **Clone the repository:**
   ```bash
   git clone https://github.com/your-org/llm-verifier.git
   cd llm-verifier
   ```

2. **Configure environment:**
   ```bash
   cp config/examples/production.yaml config/production.yaml
   # Edit config/production.yaml with your settings
   ```

3. **Start the application:**
   ```bash
   docker-compose up -d
   ```

4. **Verify deployment:**
   ```bash
   curl http://localhost:8080/health
   ```

## Configuration

### Environment Variables

Create a `.env` file in the project root:

```env
# Database
DATABASE_ENCRYPTION_KEY=your-32-character-encryption-key

# API Configuration
JWT_SECRET=your-jwt-secret-key
API_PORT=8080
RATE_LIMIT=100
BURST_LIMIT=200

# LLM Providers
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...
GOOGLE_API_KEY=...
META_API_KEY=...

# Logging
LOG_LEVEL=info
LOG_FILE=/app/logs/llm-verifier.log

# Monitoring
PROMETHEUS_ENABLED=true
HEALTH_CHECK_INTERVAL=30s
```

### Docker Compose Configuration

The `docker-compose.yml` includes:

- **llm-verifier**: Main application
- **postgres**: Database (optional, uses SQLite by default)
- **redis**: Caching and session storage
- **prometheus**: Metrics collection
- **grafana**: Dashboard for metrics
- **nginx**: Reverse proxy and load balancer

## Production Deployment

### 1. Multi-stage Docker Build

The Dockerfile uses multi-stage builds for optimal image size:

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/config ./config
CMD ["./main"]
```

### 2. Security Hardening

```dockerfile
# Use non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup
USER appuser

# Remove unnecessary packages
RUN apk del --no-cache git
```

### 3. Health Checks

```yaml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 40s
```

## Scaling and Load Balancing

### Horizontal Scaling

```yaml
services:
  llm-verifier:
    deploy:
      replicas: 3
      resources:
        limits:
          cpus: '1.0'
          memory: 1G
        reservations:
          cpus: '0.5'
          memory: 512M
```

### Load Balancer Configuration

```yaml
services:
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
    depends_on:
      - llm-verifier
```

## Monitoring and Logging

### Prometheus Metrics

Access metrics at: `http://localhost:9090/metrics`

### Grafana Dashboards

Access dashboards at: `http://localhost:3000`

Default credentials: admin/admin

### Centralized Logging

```yaml
services:
  llm-verifier:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

## Backup and Recovery

### Database Backup

```bash
# Backup SQLite database
docker exec llm-verifier-db sqlite3 /data/database.db ".backup /backups/backup-$(date +%Y%m%d-%H%M%S).db"

# Backup PostgreSQL
docker exec llm-verifier-db pg_dump -U postgres llm_verifier > backup.sql
```

### Automated Backups

```yaml
services:
  backup:
    image: alpine:latest
    command: sh -c "while true; do sleep 86400; sqlite3 /data/database.db '.backup /backups/backup-$(date +%%Y%%m%%d-%%H%%M%%S).db'; done"
    volumes:
      - ./data:/data
      - ./backups:/backups
```

## Troubleshooting

### Common Issues

1. **Port already in use:**
   ```bash
   # Check what's using the port
   lsof -i :8080
   # Change port in docker-compose.yml
   ```

2. **Database connection failed:**
   ```bash
   # Check database logs
   docker-compose logs postgres
   # Verify environment variables
   ```

3. **Memory issues:**
   ```bash
   # Increase Docker memory limit
   # Check container resource usage
   docker stats
   ```

### Logs

```bash
# View application logs
docker-compose logs llm-verifier

# Follow logs in real-time
docker-compose logs -f llm-verifier

# View logs for specific time period
docker-compose logs --since "2024-01-01T00:00:00" llm-verifier
```

## Performance Optimization

### Resource Limits

```yaml
services:
  llm-verifier:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 4G
        reservations:
          cpus: '1.0'
          memory: 2G
```

### Database Optimization

```yaml
services:
  postgres:
    environment:
      POSTGRES_SHARED_BUFFERS: 256MB
      POSTGRES_EFFECTIVE_CACHE_SIZE: 1GB
      POSTGRES_MAINTENANCE_WORK_MEM: 64MB
    command: postgres -c wal_level=logical -c max_wal_senders=4
```

## Security Considerations

### Network Security

```yaml
services:
  llm-verifier:
    networks:
      - internal
    # No external ports exposed
```

### Secrets Management

```yaml
secrets:
  db_password:
    file: ./secrets/db_password.txt
  api_keys:
    file: ./secrets/api_keys.txt
```

## Updating the Application

### Rolling Updates

```bash
# Update images
docker-compose pull

# Rolling restart
docker-compose up -d --no-deps llm-verifier
```

### Zero-downtime Deployment

```bash
# Deploy new version
docker-compose up -d --scale llm-verifier=2

# Wait for health checks
sleep 30

# Remove old instances
docker-compose up -d --scale llm-verifier=1
```

## Advanced Configuration

### Custom Networks

```yaml
networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge
    internal: true
```

### Volume Management

```yaml
volumes:
  db_data:
    driver: local
    driver_opts:
      type: tmpfs
      device: tmpfs
  logs:
    driver: local
```