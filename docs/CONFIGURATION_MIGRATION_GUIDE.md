# Configuration Migration Guide - v1 to v2

## Overview

This guide helps you migrate from LLMsVerifier v1 to v2, which introduces mandatory model verification and the (llmsvd) suffix system. The migration process ensures your configurations are updated to work with the new verification requirements and branding system.

## 🚨 Breaking Changes in v2

### Major Changes
1. **Mandatory Model Verification**: All models must pass verification before use
2. **LLMSVD Suffix**: All models and providers include "(llmsvd)" suffix
3. **Configuration Schema Updates**: New verification and branding sections
4. **API Changes**: New verification endpoints and response formats
5. **Database Schema**: Updated to support verification results

### Backward Compatibility
- **Graceful Degradation**: Systems continue to work with warnings
- **Migration Tools**: Automated migration scripts provided
- **Fallback Modes**: Non-strict modes available during transition
- **Deprecation Period**: 6-month deprecation timeline for v1 features

## 🚀 Migration Process

### Step 1: Pre-Migration Assessment

#### Current Configuration Analysis
```bash
# Analyze current configuration
./migration-assistant --analyze --config config.yaml

# Check for compatibility issues
./migration-assistant --check-compatibility --config config.yaml

# Generate migration report
./migration-assistant --generate-report --config config.yaml
```

#### Backup Current Setup
```bash
# Backup configuration
cp config.yaml config.yaml.v1.backup

# Backup database (if using)
cp llm-verifier.db llm-verifier.db.v1.backup

# Backup any custom scripts
tar -czf scripts-backup.tar.gz scripts/
```

### Step 2: Configuration Migration

#### Automatic Migration
```bash
# Run automated migration
./migrate-to-v2 --config config.yaml --output config.v2.yaml

# Validate migrated configuration
./validate-config --config config.v2.yaml --version v2

# Test migrated configuration
./test-config --config config.v2.yaml
```

#### Manual Migration

**v1 Configuration Example:**
```yaml
global:
  base_url: "https://api.openai.com/v1"
  api_key: "${OPENAI_API_KEY}"
  max_retries: 3
  request_delay: 1s
  timeout: 30s

llms:
  - name: "OpenAI GPT-4"
    endpoint: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4-turbo"
    features:
      tool_calling: true
      embeddings: false

concurrency: 5
timeout: 60s
```

**v2 Configuration (Migrated):**
```yaml
# New v2 configuration schema
version: "2.0"
profile: "production"

global:
  base_url: "https://api.openai.com/v1"
  api_key: "${OPENAI_API_KEY}"
  max_retries: 3
  request_delay: 1s
  timeout: 30s

# NEW: Model Verification Configuration
model_verification:
  enabled: true
  strict_mode: true
  require_affirmative: true
  max_retries: 3
  timeout_seconds: 30
  min_verification_score: 0.7
  verification_prompt: |
    Do you see my code? Please respond with "Yes, I can see your [language] code" 
    if you can see the code below, or "No, I cannot see your code" if you cannot.

# NEW: Branding Configuration
branding:
  enabled: true
  suffix: "(llmsvd)"
  position: "final"

database:
  path: "./llm-verifier.db"
  encryption_key: "${DB_ENCRYPTION_KEY}"

# UPDATED: LLM Configuration with Verification
llms:
  - name: "OpenAI GPT-4 (llmsvd)"              # Added (llmsvd) suffix
    provider: "openai"
    endpoint: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4-turbo"
    enabled: true
    verification_status: "pending"              # NEW: Verification status
    features:
      tool_calling: true
      embeddings: false
      streaming: true
      brotli: true                               # NEW: Feature detection
      http3: false

concurrency: 5
timeout: 60s

# NEW: API Configuration
api:
  port: 8080
  jwt_secret: "your-jwt-secret"
  rate_limit: 100
  enable_cors: true

# NEW: Monitoring Configuration
monitoring:
  enabled: true
  prometheus:
    enabled: true
    port: 9090

# NEW: Enterprise Configuration
enterprise:
  ldap:
    enabled: false
    host: "ldap.company.com"
    port: 389
  sso:
    enabled: false
    provider: "saml"
```

### Step 3: Database Migration

#### Automatic Database Migration
```bash
# Backup database first
cp llm-verifier.db llm-verifier.db.v1.backup

# Run database migration
./migrate-database --db llm-verifier.db --to-version v2

# Verify migration
./verify-database --db llm-verifier.db --version v2
```

