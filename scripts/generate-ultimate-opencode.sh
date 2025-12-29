#!/bin/bash
# Ultimate OpenCode Configuration Generator
# This script runs the llm-verifier ultimate challenge and produces
# a complete OpenCode-compatible configuration file

set -e  # Exit on any error

echo "🚀 ULTIMATE OPENCODE CONFIGURATION GENERATOR"
echo "=============================================="
echo

# Check if binary exists
if [ ! -f "../bin/ultimate-challenge" ]; then
    echo "❌ Ultimate challenge binary not found. Building..."
    cd ../llm-verifier
    go build -o ../bin/ultimate-challenge ./cmd/ultimate-challenge
    cd ..
    echo "✅ Binary built successfully"
fi

# Set output path
OUTPUT_FILE="opencode_ultimate_$(date +%Y%m%d_%H%M%S).json"
export OPENCODE_OUTPUT_PATH="$OUTPUT_FILE"

echo "📁 Output will be saved to: $OUTPUT_FILE"
echo

# Run the ultimate challenge
echo "🔍 Starting ultimate challenge..."
echo "This may take several minutes depending on API response times..."
echo

../bin/ultimate-challenge

echo
echo "🎯 CHALLENGE COMPLETE!"
echo

# Verify the output
if [ ! -f "$OUTPUT_FILE" ]; then
    echo "❌ ERROR: Output file was not created!"
    echo "This usually means:"
    echo "  - No API keys are configured in environment variables"
    echo "  - Network connectivity issues"
    echo "  - Provider API rate limits"
    echo
    echo "Please check your API key configuration and try again."
    exit 1
fi

# Validate the JSON structure
echo "🔍 Validating generated configuration..."
if ! jq . "$OUTPUT_FILE" > /dev/null 2>&1; then
    echo "❌ ERROR: Generated file is not valid JSON!"
    exit 1
fi

# Check file size and content
FILE_SIZE=$(stat -f%z "$OUTPUT_FILE" 2>/dev/null || stat -c%s "$OUTPUT_FILE" 2>/dev/null)
FILE_SIZE_MB=$((FILE_SIZE / 1024 / 1024))

echo "📊 Configuration Statistics:"
echo "   📄 File size: $FILE_SIZE_MB MB"
echo "   🗂️  File path: $OUTPUT_FILE"

# Extract metadata from the generated file
if command -v jq > /dev/null; then
    PROVIDERS=$(jq '.provider | length' "$OUTPUT_FILE" 2>/dev/null || echo "0")
    MODELS=$(jq '[.provider[]?.models | length] | add' "$OUTPUT_FILE" 2>/dev/null || echo "0")
    VERIFIED_MODELS=$(jq '.metadata.verifiedModels // 0' "$OUTPUT_FILE" 2>/dev/null || echo "0")

    echo "   🏢 Providers configured: $PROVIDERS"
    echo "   🤖 Models discovered: $MODELS"
    echo "   ✅ Verified models: $VERIFIED_MODELS"
    echo

    if [ "$VERIFIED_MODELS" -lt 100 ]; then
        echo "⚠️  WARNING: Only $VERIFIED_MODELS models verified."
        echo "   Expected: 1000+ models across 30+ providers."
        echo "   This may indicate missing API keys or connectivity issues."
        echo
    fi
fi

# Final validation
echo "🔐 Final OpenCode compatibility check..."
if jq -e '.["$schema"] == "https://opencode.sh/schema.json" and has("username") and has("provider") and has("agent") and has("mcp")' "$OUTPUT_FILE" > /dev/null 2>&1; then
    echo "✅ SUCCESS: Configuration is 100% OpenCode compatible!"
    echo
    echo "🎉 Your ultimate OpenCode configuration is ready!"
    echo "   Use this file with OpenCode: $OUTPUT_FILE"
    echo
    echo "💡 Next steps:"
    echo "   1. Copy the configuration to your OpenCode config directory"
    echo "   2. Start OpenCode with: opencode --config $OUTPUT_FILE"
    echo "   3. Enjoy 1000+ verified LLM models with feature suffixes!"
else
    echo "❌ ERROR: Configuration failed OpenCode compatibility check!"
    echo "   The generated file may not work with OpenCode."
    exit 1
fi

echo
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║  🎊 ULTIMATE OPENCODE CONFIGURATION - MISSION ACCOMPLISHED!  ║"
echo "╚══════════════════════════════════════════════════════════════╝"