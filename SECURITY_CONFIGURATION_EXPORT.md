# LLM Verifier - Security & Configuration Export Guide

## 🔐 Security-First Configuration Export System

The LLM Verifier uses a **security-first approach** for exporting configuration files with embedded API keys. This system ensures that sensitive credentials are NEVER accidentally committed to version control.

---

## 🛡️ Protection Mechanisms

### 1. **.gitignore Protection**

The `.gitignore` file includes comprehensive patterns to prevent committing sensitive files:

```
# API keys and credentials patterns
**/*api_key*
**/*secret*
**/*credential*
**/*token*

# Specific configuration files
opencode.json
opencode_*_with_keys.json
**/*opencode*keys*.json

# User Downloads directory
Downloads/opencode*.json
Downloads/*_api_keys.json
```

**Status:** ✅ **CONFIGURED** - All sensitive file patterns are protected

### 2. **Automatic File Permissions**

All exported configuration files receive restrictive permissions:
- **Mode: 600** (owner read/write only)
- **Group: None**
- **Other: None**

This prevents unauthorized access by other users on the system.

---

## 📦 Export System

### **The Official Export Tool**

**Location:** `scripts/export_opencode_config_fixed.py`

**Features:**
- ✅ Automatic security validation
- ✅ Gitignore protection verification
- ✅ Secure file permissions (600)
- ✅ Comprehensive API key embedding
- ✅ Feature detection (MCP, LCP, ACP, Embeddings)
- ✅ Performance metrics included
- ✅ Warning messages displayed

---

## 🚀 How to Export Configurations

### **Standard Export (Recommended)**

```bash
cd /media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier
python3 scripts/export_opencode_config_fixed.py
```

This will:
1. Load latest verification results
2. Extract all API keys from `.env`
3. Generate complete OpenCode configuration
4. Save to `~/Downloads/opencode_{timestamp}.json`
5. Set permissions to 600
6. Display security warnings

### **Custom Export**  

```bash
# Specify custom output location
python3 scripts/export_opencode_config_fixed.py \
  --output /path/to/safe/location/opencode.json

# Use specific verification results
python3 scripts/export_opencode_config_fixed.py \
  --verification challenges/full_verification/2025/12/28/143700/results/providers_export.json

# Use specific .env file  
python3 scripts/export_opencode_config_fixed.py \
  --env /path/to/secrets/.env
```

### **Validate Gitignore Protection**

```bash
python3 scripts/export_opencode_config_fixed.py --validate-only
```

This checks if the `.gitignore` file has proper security protections.

---

## 📊 What Gets Exported

### **Complete Configuration Includes:**

1. **All 25+ Providers** with:
   - ✅ API Keys (embedded)
   - ✅ Endpoints
   - ✅ Health status
   - ✅ Verification results

2. **All 40+ Models** with:
   - ✅ Verification status
   - ✅ Performance metrics (response time, TTFT)
   - ✅ Feature capabilities:
     - MCP (Model Capability Protocol)
     - LSP (Language Server Protocol)
     - ACP (AI Coding Protocol)
     - Embeddings
     - Streaming
     - Tool/Function Calling
     - Vision
     - Code generation
   - ✅ Comprehensive scoring

3. **MCP Servers** configured for ACP-enabled models

4. **Model Groups** for easy selection:
   - `acp_enabled` - Models supporting AI Coding Protocol
   - `lsp_enabled` - Models supporting Language Server Protocol
   - `mcp_enabled` - Models with Model Capability Protocol
   - `coding` - Code-capable models
   - `high_performance` - Models with score 75+

5. **ACP/LSP Configuration** with all capabilities enabled

---

## 🔒 Security Best Practices

### **DO ✅**

1. **Always use the official export script**
   ```bash
   python3 scripts/export_opencode_config_fixed.py
   ```

2. **Keep exported files in secure locations**
   - Default: `~/Downloads/` (already gitignored)
   - Alternative: Encrypted volumes or secure directories

3. **Set restrictive permissions manually if needed**
   ```bash
   chmod 600 opencode.json
   ```

4. **Validate gitignore before exporting**
   ```bash
   python3 scripts/export_opencode_config_fixed.py --validate-only
   ```

5. **Review exported files for sensitive data**
   ```bash
   grep -i "api_key" opencode.json | wc -l  # Should show embedded keys
   ```