#### Manual Schema Updates
```sql
-- Add verification tables
CREATE TABLE model_verifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    model_id INTEGER NOT NULL,
    verification_status TEXT NOT NULL,
    verification_score REAL,
    can_see_code BOOLEAN,
    affirmative_response BOOLEAN,
    verification_timestamp DATETIME,
    retry_count INTEGER DEFAULT 0,
    error_message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (model_id) REFERENCES models(id)
);

-- Add branding columns
ALTER TABLE models ADD COLUMN has_llmsvd_suffix BOOLEAN DEFAULT TRUE;
ALTER TABLE models ADD COLUMN original_name TEXT;
ALTER TABLE providers ADD COLUMN has_llmsvd_suffix BOOLEAN DEFAULT TRUE;
ALTER TABLE providers ADD COLUMN original_name TEXT;

-- Add verification config
CREATE TABLE verification_config (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    enabled BOOLEAN DEFAULT TRUE,
    strict_mode BOOLEAN DEFAULT TRUE,
    require_affirmative BOOLEAN DEFAULT TRUE,
    max_retries INTEGER DEFAULT 3,
    timeout_seconds INTEGER DEFAULT 30,
    min_verification_score REAL DEFAULT 0.7,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Step 4: API Migration

#### New API Endpoints
```bash
# Model verification endpoints
curl -X POST "http://localhost:8080/api/v1/models/{model_id}/verify"
curl -X GET "http://localhost:8080/api/v1/models/{model_id}/verification-status"
curl -X GET "http://localhost:8080/api/v1/models/{model_id}/verification-results"

# Verified models only
curl -X GET "http://localhost:8080/api/v1/models?verification_status=verified"

# Configuration export with verification
curl -X POST "http://localhost:8080/api/v1/config-exports/opencode" \
  -d '{"verification_status": "verified", "min_score": 80}'
```

#### Updated Response Formats
```json
// Model response (v2)
{
  "id": 1,
  "name": "GPT-4 (brotli) (http3) (llmsvd)",
  "model_id": "gpt-4",
  "verification_status": "verified",
  "verification_score": 0.85,
  "can_see_code": true,
  "affirmative_response": true,
  "overall_score": 92.5,
  "features": {
    "supports_tool_use": true,
    "supports_brotli": true,
    "supports_http3": true
  }
}
```

### Step 5: Client Updates

#### CLI Updates
```bash
# Install v2 CLI
go install ./cmd/llm-verifier@latest

# Test verification
llm-verifier verify --model gpt-4

# Get verified models only
llm-verifier models list --verified-only

# Export verified configuration
llm-verifier export opencode --verified-only
```

#### SDK Updates
```go
// Go SDK v2
import "github.com/vasic-digital/LLMsVerifier/sdk/go/v2"

client := llmverifier.NewClient("http://localhost:8080", "api-key")

// Get verified models only
verifiedModels, err := client.GetVerifiedModels()

// Verify specific model
verification, err := client.VerifyModel("gpt-4", "test code")
```

### Step 6: Testing Migration

#### Verification Testing
```bash
# Test model verification
./test_model_verification.sh

# Test suffix handling
./test_suffix_integration.sh

# Test configuration generation
./test_config_generation.sh

# Run comprehensive tests
./run_comprehensive_tests.sh
```

#### Integration Testing
```bash
# Test with real providers
./test_with_providers.sh --providers openai,anthropic

# Test configuration exports
./test_config_exports.sh --formats opencode,crush

# Test API compatibility
./test_api_compatibility.sh --version v2
```

## 📊 Migration Checklist

### Pre-Migration
- [ ] Backup all configurations
- [ ] Backup database (if applicable)
- [ ] Review current setup
- [ ] Check API usage
- [ ] Notify stakeholders

### Configuration Migration
- [ ] Run automated migration tool
- [ ] Review migrated configuration
- [ ] Update environment variables
- [ ] Test new configuration
- [ ] Validate syntax

### Database Migration
- [ ] Backup database
- [ ] Run schema updates
- [ ] Migrate existing data
- [ ] Verify data integrity
- [ ] Test queries

### Application Migration
- [ ] Update application code
- [ ] Install v2 binaries
- [ ] Update SDK versions
- [ ] Test API integration
- [ ] Verify functionality

### Post-Migration
- [ ] Run comprehensive tests
- [ ] Monitor performance
- [ ] Check logs for errors
- [ ] Validate configurations
- [ ] Document changes

## 🚨 Common Migration Issues

### Issue 1: Configuration Validation Errors
```bash
# Problem: New configuration rejected
# Solution: Run validation with detailed output
./validate-config --config config.v2.yaml --verbose

