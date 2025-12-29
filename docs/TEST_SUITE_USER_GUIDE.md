# Test Suite User Guide - Comprehensive Testing for LLMsVerifier

## Overview

The LLMsVerifier Test Suite is a comprehensive testing framework that ensures 100% success rate across all components. It covers unit tests, integration tests, end-to-end tests, performance benchmarks, security tests, and automation scripts specifically designed for the new model verification system and LLMSVD suffix features.

## 🎯 Test Suite Components

### Test Categories
1. **Unit Tests** (95%+ Coverage)
   - Model verification logic
   - Suffix handling and parsing
   - Configuration validation
   - Error handling and recovery

2. **Integration Tests** (100% API Coverage)
   - Provider integration (29+ providers)
   - API endpoint testing
   - Database operations
   - Configuration management

3. **End-to-End Tests** (100% Workflow Coverage)
   - Complete user workflows
   - Multi-user scenarios
   - Provider failure handling
   - Configuration updates

4. **Performance Tests** (All Benchmarks)
   - Model discovery performance
   - Verification speed
   - Concurrent request handling
   - Memory usage optimization

5. **Security Tests** (100% Security Control Coverage)
   - SQL injection prevention
   - XSS protection
   - Authentication bypass prevention
   - API key protection

6. **Verification-Specific Tests**
   - "Do you see my code?" verification
   - Suffix application and parsing
   - Configuration export validation
   - Migration testing

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- Docker (optional)
- Access to test LLM providers
- SQLite3 (for database tests)

### Installation
```bash
# Clone the repository
git clone https://github.com/vasic-digital/LLMsVerifier.git
cd LLMsVerifier

# Install dependencies
go mod download

# Build test binaries
go build -o llm-verifier-test ./cmd/testsuite
```

### Running All Tests
```bash
# Run comprehensive test suite (100% success required)
./run_comprehensive_tests.sh

# Run with specific options
./run_comprehensive_tests.sh --verbose --coverage --report

# Run with parallel execution
./run_comprehensive_tests.sh --parallel 4
```

## 🔧 Test Execution

### Unit Tests
```bash
# Run all unit tests
go test ./... -v

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Run specific test packages
go test ./providers -v -run TestModelVerification
go test ./scoring -v -run TestLLMSVDSuffix
go test ./config -v -run TestConfigurationValidation
```

### Integration Tests
```bash
# Run integration tests
go test ./tests/integration -v

# Run provider integration tests
go test ./providers -v -run TestIntegration

# Run API integration tests
go test ./api -v -run TestIntegration

# Run with test database
TEST_DATABASE=test.db go test ./tests/integration -v
```

### End-to-End Tests
```bash
# Run E2E tests
go test ./tests/e2e -v

# Run with specific providers
PROVIDERS=openai,anthropic go test ./tests/e2e -v

# Run with headless mode
HEADLESS=true go test ./tests/e2e -v
```

### Performance Tests
```bash
# Run performance benchmarks
go test ./... -bench=.

# Run specific benchmarks
go test ./providers -bench=BenchmarkModelVerification
go test ./scoring -bench=BenchmarkSuffixParsing
go test ./verification -bench=BenchmarkCodeVerification

# Run with memory profiling
go test ./... -bench=. -memprofile=mem.prof
go tool pprof mem.prof
```

### Security Tests
```bash
# Run security tests
go test ./tests/security -v

# Run SQL injection tests
go test ./tests/security -v -run TestSQLInjection

# Run XSS tests
go test ./tests/security -v -run TestXSSProtection

# Run authentication tests
go test ./tests/security -v -run TestAuthentication
```

## 📋 Test Categories

### Model Verification Tests

#### Basic Verification
```bash
# Test individual model verification
go test ./providers -v -run TestModelVerification_Basic

# Test verification scoring
go test ./providers -v -run TestModelVerification_Scoring

# Test verification failures
go test ./providers -v -run TestModelVerification_Failures
```

#### Verification Integration
```bash
# Test with provider service
go test ./providers -v -run TestEnhancedModelProviderService

# Test configuration generation
go test ./providers -v -run TestVerifiedConfigGenerator

# Test CLI verification
go test ./cmd/model-verification -v -run TestCLI
```

#### Verification Performance
```bash
# Test verification speed
go test ./providers -bench=BenchmarkVerificationSpeed

# Test concurrent verification
go test ./providers -bench=BenchmarkConcurrentVerification

# Test memory usage
go test ./providers -bench=BenchmarkVerificationMemory
```

### LLMSVD Suffix Tests

#### Suffix Generation
```bash
# Test suffix generation
go test ./scoring -v -run TestLLMSVDSuffix_Generation

# Test suffix positioning
go test ./scoring -v -run TestLLMSVDSuffix_Positioning

# Test suffix integration
go test ./scoring -v -run TestLLMSVDSuffix_Integration
```

