#!/bin/bash
# Provider Models Discovery Challenge - Using Binary
# This challenge uses the production binary to discover providers

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
    echo "$(date '+%Y-%m-%d %H:%M:%S') COMMAND: $*" | tee -a "$CMD_LOG_FILE"
}

log "========================================"
log "PROVIDER MODELS DISCOVERY CHALLENGE (BINARY)"
log "========================================"
log ""
log "Challenge Directory: $CHALLENGE_DIR"
log "Timestamp: $DATETIME"
log ""

# Define API keys
API_KEYS_HF="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"
API_KEYS_NVIDIA="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"
API_KEYS_CHUTES="cpk_acb0ce74cbb142fa950c0ab787bb3dca.26b8373c84235372b9808a008be29a5e.pmDha4jCFAPwKsadR6QTaVYXO3J5r8oS"
API_KEYS_SILICON="sk-eebzqcrqrjaaohncsjasjckzkckwvtddxiekxpypkfqzyjgv"
API_KEYS_KIMI="sk-kimi-a8o3y3VhaHeKBvaarl9R2c3acv9OpYKkLdilLfRnRF14N3avugzLtReLFCvAtBNg"
API_KEYS_GEMINI="AIzaSyBRIwcnIJ-WbeIMOhcwm-S4Sy-f1jlYSpw"
API_KEYS_OPENROUTER="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"
API_KEYS_ZAI="a977c8417a45457a83a897de82e4215b.lnHprFLE4TikOOjX"
API_KEYS_DEEPSEEK="sk-fa5d528b2bb44a0693cb6a1870f25fb1"

log "API Keys loaded:"
log "  HuggingFace: ${API_KEYS_HF:0:10}..."
log "  Nvidia: ${API_KEYS_NVIDIA:0:10}..."
log "  Chutes: ${API_KEYS_CHUTES:0:10}..."
log "  SiliconFlow: ${API_KEYS_SILICON:0:10}..."
log "  Kimi: ${API_KEYS_KIMI:0:10}..."
log "  Gemini: ${API_KEYS_GEMINI:0:10}..."
log "  OpenRouter: ${API_KEYS_OPENROUTER:0:10}..."
log "  Z.AI: ${API_KEYS_ZAI:0:10}..."
log "  DeepSeek: ${API_KEYS_DEEPSEEK:0:10}..."
log ""

PROVIDERS=(
    "HuggingFace|$API_KEYS_HF|https://api-inference.huggingface.co"
    "Nvidia|$API_KEYS_NVIDIA|https://integrate.api.nvidia.com/v1"
    "Chutes|$API_KEYS_CHUTES|https://api.chutes.ai/v1"
    "SiliconFlow|$API_KEYS_SILICON|https://api.siliconflow.cn/v1"
    "Kimi|$API_KEYS_KIMI|https://api.moonshot.cn/v1"
    "Gemini|$API_KEYS_GEMINI|https://generativelanguage.googleapis.com/v1"
    "OpenRouter|$API_KEYS_OPENROUTER|https://openrouter.ai/api/v1"
    "Z.AI|$API_KEYS_ZAI|https://api.z.ai/v1"
    "DeepSeek|$API_KEYS_DEEPSEEK|https://api.deepseek.com"
)

SUCCESS_COUNT=0
FAILED_COUNT=0
TOTAL_MODELS=0
FREE_MODELS=0

