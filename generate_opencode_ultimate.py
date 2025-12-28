#!/usr/bin/env python3
"""
Generate ULTIMATE OpenCode configuration with ALL API keys embedded
DO NOT COMMIT - Protected by .gitignore
"""

import json
import os
from datetime import datetime

# Load verification results
with open('/media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier/challenges/full_verification/2025/12/28/143700/results/providers_export.json', 'r') as f:
    verification_data = json.load(f)

# Load .env to get API keys
api_keys = {}
with open('/media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier/.env', 'r') as f:
    for line in f:
        line = line.strip()
        if line and not line.startswith('#') and '=' in line:
            key, value = line.split('=', 1)
            api_keys[key] = value

# Provider mapping
provider_env_map = {
    "huggingface": "ApiKey_HuggingFace", "nvidia": "ApiKey_Nvidia", "chutes": "ApiKey_Chutes",
    "siliconflow": "ApiKey_SiliconFlow", "kimi": "ApiKey_Kimi", "gemini": "ApiKey_Gemini",
    "openrouter": "ApiKey_OpenRouter", "zai": "ApiKey_ZAI", "deepseek": "ApiKey_DeepSeek",
    "mistralaistudio": "ApiKey_Mistral_AiStudio", "codestral": "ApiKey_Codestral",
    "cerebras": "ApiKey_Cerebras", "cloudflareworkersai": "ApiKey_Cloudflare_Workers_AI",
    "fireworksai": "ApiKey_Fireworks_AI", "baseten": "ApiKey_Baseten", "novitaai": "ApiKey_Novita_AI",
    "upstageai": "ApiKey_Upstage_AI", "nlpcloud": "ApiKey_NLP_Cloud", "modaltokenid": "ApiKey_Modal_Token_ID",
    "modaltokensecret": "ApiKey_Modal_Token_Secret", "inference": "ApiKey_Inference",
    "hyperbolic": "ApiKey_Hyperbolic", "sambanovaai": "ApiKey_SambaNova_AI", "replicate": "ApiKey_Replicate"
}

# Create OpenCode config
opencode = {
    "version": "2.0-ultimate",
    "generated_at": datetime.now().isoformat(),
    "security_warning": "CONTAINS EMBEDDED API KEYS - DO NOT COMMIT",
    "settings": {
        "default_timeout": 30, "max_retries": 3, "enable_acp": True, "enable_mcp": True, "enable_lsp": True
    },
    "providers": [],
    "models": [],
    "model_groups": {},
    "mcp_servers": [],
    "acp_config": {"enabled": True, "capabilities": {"tool_use": True, "streaming": True, "function_calling": True}}
}

# Process providers and models
verified_models = 0
acp_models = 0
lsp_models = 0
mcp_models = 0
embedding_models = 0
high_perf = 0

def clean_name(name):
    return name.replace("/", "_").replace("-", "_").replace(".", "_").lower()

