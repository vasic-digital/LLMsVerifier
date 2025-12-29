#!/bin/bash

set -e

echo "🔧 Building Fixed Ultimate Challenge Binary..."

# Navigate to the llm-verifier directory
cd /media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier/llm-verifier

# Copy the fixed files to the appropriate locations
echo "📁 Copying fixed files..."
cp ../fixed_model_provider_service.go ./providers/
cp ../fixed_model_verification_service.go ./providers/
cp ../fixed_enhanced_model_provider_service.go ./providers/
cp ../fixed_ultimate_challenge.go ./cmd/fixed-ultimate-challenge/

# Build the fixed ultimate challenge binary
echo "🏗️  Building fixed binary..."
cd cmd/fixed-ultimate-challenge
go build -o ../../../fixed-ultimate-challenge .

echo "✅ Fixed binary built successfully!"

# Navigate back to project root
cd /media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier

# Set up environment variables for testing
echo "🔑 Setting up environment variables..."
export HUGGINGFACE_API_KEY="hf_test_key"
export GEMINI_API_KEY="gemini_test_key"
export DEEPSEEK_API_KEY="deepseek_test_key"
export NVIDIA_API_KEY="nvidia_test_key"
export OPENROUTER_API_KEY="openrouter_test_key"
export REPLICATE_API_KEY="replicate_test_key"
export FIREWORKS_API_KEY="fireworks_test_key"
export MISTRAL_API_KEY="mistral_test_key"
export CODESTRAL_API_KEY="codestral_test_key"
export CLOUDFLARE_API_KEY="cloudflare_test_key"
export SAMBANOVA_API_KEY="sambanova_test_key"
export CEREBRAS_API_KEY="cerebras_test_key"
export MODAL_API_KEY="modal_test_key"
export INFERENCE_API_KEY="inference_test_key"
export SILICONFLOW_API_KEY="siliconflow_test_key"
export NOVITA_API_KEY="novita_test_key"
export UPSTAGE_API_KEY="upstage_test_key"
export NLP_API_KEY="nlp_test_key"
export HYPERBOLIC_API_KEY="hyperbolic_test_key"
export CHUTES_API_KEY="chutes_test_key"
export KIMI_API_KEY="kimi_test_key"

# Run the fixed ultimate challenge
echo "🚀 Running Fixed Ultimate Challenge..."
echo "This should discover models from 30+ providers with ~1000 models..."
echo

./fixed-ultimate-challenge 2>&1 | tee fixed_ultimate_challenge.log

echo
echo "📊 Analysis of results:"
echo "======================="

# Count providers and models from the log
if [ -f "fixed_ultimate_challenge.log" ]; then
    echo "📈 Results Summary:"
    grep -E "✓ Registered [0-9]+ providers" fixed_ultimate_challenge.log || echo "❌ Provider registration count not found"
    grep -E "✅ Total: [0-9]+ providers, [0-9]+ models discovered, [0-9]+ verified" fixed_ultimate_challenge.log || echo "❌ Final summary not found"
    
    echo
echo "🔍 Provider Details:"
    grep -E "Testing .+\.\.\. ✓ Found [0-9]+ verified models" fixed_ultimate_challenge.log | head -10 || echo "❌ Provider details not found"
    
    echo
    echo "📁 Output Files:"
    ls -la /home/milosvasic/Downloads/fixed_opencode.json 2>/dev/null && echo "✅ Fixed OpenCode config generated"
    ls -la /home/milosvasic/.config/opencode/opencode.json 2>/dev/null && echo "✅ Config copied to standard location"
    
    echo
    echo "📏 Configuration Size:"
    if [ -f "/home/milosvasic/Downloads/fixed_opencode.json" ]; then
        size=$(wc -c < /home/milosvasic/Downloads/fixed_opencode.json)
        echo "Size: $size bytes ($(echo "scale=2; $size/1024" | bc -l) KB)"
    fi
else
    echo "❌ Log file not found - challenge may have failed"
fi

echo
echo "🔧 Build and test complete! Check the results above."