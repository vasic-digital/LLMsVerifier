# OpenCode ProviderInitError Fix

## Problem Description

LLM Verifier was generating OpenCode configuration files that were incompatible with the actual OpenCode application, causing `ProviderInitError` when users tried to use the exported configurations.

## Root Cause

The LLM Verifier was generating configurations with:

1. **Wrong Schema**: Used `"$schema": "https://opencode.sh/schema.json"` instead of `"./opencode-schema.json"`
2. **Wrong Structure**: Used `"provider"` (singular) section instead of `"providers"` (plural)
3. **Invalid Provider Format**: Used complex nested structures with npm packages that don't exist in OpenCode
4. **Missing Required Sections**: Missing `data`, `agents`, `tui`, `shell`, and other required OpenCode sections
5. **Wrong Model References**: Used complex model objects instead of simple `provider.model` format

## Solution

### 1. Fixed Configuration Schema

**Before (Broken):**
```json
{
  "$schema": "https://opencode.sh/schema.json",
  "provider": {
    "openai": {
      "options": {
        "apiKey": "...",
        "baseURL": "https://api.openai.com/v1"
      },
      "models": {}
    }
  }
}
```

**After (Fixed):**
```json
{
  "$schema": "./opencode-schema.json",
  "data": {
    "directory": ".opencode"
  },
  "providers": {
    "openai": {
      "apiKey": "${OPENAI_API_KEY}",
      "disabled": false,
      "provider": "openai"
    }
  },
  "agents": {
    "coder": {
      "model": "openai.gpt-4o",
      "maxTokens": 5000
    }
  },
  "tui": {
    "theme": "opencode"
  },
  "shell": {
    "path": "/bin/bash",
    "args": ["-l"]
  },
  "autoCompact": true,
  "debug": false,
  "debugLSP": false
}
```

### 2. Updated Go Code

- **`createCorrectOpenCodeConfig()`**: New function that generates OpenCode-compatible configurations
- **Updated `extractProvider()`**: Returns lowercase provider names for consistency
- **Fixed validation**: New `validateCorrectOpenCodeConfigStructure()` function
- **Updated export logic**: Uses correct configuration format for OpenCode exports

### 3. Model Selection Logic

The fix includes intelligent model selection for agent roles:

- **Coder Agent**: Prioritizes GPT-4, Claude-3 models
- **Task Agent**: Uses same as coder or next best available
- **Title Agent**: Uses any available model (lighter models preferred)

### 4. Provider Name Normalization

All provider names are normalized to lowercase for consistency:
- `OpenAI` → `openai`
- `Anthropic` → `anthropic`
- `Google` → `google`
- etc.

## Files Modified

1. **`llm-verifier/llmverifier/config_export.go`**:
   - Added `createCorrectOpenCodeConfig()`
   - Added `validateCorrectOpenCodeConfigStructure()`
   - Added `getAPIKeyForProvider()`
   - Updated `extractProvider()` to return lowercase names
   - Modified export logic to use correct format

2. **`llm-verifier/cmd/fixed-ultimate-challenge/fixed_ultimate_challenge.go`**:
   - Updated `generateFixedUltimateOpenCode()` to use correct schema
   - Fixed schema validation

3. **`llm-verifier/llmverifier/config_export_test.go`**:
   - Updated test to validate new configuration format
   - Added comprehensive structure validation

## Testing

All tests pass, including:
- Configuration generation tests
- Structure validation tests
- JSON marshal/unmarshal tests
- Schema compliance tests

## Verification

To verify the fix works:

1. **Export OpenCode configuration** using LLM Verifier
2. **Check the generated JSON** matches the correct schema above
3. **Use with OpenCode** - should no longer produce `ProviderInitError`

## Migration Notes

- Existing exported configurations will not work with OpenCode
- Users should re-export configurations using the fixed LLM Verifier
- The fix is backward compatible - old code still works, new exports are correct

## Future Improvements

1. Add more comprehensive model selection logic
2. Support for custom model priorities per user
3. Integration testing with actual OpenCode binary
4. Documentation updates for correct configuration format</content>
<parameter name="filePath">OPENCODE_PROVIDERINITERROR_FIX.md