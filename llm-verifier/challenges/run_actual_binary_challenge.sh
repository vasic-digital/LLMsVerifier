#!/bin/bash
# Provider Models Discovery Challenge - Actual Binary Execution
# Uses ONLY project's binary - llm-verifier

set -e

CHALLENGE_NAME="provider_models_discovery"
TIMESTAMP=$(date +%s)
YEAR=$(date +%Y)
MONTH=$(date +%m)
DAY=$(date +%d)
DATETIME=$(date +"%Y-%m-%d %H:%M:%S")

CHALLENGE_DIR="challenges/$CHALLENGE_NAME/$YEAR/$MONTH/$DAY/$TIMESTAMP"
LOG_DIR="$CHALLENGE_DIR/logs"
RESULTS_DIR="$CHALLENGE_DIR/results"

mkdir -p "$LOG_DIR"
mkdir -p "$RESULTS_DIR"

LOG_FILE="$LOG_DIR/challenge.log"
CMD_LOG_FILE="$LOG_DIR/commands.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

log_cmd() {
    echo "$(date '+%Y-%m-%d %H:%M:%S')] COMMAND: $*" | tee -a "$CMD_LOG_FILE"
}

log "========================================"
log "PROVIDER MODELS DISCOVERY CHALLENGE (ACTUAL BINARY)"
log "========================================"
log ""
log "Challenge Directory: $CHALLENGE_DIR"
log "Timestamp: $DATETIME"
log "Binary: $(pwd)/llm-verifier"
log ""

# Create config.yaml with proper API endpoints
log "========================================"
log "CREATING CONFIGURATION"
log "========================================"
log ""

CONFIG_FILE="$CHALLENGE_DIR/config.yaml"

cat > "$CONFIG_FILE" << EOF
# LLM Verifier Configuration - Provider Models Discovery
# Based on provider API documentation

global:
  base_url: "https://api.openai.com/v1"
  max_retries: 3
  request_delay: 1s
  timeout: 30s

database:
  path: "./llm-verifier.db"

api:
  port: "8080"
  enable_cors: true

llms:
  # HuggingFace
  - name: "HuggingFace"
    endpoint: "https://api-inference.huggingface.co"
    api_key: "hf_*****"
    model: "gpt2"
    features:
      tool_calling: false
      embeddings: true
      streaming: false
      vision: false
      code_generation: true

  # Nvidia
  - name: "Nvidia"
    endpoint: "https://integrate.api.nvidia.com/v1"
    api_key: "nvapi-*****"
    model: "nvidia/nemotron-4-340b"
    features:
      tool_calling: true
      embeddings: false
      streaming: true
      vision: true
      code_generation: true

  # Chutes
  - name: "Chutes"
    endpoint: "https://api.chutes.ai/v1/chat/completions"
    api_key: "cpk_*****"
    model: "gpt-4"
    features:
      tool_calling: true
      embeddings: false
      streaming: true
      vision: true
      code_generation: true

  # SiliconFlow
  - name: "SiliconFlow"
    endpoint: "https://api.siliconflow.cn/v1"
    api_key: "sk-*****"
    model: "Qwen/Qwen2-72B-Instruct"
    features:
      tool_calling: true
      embeddings: false
      streaming: true
      vision: false
      code_generation: true

  # Kimi (Moonshot AI)
  - name: "Kimi"
    endpoint: "https://api.moonshot.cn/v1"
    api_key: "sk-kimi-*****"
    model: "moonshot-v1-128k"
    features:
      tool_calling: true
      embeddings: false
      streaming: true
      vision: false
      code_generation: true

  # Gemini
  - name: "Gemini"
    endpoint: "https://generativelanguage.googleapis.com/v1"
    api_key: "AIzaSy*****"
    model: "gemini-2.0-flash-exp"
    features:
      tool_calling: true
      embeddings: false
      streaming: true
      vision: true
      code_generation: true

  # OpenRouter
  - name: "OpenRouter"
    endpoint: "https://openrouter.ai/api/v1/chat/completions"
    api_key: "sk-or-v1-*****"
    model: "anthropic/claude-3.5-sonnet"
    features:
      tool_calling: true
      embeddings: false
      streaming: true
      vision: true
      code_generation: true

  # Z.AI
  - name: "Z.AI"
    endpoint: "https://api.z.ai/v1/chat/completions"
    api_key: "a977c8417a45457a83a897de82e4215b.*****"
    model: "zai-large"
    features:
      tool_calling: false
      embeddings: false
      streaming: true
      vision: false
      code_generation: false

  # DeepSeek
  - name: "DeepSeek"
    endpoint: "https://api.deepseek.com"
    api_key: "sk-*****"
    model: "deepseek-chat"
    features:
      tool_calling: true
      embeddings: false
      streaming: true
      vision: false
      code_generation: true

concurrency: 5
timeout: 60s
output:
  directory: "$RESULTS_DIR"
  format: "json"
EOF

log "Configuration file created: $CONFIG_FILE"
log ""

# Execute binary - ACTUAL execution, not simulation
log "========================================"
log "RUNNING BINARY - ACTUAL EXECUTION"
log "========================================"
log ""

BINARY="$(pwd)/llm-verifier"

# Log and execute binary command
log "Executing binary to discover and verify providers..."
CMD="$BINARY -c $CONFIG_FILE -o $RESULTS_DIR"
log_cmd "$CMD"

# Run binary
log "Binary output:"
log "=========================================="
$CMD 2>&1 | tee -a "$LOG_FILE"
log "=========================================="
log ""

log "========================================"
log "GENERATING RESULTS SUMMARY"
log "========================================"
log ""

# Count lines in challenge.log
log_lines=$(wc -l < "$LOG_FILE" | awk '{print $1}')
cmd_lines=$(wc -l < "$CMD_LOG_FILE" | awk '{print $1}')

log "Challenge Summary:"
log "  Challenge Directory: $CHALLENGE_DIR"
log "  Configuration File: $CONFIG_FILE"
log "  Log File: $LOG_FILE ($log_lines lines)"
log "  Commands Log: $CMD_LOG_FILE ($cmd_lines commands)"
log "  Results Directory: $RESULTS_DIR"
log ""

log "========================================"
log "CHALLENGE COMPLETE"
log "========================================"
log ""
log "Results and logs saved in: $CHALLENGE_DIR"
log ""

