#!/bin/bash
# LLM Verifier - Environment Setup Script

echo "🚀 Setting up LLM Verifier Implementation Environment"
echo "===================================================="
echo "📅 Date: $(date)"
echo "👤 User: $(whoami)"
echo "📁 Working Directory: $(pwd)"
echo ""

# Create necessary directories
echo "🔧 Creating working directories..."
mkdir -p testing/{unit,integration,e2e,performance,security,mobile}
mkdir -p testing/{fixtures,mocks,reports}
mkdir -p logs
mkdir -p backup
mkdir -p mobile/{flutter,react-native,harmony-os,aurora-os}
mkdir -p sdk/{java,dotnet,python,javascript,go}
mkdir -p docs/{api,user,developer,enterprise}
mkdir -p website/{html,css,js,assets}
mkdir -p video-course/{scripts,recordings,assets}

echo "✅ Directories created successfully"

# Check Go installation and version
echo "🔍 Checking Go environment..."
if command -v go &> /dev/null; then
    GO_VERSION=$(go version)
    echo "✅ Go found: $GO_VERSION"
    
    # Check if Go version is 1.21+
    if go version | grep -q "go1\.[2-9][0-9]\|go1\.21\|go1\.22\|go1\.23"; then
        echo "✅ Go version 1.21+ confirmed"
    else
        echo "⚠️  Go version 1.21+ recommended for optimal compatibility"
    fi
else
    echo "❌ Go not found. Please install Go 1.21+ first"
    exit 1
fi

# Install Go dependencies
echo "📦 Installing Go dependencies..."
go mod download
go mod tidy

# Install testing dependencies
echo "🧪 Installing testing dependencies..."
go install github.com/onsi/ginkgo/v2/ginkgo@latest
go install github.com/onsi/gomega/...@latest
go install github.com/golang/mock/mockgen@latest

# Install code quality tools
echo "🔧 Installing code quality tools..."
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/mvdan/gofumpt@latest
go install github.com/daixiang0/gci@latest

# Install security tools
echo "🔒 Installing security tools..."
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

echo "✅ Go dependencies installed successfully"

# Create test configuration
echo "⚙️  Creating test configuration..."
cat > testing/config/test_config.yaml << 'EOF'
testing:
  unit_tests:
    coverage_target: 95
    parallel: true
    timeout: 30s
  
  integration_tests:
    database: test_db
    api_timeout: 60s
    max_retries: 3
  
  e2e_tests:
    headless: true
    screenshot_on_failure: true
    video_recording: true
  
  performance_tests:
    load_test_duration: 300s
    concurrent_users: 100
    ramp_up_time: 60s
  
  security_tests:
    scan_timeout: 600s
    severity_threshold: medium
EOF

# Create test database setup
echo "🗄️  Creating test database setup..."
cat > testing/setup_test_db.sh << 'EOF'
#!/bin/bash
# Setup test database

echo "Setting up test database..."

