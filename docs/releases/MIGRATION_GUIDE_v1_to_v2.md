# Migration Guide: LLMsVerifier v1 to v2

## 🚨 Important Notice

This guide helps you migrate from LLMsVerifier v1 to v2, which introduces **mandatory model verification** and the **(llmsvd) suffix system**. These are **breaking changes** that require careful planning and execution.

**⚠️ WARNING**: v2 requires all models to pass verification before use. Plan your migration carefully to avoid service disruption.

## 📋 Migration Overview

### What Changes in v2?
1. **Mandatory Model Verification**: All models must pass "Do you see my code?" verification
2. **LLMSVD Suffix**: All models/providers include "(llmsvd)" suffix
3. **New Configuration Schema**: Updated configuration format
4. **Database Schema Updates**: New tables and columns
5. **API Changes**: New endpoints and response formats

### Migration Timeline
- **Planning Phase**: 1-2 weeks
- **Testing Phase**: 1 week
- **Migration Execution**: 2-4 hours
- **Validation Phase**: 1-2 hours
- **Monitoring Phase**: 1 week

## 🚀 Step-by-Step Migration

### Phase 1: Pre-Migration Planning (1-2 weeks)

#### 1.1 Assessment and Inventory
```bash
# Document current setup
./migration-assistant --analyze --config config.yaml --output assessment.json

# Check compatibility
./migration-assistant --check-compatibility --config config.yaml

# Generate migration report
./migration-assistant --generate-report --config config.yaml --output migration-report.html
```

#### 1.2 Risk Assessment
**High Risk Items:**
- Custom integrations with hardcoded model names
- Automated scripts that parse model names
- Database queries that filter by model names
- API integrations that expect specific response formats

**Medium Risk Items:**
- Configuration management tools
- Monitoring and alerting systems
- Backup and restore procedures
- Documentation and runbooks

#### 1.3 Backup Strategy
```bash
# Create comprehensive backups
# 1. Configuration backup
cp config.yaml config.yaml.v1.backup.$(date +%Y%m%d)
cp -r configs/ configs.v1.backup.$(date +%Y%m%d)

# 2. Database backup (if applicable)
sqlite3 llm-verifier.db ".backup llm-verifier.db.v1.backup.$(date +%Y%m%d)"

# 3. Custom scripts backup
tar -czf scripts-backup-$(date +%Y%m%d).tar.gz scripts/ custom/

# 4. Log backup
tar -czf logs-backup-$(date +%Y%m%d).tar.gz logs/

# 5. Verify backups
ls -la *.backup.*
file llm-verifier.db.v1.backup.*
```

#### 1.4 Stakeholder Communication
**Communication Checklist:**
- [ ] Notify development teams
- [ ] Inform operations teams
- [ ] Update documentation teams
- [ ] Communicate with management
- [ ] Inform external integrations

### Phase 2: Environment Preparation (3-5 days)

#### 2.1 Staging Environment Setup
```bash
# Clone production environment
# 1. Copy configuration
cp config.yaml staging-config.yaml

# 2. Copy database (anonymized if needed)
cp llm-verifier.db staging.db

# 3. Update staging configuration
sed -i 's/production/staging/g' staging-config.yaml
sed -i 's/8080/8081/g' staging-config.yaml
```

#### 2.2 Test Environment Validation
```bash
# Validate current setup works
./llm-verifier --config staging-config.yaml --dry-run

# Test API endpoints
curl -f http://localhost:8081/api/v1/health || echo "API not responding"

# Test model discovery
go run llm-verifier/cmd/provider-discovery/main.go --config staging-config.yaml
```

#### 2.3 Migration Tools Installation
```bash
# Install migration tools
go install ./cmd/migration-assistant@latest
go install ./cmd/config-migrator@latest
go install ./cmd/database-migrator@latest

# Verify installation
migration-assistant --version
config-migrator --version
database-migrator --version
```

### Phase 3: Staging Migration (1 week)

