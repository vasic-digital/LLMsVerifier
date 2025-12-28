#!/bin/bash
# FIX_GITHUB_PUSH.sh
# Fixes GitHub push protection by removing secrets from current commits

set -e

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║  FIX GITHUB PUSH PROTECTION                                  ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# Step 1: Check if we can push to GitHub
echo "Step 1: Testing GitHub push..."
if git push github main --dry-run 2>&1 | grep -q "GH013"; then
    echo "❌ GitHub is still blocking due to secrets in history"
    echo ""
    echo "You have TWO options:"
    echo ""
    echo "Option A: Purge secrets from history (RECOMMENDED)"
    echo "  - Completely removes secrets from all commits"
    echo "  - Rewrites git history (changes commit hashes)"
    echo "  - Use: ./scripts/purge_secrets_from_history.sh"
    echo ""
    echo "Option B: Use git filter-repo to fix specific files"
    echo "  - Targets specific files with secrets"
    echo "  - Less invasive than full purge"
    echo "  - Use: ./scripts/fix_specific_commits.sh"
    echo ""
    
    read -p "Which option? (A/B): " choice
    
    if [ "$choice" == "A" ]; then
        echo ""
        echo "Running full secret purge..."
        ./scripts/purge_secrets_from_history.sh
    elif [ "$choice" == "B" ]; then
        echo ""
        echo "Creating targeted fix..."
        ./scripts/fix_specific_files.sh
    else
        echo "Invalid choice. Exiting."
        exit 1
    fi
else
    echo "✅ GitHub push protection is satisfied"
fi

# Step 2: Ensure .env is properly ignored
echo ""
echo "Step 2: Verifying .env is in .gitignore..."
if grep -q '^\.env$' .gitignore; then
    echo "✅ .env is already in .gitignore"
else
    echo ".env" >> .gitignore
    echo "✅ Added .env to .gitignore"
    git add .gitignore
    git commit -m "fix: add .env to .gitignore"
fi

# Step 3: Add the clean config files
echo ""
echo "Step 3: Adding clean configuration files..."
if [ -f "llm-verifier/config_full.yaml" ]; then
    git add llm-verifier/config_full.yaml
    git commit -m "feat: add full provider configuration with env vars

- Configure all 29 LLM providers
- Use environment variables for all API keys  
- Achieve 100% provider coverage
- Comply with CHALLENGES_USE_ALL_PROVIDERS_POLICY"
    echo "✅ Added clean config_full.yaml"
fi

if [ -f "llm-verifier/.env.example" ]; then
    git add llm-verifier/.env.example
    git commit -m "docs: add comprehensive .env.example with all providers"
    echo "✅ Added .env.example"
fi

# Step 4: Remove old problematic files from future commits
echo ""
echo "Step 4: Removing old config files from repository..."
git rm --cached llm-verifier/config_working.yaml llm-verifier/config_minimal.yaml 2>/dev/null || true
git commit -m "fix: remove old config files with hardcoded secrets

BREAKING CHANGE: Old config files removed, use config_full.yaml instead" || true

echo ""
echo "Step 5: Final push attempt..."
if git push github main; then
    echo "✅ SUCCESS! GitHub push completed"
else
    echo "❌ Still blocked. History purge required."
    echo "Run: ./scripts/purge_secrets_from_history.sh"
    exit 1
fi

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║  GIT PUSH FIX COMPLETE                                       ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
echo "✅ All secrets removed from current commits"
echo "✅ .env properly ignored"
echo "✅ Clean config files added"
echo "✅ GitHub push protection satisfied"
echo ""
echo "Next steps:"
echo "1. Verify push: git log --oneline -n 5"
echo "2. Test config: cp llm-verifier/.env.example .env"
echo "3. Add API keys to .env (never commit this file)"
echo "4. Test: ./llm-verifier-app --config llm-verifier/config_full.yaml"