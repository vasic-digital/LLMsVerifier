#!/bin/bash
# Git History Cleanup Script for Removing Secrets
# This script uses git filter-repo to remove sensitive files from git history

echo "WARNING: This script will rewrite git history and remove sensitive files."
echo "Make sure all team members have pulled the latest changes and understand"
echo "that this will require force-pushing to all remotes."
echo ""
read -p "Are you sure you want to continue? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Operation cancelled."
    exit 1
fi

# Check if git-filter-repo is installed
if ! command -v git-filter-repo &> /dev/null; then
    echo "git-filter-repo is not installed."
    echo "Please install it first:"
    echo "  pip install git-filter-repo"
    echo "  # or"
    echo "  pipx install git-filter-repo"
    exit 1
fi

echo "Removing sensitive files from git history..."

# List of files to remove from history
FILES_TO_REMOVE=(
    "ultimate_opencode_complete.json"
    "ultimate_opencode_final.json"
    "ultimate_opencode_with_models.json"
    "test_brotli_discovery_brotli_optimized_crush_config.json"
    "test_brotli_discovery_brotli_optimized_opencode_config.json"
    "test_brotli_discovery.json"
    "llm-verifier/challenges/provider_models_discovery/2025/12/29/SUCCESS/1767004245/results/llm_verification_report.json"
)

# Create a temporary file with the list of files to remove
TEMP_FILE=$(mktemp)
for file in "${FILES_TO_REMOVE[@]}"; do
    echo "$file" >> "$TEMP_FILE"
done

# Run git filter-repo
git filter-repo --invert-paths --paths-from-file "$TEMP_FILE"

# Clean up
rm "$TEMP_FILE"

echo "Git history has been cleaned."
echo "Now you need to force push to all remotes:"
echo "  git push --force --all"
echo ""
echo "WARNING: This will require all team members to re-clone the repository"
echo "or reset their local branches."