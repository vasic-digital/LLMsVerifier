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
- **Intelligent Context Management**: 24+ hour sessions with RAG optimization
- **Supervisor/Worker Pattern**: Automated task breakdown and distributed processing
- **Vector Database Integration**: Semantic search and knowledge retrieval
- **Model Recommendations**: AI-powered model selection based on task requirements

### Production Ready
- **Docker & Kubernetes**: Production deployment with health monitoring
- **Prometheus Metrics**: Comprehensive monitoring with Grafana dashboards
- **Circuit Breaker Pattern**: Automatic failover and recovery
- **Comprehensive Testing**: 95%+ code coverage with integration tests

## 📖 Documentation

### User Guides
- [Getting Started Guide](docs/getting-started.md)
- [Configuration Guide](docs/configuration.md)
- [API Documentation](docs/api.md)
- [Deployment Guide](docs/deployment.md)
- [Security Guide](docs/security.md)

### Developer Documentation
- [Architecture Overview](docs/architecture.md)
- [API Reference](docs/api-reference.md)
- [Database Schema](docs/database-schema.md)
- [Contributing Guide](docs/contributing.md)

### Enterprise Features
- [LDAP Integration](docs/ldap-integration.md)
- [SSO Configuration](docs/sso-configuration.md)
- [Monitoring Setup](docs/monitoring-setup.md)
- [High Availability](docs/high-availability.md)

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
cp config.example.yaml config.yaml
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

We welcome contributions! Please see our [Contributing Guide](docs/contributing.md) for details.

### Development Setup
```bash
# Fork and clone
git clone https://github.com/your-username/LLMsVerifier.git
cd LLMsVerifier

# Set up development environment
make setup-dev

# Run tests
make test

# Build all components
make build-all

# Start development servers
make dev
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

- **Documentation**: [docs/](docs/)
- **Issues**: [GitHub Issues](https://github.com/vasic-digital/LLMsVerifier/issues)
- **Discussions**: [GitHub Discussions](https://github.com/vasic-digital/LLMsVerifier/discussions)
- **Email**: support@llm-verifier.com

---

**Built with ❤️ for the AI community**