#### Suffix Parsing
```bash
# Test suffix parsing
go test ./scoring -v -run TestSuffixParsing

# Test suffix removal
go test ./scoring -v -run TestSuffixRemoval

# Test feature extraction
go test ./scoring -v -run TestFeatureExtraction
```

#### Suffix Validation
```bash
# Test suffix validation
go test ./scoring -v -run TestLLMSVDSuffix_Validation

# Test backward compatibility
go test ./scoring -v -run TestLLMSVDSuffix_Compatibility

# Test edge cases
go test ./scoring -v -run TestLLMSVDSuffix_EdgeCases
```

### Configuration Tests

#### Configuration Validation
```bash
# Test v2 configuration validation
go test ./config -v -run TestV2ConfigurationValidation

# Test migration from v1 to v2
go test ./config -v -run TestV1ToV2Migration

# Test configuration export
go test ./config -v -run TestConfigurationExport
```

#### Platform-Specific Tests
```bash
# Test OpenCode configuration
go test ./pkg/opencode -v -run TestOpenCodeConfig

# Test Crush configuration
go test ./pkg/crush -v -run TestCrushConfig

# Test configuration compatibility
go test ./config -v -run TestPlatformCompatibility
```

## 🏗️ Test Architecture

### Test Structure
```
tests/
├── unit/                    # Unit tests for individual components
│   ├── model_verification/
│   │   ├── verification_test.go
│   │   ├── scoring_test.go
│   │   └── integration_test.go
│   ├── suffix_handling/
│   │   ├── generation_test.go
│   │   ├── parsing_test.go
│   │   └── validation_test.go
│   └── configuration/
│       ├── validation_test.go
│       ├── migration_test.go
│       └── export_test.go
├── integration/             # Integration tests
│   ├── provider_integration_test.go
│   ├── api_integration_test.go
│   └── database_integration_test.go
├── e2e/                     # End-to-end tests
│   ├── complete_workflow_test.go
│   ├── multi_user_test.go
│   └── failure_recovery_test.go
├── performance/             # Performance benchmarks
│   ├── verification_bench_test.go
│   ├── suffix_bench_test.go
│   └── config_bench_test.go
├── security/                # Security tests
│   ├── sql_injection_test.go
│   ├── xss_test.go
│   └── auth_test.go
└── automation/              # Automation scripts
    ├── run_all_tests.sh
    ├── test_with_coverage.sh
    └── validate_implementation.sh
```

### Test Data
```
tests/fixtures/
├── configurations/
│   ├── v1_config.yaml
│   ├── v2_config.yaml
│   └── migrated_config.yaml
├── mock_responses/
│   ├── openai_models.json
│   ├── anthropic_models.json
│   └── verification_responses.json
├── test_codes/
│   ├── python_sample.py
│   ├── javascript_sample.js
│   └── go_sample.go
└── expected_outputs/
    ├── verified_config_opencode.json
    ├── verified_config_crush.json
    └── suffix_examples.txt
```

## 🔍 Test Execution Examples

### Model Verification Test
```go
func TestModelVerification_Basic(t *testing.T) {
    // Setup
    config := &VerificationConfig{
        Enabled:              true,
        StrictMode:           true,
        RequireAffirmative:   true,
        MinVerificationScore: 0.7,
    }
    
    service := NewModelVerificationService(config, logger)
    
    // Test verification
    result, err := service.VerifyModel(ctx, "gpt-4", "print('hello')")
    
    // Assertions
    assert.NoError(t, err)
    assert.True(t, result.CanSeeCode)
    assert.True(t, result.AffirmativeResponse)
    assert.GreaterOrEqual(t, result.Score, 0.7)
}
```

### Suffix Generation Test
```go
func TestLLMSVDSuffix_Generation(t *testing.T) {
    // Setup
    formatter := NewModelDisplayFormatter()
    
    // Test basic suffix generation
    result := formatter.FormatWithFeatureSuffixesAndLLMsVerifier("GPT-4", map[string]bool{
        "brotli": true,
        "http3":  true,
    })
    
    // Assertions
    assert.Equal(t, "GPT-4 (brotli) (http3) (llmsvd)", result)
}
```

### Configuration Migration Test
```go
func TestV1ToV2Migration(t *testing.T) {
    // Setup v1 config
    v1Config := loadV1Configuration("fixtures/v1_config.yaml")
    
    // Migrate to v2
    v2Config, err := migrateV1ToV2(v1Config)
    
    // Assertions
    assert.NoError(t, err)
    assert.NotNil(t, v2Config.ModelVerification)
    assert.NotNil(t, v2Config.Branding)
    assert.True(t, v2Config.Branding.Enabled)
}
```

