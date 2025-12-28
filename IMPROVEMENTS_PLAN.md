# Implementation Improvements Plan

**Date:** 2025-12-28  
**Status:** In Progress  
**Priority:** Fix database issue first, then add providers

---

## 🚨 **CRITICAL FIX - DATABASE COLUMN MISMATCH**

**Problem:** 
```
Failed to store verification result: 61 values for 63 columns
```

**Root Cause:**
The INSERT statement in `database/crud.go` has 63 columns but only 61 values are being provided. The `VerificationResult` struct is missing 3 fields that exist in the database schema:
- `overloaded` (BOOLEAN)
- `supports_parallel_tool_use` (BOOLEAN) 
- `value_proposition_score` (REAL)

**Solution:**
Update the INSERT statement to only include columns that have corresponding struct fields (60 columns + 1 auto-increment id = 61 total).

**Files to Modify:**
- `llm-verifier/database/crud.go` - Fix INSERT statement

---

## 📊 **PROVIDER ADDITIONS**

### **Priority 1: Add Groq Provider**

**Why:**
- ✅ Completely free tier
- ✅ OpenAI-compatible API (low implementation effort)
- ✅ Fast inference (specialized hardware)
- ✅ Expected: 5-10 working models immediately

**Configuration:**
```go
// Add to providers/config.go registerDefaultProviders()
pr.providers["groq"] = &ProviderConfig{
    Name:            "groq",
    Endpoint:        "https://api.groq.com/openai/v1",
    AuthType:        "bearer",
    StreamingFormat: "sse",
    DefaultModel:    "llama3-8b-8192",
    RateLimits: RateLimitConfig{
        RequestsPerMinute: 30,
        RequestsPerHour:   1000,
        BurstLimit:        5,
    },
    // ... timeouts and retry config
    Features: map[string]interface{}{
        "supports_streaming":       true,
        "supports_functions":       false,
        "supports_vision":          false,
        "supports_acp":             true,
        "max_context_length":       8192,
        "supported_models":         []string{
            "llama3-8b-8192",
            "llama3-70b-8192",
            "mixtral-8x7b-32768",
            "gemma-7b-it",
        },
    },
}
```

**API Docs:** https://console.groq.com/docs

---

### **Priority 2: Add Together AI Provider**

**Why:**
- ✅ $5 free trial credit
- ✅ OpenAI-compatible API
- ✅ 50+ models available
- ✅ Good for large models

**Configuration:**
```go
pr.providers["togetherai"] = &ProviderConfig{
    Name:            "togetherai",
    Endpoint:        "https://api.together.xyz/v1",
    AuthType:        "bearer",
    StreamingFormat: "sse",
    DefaultModel:    "meta-llama/Llama-3-8b-chat-hf",
    // ... rate limits, timeouts
    Features: map[string]interface{}{
        "supports_streaming":       true,
        "supports_functions":       false,
        "supports_vision":          false,
        "supports_acp":             true,
        "max_context_length":       4096,
        "supported_models":         []string{
            "meta-llama/Llama-3-8b-chat-hf",
            "meta-llama/Llama-3-70b-chat-hf",
            "codellama/CodeLlama-34b-Instruct-hf",
            "wizardlm/WizardLM-13B-V1.2",
        },
    },
}
```

**API Docs:** https://docs.together.ai/reference

---

### **Priority 3: Add Poe Provider**

**Why:**
- ✅ Aggregates multiple models (Claude, GPT-4, etc.)
- ✅ Single API key reduces management complexity
- ✅ OpenAI-compatible
- ✅ Medium effort, high value

**Configuration:**
```go
pr.providers["poe"] = &ProviderConfig{
    Name:            "poe",
    Endpoint:        "https://api.poe.com/v1",
    AuthType:        "bearer",
    StreamingFormat: "sse",
    DefaultModel:    "claude-3-sonnet",
    // ... configuration
}
```

**API Docs:** https://creator.poe.com/docs/external-applications/openai-compatible-api

---

## 🔧 **ERROR HANDLING IMPROVEMENTS**

### **Better Error Messages**