# Fix common issues
./fix-config-issues --config config.v2.yaml
```

### Issue 2: Database Migration Failures
```bash
# Problem: Schema migration fails
# Solution: Check database version and permissions
./check-database --db llm-verifier.db

# Manual schema fix
./fix-database-schema --db llm-verifier.db
```

### Issue 3: API Compatibility Issues
```bash
# Problem: API calls fail after migration
# Solution: Check API version and endpoints
./test-api-compatibility --base-url http://localhost:8080

# Update API calls to v2 format
./update-api-calls --input old_calls.json --output new_calls.json
```

### Issue 4: Model Verification Failures
```bash
# Problem: Models fail verification
# Solution: Check verification configuration
./test-verification --provider openai --model gpt-4

# Adjust verification settings
./adjust-verification --config config.v2.yaml --threshold 0.6
```

### Issue 5: Suffix Not Applied
```bash
# Problem: (llmsvd) suffix missing
# Solution: Check branding configuration
./check-suffix-config --config config.v2.yaml

# Force suffix application
./apply-suffixes --config config.v2.yaml
```

## 🔧 Rollback Procedures

### Emergency Rollback
```bash
# Stop v2 services
sudo systemctl stop llm-verifier

# Restore v1 configuration
cp config.yaml.v1.backup config.yaml

# Restore v1 database
cp llm-verifier.db.v1.backup llm-verifier.db

# Start v1 services
sudo systemctl start llm-verifier-v1
```

### Gradual Rollback
```bash
# Enable compatibility mode
./enable-compatibility-mode --config config.v2.yaml

# Gradual transition
./gradual-migration --direction rollback --percentage 25
```

## 📈 Performance Considerations

### Migration Performance
- **Configuration Migration**: < 1 minute for typical configs
- **Database Migration**: 5-15 minutes depending on data size
- **API Migration**: Immediate (backward compatible)
- **Testing Phase**: 30-60 minutes for comprehensive tests

### Resource Requirements
- **Memory**: Additional 100MB for verification system
- **Storage**: Additional 20% for verification data
- **CPU**: 10-15% increase for verification processing
- **Network**: Additional API calls for verification

## 🔍 Monitoring Migration

### Key Metrics
```bash
# Monitor verification rate
watch -n 5 'curl -s http://localhost:8080/api/v1/metrics | grep verification_rate'

# Monitor model count
watch -n 5 'curl -s http://localhost:8080/api/v1/models | jq ".meta.total"'

# Monitor error rates
tail -f logs/error.log | grep -i error
```

### Health Checks
```bash
# System health
./health-check --comprehensive

# Database health
./check-database-health --db llm-verifier.db

# API health
./check-api-health --base-url http://localhost:8080
```

## 📚 Best Practices

### Planning
1. **Test in Staging**: Always test migration in staging first
2. **Schedule Downtime**: Plan for minimal downtime during migration
3. **Communicate**: Notify all stakeholders about migration timeline
4. **Prepare Rollback**: Have rollback plan ready before starting

### Execution
1. **Incremental Approach**: Migrate in stages if possible
2. **Monitor Closely**: Watch metrics and logs during migration
3. **Document Issues**: Record any issues and solutions
4. **Validate Thoroughly**: Test all functionality after migration

### Post-Migration
1. **Monitor Performance**: Track system performance after migration
2. **Gather Feedback**: Collect user feedback on new features
3. **Optimize Configuration**: Fine-tune settings based on usage
4. **Update Documentation**: Keep documentation current

## 🔗 Related Documentation

- [Model Verification Guide](MODEL_VERIFICATION_GUIDE.md)
- [LLMSVD Suffix Guide](LLMSVD_SUFFIX_GUIDE.md)
- [API Documentation](API_DOCUMENTATION.md)
- [Release Notes v2.0](RELEASE_NOTES_v2.0.md)
- [Troubleshooting Guide](../TROUBLESHOOTING.md)

## 📞 Support

For migration assistance:
- **Migration Tool Help**: `./migration-assistant --help`
- **Configuration Validation**: `./validate-config --help`
- **Database Migration**: `./migrate-database --help`
- **Community Support**: [GitHub Discussions](https://github.com/vasic-digital/LLMsVerifier/discussions)
- **Professional Support**: Contact support@llm-verifier.com

---

**The migration to v2 ensures your LLMsVerifier installation benefits from mandatory model verification and consistent branding while maintaining system reliability and performance.**