# Create test database
sqlite3 testing/test.db << 'SQL'
-- Create test schema
CREATE TABLE IF NOT EXISTS test_models (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    model_id TEXT NOT NULL,
    name TEXT NOT NULL,
    provider TEXT NOT NULL,
    overall_score REAL DEFAULT 0.0,
    score_suffix TEXT DEFAULT '',
    is_active BOOLEAN DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS test_verifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    model_id TEXT NOT NULL,
    prompt TEXT NOT NULL,
    response TEXT,
    score REAL DEFAULT 0.0,
    score_suffix TEXT DEFAULT '',
    success BOOLEAN DEFAULT 1,
    duration INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert test data
INSERT INTO test_models (model_id, name, provider, overall_score, score_suffix) VALUES 
('gpt-4-test', 'GPT-4 Test (SC:8.5)', 'OpenAI', 8.5, '(SC:8.5)'),
('claude-3-test', 'Claude-3 Test (SC:7.8)', 'Anthropic', 7.8, '(SC:7.8)'),
('gemini-pro-test', 'Gemini Pro Test (SC:7.2)', 'Google', 7.2, '(SC:7.2)');

INSERT INTO test_verifications (model_id, prompt, response, score, score_suffix, success) VALUES 
('gpt-4-test', 'Test prompt 1', 'Test response 1', 8.5, '(SC:8.5)', 1),
('claude-3-test', 'Test prompt 2', 'Test response 2', 7.8, '(SC:7.8)', 1),
('gemini-pro-test', 'Test prompt 3', 'Test response 3', 7.2, '(SC:7.2)', 1);
SQL

echo "✅ Test database setup complete"
EOF

chmod +x testing/setup_test_db.sh

# Create comprehensive test runner
echo "🎯 Creating comprehensive test runner..."
cat > testing/run_all_tests.sh << 'EOF'
#!/bin/bash
# Comprehensive test runner

echo "🧪 Running comprehensive test suite..."

# Setup test environment
export LLM_VERIFIER_ENV=test
export LLM_VERIFIER_DATABASE_PATH=testing/test.db

# Create test database if it doesn't exist
if [ ! -f "$LLM_VERIFIER_DATABASE_PATH" ]; then
    echo "Creating test database..."
    ./testing/setup_test_db.sh
fi

# Run unit tests with coverage
echo "📊 Running unit tests with coverage..."
go test ./... -v -coverprofile=testing/reports/coverage.out -covermode=atomic -tags=unit

# Run integration tests
echo "🔗 Running integration tests..."
go test ./... -tags=integration -v -coverprofile=testing/reports/integration_coverage.out

# Run E2E tests
echo "🎯 Running end-to-end tests..."
go test ./... -tags=e2e -v -coverprofile=testing/reports/e2e_coverage.out

# Run performance tests
echo "⚡ Running performance tests..."
go test ./... -bench=. -benchmem -coverprofile=testing/reports/performance_coverage.out

# Run security tests
echo "🔒 Running security tests..."
go test ./... -tags=security -v -coverprofile=testing/reports/security_coverage.out

# Generate coverage report
echo "📈 Generating coverage report..."
if [ -f testing/reports/coverage.out ]; then
    go tool cover -html=testing/reports/coverage.out -o testing/reports/coverage.html
    echo "Coverage Summary:"
    go tool cover -func=testing/reports/coverage.out | grep total
fi

echo "✅ Comprehensive test suite completed!"
echo "📊 Coverage reports available in: testing/reports/"
EOF

chmod +x testing/run_all_tests.sh

# Create mobile development directories
echo "📱 Setting up mobile development environment..."
mkdir -p mobile/flutter/{lib/{core/{services,providers,routes,themes,widgets},features/{auth,verification,dashboard,models,settings,enterprise}/{screens,widgets,models,services}},test,android,ios,web}
mkdir -p mobile/react-native/{src/{components,services,screens,utils},__tests__,android,ios}
mkdir -p mobile/harmony-os/{entry,main,resources}
mkdir -p mobile/aurora-os/{src,rpms}

# Create SDK development directories
echo "📦 Setting up SDK development environment..."
mkdir -p sdk/java/{src/{main,test}/{java/com/llmverifier/sdk,resources},docs,examples}
mkdir -p sdk/dotnet/{src/{LLMVerifier.SDK,LLMVerifier.SDK.Tests},docs,examples}
mkdir -p sdk/python/{llmverifier,tests,docs,examples}
mkdir -p sdk/javascript/{src,test,docs,examples}
mkdir -p sdk/go/{client,models,services,examples}

# Create documentation structure
echo "📚 Setting up documentation structure..."
mkdir -p docs/{api/{endpoints,schemas},user/{getting-started,tutorials,faq},developer/{architecture,contribution,testing},enterprise/{deployment,security,administration}}
mkdir -p website/{html,css,js,assets/{images,icons,videos}}
mkdir -p video-course/{scripts/{module1,module2,module3,module4,module5},recordings,assets/{slides,demos}}

# Verify Go module integrity
echo "🔍 Verifying Go module integrity..."
if go mod verify; then
    echo "✅ Go modules verified successfully"
else
    echo "⚠️  Go module verification failed - attempting to fix..."
    go mod tidy
    go mod download
fi

# Build the project to verify setup
echo "🔨 Building project to verify setup..."
if go build ./...; then
    echo "✅ Project builds successfully"
else
    echo "❌ Project build failed - please check dependencies"
    exit 1
fi

echo ""
echo "🎉 Environment setup completed successfully!"
echo ""
echo "📁 Created directory structure:"
echo "  - testing/ (unit, integration, e2e, performance, security)"
echo "  - mobile/ (flutter, react-native, harmony-os, aurora-os)"
echo "  - sdk/ (java, dotnet, python, javascript, go)"
echo "  - docs/ (api, user, developer, enterprise)"
echo "  - website/ (html, css, js, assets)"
echo "  - video-course/ (scripts, recordings, assets)"
echo "  - logs/ (implementation tracking)"
echo "  - backup/ (safe storage)"
echo ""
echo "🧪 Test infrastructure ready:"
echo "  - Run all tests: ./testing/run_all_tests.sh"
echo "  - Setup test DB: ./testing/setup_test_db.sh"
echo "  - View coverage: testing/reports/coverage.html"
echo ""
echo "📱 Mobile development ready:"
echo "  - Flutter app structure created"
echo "  - React Native structure created"
echo "  - Harmony OS structure created"
echo "  - Aurora OS structure created"
echo ""
echo "📦 SDK development ready:"
echo "  - Java SDK structure created"
echo "  - .NET SDK structure created"
echo "  - Python SDK structure created"
echo "  - JavaScript SDK structure created"
echo "  - Go SDK structure created"
echo ""
echo "🚀 Next step: Run critical fixes implementation"