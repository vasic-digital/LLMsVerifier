# LLMsVerifier Development Guidelines

Auto-generated from all feature plans. Last updated: 2025-12-28

## Active Technologies

- Go 1.21+ + Existing LLM Verifier codebase, HTTP client libraries (001-extend-llm-providers)
- Python 3.8+ for security and export tools
- SQLite with SQL Cipher for encrypted databases

## Project Structure

```text
src/
tests/
scripts/          # Security and export utilities
challenges/       # Challenge framework and results
docs/             # Documentation
data/             # Test data and fixtures
```

## 🔐 Security-Critical Commands

### Export OpenCode Configuration (Secure)

```bash
# Standard export - generates VALID OpenCode configuration with embedded API keys
# File is automatically protected with 600 permissions and gitignore rules
python3 scripts/export_opencode_config_fixed.py

# Custom output location
python3 scripts/export_opencode_config_fixed.py --output /path/to/secure/location/

# Validate gitignore protections (recommended before export)
python3 scripts/export_opencode_config_fixed.py --validate-only
```

**Security Features:**
- ✅ Automatic 600 file permissions (owner read/write only)
- ✅ Gitignore protection verification
- ✅ Comprehensive security warnings displayed
- ✅ All API keys embedded from `.env` file
- ✅ Feature detection (MCP, LSP, ACP, Embeddings)
- ✅ Performance metrics and scoring included

**Output:** `~/Downloads/opencode_{timestamp}.json` (52-60 KB typical)

### Run Model Verification Challenge

```bash
make build
cd llm-verifier
../bin/llm-verifier run_full_verification_fixed
```

**What it does:**
- Discovers all providers with API keys
- Tests 40+ models with real HTTP requests
- Verifies features: Streaming, Tool Calling, Vision, ACP, LSP, MCP
- Generates scores (0-100 scale)
- Saves results to database and JSON exports

**Results Location:**
`challenges/full_verification/{year}/{month}/{day}/{timestamp}/results/`

### Run All Challenges

```bash
cd llm-verifier
./run_challenges
```

## 🛡️ Security Requirements

### Configuration Export Rules

1. **ALWAYS use the official export script:**
   ```bash
   python3 scripts/export_opencode_config_fixed.py
   ```

2. **NEVER commit exported configurations:**
   - Files contain embedded API keys
   - Protected by `.gitignore` (lines 180-220)
   - Manual check: `git check-ignore opencode*.json`

3. **Verify permissions after export:**
   ```bash
   ls -la ~/Downloads/opencode*.json  # Should show -rw-------
   ```

4. **Rotate API keys if exposed:**
   - Immediately regenerate on provider dashboards
   - Update `.env` file
   - Re-run export script

### Gitignore Protection

The `.gitignore` file protects:
- `opencode.json` (root level)
- `opencode_*_with_keys.json` (pattern)
- `Downloads/opencode*.json` (user Downloads)
- `**/*api_key*` (any API key files)
- `**/*secret*` (any secret files)
- `**/*.env` (environment files)

**Validation:**
```bash
python3 scripts/export_opencode_config_fixed.py --validate-only
```

## 📖 Code Style

### Go Code
- Follow standard Go conventions
- Use `gofmt` for formatting
- Minimum Go version: 1.21
- Error handling: explicit, wrapped errors

### Python Scripts
- Follow PEP 8
- Use type hints where applicable
- Security-first: validate inputs, sanitize outputs
- Use `pathlib` for file operations

## 🏗️ Architecture Patterns

### Security Pattern: Secure Exports

When creating export functionality:

1. **Always validate gitignore:**
   ```python
   if not gitignore_has_protection():
       raise SecurityError("Gitignore protections missing")
   ```

2. **Always set restrictive permissions:**
   ```python
   os.chmod(output_file, 0o600)  # Owner read/write only
   ```

3. **Always include security warnings:**
   ```python
   config = {
       "security_warning": "CONTAINS API KEYS - DO NOT COMMIT",
       "safe_to_commit": False
   }
   ```

4. **Never embed keys without warnings:**
   - Display warnings during export
   - Include warnings in generated files
   - Document in generated files

### Feature Detection Pattern

When testing model capabilities:

```python
# Features to test
features = {
    "streaming": test_streaming(...),
    "tool_calling": test_tool_calling(...),
    "embeddings": test_embeddings(...),
    "vision": test_vision(...),
    "mcp": test_mcp_support(...),
    "lsp": test_lsp_support(...),
    "acp": test_acp_support(...)
}

# Validate with real HTTP requests (not just config)
# Score based on actual performance, not claims
```

### Scoring Pattern

Scoring algorithm:
```
Overall Score = 
  (Responsiveness × 0.30) +
  (Code Capability × 0.25) +
  (Feature Richness × 0.25) +
  (Reliability × 0.20)
```

Where:
- Responsiveness: 0-30 points (based on response time)
- Code Capability: 0-25 points (based on features + tests)
- Feature Richness: 0-25 points (count of supported features)
- Reliability: 0-20 points (verification status)

## 📋 Recent Changes

### 2025-12-28: Security Enhancement (v2.0-ultimate)

**Added:**
- `scripts/export_opencode_config_fixed.py` - Secure export tool with:
  - Automatic 600 permissions
  - Gitignore validation
  - Security warnings
  - Comprehensive feature detection
  - Performance metrics
  
- Enhanced `.gitignore` with expanded security patterns
  - Protects all `*api_key*` files
  - Protects all `*secret*` files
  - Protects `opencode_*_with_keys.json` patterns
  - Protects user Downloads directory exports

- Documentation: `SECURITY_CONFIGURATION_EXPORT.md`
  - Complete security guidelines
  - Incident response procedures
  - Pre-flight checklists
  - Usage examples

**What Gets Exported:**
- All 25 providers with API keys
- All 40+ models with verification status
- MCP/LSP/ACP feature detection
- Performance metrics (response time, TTFT)
- Comprehensive scoring (0-100 scale)
- MCP server configurations
- Model groups for easy selection

**Verified Models:**
- OpenRouter GPT-4: Score 80/100 ⭐
- OpenRouter Claude 3.5: Score 80/100 ⭐
- DeepSeek Chat: Score 73/100 ⭐

### 2025-12-24: Challenge Framework v1.0

**Added:**
- Challenge runner system
- Model verification with real HTTP tests
- Database schema for results storage
- Initial export functionality

## 🔧 Development Commands

```bash
# Setup
cd /media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier

# Run tests
make test

# Format code
make format

# Lint
go vet ./...
golangci-lint run

# Build
make build

# Run verifier
cd llm-verifier
./llm-verifier

# Export configuration
python3 scripts/export_opencode_config_fixed.py
```

## 📚 Documentation

- **Security Guide:** `SECURITY_CONFIGURATION_EXPORT.md`
- **Challenge Framework:** `challenges/docs/CHALLENGE_FRAMEWORK.md`
- **API Documentation:** `docs/API_DOCUMENTATION.md`
- **Database Schema:** `llm-verifier/database/schema.sql`

## 🆘 Security Contacts

If you discover security issues with exports or API key handling:

1. **DO NOT file public GitHub issues**
2. Check `SECURITY.md` for private reporting process
3. Include: Description, severity, reproduction steps
4. Expected response: Within 24 hours

---

**Last Updated:** 2025-12-28  
**Export Version:** 2.0-ultimate  
**Security Level:** Maximum (600 permissions, gitignore protected, embedded warnings)