#### 3.1 Configuration Migration (Staging)
```bash
# Create v2 configuration
config-migrator --input staging-config.yaml --output staging-config.v2.yaml --verbose

# Review migrated configuration
diff staging-config.yaml staging-config.v2.yaml

# Validate v2 configuration
config-migrator --validate --config staging-config.v2.yaml --version v2

# Test configuration
go run cmd/test-config/main.go --config staging-config.v2.yaml
```

**Example Configuration Changes:**
```yaml
# v1 Configuration
global:
  base_url: "https://api.openai.com/v1"
  api_key: "${OPENAI_API_KEY}"

llms:
  - name: "GPT-4"
    endpoint: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4"

# v2 Configuration
version: "2.0"
global:
  base_url: "https://api.openai.com/v1"
  api_key: "${OPENAI_API_KEY}"

# NEW: Model Verification
model_verification:
  enabled: true
  strict_mode: false  # Set to false for gradual migration
  require_affirmative: true
  max_retries: 3
  timeout_seconds: 30
  min_verification_score: 0.7

# NEW: Branding
branding:
  enabled: true
  suffix: "(llmsvd)"
  position: "final"

# UPDATED: Model names with suffix
llms:
  - name: "GPT-4 (llmsvd)"  # Added (llmsvd) suffix
    provider: "openai"
    endpoint: "https://api.openai.com/v1"
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4"
    verification_status: "pending"
```

#### 3.2 Database Migration (Staging)
```bash
# Backup staging database
cp staging.db staging.db.pre-migration

# Run database migration
database-migrator --db staging.db --to-version v2 --backup --verbose

# Verify migration
database-migrator --verify --db staging.db --version v2

# Check migration results
sqlite3 staging.db "SELECT COUNT(*) FROM model_verifications;"
sqlite3 staging.db "SELECT COUNT(*) FROM models WHERE has_llmsvd_suffix = 1;"
```

#### 3.3 Application Update (Staging)
```bash
# Install v2 application
go install ./cmd/llm-verifier@v2

# Test v2 application
llm-verifier --config staging-config.v2.yaml --dry-run --verbose

# Run verification in non-strict mode
llm-verifier --config staging-config.v2.yaml --verify-models --strict-mode=false
```

#### 3.4 Testing and Validation (Staging)
```bash
# Comprehensive testing
./run_comprehensive_tests.sh --config staging-config.v2.yaml --environment staging

# Test model verification
./test_model_verification.sh --config staging-config.v2.yaml --providers openai,anthropic

# Test suffix application
./test_suffix_integration.sh --config staging-config.v2.yaml

# Test configuration export
./test_config_exports.sh --config staging-config.v2.yaml --formats opencode,crush

# API compatibility testing
./test_api_compatibility.sh --base-url http://localhost:8081 --version v2
```

### Phase 4: Production Migration (2-4 hours)

#### 4.1 Pre-Migration Checklist
**Final Verification:**
- [ ] Staging migration successful
- [ ] All tests passing
- [ ] Performance acceptable
- [ ] No critical issues found
- [ ] Rollback plan tested
- [ ] Team ready and available

#### 4.2 Migration Execution
```bash
#!/bin/bash
# production-migration.sh

set -e  # Exit on error

echo "Starting production migration..."

# Step 1: Final backup
echo "Creating final backup..."
cp config.yaml config.yaml.final-backup.$(date +%Y%m%d_%H%M%S)
cp llm-verifier.db llm-verifier.db.final-backup.$(date +%Y%m%d_%H%M%S)

# Step 2: Stop services
echo "Stopping services..."
sudo systemctl stop llm-verifier
sudo systemctl stop llm-verifier-workers

# Step 3: Configuration migration
echo "Migrating configuration..."
config-migrator --input config.yaml --output config.v2.yaml --verbose
config-migrator --validate --config config.v2.yaml --version v2

# Step 4: Database migration
echo "Migrating database..."
database-migrator --db llm-verifier.db --to-version v2 --backup --verbose

# Step 5: Install v2
echo "Installing v2..."
go install ./cmd/llm-verifier@v2

# Step 6: Test migration
echo "Testing migration..."
llm-verifier --config config.v2.yaml --dry-run --verbose

# Step 7: Gradual verification (non-strict mode initially)
echo "Running initial verification..."
llm-verifier --config config.v2.yaml --verify-models --strict-mode=false --max-concurrent=5

# Step 8: Start services
echo "Starting services..."
sudo systemctl start llm-verifier
sudo systemctl start llm-verifier-workers

echo "Migration completed!"
```

