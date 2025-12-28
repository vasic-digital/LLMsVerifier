#!/usr/bin/env python3
"""
Generate OpenCode configuration from LLM verification results
"""

import json
import os
from datetime import datetime

# Load the verification results
results_path = "/media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier/challenges/full_verification/2025/12/28/143700/results/providers_export.json"

with open(results_path, 'r') as f:
    data = json.load(f)

# Create OpenCode configuration
opencode_config = {
    "version": "1.0",
    "generated_at": datetime.now().isoformat(),
    "description": "LLM Verifier - Auto-generated OpenCode configuration with verified providers and models",
    "settings": {
        "default_timeout": 30,
        "max_retries": 3,
        "request_delay": 1,
        "enable_streaming": True,
        "enable_acp": True,
        "enable_mcp": True,
        "log_level": "info"
    },
    "providers": [],
    "models": [],
    "model_groups": {},
    "acp_config": {
        "enabled": True,
        "protocol_version": "1.0",
        "capabilities": {
            "tool_use": True,
            "streaming": True,
            "function_calling": True,
            "json_mode": True
        }
    },
    "mcp_servers": [],
    "lsp_config": {
        "enabled": True,
        "diagnostics": True,
        "code_completion": True,
        "hover_info": True
    },
    "scoring": {
        "enabled": True,
        "weights": {
            "responsiveness": 0.30,
            "code_capability": 0.25,
            "reliability": 0.20,
            "feature_richness": 0.25
        }
    }
}

# Add verified providers and models
verified_count = 0
for provider in data.get("providers", []):
    provider_entry = {
        "name": provider["name"],
        "display_name": provider["name"].title(),
        "endpoint": provider["endpoint"],
        "enabled": provider.get("has_api_key", False),
        "api_key_required": True,
        "model_discovery": True,
        "verified_models": 0,
        "total_models": len(provider.get("models", [])),
        "health_status": "healthy" if any(m.get("verified") for m in provider.get("models", [])) else "degraded"
    }
    
    opencode_config["providers"].append(provider_entry)
    
    # Add models
    for model in provider.get("models", []):
        model_entry = {
            "id": model["model_id"],
            "name": model["name"],
            "provider": provider["name"],
            "enabled": model.get("verified", False),
            "verified": model.get("verified", False),
            "scores": model.get("scores", {}),
            "features": model.get("features", {}),
            "performance": {
                "response_time_ms": model.get("response_time_ms", 0),
                "ttft_ms": model.get("ttft_ms", 0),
                "last_verified": model.get("last_verified", "")
            },
            "capabilities": {
                "streaming": model.get("features", {}).get("streaming", False),
                "tool_calling": model.get("features", {}).get("tool_calling", False),
                "function_calling": model.get("features", {}).get("tool_calling", False),
                "embeddings": model.get("features", {}).get("embeddings", False),
                "vision": model.get("features", {}).get("vision", False),
                "audio": model.get("features", {}).get("audio", False),
                "mcp": model.get("features", {}).get("mcp", False),
                "lsp": model.get("features", {}).get("lsp", False),
                "acp": model.get("features", {}).get("acp", False),
                "code_generation": model.get("features", {}).get("code", False)
            }
        }
        
        if model.get("verified"):
            verified_count += 1
            provider_entry["verified_models"] += 1
            
            # Add to model groups by capability
            if model.get("features", {}).get("acp"):
                if "acp_enabled" not in opencode_config["model_groups"]:
                    opencode_config["model_groups"]["acp_enabled"] = []
                opencode_config["model_groups"]["acp_enabled"].append(model["model_id"])
            
            if model.get("features", {}).get("code"):
                if "coding" not in opencode_config["model_groups"]:
                    opencode_config["model_groups"]["coding"] = []
                opencode_config["model_groups"]["coding"].append(model["model_id"])
            
            if model.get("scores", {}).get("overall", 0) >= 75:
                if "high_performance" not in opencode_config["model_groups"]:
                    opencode_config["model_groups"]["high_performance"] = []
                opencode_config["model_groups"]["high_performance"].append(model["model_id"])
        
        opencode_config["models"].append(model_entry)

# Add MCP servers for verified ACP-enabled models
for model in opencode_config["models"]:
    if model.get("capabilities", {}).get("acp") and model.get("verified"):
        provider_upper = model['provider'].upper()
        mcp_server = {
            "name": f"{model['provider']}_{model['id']}".replace("/", "_").replace("-", "_"),
            "type": "stdio",
            "command": "node",
            "args": ["mcp-server.js"],
            "env": {
                "PROVIDER": model["provider"],
                "MODEL": model["id"],
                "API_KEY": f"${{ {provider_upper}_API_KEY }}"
            },
            "timeout": 30000,
            "enabled": True,
            "capabilities": {
                "tools": model.get("capabilities", {}).get("tool_calling", False),
                "resources": True,
                "prompts": True
            }
        }
        opencode_config["mcp_servers"].append(mcp_server)

# Add summary
opencode_config["summary"] = {
    "total_providers": len(opencode_config["providers"]),
    "total_models": len(opencode_config["models"]),
    "verified_models": verified_count,
    "acp_enabled_models": len(opencode_config["model_groups"].get("acp_enabled", [])),
    "coding_models": len(opencode_config["model_groups"].get("coding", [])),
    "high_performance_models": len(opencode_config["model_groups"].get("high_performance", []))
}

# Write to Downloads directory
output_path = "/home/milosvasic/Downloads/opencode.json"
with open(output_path, 'w') as f:
    json.dump(opencode_config, f, indent=2)

print(f"✅ Generated OpenCode configuration with {len(opencode_config['providers'])} providers and {len(opencode_config['models'])} models")
print(f"✅ {verified_count} verified models included")
print(f"✅ Saved to: {output_path}")

# Print summary
print("\n" + "="*60)
print("OpenCode Configuration Summary")
print("="*60)
print(f"Total Providers: {opencode_config['summary']['total_providers']}")
print(f"Total Models: {opencode_config['summary']['total_models']}")
print(f"Verified Models: {opencode_config['summary']['verified_models']}")
print(f"ACP-Enabled Models: {opencode_config['summary']['acp_enabled_models']}")
print(f"High Performance Models: {opencode_config['summary']['high_performance_models']}")
print("="*60)

# List verified models
print("\nVerified Models (Ready to Use):")
for model in opencode_config["models"]:
    if model.get("verified"):
        print(f"  ✅ {model['name']} ({model['provider']}) - Score: {model['scores'].get('overall', 0)}")
        if model.get("capabilities", {}).get("acp"):
            print(f"     └─ ACP Enabled")