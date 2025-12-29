#!/bin/bash
# Git History Cleanup Script for Removing Secrets (Non-interactive version)
# This script uses git filter-repo to remove sensitive files from git history

echo "🚨 CRITICAL SECURITY OPERATION: Git History Cleanup"
echo "=================================================="
echo "WARNING: This script will rewrite git history and remove sensitive files."
echo "This operation cannot be undone and will require force-pushing to all remotes."
echo "All team members will need to re-clone the repository."
echo ""

# Check if git-filter-repo is installed
if ! command -v git-filter-repo &> /dev/null && ! python3 -m git_filter_repo --help &> /dev/null; then
    echo "❌ git-filter-repo is not installed."
    echo "Installing..."
    pip install --break-system-packages git-filter-repo
fi

echo "🔍 Checking current status..."
git status --short

echo ""
echo "📋 Removing sensitive files from git history..."

# List of files to remove from history
FILES_TO_REMOVE=(
    "ultimate_opencode_complete.json"
    "ultimate_opencode_final.json"
    "ultimate_opencode_with_models.json"
    "test_brotli_discovery_brotli_optimized_crush_config.json"
    "test_brotli_discovery_brotli_optimized_opencode_config.json"
    "test_brotli_discovery.json"
    "llm-verifier/challenges/provider_models_discovery/2025/12/29/SUCCESS/1767004245/results/llm_verification_report.json"
    "opencode_complete_real_llmverifier.json"
    "opencode_complete_suffixes_llmverifier.json"
    "opencode_exact_llmverifier.json"
)

# Create a temporary file with the list of files to remove
TEMP_FILE=$(mktemp)
for file in "${FILES_TO_REMOVE[@]}"; do
    echo "$file" >> "$TEMP_FILE"
done

echo "Files to remove from history:"
cat "$TEMP_FILE"
echo ""

# Run git filter-repo
echo "Running git filter-repo..."
if git filter-repo --invert-paths --paths-from-file "$TEMP_FILE" --force; then
    echo "✅ Git history has been cleaned successfully."
else
    echo "❌ Failed to clean git history."
    rm "$TEMP_FILE"
    exit 1
fi

# Clean up
rm "$TEMP_FILE"

echo ""
echo "🔐 SECURITY STATUS:"
echo "- Files with API keys removed from git history"
echo "- Repository is now secure"
echo ""
echo "📤 NEXT STEPS:"
echo "1. Force push to all remotes: git push --force --all"
echo "2. Rotate any API keys that were in the removed files"
echo "3. Inform all team members to re-clone the repository"
echo ""
echo "⚠️  WARNING: This operation has permanently altered git history."