for provider in verification_data.get("providers", []):
    provider_name = provider["name"]
    env_key = provider_env_map.get(provider_name, "")
    api_key = api_keys.get(env_key, "NOT_FOUND")
    
    provider_entry = {
        "name": provider_name,
        "display_name": provider_name.replace("_", " ").title(),
        "endpoint": provider["endpoint"],
        "api_key": api_key,
        "api_key_available": api_key != "NOT_FOUND",
        "enabled": api_key != "NOT_FOUND",
        "verified_models": 0,
        "total_models": len(provider.get("models", [])),
        "health_status": "healthy" if any(m.get("verified") for m in provider.get("models", [])) else "degraded"
    }
    opencode["providers"].append(provider_entry)
    
    for model in provider.get("models", []):
        verified = model.get("verified", False)
        features = model.get("features", {})
        scores = model.get("scores", {})
        
        model_entry = {
            "id": model["model_id"],
            "name": model["name"],
            "provider": provider_name,
            "verified": verified,
            "enabled": verified,
            "api_key": api_key if verified else None,
            "scores": scores,
            "features": features,
            "capabilities": {
                "streaming": features.get("streaming"),
                "tool_calling": features.get("tool_calling"),
                "function_calling": features.get("tool_calling"),
                "embeddings": features.get("embeddings"),
                "vision": features.get("vision"),
                "audio": features.get("audio"),
                "mcp": features.get("mcp"),
                "lsp": features.get("lsp"),
                "acp": features.get("acp"),
                "code_generation": features.get("code")
            },
            "performance": {
                "response_time_ms": model.get("response_time_ms", 0),
                "ttft_ms": model.get("ttft_ms", 0)
            }
        }
        opencode["models"].append(model_entry)
        
        if verified:
            verified_models += 1
            provider_entry["verified_models"] += 1
            stats_acp = features.get("acp", False)
            stats_lsp = features.get("lsp", False)
            stats_mcp = features.get("mcp", False)
            stats_emb = features.get("embeddings", False)
            
            if stats_acp:
                acp_models += 1
                if "acp_enabled" not in opencode["model_groups"]:
                    opencode["model_groups"]["acp_enabled"] = []
                opencode["model_groups"]["acp_enabled"].append(model["model_id"])
            
            if stats_lsp:
                lsp_models += 1
                if "lsp_enabled" not in opencode["model_groups"]:
                    opencode["model_groups"]["lsp_enabled"] = []
                opencode["model_groups"]["lsp_enabled"].append(model["model_id"])
            
            if stats_mcp:
                mcp_models += 1
                if "mcp_enabled" not in opencode["model_groups"]:
                    opencode["model_groups"]["mcp_enabled"] = []
                opencode["model_groups"]["mcp_enabled"].append(model["model_id"])
            
            if stats_emb:
                embedding_models += 1
                opencode["embeddings_config"]["providers"].append({
                    "provider": provider_name,
                    "model": model["model_id"],
                    "enabled": True
                })
                if not opencode["embeddings_config"]["default_model"]:
                    opencode["embeddings_config"]["default_model"] = model["model_id"]
            
            if scores.get("overall", 0) >= 75:
                high_perf += 1
                if "high_performance" not in opencode["model_groups"]:
                    opencode["model_groups"]["high_performance"] = []
                opencode["model_groups"]["high_performance"].append(model["model_id"])
            
            if features.get("code"):
                if "coding" not in opencode["model_groups"]:
                    opencode["model_groups"]["coding"] = []
                opencode["model_groups"]["coding"].append(model["model_id"])
            
            # MCP Server for ACP-enabled models
            if stats_acp:
                mcp_server = {
                    "name": f"{provider_name}_{clean_name(model['model_id'])}_mcp",
                    "type": "stdio",
                    "command": "node",
                    "args": ["mcp-server.js"],
                    "env": {
                        "PROVIDER": provider_name,
                        "MODEL": model["model_id"],
                        "API_KEY": api_key,
                        "API_ENDPOINT": provider["endpoint"]
                    },
                    "timeout": 30000,
                    "enabled": True,
                    "capabilities": {
                        "tools": features.get("tool_calling", False),
                        "resources": True,
                        "prompts": True
                    }
                }
                opencode["mcp_servers"].append(mcp_server)

# Update metadata
opencode["metadata"] = {
    "total_providers": len(opencode["providers"]),
    "total_models": len(opencode["models"]),
    "verified_models": verified_models,
    "acp_enabled_models": acp_models,
    "lsp_enabled_models": lsp_models,
    "mcp_enabled_models": mcp_models,
    "embedding_models": embedding_models,
    "high_performance_models": high_perf
}

# Save to Downloads
output_path = "/home/milosvasic/Downloads/opencode.json"
with open(output_path, 'w') as f:
    json.dump(opencode, f, indent=2)

# Set restrictive permissions for security
os.chmod(output_path, 0o600)

print("="*70)
print("✅ ULTIMATE OPENCODE CONFIGURATION GENERATED")
print("="*70)
print(f"File: {output_path}")
print(f"Size: {os.path.getsize(output_path) / 1024:.1f} KB")
print(f"Permissions: 600 (owner read/write only)")
print(f"WARNING: CONTAINS EMBEDDED API KEYS")
print("="*70)
print(f"\n📊 Statistics:")
print(f"   Total Providers: {len(opencode['providers'])}")
print(f"   Total Models: {len(opencode['models'])}")
print(f"   ✅ Verified Models: {verified_models}")
print(f"   🎯 ACP-Enabled: {acp_models}")
print(f"   🔧 LSP-Enabled: {lsp_models}")
print(f"   🔌 MCP-Enabled: {mcp_models}")
print(f"   📝 Embedding Models: {embedding_models}")
print(f"   ⭐ High Performance (75+): {high_perf}")
print(f"   🔌 MCP Servers Configured: {len(opencode['mcp_servers'])}")
print("="*70)
print("\n✅ VERIFIED MODELS (Ready to Use):")
for model in opencode["models"]:
    if model["verified"]:
        print(f"\n   🎯 {model['name']} ({model['provider']})")
        print(f"      Score: {model['scores']['overall']}/100")
        print(f"      Features: ACP={model['capabilities']['acp']} LSP={model['capabilities']['lsp']} MCP={model['capabilities']['mcp']}")
        print(f"      API Key: {'✅ Embedded' if model['api_key'] else '❌ None'}")
        if model['capabilities']['tool_calling']:
            print(f"      ⚡ Tool Calling: Enabled")
        if model['capabilities']['streaming']:
            print(f"      🌊 Streaming: Enabled")
        if model['capabilities']['embeddings']:
            print(f"      📊 Embeddings: Enabled")
        print(f"      Speed: {model['performance']['response_time_ms']}ms")
print("="*70)
print("\n⚠️  SECURITY REMINDER:")
print("   - File contains embedded API keys")
print("   - Protected by .gitignore (line 191)")
print("   - Do NOT commit to version control")
print("   - Safe for local development use only")
print("="*70)