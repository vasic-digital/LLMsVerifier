# GitHub Push Protection Resolution Guide

## Problem
GitHub's push protection detected and blocked commits containing API tokens:
- Hugging Face token: `${HUGGINGFACE_API_KEY}`
- Replicate API token: `${REPLICATE_API_KEY}lG`

## Solution Applied

✅ **Fixed configuration files:**
- `llm-verifier/config_working.yaml` - Replaced all hardcoded API keys with environment variables
- `llm-verifier/config_minimal.yaml` - Replaced hardcoded keys with environment variables
- `llm-verifier/.env.example` - Created template for required environment variables
- `llm-verifier/SECURITY_SETUP.md` - Added comprehensive security documentation
- `llm-verifier/.gitignore` - Added additional security exclusions

## Quick Start to Resolve GitHub Push Issue

### Option 1: Clean the Commit (Recommended)

If you haven't pushed the secrets yet:

```bash
cd /media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier

# 1. Reset the problematic commit (if exists)
git reset HEAD~1  # Adjust number as needed

# 2. Add the clean configuration
git add llm-verifier/config_working.yaml

# 3. Add the .env.example template
git add llm-verifier/.env.example

# 4. Commit with security message
git commit -m 'fix: remove hardcoded secrets, use environment variables

- Replace hardcoded API tokens with ${ENV_VAR} placeholders
- Add .env.example template for configuration
- Update SECURITY_SETUP.md documentation
- Fixes GitHub push protection warnings'

# 5. Push to GitHub
git push github main  # or your branch name
```

### Option 2: If Secrets Were Already Committed

If secrets exist in previous commits, clean the history:

```bash
# Check if secrets exist in history
git log -p --all -S 'hf_eSWSEHRcCy' -- llm-verifier/config_working.yaml

# If found, use git filter-repo to clean history
git filter-repo --force \
  --replace-text <(cat <<'EOF'
${HUGGINGFACE_API_KEY}==>${HUGGINGFACE_API_KEY}
${REPLICATE_API_KEY}lG==>${REPLICATE_API_KEY}
${DEEPSEEK_API_KEY}==>${DEEPSEEK_API_KEY}
REDACTED_API_KEY==>${NVIDIA_API_KEY}
EOF
) --force

# Force push to all remotes
git push --force --all
git push --force --tags
```

## Setup Local Development Environment

After fixing the GitHub push, set up your environment:

```bash
cd /media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier/llm-verifier

# 1. Copy the example file
cp .env.example .env

# 2. Add your real API keys to .env
echo 'HUGGINGFACE_API_KEY=your_actual_huggingface_token_here' >> .env
echo 'REPLICATE_API_KEY=your_actual_replicate_token_here' >> .env
echo 'DEEPSEEK_API_KEY=your_actual_deepseek_key' >> .env
echo 'NVIDIA_API_KEY=your_actual_nvidia_key' >> .env
# ... add all other keys

# 3. Source the variables
source .env

# 4. Test the application
../llm-verifier-app --config config_working.yaml

# 5. Verify environment is loaded
echo $HUGGINGFACE_API_KEY | awk '{print "Token length: " length($0), "chars"}'
```

## Verify Fix Before Pushing

```bash
# Check for remaining hardcoded secrets
grep -rn "api_key: [a-zA-Z0-9_\-]\{20,\}" llm-verifier/ --include="*.yaml" --include="*.yml"

# Should return nothing! If it does, update those files too.

# Check for common token patterns
grep -rn "hf_" llm-verifier/config*.yaml  # Hugging Face tokens
grep -rn "r8_" llm-verifier/config*.yaml  # Replicate tokens
grep -rn "sk-" llm-verifier/config*.yaml  # Various API keys

# Check if .env is tracked (should NOT be)
git ls-files | grep -E "\.env$"
# Should return nothing!
```

## Platform-Specific Instructions

### Push to Other Remotes (Already Working)

GitLab, GitFlic, and GitVerse are not blocking, so your existing workflow continues:

```bash
git push gitlab
git push gitflic
git push gitverse
```

### GitHub Protected Push

For GitHub only, you must use environment variables:

```bash
# Add and push to GitHub only when secrets are removed
git add llm-verifier/config*.yaml
git commit -m "fix: secure configuration files"
git push github
```

## Troubleshooting

### GitHub Still Blocking Push

If GitHub still detects secrets:

```bash
# 1. Check entire repository for the pattern
git log --all -p --source --all -S 'hf_eSWSEHRcCy'

# 2. Clean specific files from git history
git filter-repo --path llm-verifier/config_working.yaml --invert-paths --force

# 3. Re-add the clean file
git add llm-verifier/config_working.yaml
git commit -m "Add clean config file"

# 4. Force push
git push --force github main
```

### False Positive Detection

If GitHub detects a false positive:

```bash
# Option 1: Use GitHub's UI to bypass for this push
# (Look for "Bypass protection" option)

# Option 2: Use git push with force after confirming it's safe
git push github main --force-with-lease
```

### Recover Original Tokens

If you need to recover the original tokens (for rotation):

```bashn# Check git reflog if commit was recent
git reflog | grep -i "secret\|config"

# Check backup if exists
git show HEAD@{1}:llm-verifier/config_working.yaml > config_backup.yaml
```

## Security Checklist

Before pushing to GitHub, verify:

- [ ] No hardcoded API keys in any config files
- [ ] All API keys use `${VARIABLE_NAME}` syntax
- [ ] `.env` file is in `.gitignore`
- [ ] `.env.example` exists with all required variables documented
- [ ] Committed files don't contain actual secrets
- [ ] Git history cleaned if secrets were committed
- [ ] Application correctly loads environment variables
- [ ] Tested with `source .env && ./llm-verifier-app`

## Rotating Compromised Tokens

Since tokens were exposed, rotate them immediately:

1. **Hugging Face:**
   - Go to https://huggingface.co/settings/tokens
   - Delete old token: `${HUGGINGFACE_API_KEY}`
   - Generate new token
   - Update your `.env` file

2. **Replicate:**
   - Go to https://replicate.com/account
   - Delete old token: `${REPLICATE_API_KEY}lG`
   - Generate new API token
   - Update your `.env` file

3. **Other tokens** in config_minimal.yaml:
   - DeepSeek: https://platform.deepseek.com/api_keys
   - NVIDIA: https://build.nvidia.com/nim

## Additional Resources

- See `llm-verifier/SECURITY_SETUP.md` for detailed environment setup
- Review `.env.example` for all required variables
- Check GitHub's secret scanning documentation: https://docs.github.com/en/code-security/secret-scanning

## Support

If issues persist:
1. Review GitHub secret scanning alerts in repository Security tab
2. Check detailed verification output above
3. Contact repository maintainers with error details