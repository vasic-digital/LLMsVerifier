#!/bin/bash

# Ultimate OpenCode Configuration Generator
# This script sets all API keys from .env and generates the most comprehensive OpenCode configuration

cd /media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier

echo "🚀 Generating Ultimate OpenCode Configuration..."

# Export all API keys from .env file
echo "📋 Loading API keys from .env file..."
export $(grep -v '^#' .env | grep -v '^$' | sed 's/ApiKey_//' | xargs)

# Set specific environment variables that llm-verifier expects
echo "🔧 Setting environment variables for llm-verifier..."

# Core providers
export ANTHROPIC_API_KEY="$HuggingFace"
export OPENAI_API_KEY="$HuggingFace"
export HUGGINGFACE_API_KEY="$HuggingFace"
export NVIDIA_API_KEY="$Nvidia"
export CHUTES_API_KEY="$Chutes"
export SILICONFLOW_API_KEY="$SiliconFlow"
export KIMI_API_KEY="$Kimi"
export GEMINI_API_KEY="$Gemini"
export OPENROUTER_API_KEY="$OpenRouter"
export ZAI_API_KEY="$ZAI"
export DEEPSEEK_API_KEY="$DeepSeek"
export MISTRAL_API_KEY="$Mistral_AiStudio"
export CODESTRAL_API_KEY="$Codestral"
export CEREBRAS_API_KEY="$Cerebras"
export CLOUDFLARE_API_KEY="$Cloudflare_Workers_AI"
export FIREWORKS_API_KEY="$Fireworks_AI"
export BASETEN_API_KEY="$Baseten"
export NOVITA_API_KEY="$Novita_AI"
export UPSTAGE_API_KEY="$Upstage_AI"
export NLP_API_KEY="$NLP_Cloud"
export MODAL_API_KEY="$Modal_Token_Secret"
export MODAL_API_KEY_ID="$Modal_Token_ID"
export INFERENCE_API_KEY="$Inference"
export HYPERBOLIC_API_KEY="$Hyperbolic"
export SAMBANOVA_API_KEY="$SambaNova_AI"
export REPLICATE_API_KEY="$Replicate"
export SARVAM_API_KEY="$Sarvam_AI_India"
export VULAVULA_API_KEY="$Vulavula"

# Additional providers that might be expected
export VERTEX_API_KEY="$Gemini"
export GROQ_API_KEY="$HuggingFace"
export TOGETHER_API_KEY="$HuggingFace"
export PERPLEXITY_API_KEY="$HuggingFace"
export MISTRAL_AI_API_KEY="$Mistral_AiStudio"

# Generate the ultimate configuration
echo "🔥 Generating OpenCode configuration with all providers..."
./bin/llm-verifier ai-config export opencode ultimate_opencode_config.json

# Validate the configuration
echo "🔍 Validating configuration..."
./bin/llm-verifier ai-config validate ultimate_opencode_config.json

echo "✅ Ultimate OpenCode configuration generated!"
echo "📁 File: ultimate_opencode_config.json"
echo "🔒 Remember: This file contains embedded API keys - DO NOT COMMIT!"