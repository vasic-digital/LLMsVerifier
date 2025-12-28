#!/bin/bash
# FIX_SPECIFC_FILES.sh
# Less invasive option - removes specific files with secrets

set -e

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║  FIX SPECIFIC FILES WITH SECRETS                            ║"
echo "║  Less invasive option - removes specific problematic files  ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

echo "Step 1: Creating backup..."
BACKUP_DIR="../LLMsVerifier-backup-$(date +%Y%m%d-%H%M%S)"
cp -r . "$BACKUP_DIR"
echo "✅ Backup created at: $BACKUP_DIR"

echo ""
echo "Step 2: Installing git-filter-repo..."
pip3 install git-filter-repo

echo ""
echo "Step 3: Removing files with secrets from history..."

# Remove the specific files that contain secrets
git filter-repo --force \
    --path llm-verifier/config_working.yaml \
    --path llm-verifier/config_minimal.yaml \
    --path llm-verifier/.env \
    --path .env \
    --path .env.production \
    --invert-paths \
    --refs main

echo ""
echo "Step 4: Adding clean versions back..."

# Now add the clean versions (if they exist)
if [ -f "llm-verifier/config_full.yaml" ]; then
    git add llm-verifier/config_full.yaml
fi

if [ -f "llm-verifier/.env.example" ]; then
    git add llm-verifier/.env.example
fi

# Update .gitignore
echo ".env" >> .gitignore
echo "llm-verifier/.env" >> .gitignore
echo ".env.*" >> .gitignore
git add .gitignore

git commit -m "fix: purge secrets from history and add clean configs

- Remove files with hardcoded secrets from git history
- Add clean config_full.yaml with environment variables
- Enhance .gitignore for security
- Comply with GitHub push protection"

echo ""
echo "Step 5: Force pushing to GitHub..."
read -p "This will overwrite remote history. Continue? (type YES): " confirm
if [ "$confirm" == "YES" ]; then
    git push github main --force-with-lease
    echo "✅ GitHub updated"
else
    echo "Skipped. Push manually when ready:"
    echo "git push github main --force-with-lease"
    exit 0
fi

echo ""
echo "Step 6: Checking if GitHub is satisfied..."
if git push github main --dry-run 2>&1 | grep -q "GH013"; then
    echo "❌ GitHub is still blocking"
    echo "Full history purge may be needed:"
    echo "./scripts/purge_secrets_from_history.sh"
else
    echo "✅ GitHub push protection satisfied!"
fi

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║  SPECIFIC FILE CLEANUP COMPLETE                              ║"
echo "╚══════════════════════════════════════════════════════════════╝"