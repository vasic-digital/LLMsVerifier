#!/bin/bash
# CLEAN_WORKING_DIRECTORY.sh
# Removes secrets from current working directory files

set -e

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║  CLEAN WORKING DIRECTORY - Remove Secrets                    ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# Count files that will be affected
echo "Scanning for files with secrets..."

ENV_COUNT=$(find . -name ".env" -type f 2>/dev/null | wc -l)
LOG_COUNT=$(find challenges/ -name "*.log" -type f 2>/dev/null | wc -l)
JSON_COUNT=$(find results/ -name "*.json" -type f 2>/dev/null | wc -l)

echo "  - .env files found: $ENV_COUNT"
echo "  - Challenge log files: $LOG_COUNT"
echo "  - Results JSON files: $JSON_COUNT"
echo ""

# Remove .env files
echo "Step 1: Removing .env files..."
find . -name ".env" -type f -exec rm -vf {} \;
find . -name ".env.local" -type f -exec rm -vf {} \;
find . -name "*.env.backup" -type f -exec rm -vf {} \;
echo "✅ .env files removed"
echo ""

# Clean challenge logs (redact secrets but keep files)
echo "Step 2: Redacting secrets from challenge logs..."
if [ -d "challenges" ]; then
    find challenges/ -name "*.log" -type f -print0 | while IFS= read -r -d '' file; do
        # Only process if file contains secrets
        if grep -qE "(hf_[a-zA-Z0-9]{34}|sk-[a-zA-Z0-9]{20,}|r8_[a-zA-Z0-9]{30,}|nvapi-[a-zA-Z0-9]{80,}|cpk_[a-zA-Z0-9]{90,})" "$file" 2>/dev/null; then
            sed -i 's/hf_[a-zA-Z0-9]\{34\}/hf_***REDACTED***/g' "$file" 2>/dev/null || true
            sed -i 's/sk-[a-zA-Z0-9]\{20,\\}/sk-***REDACTED***/g' "$file" 2>/dev/null || true
            sed -i 's/r8_[a-zA-Z0-9]\{30,\\}/r8_***REDACTED***/g' "$file" 2>/dev/null || true
            sed -i 's/nvapi-[a-zA-Z0-9]\{80,\\}/nvapi-***REDACTED***/g' "$file" 2>/dev/null || true
            sed -i 's/cpk_[a-zA-Z0-9]\{90,\\}/cpk_***REDACTED***/g' "$file" 2>/dev/null || true
            echo "  ✅ Cleaned: $file"
        fi
    done
    echo "✅ Challenge logs redacted"
else
    echo "  No challenges directory found"
fi
echo ""

# Clean results JSON files
echo "Step 3: Redacting secrets from results JSON files..."
if [ -d "results" ]; then
    find results/ -name "*.json" -type f -print0 | while IFS= read -r -d '' file; do
        # Only process if file contains secrets
        if grep -qE "(hf_[a-zA-Z0-9]{34}|sk-[a-zA-Z0-9]{20,}|r8_[a-zA-Z0-9]{30,}|nvapi-[a-zA-Z0-9]{80,}|cpk_[a-zA-Z0-9]{90,})" "$file" 2>/dev/null; then
            sed -i 's/hf_[a-zA-Z0-9]\{34\}/hf_***REDACTED***/g' "$file" 2>/dev/null || true
            sed -i 's/sk-[a-zA-Z0-9]\{20,\\}/sk-***REDACTED***/g' "$file" 2>/dev/null || true
            sed -i 's/r8_[a-zA-Z0-9]\{30,\\}/r8_***REDACTED***/g' "$file" 2>/dev/null || true
            sed -i 's/nvapi-[a-Z0-9]\{80,\\}/nvapi-***REDACTED***/g' "$file" 2>/dev/null || true
            sed -i 's/cpk_[a-zA-Z0-9]\{90,\\}/cpk_***REDACTED***/g' "$file" 2>/dev/null || true
            echo "  ✅ Cleaned: $file"
        fi
    done
    echo "✅ Results JSON files redacted"
else
    echo "  No results directory found"
fi
echo ""

# Remove or clean Go files with test keys
echo "Step 4: Cleaning Go test files..."
find . -name "*.go" -type f -exec grep -l 'REDACTED_API_KEY' {} \; 2>/dev/null | while read -r file; do
    if [[ "$file" != *"/cmd/"* ]]; then
        sed -i 's/REDACTED_API_KEY/"${DEEPSEEK_API_KEY}"/g' "$file" 2>/dev/null || true
        echo "  ✅ Cleaned: $file"
    fi
done
echo ""

# Verify
echo "Step 5: Verification..."
REMAINING=$(grep -rE "(hf_[a-zA-Z0-9]{34}|sk-[a-zA-Z0-9]{20,}|r8_[a-zA-Z0-9]{30,}|nvapi-[a-zA-Z0-9]{80,}|cpk_[a-zA-Z0-9]{90,})" \
  --exclude-dir=.git \
  --exclude-dir=node_modules \
  --exclude="*.sh" \
  --exclude="*.md" \
  . 2>/dev/null | wc -l)

echo "  Remaining secret instances: $REMAINING"

if [ "$REMAINING" -eq 0 ]; then
    echo "✅ Working directory is clean"
else
    echo "⚠️  $REMAINING instances may remain in excluded files"
fi

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║  WORKING DIRECTORY CLEANUP COMPLETE                          ║"
echo "╚══════════════════════════════════════════════════════════════╝"