6. **Use environment variables when possible**
   The export script can generate configs with env var references instead of embedded keys:
   - Use `--no-embed-keys` flag (if implemented)

### **DO NOT ❌**

1. **NEVER commit opencode.json to Git**
   ```bash
   # This will be blocked by .gitignore
   git add opencode.json  # Will be ignored
   ```

2. **NEVER share exported configuration files**
   - They contain raw API keys
   - Even with permissions 600, sharing is risky

3. **NEVER store in cloud-synced folders**
   - Avoid Dropbox, Google Drive, OneDrive
   - These may sync without encryption

4. **NEVER push to public repositories**
   - Even private repos can be compromised
   - Rotate API keys if accidentally exposed

5. **NEVER embed in Docker images**
   - Images can be extracted and inspected
   - Use secrets management instead

---

## 🚨 Incident Response

### **If API Keys Are Exposed:**

1. **Immediately rotate all exposed API keys**
   ```bash
   # For each exposed key in .env
   # 1. Generate new key on provider's website
   # 2. Update .env file
   # 3. Re-export configuration
   ```

2. **Check for unauthorized usage**
   - Review provider dashboards
   - Check usage logs
   - Monitor billing

3. **Verify git history is clean**
   ```bash
   git log --all --full-history --source -- **/opencode*.json
   ```

4. **If committed to Git, use BFG Repo Cleaner**
   ```bash
   bfg --delete-files opencode.json
   git reflog expire --expire=now --all
   git gc --prune=now --aggressive
   ```

---

## 📋 Pre-Flight Checklist

Before exporting configurations, verify:

- [ ] `.gitignore` has security protections
- [ ] Export location is NOT in project directory
- [ ] API keys in `.env` are current and valid
- [ ] Latest verification results available
- [ ] Umask is set to 077 (restrictive file creation)
- [ ] Export script is up to date
- [ ] No cloud sync on target directory

---

## 🔧 Configuration Validation

### **Validate Export Integrity**

```python
import json

with open('opencode.json', 'r') as f:
    config = json.load(f)

# Verify security settings
assert config['security']['api_keys_embedded'] == True
assert config['security']['safe_to_commit'] == False
assert config['security']['gitignore_protected'] == True

# Verify permissions
import os
assert os.stat('opencode.json').st_mode & 0o777 == 0o600

print("✅ Configuration export is secure")
```

---

## 🎯 Usage Example

```python
from pathlib import Path
import json

# Load secure configuration
config_path = Path.home() / "Downloads" / "opencode.json"

with open(config_path, 'r') as f:
    config = json.load(f)

# Use verified models
for model in config['models']:
    if model['verified']:
        print(f"✅ {model['name']} - Score: {model['scores']['overall']}")
        print(f"   Features: {', '.join([k for k,v in model['capabilities'].items() if v])}")

# Access MCP server config
for server in config['mcp_servers']:
    if server['enabled']:
        print(f"🔌 {server['name']} ready")
```

---

## 📞 Support & Security Contacts

If you discover security issues:

1. **Do NOT file public issues**
2. Email: security@llm-verifier.local (replace with actual contact)
3. Include: Description, reproduction steps, severity
4. Response time: Within 24 hours

---

## ✅ Verification Checklist

Verify complete security setup:

```bash
# 1. Gitignore protection
python3 scripts/export_opencode_config_fixed.py --validate-only

# 2. Export configuration
python3 scripts/export_opencode_config_fixed.py

# 3. Check file permissions
ls -la ~/Downloads/opencode*.json  # Should show -rw-------

# 4. Verify API keys embedded
grep -c '"api_key": "sk-' ~/Downloads/opencode*.json  # Should show count

# 5. Test gitignore
git check-ignore --verbose ~/Downloads/opencode*.json  # Should show ignore rule
```

All checks should pass ✅

---

## 📚 Related Documentation

- [LLM Verifier Security Policy](SECURITY.md)
- [API Configuration Guide](docs/API_CONFIGURATION.md)
- [Challenge Framework](challenges/docs/CHALLENGE_FRAMEWORK.md)
- [Best Practices for API Key Management](docs/SECURITY_BEST_PRACTICES.md)

---

**Last Updated:** 2025-12-28  
**Version:** 2.0-ultimate  
**Security Level:** Maximum (600 permissions, gitignore protected, embedded warnings)