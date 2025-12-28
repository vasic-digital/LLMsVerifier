# Security Setup Guide - Environment Variables

## Overview
This guide explains how to securely configure API keys using environment variables to avoid committing secrets to GitHub.

## Step 1: Create Environment File

```bash
cp llm-verifier/.env.example llm-verifier/.env
```

## Step 2: Add Your API Keys

Edit `llm-verifier/.env` and replace the placeholder values with your actual API keys:

```bash
# Required API Keys
DEEPSEEK_API_KEY=your_actual_deepseek_key_here
NVIDIA_API_KEY=your_actual_nvidia_key_here
HUGGINGFACE_API_KEY=your_actual_huggingface_token_here
GROQ_API_KEY=your_actual_groq_key_here
OPENROUTER_API_KEY=your_actual_openrouter_key_here
REPLICATE_API_KEY=your_actual_replicate_key_here
```

**IMPORTANT**: Never commit the `.env` file to Git. It's already in `.gitignore`.

## Step 3: Load Environment Variables

```bash
# Method 1: Source the file (Unix/Linux/macOS)
source llm-verifier/.env

# Method 2: Use a dotenv package (cross-platform)
# The application should auto-load .env if dotenv is configured

# Method 3: Export manually
export HUGGINGFACE_API_KEY="your_actual_token_here"
```

## Step 4: Verify Configuration

```bash
# Check that variables are set
echo $HUGGINGFACE_API_KEY | sed 's/./•/g'  # Shows masked version

# Test the application
./llm-verifier-app --config llm-verifier/config_working.yaml
```

## Environment Variables Reference

| Variable | Provider | Required? | Description |
|----------|----------|-----------|-------------|
| `DEEPSEEK_API_KEY` | DeepSeek | Yes | DeepSeek API access token |
| `NVIDIA_API_KEY` | NVIDIA | Yes | NVIDIA NIM API key |
| `HUGGINGFACE_API_KEY` | Hugging Face | Yes | Hugging Face token (prefixed with `hf_`) |
| `GROQ_API_KEY` | Groq | Yes | Groq API key |
| `OPENROUTER_API_KEY` | OpenRouter | Yes | OpenRouter API key |
| `REPLICATE_API_TOKEN` | Replicate | Yes | Replicate API token (referenced in GitHub push) |

## Previous Secret Cleanup

If you already committed secrets, you need to clean Git history:

### Option A: Remove Entire File from History

```bash
git filter-repo --force --path llm-verifier/config_working.yaml --invert-paths
```

### Option B: Replace Secrets in History

```bash
# Create a replacement file
cat > replacements.txt <<EOF
${HUGGINGFACE_API_KEY}==>${HUGGINGFACE_API_KEY}
${REPLICATE_API_KEY}lG==>${REPLICATE_API_KEY}
EOF

# Apply replacements
git filter-repo --replace-text replacements.txt --force
```

### Option C: BFG Repo Cleaner (Easier)

```bash
# Install BFG
brew install bfg  # macOS
# Or download from https://rtyley.github.io/bfg-repo-cleaner/

# Clean secrets
bfg --replace-text secrets.txt your-repo.git
```

## Update All Config Files

Run this to check for other config files with secrets:

```bash
# Check for hardcoded secrets
find . -name "config*.yaml" -exec grep -l "api_key: [a-zA-Z0-9_\-]\{20,\}" {} \;

# Replace with environment variables
find . -name "config*.yaml" -exec sed -i 's/api_key: [a-zA-Z0-9_\-]\{20,\}/api_key: ${PROVIDER_API_KEY}/g' {} \;
```

## Best Practices

1. ✅ **Never** commit `.env` files
2. ✅ Use `.env.example` with placeholders
3. ✅ Use environment-specific config files (`.env.production`, `.env.staging`)
4. ✅ Rotate compromised API keys immediately
5. ✅ Use secret management services (AWS Secrets Manager, Azure Key Vault)
6. ✅ Regularly scan for secrets in code with tools like:
   - `truffleHog`
   - `git-secrets`
   - GitHub secret scanning

## Troubleshooting

### "api_key is empty" error
- Ensure you've set the environment variables
- Check variable names match exactly (case-sensitive)
- Verify `.env` file is in the correct location

### "config file not found"
- Check path to config file is correct
- Ensure config file uses `${VARIABLE_NAME}` syntax

### GitHub still blocking push
- Check entire commit history: `git log -p --all`
- Verify `.env` is in `.gitignore`
- Clean repository with filter-repo if needed

## Support

If you continue to have issues with GitHub push protection:
1. Check GitHub's secret scanning alerts in repository settings
2. Contact GitHub support for false positives
3. Consider using GitHub's Codespaces secrets feature for development

## Security Contact

For security issues, please:
1. Rotate compromised API keys immediately
2. Check GitHub security advisories
3. Contact the security team if available