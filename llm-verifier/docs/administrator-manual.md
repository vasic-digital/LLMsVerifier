# LLM Verifier Administrator Manual

## System Overview

LLM Verifier is an enterprise-grade platform for comprehensive testing, benchmarking, and verification of Large Language Models across multiple providers. This manual provides administrators with complete instructions for installation, configuration, operation, and maintenance.

## Prerequisites

### System Requirements
- **Operating System**: Linux (Ubuntu 20.04+, CentOS 8+, RHEL 8+)
- **Memory**: 8GB RAM minimum, 16GB recommended
- **Storage**: 50GB available disk space
- **Network**: Stable internet connection for provider API access

### Software Dependencies
- **Go**: Version 1.21 or higher
- **SQLite**: Version 3.35 or higher (with SQLCipher support)
- **Docker**: Version 20.10 or higher (for containerized deployment)
- **Git**: Version 2.25 or higher

### Network Requirements
- **Outbound HTTPS**: Access to LLM provider APIs (OpenAI, Anthropic, etc.)
- **Inbound HTTPS**: Port 443 for web interface and API access
- **Database Port**: Port 5432 (if using external PostgreSQL)

## Installation

### Option 1: Docker Deployment (Recommended)

```bash
# Clone the repository
git clone https://github.com/your-org/llm-verifier.git
cd llm-verifier

# Create environment file
cat > .env << EOF
# Database Configuration
DB_PATH=/data/llm-verifier.db
DB_ENCRYPTION_KEY=your-encryption-key-here

# API Configuration
JWT_SECRET=your-jwt-secret-here
API_PORT=8080

# Provider API Keys
OPENAI_API_KEY=your-openai-key
ANTHROPIC_API_KEY=your-anthropic-key
GOOGLE_API_KEY=your-google-key
GROQ_API_KEY=your-groq-key
# Add other provider keys as needed
EOF

# Start with Docker Compose
docker-compose up -d
```

### Option 2: Native Installation

```bash
# Install Go dependencies
go mod download

# Build the application
go build -o llm-verifier ./cmd

# Create data directory
mkdir -p /var/lib/llm-verifier

# Initialize database
./llm-verifier --init-db --db-path /var/lib/llm-verifier/data.db

# Create systemd service
cat > /etc/systemd/system/llm-verifier.service << EOF
[Unit]
Description=LLM Verifier Service
After=network.target

[Service]
Type=simple
User=llm-verifier
Group=llm-verifier
WorkingDirectory=/opt/llm-verifier
ExecStart=/opt/llm-verifier/llm-verifier --config /etc/llm-verifier/config.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Enable and start service
systemctl enable llm-verifier
systemctl start llm-verifier
```

## Configuration

### Core Configuration File

Create `/etc/llm-verifier/config.yaml`:

```yaml
# Global Configuration
global:
  default_model: "gpt-4"
  max_retries: 3
  log_level: "info"

# Database Configuration
database:
  path: "/var/lib/llm-verifier/data.db"
  encryption_key: "your-encryption-key"

# API Configuration
api:
  port: 8080
  jwt_secret: "your-jwt-secret"
  rate_limit: 100
  burst_limit: 200
  cors_origins:
    - "https://your-domain.com"

# Provider Configurations
providers:
  openai:
    enabled: true
    api_key: "${OPENAI_API_KEY}"
    base_url: "https://api.openai.com/v1"
    models:
      - "gpt-4"
      - "gpt-3.5-turbo"
    limits:
      requests_per_minute: 100
      tokens_per_minute: 10000

  anthropic:
    enabled: true
    api_key: "${ANTHROPIC_API_KEY}"
    base_url: "https://api.anthropic.com"
    models:
      - "claude-3-opus"
      - "claude-3-sonnet"
    limits:
      requests_per_minute: 50
      tokens_per_minute: 5000

  groq:
    enabled: true
    api_key: "${GROQ_API_KEY}"
    base_url: "https://api.groq.com/openai/v1"
    models:
      - "llama2-70b-4096"
    limits:
      requests_per_minute: 30
      tokens_per_minute: 10000

# Add configurations for all 17 supported providers...
```

