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