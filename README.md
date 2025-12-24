# LLM Verifier - Enterprise-Grade LLM Verification Platform

[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org)
[![Docker](https://img.shields.io/badge/docker-ready-blue.svg)](https://docker.com)
[![Kubernetes](https://img.shields.io/badge/kubernetes-ready-blue.svg)](https://kubernetes.io)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

**LLM Verifier** is the most comprehensive, enterprise-grade platform for verifying, monitoring, and optimizing Large Language Model (LLM) performance across multiple providers. Built with production reliability, advanced AI capabilities, and seamless enterprise integration.

## 🌟 Key Features

### Core Capabilities
- **20+ LLM Verification Tests**: Comprehensive capability assessment across all major providers
- **Multi-Provider Support**: OpenAI, Anthropic, Google, Cohere, Meta, and more
- **Real-Time Monitoring**: 99.9% uptime with intelligent failover and health checking
- **Advanced Analytics**: AI-powered insights, trend analysis, and optimization recommendations

### Enterprise Features
- **LDAP/SSO Integration**: Enterprise authentication with SAML/OIDC support
- **SQL Cipher Encryption**: Database-level encryption for sensitive data
- **Enterprise Monitoring**: Splunk, DataDog, New Relic, ELK integration
- **Multi-Platform Clients**: CLI, TUI, Web, Desktop, and Mobile interfaces

### Advanced AI Capabilities
- **Intelligent Context Management**: 24+ hour sessions with LLM-powered summarization and RAG optimization
- **Supervisor/Worker Pattern**: Automated task breakdown using LLM analysis and distributed processing
- **Vector Database Integration**: Semantic search and knowledge retrieval
- **Model Recommendations**: AI-powered model selection based on task requirements
- **Cloud Backup Integration**: Multi-provider cloud storage for checkpoints (AWS S3, Google Cloud, Azure)

### Production Ready
- **Docker & Kubernetes**: Production deployment with health monitoring
- **Prometheus Metrics**: Comprehensive monitoring with Grafana dashboards
- **Circuit Breaker Pattern**: Automatic failover and recovery
- **Comprehensive Testing**: 95%+ code coverage with integration tests

## 📖 Documentation

### User Guides
- [Complete User Guide](llm-verifier/docs/COMPLETE_USER_MANUAL.md)
- [User Manual](llm-verifier/docs/USER_MANUAL.md)
- [API Documentation](llm-verifier/docs/API_DOCUMENTATION.md)
- [Deployment Guide](llm-verifier/docs/DEPLOYMENT_GUIDE.md)
- [Environment Variables](llm-verifier/docs/ENVIRONMENT_VARIABLES.md)

### Developer Documentation
- [Architecture Overview](docs/ARCHITECTURE_OVERVIEW.md)
- [System Documentation](docs/COMPLETE_SYSTEM_DOCUMENTATION.md)
- [API Changelog](llm-verifier/docs/CHANGELOG.md)

### Deployment Guides
- [Docker Deployment](llm-verifier/docs/deployment/docker.md)
- [Kubernetes Deployment](llm-verifier/docs/deployment/kubernetes.md)
- [AWS Deployment](llm-verifier/docs/deployment/aws.md)

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- SQLite3
- Docker (optional)
- Kubernetes (optional)

### Installation

#### Option 1: Docker (Recommended)
```bash
# Clone the repository
git clone https://github.com/vasic-digital/LLMsVerifier.git
cd LLMsVerifier

# Start with Docker Compose
docker-compose up -d

# Access the web interface at http://localhost:8080
```

#### Option 2: Local Development
```bash
# Clone the repository
git clone https://github.com/vasic-digital/LLMsVerifier.git
cd LLMsVerifier/llm-verifier

# Install dependencies
go mod download

# Configure environment
cp llm-verifier/config.yaml.example config.yaml
# Edit config.yaml with your settings

# Run the application
go run cmd/main.go
```

### Basic Configuration

Create a `config.yaml` file:

```yaml
profile: "production"
global:
  log_level: "info"
  log_file: "/var/log/llm-verifier.log"

database:
  path: "/data/llm-verifier.db"
  encryption_key: "your-encryption-key-here"

llms:
  - name: "openai-gpt4"
    provider: "openai"
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4"
    enabled: true

  - name: "anthropic-claude"
    provider: "anthropic"
    api_key: "${ANTHROPIC_API_KEY}"
    model: "claude-3-sonnet-20240229"
    enabled: true

api:
  port: 8080
  jwt_secret: "your-jwt-secret"
  enable_cors: true

### Configuration Management

The LLM Verifier includes tools for managing LLM configurations for different platforms:

#### Crush Configuration
- **Auto-Generated Configs**: Use the built-in converter to generate valid Crush configurations from discovery results
- **Streaming Support**: Configurations automatically include streaming flags when LLMs support it
- **Cost Estimation**: Realistic cost calculations based on provider and model type

```bash
# Generate Crush config from discovery
go run crush_config_converter.go path/to/discovery.json
```

#### OpenCode Configuration
- **Streaming Enabled**: All compatible models have streaming support enabled by default
- **Model Verification**: Configurations are validated to ensure consistency

#### Sensitive File Handling

The LLM Verifier implements secure configuration management:

- **Full Files**: Contain actual API keys - **gitignored** (e.g., `*_config.json`)
- **Redacted Files**: API keys as `""` - **versioned** (e.g., `*_config_redacted.json`)
- **Platform Formats**: Generates Crush and OpenCode configs per official specs

**Security**: Never commit files with real API keys. Use redacted versions for sharing.

#### Platform Configuration Formats

- **Crush**: Full JSON schema compliance with providers, models, costs, and options
- **OpenCode**: Official format with `$schema`, `provider` object containing `options.apiKey` and empty `models`

### Running Challenges

To run LLM verification challenges:

```bash
# Run provider discovery
go run llm-verifier/challenges/codebase/go_files/provider_models_discovery.go

# Run model verification
go run llm-verifier/challenges/codebase/go_files/run_model_verification.go
```

monitoring:
  enabled: true
  prometheus:
    enabled: true
    port: 9090

enterprise:
  ldap:
    enabled: true
    host: "ldap.company.com"
    port: 389
    base_dn: "dc=company,dc=com"
    bind_user: "cn=service,ou=users,dc=company,dc=com"
    bind_password: "${LDAP_BIND_PASSWORD}"

  sso:
    provider: "saml"
    saml:
      entity_id: "llm-verifier"
      sso_url: "https://sso.company.com/saml/sso"
      certificate: "/path/to/cert.pem"
      private_key: "/path/to/key.pem"
```

## 🔧 API Usage

### REST API

The LLM Verifier provides a comprehensive REST API for all operations:

```bash
# Verify a model
curl -X POST http://localhost:8080/api/v1/verify \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "model_id": "gpt-4",
    "prompt": "Explain quantum computing in simple terms"
  }'

# Get verification results
curl -X GET http://localhost:8080/api/v1/results/gpt-4 \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# Start real-time chat
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "model_id": "claude-3-sonnet",
    "messages": [
      {"role": "user", "content": "Hello, how are you?"}
    ],
    "stream": true
  }'
```

### SDK Usage

#### Go SDK
```go
package main

import (
    "fmt"
    "log"

    "github.com/vasic-digital/LLMsVerifier/sdk/go"
)

func main() {
    client := llmverifier.NewClient("http://localhost:8080", "your-api-key")

    result, err := client.VerifyModel("gpt-4", "Test prompt")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Score: %.2f, Capabilities: %v\n",
        result.OverallScore, result.Capabilities)
}
```

#### JavaScript SDK
```javascript
const { LLMVerifier } = require('@llm-verifier/sdk');

const client = new LLMVerifier({
    baseURL: 'http://localhost:8080',
    apiKey: 'your-api-key'
});

async function verifyModel() {
    try {
        const result = await client.verifyModel('gpt-4', 'Test prompt');
        console.log(`Score: ${result.overallScore}`);
        console.log(`Capabilities:`, result.capabilities);
    } catch (error) {
        console.error('Verification failed:', error);
    }
}

verifyModel();
```

## 🏗️ Architecture

### System Components

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   CLI/TUI/Web   │    │   API Server    │    │   Mobile Apps   │
│   Interfaces    │◄──►│   (Gin/Rest)    │◄──►│   (React Native)│
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  LLM Verifier   │    │  Context Mgmt   │    │  Vector DB      │
│  (Core Logic)   │◄──►│  (RAG/Summary)  │◄──►│  (Embeddings)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Supervisor    │    │   Workers       │    │   Providers     │
│   (Task Mgmt)   │◄──►│   (Processing)  │◄──►│   (OpenAI, etc) │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Database      │    │   Monitoring    │    │   Enterprise    │
│   (SQL Cipher)  │◄──►│   (Prometheus)  │◄──►│   (LDAP/SSO)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Key Design Patterns

- **Circuit Breaker**: Automatic failover for provider outages
- **Supervisor/Worker**: Distributed task processing with load balancing
- **Repository Pattern**: Clean data access layer
- **Observer Pattern**: Event-driven architecture
- **Strategy Pattern**: Pluggable provider adapters
- **Decorator Pattern**: Middleware for authentication and logging

## 🎯 Advanced Features

### Intelligent Model Selection
```go
// AI-powered model recommendation
requirements := analytics.TaskRequirements{
    TaskType:         "coding",
    Complexity:       "medium",
    SpeedRequirement: "normal",
    BudgetLimit:      0.50, // $0.50 per request
    RequiredFeatures: []string{"function_calling", "json_mode"},
}

recommendation, _ := recommender.RecommendModel(requirements)
fmt.Printf("Recommended: %s (Score: %.1f, Cost: $%.4f)\n",
    recommendation.BestChoice.ModelID,
    recommendation.BestChoice.Score,
    recommendation.BestChoice.CostEstimate)
```

### Context Management with RAG
```go
// Advanced context with vector search
contextMgr := context.NewConversationManager(100, time.Hour)
rag := vector.NewRAGService(vectorDB, embeddings, contextMgr)

// Index conversation messages
for _, msg := range messages {
    rag.IndexMessage(ctx, msg)
}

// Retrieve relevant context
relevantDocs, _ := rag.RetrieveContext(ctx, query, conversationID)

// Optimize prompts with context
optimizedPrompt, _ := rag.OptimizePrompt(ctx, userPrompt, conversationID)
```

### Enterprise Monitoring
```yaml
# Prometheus metrics endpoint: http://localhost:9090/metrics
# Grafana dashboard: Import dashboard ID 1860

monitoring:
  enabled: true
  prometheus:
    enabled: true
    port: 9090

enterprise:
  monitoring:
    enabled: true
    splunk:
      host: "splunk.company.com"
      token: "${SPLUNK_TOKEN}"
    datadog:
      api_key: "${DD_API_KEY}"
      service_name: "llm-verifier"
```

## 🚀 Deployment

### Docker Deployment
```bash
# Build and run
docker build -t llm-verifier .
docker run -p 8080:8080 -v /data:/data llm-verifier

# With Docker Compose
docker-compose up -d
```

### Kubernetes Deployment
```bash
# Deploy to Kubernetes
kubectl apply -f k8s-manifests/

# Check status
kubectl get pods
kubectl get services
```

### High Availability Setup
```yaml
# Multi-zone deployment with load balancing
apiVersion: apps/v1
kind: Deployment
metadata:
  name: llm-verifier
spec:
  replicas: 3
  selector:
    matchLabels:
      app: llm-verifier
  template:
    spec:
      containers:
      - name: llm-verifier
        image: llm-verifier:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_PATH
          value: "/data/llm-verifier.db"
        volumeMounts:
        - name: data
          mountPath: /data
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: llm-verifier-data
```

## 🤝 Contributing

We welcome contributions! Please see our documentation for details on how to contribute to the project.

### Development Setup
```bash
# Clone and set up
git clone https://github.com/vasic-digital/LLMsVerifier.git
cd LLMsVerifier/llm-verifier

# Install dependencies
go mod download

# Run tests
go test ./...

# Build application
go build -o llm-verifier cmd/main.go

# Run application
./llm-verifier
```

### Code Quality
- Go: `gofmt`, `go vet`, `golint`
- TypeScript: ESLint, Prettier
- Tests: 95%+ coverage required
- Documentation: Auto-generated API docs

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- OpenAI, Anthropic, Google, and other LLM providers for their APIs
- The Go community for excellent libraries and tools
- Contributors and users for their valuable feedback

## 📞 Support

- **Documentation**: [llm-verifier/docs/](llm-verifier/docs/)
- **Issues**: [GitHub Issues](https://github.com/vasic-digital/LLMsVerifier/issues)
- **Discussions**: [GitHub Discussions](https://github.com/vasic-digital/LLMsVerifier/discussions)

---

**Built with ❤️ for the AI community**