### Environment Variables

Create `/etc/llm-verifier/environment`:

```bash
# API Keys for all providers
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
export GOOGLE_API_KEY="..."
export GROQ_API_KEY="..."
export TOGETHER_API_KEY="..."
export FIREWORKS_API_KEY="..."
export POE_API_KEY="..."
export NAVIGATOR_API_KEY="..."
export MISTRAL_API_KEY="..."
export XAI_API_KEY="..."

# System Configuration
export LLM_VERIFIER_DB_PATH="/var/lib/llm-verifier/data.db"
export LLM_VERIFIER_JWT_SECRET="your-jwt-secret"
export LLM_VERIFIER_PORT="8080"
```

## User Management

### Creating Administrative Users

```bash
# Create admin user
./llm-verifier user create --email admin@your-org.com --role admin

# Set initial password
./llm-verifier user password --email admin@your-org.com --password "temporary-password"

# User will be prompted to change password on first login
```

### Role-Based Access Control

**Available Roles:**
- **admin**: Full system access, user management, configuration
- **operator**: Verification execution, report generation, provider management
- **viewer**: Read-only access to results and reports

```bash
# Assign role to user
./llm-verifier user role --email user@domain.com --role operator

# List all users
./llm-verifier user list

# Remove user
./llm-verifier user delete --email user@domain.com
```

## Provider Management

### Adding New Providers

```bash
# Add a provider interactively
./llm-verifier provider add

# Add provider from configuration
./llm-verifier provider add --config provider-config.yaml

# Example provider configuration file
cat > groq-provider.yaml << EOF
name: groq
endpoint: https://api.groq.com/openai/v1
api_key: ${GROQ_API_KEY}
models:
  - llama2-70b-4096
  - llama2-7b-2048
limits:
  requests_per_minute: 30
  tokens_per_minute: 10000
EOF

./llm-verifier provider add --file groq-provider.yaml
```

### Managing Provider Configurations

```bash
# List all providers
./llm-verifier provider list

# Update provider configuration
./llm-verifier provider update groq --api-key "new-key"

# Disable a provider
./llm-verifier provider disable groq

# Enable a provider
./llm-verifier provider enable groq

# Remove a provider
./llm-verifier provider remove groq
```

### Health Monitoring

```bash
# Check all provider health
./llm-verifier provider health

# Check specific provider
./llm-verifier provider health groq

# Get provider statistics
./llm-verifier provider stats groq
```

## Verification Operations

### Running Verifications

```bash
# Run verification on all providers
./llm-verifier verify --all

# Run verification on specific providers
./llm-verifier verify --providers openai,anthropic,groq

# Run verification on specific models
./llm-verifier verify --models gpt-4,claude-3-opus

# Run with custom test suite
./llm-verifier verify --suite comprehensive --output-dir ./results
```

### Scheduling Automated Verifications

```bash
# Create daily verification schedule
./llm-verifier schedule create --name "daily-all" --cron "0 2 * * *" --providers all

# Create weekly comprehensive test
./llm-verifier schedule create --name "weekly-full" --cron "0 3 * * 1" --suite full

# List active schedules
./llm-verifier schedule list

# Disable schedule
./llm-verifier schedule disable daily-all
```

## Monitoring and Maintenance

### System Health Checks

```bash
# Overall system health
./llm-verifier health

# Database health
./llm-verifier db health

# Provider connectivity
./llm-verifier provider health --all

# API responsiveness
curl http://localhost:8080/api/health
```

### Log Management

```bash
# View recent logs
./llm-verifier logs --tail 100

# Search logs
./llm-verifier logs --grep "error" --since "1h"

# Export logs
./llm-verifier logs --export logs.json --since "24h"
```

