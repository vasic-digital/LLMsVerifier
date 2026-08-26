# Environment Variables Configuration

This document explains how to configure the LLM Verifier using environment variables for secure deployment.

## Required Environment Variables for Production

### JWT Secret
```bash
export LLM_VERIFIER_API_JWT_SECRET="your-secure-random-jwt-secret-here"
```

**⚠️ SECURITY WARNING**: Never use the default JWT secret in production. Always set a secure, random secret.

### Database Path (Optional)
```bash
export LLM_VERIFIER_DATABASE_PATH="/app/data/llm-verifier.db"
```

### API Configuration (Optional)
```bash
export LLM_VERIFIER_API_PORT="8080"
export LLM_VERIFIER_API_RATE_LIMIT="1000"
export LLM_VERIFIER_API_ENABLE_CORS="true"
export LLM_VERIFIER_API_JWT_SECRET="your-secure-jwt-secret"
export LLM_VERIFIER_API_RATE_LIMIT_WINDOW="60"
```

### Supervisor System Configuration
```bash
export LLM_VERIFIER_SUPERVISOR_ENABLED="true"
export LLM_VERIFIER_SUPERVISOR_MAX_WORKERS="5"
export LLM_VERIFIER_SUPERVISOR_TASK_TIMEOUT="30m"
export LLM_VERIFIER_SUPERVISOR_RETRY_ATTEMPTS="3"
```

### Context Management Configuration
```bash
export LLM_VERIFIER_CONTEXT_LONG_TERM_ENABLED="true"
export LLM_VERIFIER_CONTEXT_SUMMARIZATION_ENABLED="true"
export LLM_VERIFIER_CONTEXT_MAX_AGE="168h"
export LLM_VERIFIER_CONTEXT_SUMMARIZATION_INTERVAL="1h"
```

### Cloud Backup Configuration
```bash
# AWS S3
export AWS_ACCESS_KEY_ID="your-aws-key"
export AWS_SECRET_ACCESS_KEY="your-aws-secret"
export AWS_REGION="us-east-1"

# Google Cloud
export GCP_SERVICE_ACCOUNT_JSON="path/to/service-account.json"

# Azure
export AZURE_STORAGE_KEY="your-azure-key"
export AZURE_STORAGE_CONNECTION_STRING="your-connection-string"
```

### Notification System Configuration
```bash
export LLM_VERIFIER_NOTIFICATIONS_SLACK_WEBHOOK_URL="https://hooks.slack.com/..."
export LLM_VERIFIER_NOTIFICATIONS_EMAIL_SMTP_HOST="smtp.gmail.com"
export LLM_VERIFIER_NOTIFICATIONS_EMAIL_SMTP_PORT="587"
export LLM_VERIFIER_NOTIFICATIONS_TELEGRAM_BOT_TOKEN="your-bot-token"
```

### Monitoring and Metrics Configuration
```bash
export LLM_VERIFIER_MONITORING_PROMETHEUS_ENABLED="true"
export LLM_VERIFIER_MONITORING_PROMETHEUS_PORT="9090"
export LLM_VERIFIER_MONITORING_CLOUDWATCH_ENABLED="true"
export LLM_VERIFIER_MONITORING_CLOUDWATCH_REGION="us-east-1"
```

## Development Environment Setup

For development, you can use the test configuration:

```bash
# Use test secret for development
export LLM_VERIFIER_API_JWT_SECRET="test-secret-key"
export LLM_VERIFIER_PROFILE="dev"
```

## Production Deployment Example

### Docker Environment Variables
```yaml
# docker-compose.yml
services:
  llm-verifier:
    image: llm-verifier:latest
    environment:
      - LLM_VERIFIER_API_JWT_SECRET=${JWT_SECRET}
      - LLM_VERIFIER_DATABASE_PATH=/app/data/llm-verifier.db
      - LLM_VERIFIER_API_PORT=8080
      - LLM_VERIFIER_PROFILE=prod
    volumes:
      - ./data:/app/data
```

### Kubernetes Secret Management
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: llm-verifier-secrets
type: Opaque
stringData:
  jwt-secret: "your-secure-random-jwt-secret-here"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: llm-verifier