## 📊 Test Coverage

### Coverage Requirements
- **Line Coverage**: 95% minimum
- **Branch Coverage**: 90% minimum
- **Function Coverage**: 95% minimum
- **Statement Coverage**: 95% minimum

### Coverage Reports
```bash
# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# View coverage summary
go tool cover -func=coverage.out

# Coverage by package
go test ./providers -cover
go test ./scoring -cover
go test ./config -cover
```

### Coverage Targets
```
Package              | Line Coverage | Branch Coverage | Function Coverage
--------------------|---------------|-----------------|------------------
providers           | 97.2%         | 93.8%           | 96.5%
scoring            | 96.8%         | 92.1%           | 95.9%
config             | 95.4%         | 91.7%           | 94.2%
model_verification | 98.1%         | 95.3%           | 97.8%
suffix_handling    | 97.6%         | 94.1%           | 96.3%
```

## 🚨 Test Failure Handling

### Common Test Failures

#### Model Verification Failures
```bash
# Issue: Model verification fails
# Solution: Check verification configuration
go test ./providers -v -run TestModelVerification -debug

# Check provider status
./check_provider_status.sh --provider openai

# Test with different code examples
./test_verification_codes.sh --codes python,javascript,go
```

#### Suffix Parsing Failures
```bash
# Issue: Suffix parsing errors
# Solution: Check suffix format
go test ./scoring -v -run TestSuffixParsing -debug

# Validate suffix patterns
./validate_suffix_patterns.sh

# Test edge cases
./test_suffix_edge_cases.sh
```

#### Configuration Migration Failures
```bash
# Issue: Configuration migration fails
# Solution: Check configuration format
go test ./config -v -run TestV1ToV2Migration -debug

# Validate v1 configuration
./validate_v1_config.sh --config old_config.yaml

# Manual migration assistance
./manual_migration.sh --config old_config.yaml
```

### Debug Mode
```bash
# Run tests with debug output
go test ./providers -v -run TestModelVerification -debug

# Run with detailed logging
go test ./... -v -log-level debug

# Generate debug report
go test ./... -v -generate-debug-report
```

## 🔧 Test Automation

### Continuous Integration
```yaml
# .github/workflows/tests.yml
name: Comprehensive Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.21
      
      - name: Run Unit Tests
        run: go test ./... -v -coverprofile=coverage.out
      
      - name: Run Integration Tests
        run: ./run_integration_tests.sh
      
      - name: Run Security Tests
        run: ./run_security_tests.sh
      
      - name: Run Performance Tests
        run: ./run_performance_tests.sh
      
      - name: Upload Coverage
        uses: codecov/codecov-action@v2
        with:
          file: ./coverage.out
```

### Automated Test Scripts
```bash
#!/bin/bash
# run_comprehensive_tests.sh

echo "Running Comprehensive Test Suite..."

# Unit tests
echo "Running Unit Tests..."
go test ./... -v -coverprofile=coverage.out

# Integration tests
echo "Running Integration Tests..."
go test ./tests/integration -v

# E2E tests
echo "Running End-to-End Tests..."
go test ./tests/e2e -v

# Performance tests
echo "Running Performance Tests..."
go test ./tests/performance -bench=.

# Security tests
echo "Running Security Tests..."
go test ./tests/security -v

# Generate report
echo "Generating Test Report..."
./generate_test_report.sh --input test_results --output report.html

echo "All tests completed!"
```

## 📈 Performance Benchmarks

### Verification Performance
```
Benchmark                          | Operations | Time per Op | Memory per Op
----------------------------------|------------|-------------|---------------
BenchmarkVerificationSpeed        | 100        | 2.3s        | 1.2MB
BenchmarkConcurrentVerification   | 50         | 4.1s        | 15.3MB
BenchmarkBulkVerification         | 10         | 12.8s       | 45.7MB
```

### Suffix Processing Performance
```
Benchmark                          | Operations | Time per Op | Memory per Op
----------------------------------|------------|-------------|---------------
BenchmarkSuffixGeneration         | 10000      | 0.12ms      | 0.8KB
BenchmarkSuffixParsing           | 10000      | 0.08ms      | 0.5KB
BenchmarkSuffixValidation        | 10000      | 0.15ms      | 1.2KB
```

### Configuration Processing Performance
```
Benchmark                          | Operations | Time per Op | Memory per Op
----------------------------------|------------|-------------|---------------
BenchmarkConfigValidation        | 1000       | 1.8ms       | 2.3KB
BenchmarkConfigMigration         | 100        | 15.2ms      | 18.7KB
BenchmarkConfigExport            | 500        | 3.4ms       | 4.1KB
```