#### 4.3 Post-Migration Validation
```bash
#!/bin/bash
# post-migration-validation.sh

echo "Validating production migration..."

# Check service status
echo "Checking service status..."
sudo systemctl status llm-verifier
sudo systemctl status llm-verifier-workers

# Test API health
echo "Testing API health..."
curl -f http://localhost:8080/api/v1/health || exit 1

# Test model list
echo "Testing model list..."
curl -f http://localhost:8080/api/v1/models | jq '.data[0].name' | grep -q "(llmsvd)" || exit 1

# Test verification status
echo "Testing verification status..."
VERIFIED_COUNT=$(curl -s http://localhost:8080/api/v1/models?verification_status=verified | jq '.meta.total')
echo "Verified models: $VERIFIED_COUNT"

# Test configuration export
echo "Testing configuration export..."
curl -X POST http://localhost:8080/api/v1/config-exports/opencode \
  -H "Content-Type: application/json" \
  -d '{"verification_status": "verified"}' \
  -f || exit 1

echo "Validation completed successfully!"
```

### Phase 5: Gradual Transition (1 week)

#### 5.1 Monitoring and Observation
```bash
# Monitor verification rates
watch -n 30 'curl -s http://localhost:8080/api/v1/metrics | grep verification_rate'

# Monitor system performance
watch -n 60 'curl -s http://localhost:8080/api/v1/system/info | jq .data.system_stats'

# Monitor error rates
tail -f /var/log/llm-verifier/error.log | grep -i error

# Monitor application logs
tail -f /var/log/llm-verifier/app.log
```

#### 5.2 Performance Tuning
```bash
# Adjust verification concurrency
./adjust-verification-concurrency --max-concurrent 10

# Optimize database performance
./optimize-database --db llm-verifier.db

# Tune rate limiting
./update-rate-limits --requests-per-minute 500
```

#### 5.3 Enable Strict Mode (After 1 week)
```bash
# Enable strict verification mode
sed -i 's/strict_mode: false/strict_mode: true/' config.v2.yaml

# Restart services
sudo systemctl restart llm-verifier
sudo systemctl restart llm-verifier-workers

# Verify strict mode is working
./test_strict_mode.sh --config config.v2.yaml
```

## 🚨 Emergency Rollback Procedures

### Immediate Rollback
```bash
#!/bin/bash
# emergency-rollback.sh

echo "EMERGENCY ROLLBACK INITIATED"

# Stop v2 services
sudo systemctl stop llm-verifier
sudo systemctl stop llm-verifier-workers

# Restore v1 configuration
cp config.yaml.final-backup.* config.yaml

# Restore v1 database
cp llm-verifier.db.final-backup.* llm-verifier.db

# Install v1
go install ./cmd/llm-verifier@v1

# Start v1 services
sudo systemctl start llm-verifier
sudo systemctl start llm-verifier-workers

echo "Rollback completed - v1 restored"
```

### Gradual Rollback
```bash
#!/bin/bash
# gradual-rollback.sh

# Disable strict mode first
sed -i 's/strict_mode: true/strict_mode: false/' config.v2.yaml

# Disable verification
sed -i 's/enabled: true/enabled: false/' config.v2.yaml

# Restart with relaxed settings
sudo systemctl restart llm-verifier

# Monitor and plan full rollback
echo "Relaxed v2 settings applied - monitor and plan full rollback"
```

## 📊 Migration Validation Checklist

### Configuration Validation
- [ ] v2 configuration format is valid
- [ ] Model verification settings are appropriate
- [ ] Branding configuration is correct
- [ ] All required sections are present