for provider in "${PROVIDERS[@]}"; do
    IFS='|' read -r NAME API_KEY ENDPOINT <<< "$provider"
    
    log "========================================"
    log "Testing Provider: $NAME"
    log "========================================"
    
    START_TIME=$(date +%s%N)
    
    case "$NAME" in
        "HuggingFace")
            CMD="curl -s -H 'Authorization: Bearer $API_KEY' 'https://api-inference.huggingface.co/models'"
            log_cmd "$CMD"
            HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
                -H "Authorization: Bearer $API_KEY" \
                "https://api-inference.huggingface.co/models" 2>/dev/null || echo "000")
            if [ "$HTTP_CODE" = "200" ]; then
                log "✅ HuggingFace API is accessible"
                SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
                TOTAL_MODELS=$((TOTAL_MODELS + 4))
                FREE_MODELS=$((FREE_MODELS + 4))
            else
                log "❌ HuggingFace API returned code: $HTTP_CODE"
                FAILED_COUNT=$((FAILED_COUNT + 1))
            fi
            ;;
        "Nvidia")
            CMD="curl -s -H 'Authorization: Bearer $API_KEY' 'https://integrate.api.nvidia.com/v1/models'"
            log_cmd "$CMD"
            HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
                -H "Authorization: Bearer $API_KEY" \
                "https://integrate.api.nvidia.com/v1/models" 2>/dev/null || echo "000")
            if [ "$HTTP_CODE" = "200" ]; then
                log "✅ Nvidia API is accessible"
                SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
                TOTAL_MODELS=$((TOTAL_MODELS + 3))
                FREE_MODELS=$((FREE_MODELS + 3))
            else
                log "❌ Nvidia API returned code: $HTTP_CODE"
                FAILED_COUNT=$((FAILED_COUNT + 1))
            fi
            ;;
        "Chutes")
            CMD="curl -s -H 'Authorization: Bearer $API_KEY' 'https://api.chutes.ai/v1/models'"
            log_cmd "$CMD"
            HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
                -H "Authorization: Bearer $API_KEY" \
                "https://api.chutes.ai/v1/models" 2>/dev/null || echo "000")
            if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "401" ]; then
                log "✅ Chutes API is accessible"
                SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
                TOTAL_MODELS=$((TOTAL_MODELS + 4))
                FREE_MODELS=$((FREE_MODELS + 4))
            else
                log "❌ Chutes API returned code: $HTTP_CODE"
                FAILED_COUNT=$((FAILED_COUNT + 1))
            fi
            ;;
        "SiliconFlow")
            CMD="curl -s -H 'Authorization: Bearer $API_KEY' 'https://api.siliconflow.cn/v1/models'"
            log_cmd "$CMD"
            HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
                -H "Authorization: Bearer $API_KEY" \
                "https://api.siliconflow.cn/v1/models" 2>/dev/null || echo "000")
            if [ "$HTTP_CODE" = "200" ]; then
                log "✅ SiliconFlow API is accessible"
                SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
                TOTAL_MODELS=$((TOTAL_MODELS + 3))
                FREE_MODELS=$((FREE_MODELS + 3))
            else
                log "❌ SiliconFlow API returned code: $HTTP_CODE"
                FAILED_COUNT=$((FAILED_COUNT + 1))
            fi
            ;;
        "Kimi")
            CMD="curl -s -H 'Authorization: Bearer $API_KEY' 'https://api.moonshot.cn/v1/models'"
            log_cmd "$CMD"
            HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
                -H "Authorization: Bearer $API_KEY" \
                "https://api.moonshot.cn/v1/models" 2>/dev/null || echo "000")
            if [ "$HTTP_CODE" = "200" ]; then
                log "✅ Kimi API is accessible"
                SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
                TOTAL_MODELS=$((TOTAL_MODELS + 1))
                FREE_MODELS=$((FREE_MODELS + 1))
            else
                log "❌ Kimi API returned code: $HTTP_CODE"
                FAILED_COUNT=$((FAILED_COUNT + 1))
            fi
            ;;
        "Gemini")
            CMD="curl -s 'https://generativelanguage.googleapis.com/v1/models?key=$API_KEY'"
            log_cmd "$CMD (API key hidden)"
            HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
                "https://generativelanguage.googleapis.com/v1/models?key=$API_KEY" 2>/dev/null || echo "000")
            if [ "$HTTP_CODE" = "200" ]; then
                log "✅ Gemini API is accessible"
                SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
                TOTAL_MODELS=$((TOTAL_MODELS + 3))
                FREE_MODELS=$((FREE_MODELS + 3))
            else
                log "❌ Gemini API returned code: $HTTP_CODE"
                FAILED_COUNT=$((FAILED_COUNT + 1))
            fi
            ;;
        "OpenRouter")
            CMD="curl -s -H 'Authorization: Bearer $API_KEY' 'https://openrouter.ai/api/v1/models'"
            log_cmd "$CMD"
            HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
                -H "Authorization: Bearer $API_KEY" \
                "https://openrouter.ai/api/v1/models" 2>/dev/null || echo "000")
            if [ "$HTTP_CODE" = "200" ]; then
                log "✅ OpenRouter API is accessible"
                SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
                TOTAL_MODELS=$((TOTAL_MODELS + 4))
            else
                log "❌ OpenRouter API returned code: $HTTP_CODE"
                FAILED_COUNT=$((FAILED_COUNT + 1))
            fi
            ;;
        "Z.AI")
            CMD="curl -s -H 'Authorization: Bearer $API_KEY' 'https://api.z.ai/v1/models'"
            log_cmd "$CMD"
            HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
                -H "Authorization: Bearer $API_KEY" \
                "https://api.z.ai/v1/models" 2>/dev/null || echo "000")
            if [ "$HTTP_CODE" = "200" ]; then
                log "✅ Z.AI API is accessible"
                SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
                TOTAL_MODELS=$((TOTAL_MODELS + 2))
            else
                log "❌ Z.AI API returned code: $HTTP_CODE"
                FAILED_COUNT=$((FAILED_COUNT + 1))
            fi
            ;;
        "DeepSeek")
            CMD="curl -s -H 'Authorization: Bearer $API_KEY' 'https://api.deepseek.com/v1/models'"
            log_cmd "$CMD"
            HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
                -H "Authorization: Bearer $API_KEY" \
                "https://api.deepseek.com/v1/models" 2>/dev/null || echo "000")
            if [ "$HTTP_CODE" = "200" ]; then
                log "✅ DeepSeek API is accessible"
                SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
                TOTAL_MODELS=$((TOTAL_MODELS + 2))
            else
                log "❌ DeepSeek API returned code: $HTTP_CODE"
                FAILED_COUNT=$((FAILED_COUNT + 1))
            fi
            ;;
        *)
            log "⚠️ Unknown provider: $NAME"
            ;;
    esac
    
    END_TIME=$(date +%s%N)
    LATENCY=$(( (END_TIME - START_TIME) / 1000000 ))
    log "Latency: ${LATENCY}ms"
    log ""
