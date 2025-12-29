#!/usr/bin/env python3
"""
Fix OpenCode Configuration Structure

This script fixes the structural issues in our OpenCode configuration
to match the expected format.
"""

import json
import os

def fix_model_structure(models_dict):
    """Fix model structure to include required fields"""
    fixed_models = {}
    
    for model_key, model_data in models_dict.items():
        # Create proper model structure
        fixed_model = {
            "id": model_key,  # Use the model key as ID
            "name": model_data.get("name", model_key),
            "displayName": model_data.get("name", model_key),  # Use name as displayName
            "provider": {
                "id": "unknown",  # Will be set by parent provider
                "npm": "@openrouter/unknown"
            },
            "maxTokens": model_data.get("maxTokens", 4096),
            "supportsHTTP3": model_data.get("supportsHTTP3", False)
        }
        
        # Add optional fields if they exist
        if "cost_per_1m_in" in model_data:
            fixed_model["costPer1MInput"] = model_data["cost_per_1m_in"]
        if "cost_per_1m_out" in model_data:
            fixed_model["costPer1MOutput"] = model_data["cost_per_1m_out"]
        if "supports_brotli" in model_data:
            fixed_model["supportsBrotli"] = model_data["supports_brotli"]
        
        # Add features if any
        features = {}
        if model_data.get("supports_brotli"):
            features["brotli"] = True
        if model_data.get("supportsHTTP3"):
            features["http3"] = True
        if model_data.get("is_free"):
            features["freeToUse"] = True
        if model_data.get("is_open_source"):
            features["openSource"] = True
        
        if features:
            fixed_model["features"] = features
        
        fixed_models[model_key] = fixed_model
    
    return fixed_models

def fix_provider_structure(providers_dict):
    """Fix provider structure"""
    fixed_providers = {}
    
    for provider_key, provider_data in providers_dict.items():
        fixed_provider = {
            "displayName": provider_data.get("displayName", provider_key.title()),
            "options": provider_data.get("options", {})
        }
        
        # Fix models if they exist
        if "models" in provider_data:
            fixed_provider["models"] = fix_model_structure(provider_data["models"])
            
            # Fix provider references in models
            for model_key, model_data in fixed_provider["models"].items():
                model_data["provider"]["id"] = provider_key
                model_data["provider"]["npm"] = f"@openrouter/{provider_key}-provider"
        
        fixed_providers[provider_key] = fixed_provider
    
    return fixed_providers

def main():
    # Load the current configuration
    input_file = "/media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier/ultimate_opencode_config.json"
    output_file = "/media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier/opencode_fixed_structure.json"
    
    print(f"🔧 Loading configuration from {input_file}")
    
    with open(input_file, 'r') as f:
        config = json.load(f)
    
    print("🔍 Analyzing current structure...")
    
    # Fix the provider structure
    if "provider" in config:
        config["provider"] = fix_provider_structure(config["provider"])
    
    # Ensure required top-level fields
    if "$schema" not in config:
        config["$schema"] = "https://opencode.sh/schema.json"
    
    if "username" not in config:
        config["username"] = "OpenCode AI Assistant"
    
    # Add agent configuration if missing
    if "agent" not in config:
        config["agent"] = {
            "code": {
                "model": "openai/gpt-4",
                "prompt": "You are a senior software engineer specializing in code development, debugging, and optimization. You have deep expertise in multiple programming languages and frameworks. Help the user write clean, efficient, and well-documented code.",
                "tools": {
                    "bash": True,
                    "docker": True,
                    "git": True,
                    "lsp": True,
                    "webfetch": True
                },
                "temperature": 0.2,
                "maxSteps": 10
            }
        }
    
    # Add MCP configuration if missing
    if "mcp" not in config:
        config["mcp"] = {
            "github": {
                "type": "local",
                "command": ["npx", "-y", "@modelcontextprotocol/server-github"],
                "enabled": True,
                "environment": {
                    "GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN}"
                },
                "timeout": 30000
            },
            "filesystem": {
                "type": "local",
                "command": ["npx", "-y", "@modelcontextprotocol/server-filesystem"],
                "enabled": True,
                "args": ["."],
                "timeout": 30000
            }
        }
    
    print(f"💾 Saving fixed configuration to {output_file}")
    
    with open(output_file, 'w') as f:
        json.dump(config, f, indent=2)
    
    print(f"✅ Configuration structure fixed!")
    print(f"📊 Total providers: {len(config.get('provider', {}))}")
    
    # Count total models
    total_models = sum(len(provider.get("models", {})) for provider in config.get("provider", {}).values())
    print(f"📊 Total models: {total_models}")
    
    return output_file

if __name__ == "__main__":
    output = main()
    print(f"\n🎯 Fixed configuration saved to: {output}")