### Database Validation
- [ ] Migration completed successfully
- [ ] New tables are created and populated
- [ ] Data integrity is maintained
- [ ] Performance is acceptable

### Application Validation
- [ ] Services start without errors
- [ ] API endpoints respond correctly
- [ ] Model verification works
- [ ] Suffixes are applied correctly

### Integration Validation
- [ ] External integrations still work
- [ ] Custom scripts handle new format
- [ ] Monitoring systems function
- [ ] Backup/restore procedures work

## 🎯 Common Migration Issues and Solutions

### Issue 1: High Verification Failure Rate
**Problem**: Many models fail verification in strict mode
**Solution**:
```bash
# Check verification logs
tail -f logs/verification.log

# Lower verification threshold temporarily
sed -i 's/min_verification_score: 0.7/min_verification_score: 0.6/' config.v2.yaml

# Increase timeout for slow providers
sed -i 's/timeout_seconds: 30/timeout_seconds: 60/' config.v2.yaml

# Test with different code examples
./test_verification_codes.sh --codes python,javascript,go
```

### Issue 2: External Integration Breaks
**Problem**: External systems can't handle new model names with suffixes
**Solution**:
```bash
# Create compatibility layer
./create_compatibility_layer.sh --integrations external_systems/

# Update external integrations
./update_external_integrations.sh --suffix-handling

# Provide API endpoint for clean names
# Implement /api/v1/models?strip_suffixes=true
```

### Issue 3: Performance Degradation
**Problem**: System performance degrades after migration
**Solution**:
```bash
# Optimize database queries
./optimize_database_queries.sh --db llm-verifier.db

# Adjust verification concurrency
./adjust_concurrency.sh --max-concurrent 5

# Enable result caching
./enable_caching.sh --cache-verification-results
```

### Issue 4: Database Migration Fails
**Problem**: Database migration fails or corrupts data
**Solution**:
```bash
# Restore from backup
cp llm-verifier.db.final-backup.* llm-verifier.db

# Manual migration with assistance
./manual_database_migration.sh --assisted

# Contact support with migration logs
./collect_migration_logs.sh --output support-package.tar.gz
```

## 📈 Post-Migration Best Practices

### Monitoring
1. **Verification Rate**: Monitor percentage of verified models
2. **System Performance**: Track response times and resource usage
3. **Error Rates**: Watch for increased error rates
4. **User Feedback**: Collect feedback from system users

### Optimization
1. **Verification Thresholds**: Fine-tune based on experience
2. **Concurrency Settings**: Optimize for your workload
3. **Caching Strategy**: Implement intelligent caching
4. **Resource Allocation**: Adjust based on usage patterns

### Maintenance
1. **Regular Verification**: Re-verify models periodically
2. **Database Maintenance**: Regular cleanup and optimization
3. **Configuration Reviews**: Review and update configurations
4. **Documentation Updates**: Keep documentation current

## 🔗 Related Resources

### Documentation
- [Model Verification Guide](../MODEL_VERIFICATION_GUIDE.md)
- [LLMSVD Suffix Guide](../LLMSVD_SUFFIX_GUIDE.md)
- [Configuration Migration Guide](../CONFIGURATION_MIGRATION_GUIDE.md)
- [Test Suite User Guide](../TEST_SUITE_USER_GUIDE.md)

### Tools and Scripts
- `migration-assistant`: Comprehensive migration analysis
- `config-migrator`: Configuration migration tool
- `database-migrator`: Database migration utility
- `validate-migration`: Migration validation tool

### Support
- **Migration Support**: migration-support@llm-verifier.com
- **Emergency Hotline**: +1-800-LLM-HELP
- **Community Forum**: [GitHub Discussions](https://github.com/vasic-digital/LLMsVerifier/discussions)
- **Documentation**: [Complete Docs](https://docs.llm-verifier.com)

---

**Remember: Migration to v2 is a significant upgrade that requires careful planning and execution. Take your time, test thoroughly, and don't hesitate to seek support when needed.**

**Good luck with your migration!** 🚀