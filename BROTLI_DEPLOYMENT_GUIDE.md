# Brotli Compression Implementation - Deployment Guide

## 📚 Table of Contents
1. [Prerequisites](#prerequisites)
2. [Installation](#installation)
3. [Configuration](#configuration)
4. [Deployment Steps](#deployment-steps)
5. [Monitoring Setup](#monitoring-setup)
6. [Verification](#verification)
7. [Troubleshooting](#troubleshooting)

---

## 🎯 Prerequisites

### System Requirements
- Go 1.21 or higher
- SQLite 3.x
- Grafana 8.x or higher (optional, for monitoring)
- Prometheus 2.x or higher (optional, for monitoring)

### API Keys
- Valid API keys for LLM providers (OpenAI, Anthropic, etc.)
- API keys should have sufficient permissions to make model requests

---

## 📦 Installation

### 1. Clone and Build
```bash
cd /path/to/llm-verifier
go build ./llm-verifier
```

### 2. Verify Database Migration
```bash
# Run verification to apply migrations
./llm-verifier --config config.yaml
```

**Expected Output:**
```
2025/12/25 01:20:17 Applying migration 5: Add Brotli compression support field to verification_results table
2025/12/25 01:20:17 Successfully applied migration 5
```

---

## ⚙️ Configuration

### 1. Basic Configuration
Create or update `config.yaml`:

```yaml
providers:
  openai:
    api_key: "your-openai-api-key"
    models:
      - "gpt-4"
      - "gpt-3.5-turbo"
  
  anthropic:
    api_key: "your-anthropic-api-key"
    models:
      - "claude-3-opus"
      - "claude-3-sonnet"

database:
  path: "./llm-verifier.db"

monitoring:
  enabled: true
  metrics_endpoint: "/metrics"
  health_endpoint: "/health"
```

### 2. Advanced Configuration (Optional)
```yaml
performance:
  cache_ttl: 24h
  max_concurrent_requests: 10
  request_timeout: 30s
  
brotli:
  enabled: true
  auto_detection: true
  cache_enabled: true
  monitor_performance: true
```

---

## 🚀 Deployment Steps

### Step 1: Deploy Application
```bash
# Build for production
cd llm-verifier
CGO_ENABLED=1 go build -o llm-verifier main.go

# Run application
./llm-verifier --config /path/to/config.yaml
```

### Step 2: Verify Brotli Integration
```bash
# Run a test verification
curl http://localhost:8080/health

# Expected response should include brotli_metrics
```

### Step 3: Generate Brotli-Optimized Configurations
```bash
# From project root
go run crush_config_converter.go discovery.json --brotli-only

# This creates:
# - discovery_brotli_optimized_crush_config.json (with API keys)
# - discovery_brotli_optimized_crush_config_redacted.json (without API keys)
# - discovery_brotli_optimized_opencode_config.json (OpenCode format)
# - discovery_brotli_optimized_brotli_stats.json (statistics)
```

---

## 📊 Monitoring Setup

### Option 1: Built-in Metrics Endpoint

The application exposes metrics at `/metrics` endpoint:

```bash
# View all metrics
curl http://localhost:8080/metrics

# View Brotli-specific metrics
curl http://localhost:8080/metrics | grep brotli
```

**Key Brotli Metrics:**
- `llm_verifier_brotli_tests_performed` - Total Brotli detection tests
- `llm_verifier_brotli_supported_models` - Number of models supporting Brotli
- `llm_verifier_brotli_support_rate_percent` - Percentage of models supporting Brotli
- `llm_verifier_brotli_cache_hits` - Cache hit count
- `llm_verifier_brotli_cache_misses` - Cache miss count
- `llm_verifier_brotli_cache_hit_rate` - Cache hit rate percentage
- `llm_verifier_brotli_avg_detection_time_seconds` - Average detection time

### Option 2: Grafana Dashboard (Recommended)

#### 1. Deploy Grafana
```bash
docker run -d \
  -p 3000:3000 \
  --name=grafana \
  grafana/grafana
```

#### 2. Import Brotli Dashboard
1. Access Grafana at `http://localhost:3000`
2. Login (default: admin/admin)
3. Navigate to Dashboards → Import
4. Upload `llm-verifier/monitoring/grafana/brotli_dashboard.json`
5. Select Prometheus data source

#### 3. Configure Prometheus
```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'llm-verifier'
    static_configs:
      - targets: ['localhost:8080']
```

Run Prometheus:
```bash
docker run -d \
  -p 9090:9090 \
  -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus
```

---

## ✅ Verification

### 1. Database Verification
```bash
# Check if Brotli migration is applied
sqlite3 llm-verifier.db "SELECT version FROM schema_migrations WHERE id = 5;"

# Expected output: 5
```

### 2. API Verification
```bash
# Test Brotli detection with real API
curl -X POST http://localhost:8080/api/verify \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "openai",
    "api_key": "your-key",
    "model": "gpt-4",
    "check_brotli": true
  }'
```

### 3. Performance Verification
Run benchmark tests:
```bash
cd llm-verifier
go test ./performance -v -run TestBrotli
```

---

## 🔧 Troubleshooting

### Issue: Migration Fails
**Symptom:** Database migration error
**Solution:**
```bash
# Check current database version
sqlite3 llm-verifier.db "SELECT * FROM schema_migrations;"

# If migration 5 is missing, manually apply
sqlite3 llm-verifier.db "ALTER TABLE verification_results ADD COLUMN supports_brotli BOOLEAN DEFAULT 0;"

# Mark migration as applied
sqlite3 llm-verifier.db "INSERT INTO schema_migrations (id, description, applied_at) VALUES (5, 'Add Brotli compression support field', datetime('now'));"
```

### Issue: Brotli Detection Fails
**Symptom:** All Brotli tests return false
**Solution:**
1. Verify API keys are valid
2. Check network connectivity
3. Review provider-specific requirements
4. Check application logs for errors

### Issue: High Cache Miss Rate
**Symptom:** Low cache hit rate
**Solution:**
1. Increase cache TTL in configuration
2. Verify cache persistence between restarts
3. Check for concurrent cache clear operations
4. Monitor cache performance via metrics

### Issue: Metrics Not Exposed
**Symptom:** `/metrics` endpoint returns 404
**Solution:**
1. Verify monitoring is enabled in configuration
2. Check application logs for endpoint registration
3. Confirm no port conflicts
4. Verify health endpoints are registered

### Issue: Grafana Dashboard No Data
**Symptom:** Dashboard shows no metrics
**Solution:**
1. Verify Prometheus is scraping metrics
2. Check Prometheus targets: `http://localhost:9090/targets`
3. Confirm Grafana data source is configured correctly
4. Verify time range in Grafana dashboard

---

## 📈 Performance Optimization

### Cache Tuning
```yaml
# For high-traffic scenarios
performance:
  cache_ttl: 48h  # Increase TTL
  
# For rapid testing environments
performance:
  cache_ttl: 1h   # Decrease TTL for fresh data
```

### Concurrent Testing
```bash
# Run concurrent benchmarks
cd llm-verifier/performance
go run benchmark.go --providers 5 --iterations 100 --concurrency 10
```

---

## 🔒 Security Considerations

### API Key Management
1. **Never commit API keys** to version control
2. Use environment variables for sensitive data
3. Rotate API keys regularly
4. Implement key rotation procedures

### Configuration Security
```bash
# Set environment variables
export OPENAI_API_KEY="your-key"
export ANTHROPIC_API_KEY="your-key"

# Use in configuration
providers:
  openai:
    api_key: "${OPENAI_API_KEY}"
```

### Redacted Configurations
Always use redacted configurations for sharing:
```bash
# Generate redacted config (removes API keys)
go run crush_config_converter.go discovery.json --brotli-only
# Result: *_redacted.json files are safe to share
```

---

## 📚 Additional Resources

### Documentation
- [Brotli Implementation Summary](./BROTLI_IMPLEMENTATION_FINAL_SUMMARY.md)
- [User Documentation](./BROTLI_USER_DOCUMENTATION.md)
- [Implementation Complete](./BROTLI_IMPLEMENTATION_COMPLETE.md)

### Code Reference
- `llm-verifier/client/http_client.go` - Brotli detection implementation
- `llm-verifier/monitoring/health.go` - Health monitoring
- `llm-verifier/performance/brotli_benchmark.go` - Benchmarking tools

### API Reference
See [API Documentation](./llm-verifier/api/docs/) for full API reference

---

## 🎯 Next Steps After Deployment

1. **Monitor Performance** - Review Grafana dashboard regularly
2. **Benchmark Regularly** - Run performance benchmarks weekly
3. **Update Configurations** - Regenerate Brotli-optimized configs as providers change
4. **Review Logs** - Monitor for Brotli-related errors
5. **Optimize TTL** - Adjust cache settings based on usage patterns

---

## 📞 Support

For issues or questions:
1. Check troubleshooting section above
2. Review application logs
3. Verify configuration syntax
4. Check API key validity
5. Consult project documentation

---

**Last Updated:** 2025-12-25  
**Version:** 1.0.0  
**Status:** Production Ready ✅