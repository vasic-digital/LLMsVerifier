#!/bin/bash
# Security Audit Script
# Checks for potential secrets and security issues in the codebase

echo "🔒 LLMsVerifier Security Audit"
echo "================================"
echo

# Check for common secret patterns
echo "Checking for potential secrets in files..."

# API Keys
echo "API Keys found:"
grep -r -i "sk-\|pk_\|api_key\|apikey\|secret" --include="*.go" --include="*.py" --include="*.json" --include="*.yaml" --include="*.yml" . 2>/dev/null | grep -v ".git" | grep -v "__pycache__" | head -10

echo
echo "Potential token patterns:"
grep -r -i "hf_\|r8_\|xoxp-\|xoxb-" --include="*.go" --include="*.py" --include="*.json" . 2>/dev/null | grep -v ".git" | head -5

echo
echo "Environment files:"
find . -name ".env*" -type f | head -5

echo
echo "Checking .gitignore coverage..."

# Check if sensitive files are gitignored
SENSITIVE_FILES=(
    "opencode.json"
    "opencode_with_keys.json"
    ".env"
    ".env.local"
)

echo "Checking if sensitive files are properly ignored:"
for file in "${SENSITIVE_FILES[@]}"; do
    if git check-ignore "$file" 2>/dev/null; then
        echo "✅ $file is properly ignored"
    else
        echo "❌ $file is NOT ignored - this is a security risk!"
    fi
done

echo
echo "Git history check (last 10 commits):"
echo "Files that might contain secrets:"
git log --name-only -10 | grep -E "\.(json|yaml|yml)$" | grep -E "(opencode|config|key|secret)" | head -5

echo
echo "Security recommendations:"
echo "1. Never commit API keys or secrets to git history"
echo "2. Use .env files for local development"
echo "3. Rotate any keys that may have been exposed"
echo "4. Run this audit script regularly"
echo "5. Enable secret scanning on your GitHub repository"

echo
echo "Audit completed."