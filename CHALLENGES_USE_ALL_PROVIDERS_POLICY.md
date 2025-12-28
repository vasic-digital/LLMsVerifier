# 📋 CHALLENGES MUST USE ALL PROVIDERS POLICY

## 🔒 **MANDATE: 100% Provider Coverage**

**Effective Immediately**: ALL challenges in the LLM Verifier project **MUST** use 100% of available providers. No exceptions.

---

## 📊 **BACKGROUND: THE DISCREPANCY**

### What Was Discovered
- **27 API Keys** available across multiple providers
- **6 Providers** actually tested (22%)
- **21 Providers** never configured (78%)
- **Violation of core principle**: "Challenges must use all resources"

### Affected Projects
- `config_working.yaml` - Only 6 providers
- `ultimate_opencode_config.json` - Only 6 providers
- Challenge test suite - Only validated subset

---

## 🎯 **NEW REQUIREMENTS**

### 1. **100% Provider Coverage**
```yaml
# BEFORE (VIOLATION):
llms:
  - name: "HuggingFace"  # Only 6 providers
  - name: "Nvidia"
  - name: "DeepSeek"
  - name: "Groq"
  - name: "OpenRouter"
  - name: "Replicate"

# AFTER (CORRECT):
llms:
  - name: "HuggingFace"  # All 27 providers configured
  - name: "Nvidia"
  - name: "DeepSeek"
  - name: "Groq"
  - name: "OpenRouter"
  - name: "Replicate"
  - name: "Anthropic"
  - name: "OpenAI"
  - name: "Google/Gemini"
  - name: "Perplexity"
  - name: "Together AI"
  - name: "Cerebras"
  - name: "SambaNova AI"
  - name: "Fireworks AI"
  - name: "Mistral AI"
  - name: "Codestral"
  - name: "Kimi (Moonshot)"
  - name: "Cloudflare Workers AI"
  - name: "Modal"
  - name: "Chutes"
  - name: "SiliconFlow"
  - name: "Novita AI"
  - name: "Upstage AI"
  - name: "NLP Cloud"
  - name: "Hyperbolic"
  - name: "ZAI"
  - name: "Baseten"
  - name: "TwelveLabs"
```

---

## 🔑 **ALL 27 PROVIDERS THAT MUST BE CONFIGURED**

### Tier 1: Major AI Providers (6)
1. **HuggingFace** - hf_***REDACTED***
2. **NVIDIA** - REDACTED_API_KEY
3. **DeepSeek** - REDACTED_API_KEY
4. **Groq** - gsk_test_placeholder_key_for_groq_provider
5. **Replicate** - r8_***REDACTED***
6. **OpenRouter** - REDACTED_API_KEY

### Tier 2: Commercial Providers (8)
7. **Anthropic** - Required for Claude models
8. **OpenAI** - GPT-4, GPT-3.5 series
9. **Google/Gemini** - Gemini Pro, Flash, Ultra
10. **Mistral AI** - Mistral, Mixtral models
11. **Codestral** - Code generation models
12. **Kimi (Moonshot)** - Chinese market leader
13. **Together AI** - Open source model hosting
14. **Perplexity** - Search + AI

### Tier 3: Specialized Platforms (6)
15. **Cerebras** - Hardware-accelerated AI
16. **SambaNova AI** - Enterprise AI
17. **Fireworks AI** - Model serving platform
18. **Modal** - Serverless model deployment
19. **Cloudflare Workers AI** - Edge AI inference
20. **Baseten** - ML model deployment

### Tier 4: Regional & Emerging (7)
21. **Chutes** - Decentralized AI
22. **SiliconFlow** - Chinese AI platform
23. **Novita AI** - Affordable inference
24. **Upstage AI** - Korean AI leader
25. **NLP Cloud** - Specialized NLP
26. **Hyperbolic** - Decentralized compute
27. **ZAI** - AI model marketplace
28. **TwelveLabs** - Video understanding
29. **Inference** - AI inference platform

---

## ✅ **CONFIGURATION FILES TO UPDATE**

### 1. `config_full.yaml` (NEW - Master Config)
**Location**: `llm-verifier/config_full.yaml`
**Status**: ✅ **CREATED** with all 27 providers
**Action**: Use this as primary configuration

### 2. `ultimate_opencode_config_FULL.json` (NEW)
**Location**: `ultimate_opencode_config_FULL.json`
**Status**: ✅ **GENERATED** with all providers
**Action**: Use for OpenCode schema validation

### 3. `.env.example` (UPDATED)
**Location**: `llm-verifier/.env.example`
**Status**: ✅ **ENHANCED** with all provider variables
**Action**: Copy to `.env` and add all keys

### 4. Challenge Templates (TO UPDATE)
**Location**: `challenges/codebase/`
**Action**: Regenerate all challenge configs to use full provider list

---

## 🛡️ **VALIDATION RULES**

### Pre-Commit Hook (REQUIRED)
```bash
#!/bin/bash
# .git/hooks/pre-commit

# Check provider coverage
python3 scripts/validate_provider_coverage.py

if [ $? -ne 0 ]; then
    echo "❌ PROVIDER VALIDATION FAILED"
    echo "Configuration does not include all 27 providers"
    exit 1
fi
```

### CI/CD Pipeline Check
```yaml
# .github/workflows/verify-providers.yml
name: Verify 100% Provider Coverage

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Validate all providers configured
        run: |
          python scripts/validate_provider_coverage.py --strict
          echo "✅ All 27 providers configured correctly"
```

---

## 📋 **MANDATORY VALIDATION SCRIPT**

Create `scripts/validate_provider_coverage.py`:

```python
#!/usr/bin/env python3
"""
Validator: Ensures ALL challenges use ALL available providers
"""

import yaml
import json
import sys

REQUIRED_PROVIDERS = 27  # MUST match available API keys

def validate_config(config_path, config_type):
    """Validate configuration has all providers"""
    if config_type == 'yaml':
        with open(config_path) as f:
            config = yaml.safe_load(f)
        providers = [p['name'] for p in config.get('llms', [])]
    elif config_type == 'json':
        with open(config_path) as f:
            config = json.load(f)
        providers = list(config.get('provider', {}).keys())
    
    actual = len(providers)
    if actual < REQUIRED_PROVIDERS:
        print(f"❌ {config_path}: Only {actual}/{REQUIRED_PROVIDERS} providers")
        missing = get_missing_providers(providers)
        print(f"   Missing: {missing}")
        return False
    
    print(f"✅ {config_path}: {actual}/{REQUIRED_PROVIDERS} providers")
    return True

def get_missing_providers(configured):
    """Get list of missing providers"""
    all_providers = {
        'HuggingFace', 'Nvidia', 'DeepSeek', 'Groq', 'OpenRouter', 'Replicate',
        'Anthropic', 'OpenAI', 'Google', 'Gemini', 'Perplexity', 'Together AI',
        'Cerebras', 'SambaNova AI', 'Fireworks AI', 'Mistral AI', 'Codestral',
        'Kimi', 'Cloudflare Workers AI', 'Modal', 'Chutes', 'SiliconFlow',
        'Novita AI', 'Upstage AI', 'NLP Cloud', 'Hyperbolic', 'ZAI', 'Baseten',
        'TwelveLabs'
    }
    return sorted(all_providers - set(configured))

if __name__ == '__main__':
    results = []
    
    # Validate all configs
    results.append(validate_config('llm-verifier/config_full.yaml', 'yaml'))
    results.append(validate_config('ultimate_opencode_config_FULL.json', 'json'))
    
    if all(results):
        print(f"\n🎉 ALL VALIDATIONS PASSED - {REQUIRED_PROVIDERS} providers configured")
        sys.exit(0)
    else:
        print(f"\n❌ VALIDATION FAILED - Not all providers configured")
        sys.exit(1)
```

---

## 🚨 **ENFORCEMENT**

### Git Hooks (MANDATORY)
```bash
cd /media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier

# Install pre-commit hook
ln -sf scripts/validate_provider_coverage.py .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit

# Verify it works
python3 scripts/validate_provider_coverage.py
```

### Review Checklist
Every PR must include:
- [ ] All 27 providers configured in config files
- [ ] New providers added to `.env.example`
- [ ] Validation script passes
- [ ] Documentation updated with new providers
- [ ] API key rotation if adding new providers

---

## 💰 **COST OPTIMIZATION**

### Free Tier Providers (Test These First)
- Groq - Unlimited free tier
- Fireworks AI - $5 free credit
- Together AI - $5 free credit
- Replicate - Pay per second
- Modal - Free tier available
- Cloudflare Workers AI - Free tier
- Baseten - Free tier
- Cerebras - Free tier
- SambaNova AI - Free tier

### Rotation Strategy
Rotate through providers to maximize free tier usage:
```bash
# Day 1-3: Use Groq (free)
# Day 4-5: Use Fireworks + Together (free credits)
# Day 6-7: Use Cloudflare + Modal (free tier)
# Week 2: Rotate to premium for specific tests
```

---

## 📚 **DOCUMENTATION**

### Update These Files
- ✅ `GITHUB_PUSH_RESOLUTION.md` - Add provider coverage section
- ✅ `SECURITY_SETUP.md` - Document all 27 providers
- 🔄 `README.md` - Add provider count badge
- 🔄 `PROVIDER_DISCREPANCY_REPORT.md` - Document resolution
- 🔄 `CHALLENGES_FINAL_STATUS.md` - Update to 27 providers

---

## 🔐 **SECURITY REMINDER**

### All 27 API Keys Must Be:
1. **Stored in `.env` ONLY** (NEVER commit)
2. **Referenced as `${VARIABLE}`** in configs
3. **Rotated immediately** if exposed
4. **Documented in `.env.example`** with placeholders
5. **Validated in CI/CD** before deployment

### Exposed Keys (Must Rotate ALL 27)
Every key in `.env` was hardcoded and exposed. Rotate ALL:
- HuggingFace token (hf_...)
- NVIDIA API key (nvapi-...)
- Replicate token (r8_...)
- DeepSeek key (sk-...)
- And all 23 others...

---

## ✅ **ACCEPTANCE CRITERIA**

Project is compliant when:

1. ✅ **Configuration Files**: All use 27/27 providers
2. ✅ **Environment Variables**: All 27 mapped correctly
3. ✅ **Validation Script**: Passes in CI/CD
4. ✅ **Documentation**: All 27 providers documented
5. ✅ **Security**: All keys rotated and using env vars
6. ✅ **Git Hooks**: Pre-commit validation active
7. ✅ **Challenges**: Execute across all providers
8. ✅ **Reports**: Show 100% provider coverage

---

## 📞 **ESCALATION**

### If You Cannot Add a Provider
1. **Rotate the API key** (security)
2. **Mark as DISABLED** in comments
3. **Document why** in `PROVIDER_DISCREPANCY_REPORT.md`
4. **Create ticket** to re-enable when ready

Example:
```yaml
# - name: "ProblemProvider"  # DISABLED: API endpoint unstable
#   api_key: ${PROBLEM_API_KEY}
#   endpoint: https://api.problem.com/v1
```

---

**Policy Created**: 2025-12-28
**Policy Version**: 1.0
**Enforcement**: IMMEDIATE
**Compliance Deadline**: 2025-12-30
**Owner**: LLM Verifier Core Team