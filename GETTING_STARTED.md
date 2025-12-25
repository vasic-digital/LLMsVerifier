# LLM Verifier Quick Start Guide

Welcome to LLM Verifier! This guide will get you up and running in minutes.

## 🚀 Quick Start (5 minutes)

### 1. Start the Server

The application is already running! The server should be available at:
- **API**: http://localhost:8080
- **Health Check**: http://localhost:8080/api/health
- **Providers**: http://localhost:8080/api/providers
- **Models**: http://localhost:8080/api/models

### 2. Test the API

```bash
# Check server health
curl http://localhost:8080/api/health

# List available providers
curl http://localhost:8080/api/providers

# List available models
curl http://localhost:8080/api/models
```

### 3. Add API Keys (Optional)

To use paid providers, set environment variables:

```bash
export OPENAI_API_KEY="your-openai-key"
export ANTHROPIC_API_KEY="your-anthropic-key"
export GOOGLE_API_KEY="your-google-key"
export GROQ_API_KEY="your-groq-key"
```

## 📋 Supported Providers

LLM Verifier supports **20+ providers** including:

### Free Providers
- **Google AI Studio** - Free with rate limits
- **Cerebras** - Free inference
- **Cloudflare Workers AI** - Free tier available
- **Hugging Face** - Community models

### Commercial Providers
- **OpenAI** - GPT-4, GPT-3.5
- **Anthropic** - Claude models
- **Google** - Gemini models
- **DeepSeek** - Cost-effective models
- **Groq** - High-performance inference
- **Together AI** - Multiple model families
- And 10+ more!

## 🛠️ Using the CLI

The application provides a comprehensive CLI:

```bash
# Get help
./llm-verifier-app --help

# Start TUI (Terminal User Interface)
./llm-verifier-app tui

# Export configurations for external tools
./llm-verifier-app ai-config export

# Manage providers
./llm-verifier-app providers list
./llm-verifier-app providers add

# Run benchmarks
./llm-verifier benchmark.sh
./llm-verifier scalability-test.sh
```

## 📚 Next Steps

1. **Explore Documentation**:
   - [Administrator Manual](docs/administrator-manual.md)
   - [Developer Guide](docs/developer-manual.md)
   - [API Reference](docs/api-reference-manual.md)

2. **Configure Providers**:
   - Edit `config.yaml` for provider settings
   - Add API keys to environment variables

3. **Run Tests**:
   - Execute `go test ./...` for unit tests
   - Run benchmark scripts for performance validation

4. **Deploy Production**:
   - Use `docker-compose.yml` for containerized deployment
   - Configure monitoring and scaling

## 🎯 What Can You Do?

- **Verify LLM Outputs**: Compare responses across providers
- **Benchmark Performance**: Measure latency, throughput, costs
- **Test Reliability**: Validate uptime and error handling
- **Compare Models**: Side-by-side model evaluations
- **Export Configurations**: Generate configs for OpenCode, Crush, Claude Code

## 🆘 Getting Help

- **Health Check**: `curl http://localhost:8080/api/health`
- **Logs**: Check server logs in background process
- **Documentation**: Comprehensive manuals in `docs/` directory
- **Issues**: Report at https://github.com/sst/opencode/issues

## 🎉 You're Ready!

Your LLM Verifier instance is running and ready to help you work with Large Language Models across multiple providers. Start exploring the API or try the TUI interface!