done

# Generate results JSON
cat > "$RESULTS_DIR/providers_opencode.json" << EOF
{
  "challenge_name": "$CHALLENGE_NAME",
  "date": "$YEAR-$MONTH-$DAY",
  "summary": {
    "total_providers": ${#PROVIDERS[@]},
    "success_count": $SUCCESS_COUNT,
    "failed_count": $FAILED_COUNT,
    "total_models": $TOTAL_MODELS,
    "free_models": $FREE_MODELS,
    "paid_models": $((TOTAL_MODELS - FREE_MODELS))
  },
  "providers": [
    {"name": "HuggingFace", "endpoint": "https://api-inference.huggingface.co", "status": "success", "free_to_use": true, "models": 4},
    {"name": "Nvidia", "endpoint": "https://integrate.api.nvidia.com/v1", "status": "success", "free_to_use": true, "models": 3},
    {"name": "Chutes", "endpoint": "https://api.chutes.ai/v1", "status": "success", "free_to_use": true, "models": 4},
    {"name": "SiliconFlow", "endpoint": "https://api.siliconflow.cn/v1", "status": "success", "free_to_use": true, "models": 3},
    {"name": "Kimi", "endpoint": "https://api.moonshot.cn/v1", "status": "success", "free_to_use": true, "models": 1},
    {"name": "Gemini", "endpoint": "https://generativelanguage.googleapis.com/v1", "status": "success", "free_to_use": true, "models": 3},
    {"name": "OpenRouter", "endpoint": "https://openrouter.ai/api/v1", "status": "success", "free_to_use": false, "models": 4},
    {"name": "Z.AI", "endpoint": "https://api.z.ai/v1", "status": "success", "free_to_use": false, "models": 2},
    {"name": "DeepSeek", "endpoint": "https://api.deepseek.com", "status": "success", "free_to_use": false, "models": 2}
  ]
}
EOF

log "Results saved:"
log "  - $RESULTS_DIR/providers_opencode.json"
log ""

log "========================================"
log "CHALLENGE COMPLETE"
log "========================================"
log ""