### Database Maintenance

```bash
# Run database migrations
./llm-verifier db migrate

# Backup database
./llm-verifier db backup --output backup-$(date +%Y%m%d).db

# Optimize database
./llm-verifier db optimize

# Check database integrity
./llm-verifier db integrity
```

### Performance Monitoring

```bash
# View system metrics
./llm-verifier metrics

# Performance benchmarks
./llm-verifier benchmark --providers openai,groq --iterations 100

# Resource usage
./llm-verifier stats --period 24h
```

## Backup and Recovery

### Automated Backups

```bash
# Configure automated backups
./llm-verifier backup config --schedule "0 1 * * *" --retention 30

# Manual backup
./llm-verifier backup create --type full --destination /backup/

# List backups
./llm-verifier backup list

# Restore from backup
./llm-verifier backup restore --file /backup/backup-20251225.db
```

### Disaster Recovery

```bash
# Emergency stop
./llm-verifier emergency stop

# Recovery mode startup
./llm-verifier start --recovery-mode

# Data integrity check
./llm-verifier integrity check

# Failover activation
./llm-verifier failover activate --provider groq
```

## Troubleshooting

### Common Issues

#### Provider Connection Failures
```bash
# Check provider configuration
./llm-verifier provider test groq

# Verify API key validity
./llm-verifier provider validate groq

# Check network connectivity
curl -H "Authorization: Bearer ${GROQ_API_KEY}" https://api.groq.com/openai/v1/models
```

#### Database Issues
```bash
# Check database connectivity
./llm-verifier db ping

# Repair corrupted database
./llm-verifier db repair

# Reset database (CAUTION: destroys all data)
./llm-verifier db reset --confirm
```

#### Performance Problems
```bash
# Check system resources
./llm-verifier diagnostics

# Clear caches
./llm-verifier cache clear

# Restart services
./llm-verifier restart
```

### Log Analysis

```bash
# Find error patterns
./llm-verifier logs --grep "ERROR" --since "1h" | head -20

# Performance bottlenecks
./llm-verifier logs --grep "slow" --stats

# Security events
./llm-verifier logs --grep "auth" --security
```

## Security Configuration

### SSL/TLS Setup

```bash
# Generate SSL certificate
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365

# Configure SSL in config.yaml
api:
  ssl_cert: /etc/llm-verifier/cert.pem
  ssl_key: /etc/llm-verifier/key.pem
  port: 8443
```

### Firewall Configuration

```bash
# Allow API access
ufw allow 8080/tcp
ufw allow 8443/tcp

# Restrict database access
ufw deny 5432/tcp
ufw allow from 127.0.0.1 to any port 5432
```

### Audit Logging

```bash
# Enable detailed audit logging
./llm-verifier config set audit.enabled true
./llm-verifier config set audit.level detailed

# View audit logs
./llm-verifier audit log --since "24h"

# Export audit reports
./llm-verifier audit export --format pdf --period "month"
```

## Scaling and High Availability

### Load Balancing Setup

```bash
# Configure multiple instances
# Instance 1
./llm-verifier start --port 8081 --node-id node1

# Instance 2
./llm-verifier start --port 8082 --node-id node2

# Load balancer configuration (nginx example)
upstream llm_verifier {
    server localhost:8081;
    server localhost:8082;
}

server {
    listen 80;
    location / {
        proxy_pass http://llm_verifier;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Database Clustering

```bash
# Enable read replicas
./llm-verifier db cluster enable --master localhost:5432 --replicas replica1:5432,replica2:5432

# Monitor cluster health
./llm-verifier db cluster status

# Failover to replica
./llm-verifier db cluster failover --to replica1
```

This administrator manual provides comprehensive guidance for deploying, configuring, and maintaining LLM Verifier in production environments. Regular updates and security patches should be applied to maintain system integrity.