Current: `Model existence check failed: 401 Unauthorized`
Improved: `Model existence check failed: 401 Unauthorized - API key invalid. Please regenerate at https://provider.com/dashboard`

**Implementation:**
```go
func (vr *VerificationRunner) verifyModel(...) ModelVerification {
    // ...
    if err != nil || !exists {
        if statusCode == 401 {
            result.Error = fmt.Sprintf("API key invalid (401). Please regenerate at %s", 
                vr.getProviderDashboardURL(providerName))
        } else if statusCode == 402 {
            result.Error = fmt.Sprintf("Insufficient credits (402). Add payment info at %s",
                vr.getProviderDashboardURL(providerName))
        } else if statusCode == 404 {
            result.Error = fmt.Sprintf("Model not found (404). Check model ID at %s",
                vr.getProviderDocsURL(providerName))
        }
        return result
    }
}
```

**Provider Dashboard Mapping:**
```go
var providerDashboards = map[string]string{
    "openrouter": "https://openrouter.ai/keys",
    "huggingface": "https://huggingface.co/settings/tokens",
    "nvidia": "https://build.nvidia.com/nim",
    "gemini": "https://aistudio.google.com/app/apikey",
    "mistral": "https://console.mistral.ai/api-keys",
    "groq": "https://console.groq.com/keys",
    "togetherai": "https://api.together.xyz/settings/api-key",
    "deepseek": "https://platform.deepseek.com/api-keys",
}
```

---

## 📋 **API KEY REGENERATION HELPER**

### **Script: `scripts/regenerate_keys.sh`**

Generates commands to help users quickly regenerate keys:

```bash
#!/bin/bash
echo "Open these URLs to regenerate API keys:"
echo ""
echo "1. OpenRouter: https://openrouter.ai/keys"
echo "2. HuggingFace: https://huggingface.co/settings/tokens"
echo "3. NVIDIA: https://build.nvidia.com/nim"
echo "4. Google AI Studio: https://aistudio.google.com/app/apikey"
echo "5. Mistral: https://console.mistral.ai/api-keys"
echo ""
echo "After regenerating, update .env and run:"
echo "cd llm-verifier/cmd/model-verification && go run ."
```

---

## 🎯 **EXPECTED RESULTS AFTER IMPROVEMENTS**

### **Database Fix:**
- ✅ All verification results store successfully
- ✅ No more "61 values for 63 columns" errors

### **Adding Groq:**
- Expected: 5-10 new working models
- Models likely to work:
  - llama3-8b-8192
  - llama3-70b-8192
  - mixtral-8x7b-32768
  - gemma-7b-it

### **Adding Together AI:**
- Expected: 10-15 new working models
- Models likely to work:
  - Llama-3-8b-chat-hf
  - Llama-3-70b-chat-hf
  - CodeLlama variants

### **After Key Regeneration:**
- Expected total: 35-42 working models
- Success rate: 83-100%

---

## 📦 **FILES TO CREATE/MODIFY**

### **Create:**
1. ✅ `scripts/api_key_audit.sh` - Audit tool (done)
2. 📝 `scripts/regenerate_keys.sh` - Key regeneration helper
3. 📝 `llm-verifier/providers/groq.go` - Groq provider implementation
4. 📝 `llm-verifier/providers/togetherai.go` - Together AI implementation

### **Modify:**
1. 📝 `llm-verifier/database/crud.go` - Fix INSERT statement
2. 📝 `llm-verifier/providers/config.go` - Add provider configs
3. 📝 `llm-verifier/cmd/model-verification/run_full_verification.go` - Better error messages

---

## 🚀 **IMPLEMENTATION ORDER**

1. **Fix database column mismatch** (15 minutes)
2. **Add Groq provider** (30 minutes)
3. **Test verification with Groq** (15 minutes)
4. **Add Together AI provider** (30 minutes)
5. **Test verification with Together AI** (15 minutes)
6. **Improve error messages** (30 minutes)
7. **Create key regeneration helper** (15 minutes)
8. **Final test run** (15 minutes)

**Total time:** ~2.5 hours

---

*Plan created: 2025-12-28*
