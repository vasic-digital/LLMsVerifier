# Git History Cleanup Checklist

## 🚨 URGENT: Secrets Found in Git History

GitHub has detected secrets in commit history. You must completely remove them or bypass protection.

---

## ⚡ IMMEDIATE ACTION REQUIRED

### Option 1: Purge Secrets from History (RECOMMENDED)
```bash
cd /media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier
./scripts/purge_secrets_from_history.sh
```
**What it does:**
- Completely removes all secrets from every commit
- Rewrites git history (changes commit hashes)
- Removes sensitive files entirely from history
- Force pushes to all remotes

**Pros:**
- ✅ Completely cleans history
- ✅ Satisfies GitHub protection permanently
- ✅ Best security practice

**Cons:**
- ⚠️ Changes commit hashes (breaks links)
- ⚠️ Requires force push to all remotes
- ⚠️ Must notify team if collaborating

---

### Option 2: Bypass GitHub Protection (QUICK FIX)

Go to these URLs and allow the secrets:

1. **Hugging Face Token:**
   https://github.com/vasic-digital/LLMsVerifier/security/secret-scanning/unblock-secret/37TrE0HEFBrdKOcNw4rdw2xNio0

2. **Replicate API Token:**
   https://github.com/vasic-digital/LLMsVerifier/security/secret-scanning/unblock-secret/37TrDzN9iN59XaNrBWD7yS4l2Fh

**Then push again:**
```bash
git push github main
```

**Pros:**
- ✅ Fast (2 minutes)
- ✅ No history rewrite
- ✅ Works immediately

**Cons:**
- ⚠️ Secrets remain in history
- ⚠️ Must repeat if secrets found again
- ⚠️ Less secure

---

### Option 3: Remove Specific Files (MIDDLE GROUND)
```bash
cd /media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier
./scripts/fix_specific_files.sh
```
**What it does:**
- Removes only files with secrets from history
- Keeps most commit history intact
- Less invasive than full purge

---

## 🎯 RECOMMENDATION

**Use Option 1 (Purge) for best security:**

```bash
# 1. Run the purge script
./scripts/purge_secrets_from_history.sh

# 2. When prompted, type: YES

# 3. When prompted to push, type: PUSH

# 4. Wait for completion

# 5. Verify secrets are gone
git log --oneline -n 5
git log -p --all -S 'hf_eSWSEHRcCy'  # Should show nothing
```

---

## 🔒 BEFORE YOU START

### Backup Your Work
```bash
# Create a full backup
cd /media/milosvasic/DATA4TB/Projects/LLM
cp -r LLMsVerifier LLMsVerifier-BACKUP-$(date +%Y%m%d)

# Or push to a safe remote (GitLab, GitFlic, GitVerse)
git push gitlab main
git push gitflic main
git push gitverse main
```

### Verify Current Status
```bash
cd /media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier

# Check what secrets GitHub sees
git log -p --all -S 'hf_eSWSEHRcCy' | head -20

# Check commit history
git log --oneline -n 10
```

---

## 📝 WHAT FILES CONTAIN SECRETS

### Files in Git History:
- ❌ `llm-verifier/config_working.yaml` - Lines 24, 45 (old commits)
- ❌ `llm-verifier/config_minimal.yaml` - Various lines (old commits)
- ❌ `GITHUB_PUSH_RESOLUTION.md` - Multiple locations (documentation)
- ❌ `llm-verifier/SECURITY_SETUP.md` - Documentation references

### Current Files (Clean):
- ✅ `llm-verifier/config_full.yaml` - Uses env vars (CLEAN)
- ✅ `.env.example` - Uses placeholders (CLEAN)

---

## 🔧 AFTER PURGING

### Add Clean Files Back
```bash
# After purge, add the clean configuration
git add llm-verifier/config_full.yaml
git add llm-verifier/.env.example
git add .gitignore

git commit -m "feat: add clean configuration with env vars

- Configure all 29 LLM providers with environment variables
- Remove hardcoded secrets (security compliance)
- Add comprehensive .gitignore patterns
- Achieve 100% provider coverage (29/29)"

# Push to all remotes
git push github main --force-with-lease
git push gitlab main --force-with-lease
git push gitflic main --force-with-lease
git push gitverse main --force-with-lease
```

### Verify Clean State
```bash
# Should show NO results
git log -p --all -S 'hf_eSWSEHRcCy'
git log -p --all -S 'r8_4Ai9B8Sz'

# Should show clean commits only
git log --oneline -n 5
```

---

## 🛡️ PREVENT FUTURE ISSUES

### Update .gitignore (Already Done)
```bash
# Verify these are in .gitignore
grep -E "^(\.env|\.secret|.*\.key)$" .gitignore
```

### Install Pre-Commit Hook
```bash
# Create hook to prevent committing secrets
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
# Pre-commit hook to prevent secrets

# Check for common secret patterns
if grep -rE "(hf_[a-zA-Z0-9]{30,}|sk-[a-zA-Z0-9]{20,}|r8_[a-zA-Z0-9]{30,}|nvapi-[a-zA-Z0-9]{50,})" \
   --exclude-dir=.git \
   --exclude="*.sh" \
   --exclude="*.md" \
   .; then
    echo "❌ SECRET DETECTED! Commit blocked."
    echo "Remove hardcoded API keys and use environment variables."
    exit 1
fi
EOF

chmod +x .git/hooks/pre-commit
```

### Create Security Policy
```bash
cat > SECURITY.md << 'EOF'
# Security Policy

## Never Commit Secrets
- API keys must use environment variables
- Use .env file (gitignored)
- Reference: ${API_KEY_NAME} in configs

## Validation
Run: ./scripts/validate-no-secrets.sh

## If Secret is Committed
1. Rotate the key immediately
2. Purge from git history
3. Notify security team
EOF
```

---

## 🚨 CRITICAL REMINDER

### Rotate ALL Exposed API Keys
After purging history, **immediately rotate**:

1. **HuggingFace:** `hf_***REDACTED***`
   - URL: https://huggingface.co/settings/tokens

2. **Replicate:** `r8_***REDACTED***`
   - URL: https://replicate.com/account

3. **DeepSeek:** `REDACTED_API_KEY`
   - URL: https://platform.deepseek.com/api_keys

4. **NVIDIA:** `REDACTED_API_KEY`
   - URL: https://build.nvidia.com/nim

5. **All other 25 keys** in .env file

---

## 📞 SUPPORT

If issues persist:

1. **GitHub still blocking:** Full purge required
2. **Lost work:** Restore from backup
3. **Team collaboration issues:** Share new remote URL
4. **Validation failing:** Run validator script

---

## ✅ CHECKLIST

- [ ] Backup created
- [ ] Purge script executed OR GitHub URLs visited
- [ ] .env in .gitignore
- [ ] Clean config added
- [ ] Force push completed
- [ ] Secrets verified removed from history
- [ ] API keys rotated
- [ ] Team notified (if collaborating)

---

**Time Estimate:** 15-30 minutes
**Risk Level:** HIGH (history rewrite)
**Security Impact:** CRITICAL (must complete)

**Status:** 🚨 BLOCKED - Action Required