spec:
  template:
    spec:
      containers:
      - name: llm-verifier
        image: llm-verifier:latest
        env:
        - name: LLM_VERIFIER_API_JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: llm-verifier-secrets
              key: jwt-secret
        - name: LLM_VERIFIER_PROFILE
          value: "prod"
```

## Security Best Practices

1. **Use Strong JWT Secrets**: Generate cryptographically strong secrets (32+ characters)
2. **Rotate Secrets Regularly**: Change JWT secrets periodically
3. **Use Secret Management**: Employ proper secret management in production
4. **Environment Separation**: Use different secrets for dev/staging/prod
5. **Never Commit Secrets**: Never include secrets in version control

## Generating a Secure JWT Secret

### Using OpenSSL
```bash
openssl rand -base64 32
```

### Using Python
```python
import secrets
print(secrets.token_urlsafe(32))
```

### Using Node.js
```javascript
const crypto = require('crypto');
console.log(crypto.randomBytes(32).toString('base64'));
```

## Configuration Precedence

The LLM Verifier uses the following configuration precedence:

1. **Environment Variables** (highest priority)
2. **Configuration File** (config.yaml)
3. **Default Values** (lowest priority)

Environment variables follow the pattern: `LLM_VERIFIER_<SECTION>_<KEY>`

Examples:
- `LLM_VERIFIER_API_JWT_SECRET` → `api.jwt_secret`
- `LLM_VERIFIER_DATABASE_PATH` → `database.path`
- `LLM_VERIFIER_GLOBAL_BASE_URL` → `global.base_url`

## Complete Configuration Reference

### Core System Configuration

#### Global Settings
```bash
export LLM_VERIFIER_PROFILE="prod"                    # dev, prod, test
export LLM_VERIFIER_GLOBAL_BASE_URL="https://api.example.com"
export LLM_VERIFIER_GLOBAL_API_KEY="${API_KEY}"
export LLM_VERIFIER_GLOBAL_DEFAULT_MODEL="gpt-4"
export LLM_VERIFIER_GLOBAL_MAX_RETRIES="3"
export LLM_VERIFIER_GLOBAL_REQUEST_DELAY="100ms"
export LLM_VERIFIER_GLOBAL_TIMEOUT="30s"
```

#### Database Configuration
```bash
export LLM_VERIFIER_DATABASE_PATH="/app/data/llm-verifier.db"
export LLM_VERIFIER_DATABASE_ENCRYPTION_KEY="${DB_KEY}"
export LLM_VERIFIER_DATABASE_MAX_OPEN_CONNS="25"
export LLM_VERIFIER_DATABASE_MAX_IDLE_CONNS="5"
export LLM_VERIFIER_DATABASE_CONN_MAX_LIFETIME="1h"
```

#### Concurrency and Performance
```bash
export LLM_VERIFIER_CONCURRENCY="10"
export LLM_VERIFIER_TIMEOUT="60s"
export LLM_VERIFIER_VERIFICATION_TIMEOUT="300s"
```

### Advanced Features Configuration

#### Supervisor System
```bash
export LLM_VERIFIER_SUPERVISOR_ENABLED="true"
export LLM_VERIFIER_SUPERVISOR_MAX_WORKERS="10"
export LLM_VERIFIER_SUPERVISOR_TASK_TIMEOUT="30m"
export LLM_VERIFIER_SUPERVISOR_RETRY_ATTEMPTS="3"
export LLM_VERIFIER_SUPERVISOR_QUALITY_CHECKS_ENABLED="true"
export LLM_VERIFIER_SUPERVISOR_HUMAN_REVIEW_THRESHOLD="0.8"
```

#### Context Management
```bash
export LLM_VERIFIER_CONTEXT_LONG_TERM_ENABLED="true"
export LLM_VERIFIER_CONTEXT_SUMMARIZATION_ENABLED="true"
export LLM_VERIFIER_CONTEXT_MAX_AGE="168h"
export LLM_VERIFIER_CONTEXT_SUMMARIZATION_INTERVAL="1h"
export LLM_VERIFIER_CONTEXT_COMPRESSION_THRESHOLD="0.8"
export LLM_VERIFIER_CONTEXT_QUALITY_PRESERVATION="true"
```

#### Failover System
```bash
export LLM_VERIFIER_FAILOVER_ENABLED="true"
export LLM_VERIFIER_FAILOVER_CIRCUIT_BREAKER_FAILURE_THRESHOLD="5"
export LLM_VERIFIER_FAILOVER_CIRCUIT_BREAKER_RECOVERY_TIMEOUT="30s"
export LLM_VERIFIER_FAILOVER_LATENCY_ROUTING_ENABLED="true"
export LLM_VERIFIER_FAILOVER_MAX_LATENCY="5s"
export LLM_VERIFIER_FAILOVER_WEIGHTED_ROUTING_ENABLED="true"
```

#### Vector Database Integration
```bash
export LLM_VERIFIER_VECTOR_ENABLED="true"
export LLM_VERIFIER_VECTOR_PROVIDER="cognee"
export LLM_VERIFIER_VECTOR_ENDPOINT="http://localhost:8000"
export LLM_VERIFIER_VECTOR_API_KEY="${VECTOR_API_KEY}"
export LLM_VERIFIER_VECTOR_DIMENSION="1536"
export LLM_VERIFIER_VECTOR_TOP_K="5"
export LLM_VERIFIER_VECTOR_SCORE_THRESHOLD="0.7"
```

### Cloud Integration Configuration

#### AWS Integration
```bash
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="us-east-1"
export AWS_SESSION_TOKEN="your-session-token"  # Optional
```

#### Google Cloud Integration
```bash
export GCP_PROJECT_ID="your-project-id"
export GCP_SERVICE_ACCOUNT_KEY_PATH="/path/to/service-account.json"
export GCP_SERVICE_ACCOUNT_JSON="${SERVICE_ACCOUNT_JSON}"
```

#### Azure Integration
```bash
export AZURE_TENANT_ID="your-tenant-id"
export AZURE_CLIENT_ID="your-client-id"
export AZURE_CLIENT_SECRET="your-client-secret"
export AZURE_STORAGE_KEY="your-storage-key"
export AZURE_STORAGE_CONNECTION_STRING="your-connection-string"
```

### Notification System Configuration

#### Slack Notifications
```bash
export LLM_VERIFIER_NOTIFICATIONS_SLACK_ENABLED="true"
export LLM_VERIFIER_NOTIFICATIONS_SLACK_WEBHOOK_URL="https://hooks.slack.com/..."
export LLM_VERIFIER_NOTIFICATIONS_SLACK_CHANNEL="#llm-alerts"
export LLM_VERIFIER_NOTIFICATIONS_SLACK_USERNAME="LLM-Verifier"
```

#### Email Notifications
```bash
export LLM_VERIFIER_NOTIFICATIONS_EMAIL_ENABLED="true"
export LLM_VERIFIER_NOTIFICATIONS_EMAIL_SMTP_HOST="smtp.gmail.com"
export LLM_VERIFIER_NOTIFICATIONS_EMAIL_SMTP_PORT="587"
export LLM_VERIFIER_NOTIFICATIONS_EMAIL_SMTP_USERNAME="your-email@gmail.com"
export LLM_VERIFIER_NOTIFICATIONS_EMAIL_SMTP_PASSWORD="your-app-password"
export LLM_VERIFIER_NOTIFICATIONS_EMAIL_FROM="llm-verifier@yourdomain.com"
```

#### Telegram Notifications
```bash
export LLM_VERIFIER_NOTIFICATIONS_TELEGRAM_ENABLED="true"
export LLM_VERIFIER_NOTIFICATIONS_TELEGRAM_BOT_TOKEN="your-bot-token"
export LLM_VERIFIER_NOTIFICATIONS_TELEGRAM_CHAT_ID="your-chat-id"
```

#### Matrix Notifications
```bash
export LLM_VERIFIER_NOTIFICATIONS_MATRIX_ENABLED="true"
export LLM_VERIFIER_NOTIFICATIONS_MATRIX_HOMESERVER="https://matrix.org"
export LLM_VERIFIER_NOTIFICATIONS_MATRIX_ACCESS_TOKEN="your-access-token"
export LLM_VERIFIER_NOTIFICATIONS_MATRIX_ROOM_ID="!room:matrix.org"
```

#### WhatsApp Notifications
```bash
export LLM_VERIFIER_NOTIFICATIONS_WHATSAPP_ENABLED="true"
export LLM_VERIFIER_NOTIFICATIONS_WHATSAPP_ACCESS_TOKEN="your-access-token"
export LLM_VERIFIER_NOTIFICATIONS_WHATSAPP_PHONE_NUMBER_ID="your-phone-number-id"
export LLM_VERIFIER_NOTIFICATIONS_WHATSAPP_RECIPIENT="1234567890"
```

### Monitoring and Observability

#### Prometheus Metrics
```bash
export LLM_VERIFIER_MONITORING_PROMETHEUS_ENABLED="true"
export LLM_VERIFIER_MONITORING_PROMETHEUS_PORT="9090"
export LLM_VERIFIER_MONITORING_PROMETHEUS_PATH="/metrics"
```

#### CloudWatch Integration
```bash
export LLM_VERIFIER_MONITORING_CLOUDWATCH_ENABLED="true"
export LLM_VERIFIER_MONITORING_CLOUDWATCH_REGION="us-east-1"
export LLM_VERIFIER_MONITORING_CLOUDWATCH_NAMESPACE="LLM-Verifier"
```

#### Health Checks
```bash
export LLM_VERIFIER_MONITORING_HEALTH_ENABLED="true"
export LLM_VERIFIER_MONITORING_HEALTH_PORT="8081"
export LLM_VERIFIER_MONITORING_HEALTH_PATH="/health"
```

### Backup and Recovery

#### Cloud Backup Configuration
```bash
export LLM_VERIFIER_BACKUP_ENABLED="true"
export LLM_VERIFIER_BACKUP_PROVIDER="aws"  # aws, gcp, azure
export LLM_VERIFIER_BACKUP_BUCKET="llm-verifier-backups"
export LLM_VERIFIER_BACKUP_REGION="us-east-1"
export LLM_VERIFIER_BACKUP_PREFIX="backups/"
export LLM_VERIFIER_BACKUP_SCHEDULE="0 2 * * *"  # Daily at 2 AM
export LLM_VERIFIER_BACKUP_COMPRESSION="true"
export LLM_VERIFIER_BACKUP_ENCRYPTION="true"
export LLM_VERIFIER_BACKUP_RETENTION_DAYS="30"
```

### Security Configuration

#### Rate Limiting
```bash
export LLM_VERIFIER_SECURITY_RATE_LIMITING_ENABLED="true"
export LLM_VERIFIER_SECURITY_RATE_LIMIT_REQUESTS_PER_MINUTE="1000"
export LLM_VERIFIER_SECURITY_RATE_LIMIT_BURST="100"
export LLM_VERIFIER_SECURITY_RATE_LIMIT_WINDOW="60"
```

#### CORS Configuration
```bash
export LLM_VERIFIER_SECURITY_CORS_ENABLED="true"
export LLM_VERIFIER_SECURITY_CORS_ALLOWED_ORIGINS="https://yourdomain.com"
export LLM_VERIFIER_SECURITY_CORS_ALLOWED_METHODS="GET,POST,PUT,DELETE"
export LLM_VERIFIER_SECURITY_CORS_ALLOWED_HEADERS="Content-Type,Authorization"
```

#### IP Whitelisting
```bash
export LLM_VERIFIER_SECURITY_IP_WHITELIST_ENABLED="true"
export LLM_VERIFIER_SECURITY_IP_WHITELIST="192.168.1.0/24,10.0.0.0/8"
```

#### Trusted Proxies (X-Forwarded-For / X-Real-IP)

This is a distinct concern from IP Whitelisting above: whitelisting gates which
clients may connect to the API at all; `LLM_VERIFIER_TRUSTED_PROXIES` gates which
**already-connected peer** the API will trust to tell it who the *real* client is,
via the `X-Forwarded-For` / `X-Real-IP` headers.

```bash
# Comma-separated bare IPs and/or CIDRs. Read directly by api/middleware.go
# (not the LLM_VERIFIER_SECURITY_* structured-config namespace above).
export LLM_VERIFIER_TRUSTED_PROXIES="172.20.0.0/16"
```

**Why this exists.** `getClientIP` (`api/middleware.go`) feeds both the rate
limiter (`RateLimitMiddleware`) and the audit log
(`api/audit_logger.go`'s `ClientIP` field). Without a trusted-proxy allowlist,
any caller that can reach this service — not only one behind your reverse
proxy — could set `X-Forwarded-For`/`X-Real-IP` to an arbitrary value and
evade its own rate-limit bucket or forge its identity in the audit trail.

**Default: unset = nothing trusted.** With `LLM_VERIFIER_TRUSTED_PROXIES`
unset (the default), `X-Forwarded-For` and `X-Real-IP` are ignored entirely
and every caller's identity is its own TCP peer address (`RemoteAddr`). This
is deliberate, industry-standard "deny by default" (the same posture Express's
`trust proxy`, Django, and Rails all ship with) — this repository documents
BOTH a direct-exposure quick-start topology (root `docker-compose.yml`, no
proxy) and reverse-proxied production topologies
(`docker-compose.prod.yml` + Traefik, `llm-verifier/docker-compose.yml` +
nginx, `k8s-manifests.yaml` + nginx-ingress — see `docs/deployment/docker.md`
and `docs/deployment/kubernetes.md`), and the *specific* trusted address or
CIDR varies by which of those you actually run, so no single default could be
both safe and correct for all of them. A hardcoded guess (e.g. "trust all of
RFC 1918") would also be unsafe on a shared bridge network, where every other
container on that network — not only your reverse proxy — could reach this
service directly.

**The honest cost of leaving it unset behind a real proxy.** If you deploy
behind nginx/Traefik/an ingress controller and do NOT set this variable,
every request arrives from the proxy's own peer address, so every real user
behind that proxy is treated as ONE caller for rate-limiting purposes. Under
load this is not a cosmetic degradation — an aggregate request volume that
exceeds one client's rate-limit budget will start returning `429 Too Many
Requests` to *every* legitimate user sharing that proxy, which is
operationally an outage for a busy deployment, even though the service
itself is healthy and answering. Audit-log `ClientIP` values likewise
collapse to the proxy's own address. **Set this variable as part of standing
up any reverse-proxied deployment** — it is not optional tuning, it is part
of making that topology work correctly.

**Per-topology example values** (illustrative — always use YOUR deployment's
actual proxy address/CIDR, never copy these verbatim):

```bash
# docker-compose.prod.yml topology: nginx + Traefik share the bridge network
# whose subnet that compose file itself declares (see its "subnet:" line under
# the `llm-verifier-network` definition). Trust that subnet, not a guess.
export LLM_VERIFIER_TRUSTED_PROXIES="172.20.0.0/16"

