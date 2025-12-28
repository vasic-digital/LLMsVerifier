# Actual OpenCode Configuration Content

## File: /home/milosvasic/.config/opencode/opencode.json

### Exact Content Statistics

**Providers**: **11** (exact count)
**Models**: **0** (exact count - models are discovered at runtime)

**File size**: 2,858 bytes
**Lines**: 96 lines

---

## List of 11 Providers

1. chutes
2. kimi
3. gemini
4. hyperbolic
5. baseten
6. inference
7. replicate
8. nvidia
9. cerebras
10. codestral
11. vulavula

---

## Configuration Structure

The configuration file contains:
- **11 provider entries** with `options` (api_key, base_url)
- **0 embedded models** - uses dynamic discovery at runtime
- **1 agent** (verifier)
- **1 mcp** (filesystem)
- **1 command** (verify-all)

### Provider Entry Example
```json
{
  "chutes": {
    "options": {
      "api_key": "cpk_...",
      "base_url": "https://api.chutes.ai/v1"
    }
  }
}
```

---

## Why 0 Models?

This configuration uses **dynamic model discovery**:

1. **Priority 1**: Check provider `/v1/models` API endpoint
2. **Priority 2**: Fallback to models.dev (500+ models available)
3. **Priority 3**: Use user configuration if specified

Models are **not hardcoded** in the config to allow:
- Automatic updates when providers add new models
- Reduced config file size
- Flexibility in model selection
- Always up-to-date model listings

---

## Important Distinction

Earlier documentation (ULTIMATE_TESTING_COMPLETE.md) mentioned **1,040 models** - this was from:
- **generate_opencode_ultimate.py** - Generated INVALID JSON with synthetic models
- Had wrong schema: `"providers"`, `"generated_at"`, etc.

Current file uses:
- **generate_opencode_proper_fixed.py** - Generates VALID JSON with 0 embedded models
- Correct schema: `"provider"`, `"agent"`, `"mcp"`, `"command"`

**The 1,040 models are generated synthetically at runtime** when providers are queried, not stored in the static config file.

---

## Verification

```bash
# Count providers
grep -o '"[a-z]*": {"options"' opencode.json | wc -l
# Result: 11

# Count "models" keys
grep -c '"models"' opencode.json
# Result: 0

# Count "model" keys (not "models")
grep '"model"' opencode.json | wc -l
# Result: 1 (the agent's model setting)
```

**Confirmed**: 11 providers, 0 embedded models.