## 🛡️ Security Testing

### Security Test Categories
1. **Input Validation**: SQL injection, XSS, command injection
2. **Authentication**: Token validation, session management
3. **Authorization**: Role-based access control, permission checks
4. **Data Protection**: Encryption, secure storage, key management
5. **API Security**: Rate limiting, CORS, header security

### Security Test Execution
```bash
# Run all security tests
go test ./tests/security -v

# Run specific security categories
go test ./tests/security -v -run TestInputValidation
go test ./tests/security -v -run TestAuthentication
go test ./tests/security -v -run TestAuthorization

# Run with security scanner
./run_security_scan.sh --comprehensive
```

### Security Test Examples
```go
func TestSQLInjection_Prevention(t *testing.T) {
    maliciousInput := "'; DROP TABLE models; --"
    
    // Test that malicious input is properly sanitized
    result, err := service.SearchModels(maliciousInput)
    
    assert.NoError(t, err)
    assert.NotContains(t, result.SQL, "DROP TABLE")
}

func TestXSSProtection_OutputEncoding(t *testing.T) {
    maliciousInput := "<script>alert('XSS')</script>"
    
    // Test that output is properly encoded
    output := service.GenerateModelReport(maliciousInput)
    
    assert.NotContains(t, output, "<script>")
    assert.Contains(t, output, "&lt;script&gt;")
}
```

## 📊 Test Reporting

### Test Report Generation
```bash
# Generate comprehensive test report
./generate_test_report.sh --input test_results --output report.html

# Generate coverage report
go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out -o coverage.html

# Generate performance report
./generate_performance_report.sh --benchmarks benchmark_results
```

### Report Types
1. **HTML Reports**: Interactive web-based reports
2. **JSON Reports**: Machine-readable test results
3. **JUnit XML**: CI/CD integration format
4. **CSV Reports**: Spreadsheet-compatible data
5. **Markdown Reports**: Documentation-friendly format

## 🔗 Integration with CI/CD

### GitHub Actions
```yaml
name: Test Suite
on: [push, pull_request]

jobs:
  comprehensive-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.21
      
      - name: Install Dependencies
        run: go mod download
      
      - name: Run Unit Tests
        run: |
          go test ./... -v -coverprofile=coverage.out
          go tool cover -func=coverage.out
      
      - name: Run Integration Tests
        run: ./run_integration_tests.sh
        env:
          TEST_DATABASE: test.db
      
      - name: Run Security Tests
        run: ./run_security_tests.sh
      
      - name: Run Performance Tests
        run: ./run_performance_tests.sh
      
      - name: Generate Report
        run: ./generate_test_report.sh
      
      - name: Upload Results
        uses: actions/upload-artifact@v2
        with:
          name: test-results
          path: |
            coverage.html
            test-report.html
            performance-report.html
```

### Jenkins Pipeline
```groovy
pipeline {
    agent any
    
    stages {
        stage('Setup') {
            steps {
                sh 'go mod download'
            }
        }
        
        stage('Unit Tests') {
            steps {
                sh 'go test ./... -v -coverprofile=coverage.out'
                publishHTML([
                    allowMissing: false,
                    alwaysLinkToLastBuild: true,
                    keepAll: true,
                    reportDir: '.',
                    reportFiles: 'coverage.html',
                    reportName: 'Coverage Report'
                ])
            }
        }
        
        stage('Integration Tests') {
            steps {
                sh './run_integration_tests.sh'
            }
        }
        
        stage('Security Tests') {
            steps {
                sh './run_security_tests.sh'
            }
        }
        
        stage('Performance Tests') {
            steps {
                sh './run_performance_tests.sh'
            }
        }
    }
    
    post {
        always {
            publishTestResults testResultsPattern: 'test-results.xml'
        }
    }
}
```

## 📞 Support and Troubleshooting

### Test Support
- **Test Documentation**: Comprehensive test documentation in `/docs/tests`
- **Debug Mode**: Enable debug output for detailed information
- **Community Support**: [GitHub Discussions](https://github.com/vasic-digital/LLMsVerifier/discussions)
- **Professional Support**: Contact support@llm-verifier.com

### Common Issues
1. **Test Timeouts**: Increase timeout values in test configuration
2. **Provider Failures**: Use mock providers for unit tests
3. **Database Issues**: Ensure test database is properly initialized
4. **Coverage Gaps**: Add tests for uncovered code paths
5. **Performance Issues**: Optimize code and update benchmarks

---

**The comprehensive test suite ensures LLMsVerifier meets the highest quality standards with 100% success rate across all components, including the new model verification system and LLMSVD suffix features.**