# Multiple trusted hops (e.g. an internal load balancer AND nginx in front of
# it) — comma-separated, mixing bare IPs and CIDRs is fine:
export LLM_VERIFIER_TRUSTED_PROXIES="172.20.0.5,10.1.0.0/16"

# Kubernetes: trust your ingress-controller's actual pod/Service CIDR for
# your cluster — this is cluster-specific; find yours with
# `kubectl get pods -n ingress-nginx -o wide` or your CNI's documented pod
# CIDR, then narrow to the smallest range that actually covers it.
export LLM_VERIFIER_TRUSTED_PROXIES="10.244.0.0/16"
```

**How the header is used once a peer is trusted.** `X-Forwarded-For` is a
comma-separated list built left-to-right as each hop appends its own
observed peer to the right of whatever it received. This service walks that
list **right to left**, skipping any entry that is *itself* on the trusted
allowlist, and uses the first entry that is not — never the leftmost entry
unconditionally. (An earlier draft of this feature picked the leftmost entry
outright; that let an attacker prepend an arbitrary forged address ahead of
whatever a trusted, header-*appending* proxy — e.g. nginx's
`$proxy_add_x_forwarded_for` — added for its own observed peer. That defect
was caught in review before it shipped and is covered by a permanent
regression guard: see `api/hxc292_client_ip_trust_red_test.go`.) Every
candidate is validated as a real IP address before being trusted or
returned, so a malformed or non-IP entry can never reach the audit log.

## Troubleshooting

### JWT Secret Warning
If you see this warning:
```
WARNING: Using default JWT secret. Please set LLM_VERIFIER_API_JWT_SECRET environment variable in production.
```

Set the environment variable:
```bash
export LLM_VERIFIER_API_JWT_SECRET="your-secure-secret"
```

### Configuration Not Applied
If environment variables are not being applied, check:
1. Variable name follows the pattern `LLM_VERIFIER_<SECTION>_<KEY>`
2. Variable is exported/set for the process
3. No conflicting values in config file (unless intentional)