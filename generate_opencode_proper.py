#!/usr/bin/env python3
"""
Generate VALID OpenCode configuration following the correct schema
"""

import json
import sys
import os

def generate_proper_opencode(providers_json, env_file):
    """Generate proper OpenCode config"""
    
    # Load providers
    with open(providers_json, 'r') as f:
        providers = json.load(f)
    
    # Load env vars
    env_vars = {}
    with open(env_file, 'r') as f:
        for line in f:
            line = line.strip()
            if line.startswith('ApiKey_'):
                # Parse ApiKey_PROVIDER_NAME=value
                parts = line.split('=', 1)
                if len(parts) == 2:
                    key = parts[0]
                    value = parts[1]
                    # Skip if it's a reference (starts with $)
                    if not value.startswith('$'):
                        provider_name = key.replace('ApiKey_', '')
                        env_vars[provider_name] = value
    
    # Create proper OpenCode config
    config = {
        "provider": {},  # Must be "provider" not "providers"
        "agent": {
            "verifier": {
                "model": "openai/gpt-4",
                "prompt": "You are an LLM verifier agent. Verify configurations and test providers.",
                "tools": {
                    "webfetch": True,
                    "bash": True
                }
            }
        },
        "mcp": {
            "filesystem": {
                "type": "local",
                "command": ["npx", "@modelcontextprotocol/server-filesystem"],
                "enabled": True
            }
        },
        "command": {
            "verify-all": {
                "template": "Verify all providers and models",
                "agent": "verifier"
            }
        }
    }
    
    # Add each provider
    for provider_name, provider_data in providers.items():
        # Skip if no API key
        env_provider_name = provider_name.replace('-', '_').replace(' ', '_').title()
        if env_provider_name not in env_vars:
            print(f"Skipping {provider_name} - no API key")
            continue
            
        api_key = env_vars[env_provider_name]
        base_url = provider_data.get('chat_endpoint', '').split('/v1/')[0] + '/v1'
        
        # Clean up base_url for special cases
        if '{model}' in base_url or '{account_id}' in base_url:
            if 'gemini' in provider_name:
                base_url = 'https://generativelanguage.googleapis.com/v1'
            elif 'cloudflare' in provider_name:
                base_url = 'https://api.cloudflare.com/client/v4'
            elif 'huggingface' in provider_name:
                base_url = 'https://api-inference.huggingface.co'
        
        # Add provider to config
        provider_entry = {
            "options": {
                "api_key": api_key,
                "base_url": base_url
            }
        }
        
        config["provider"][provider_name] = provider_entry
    
    print(f"✅ Generated OpenCode config with {len(config['provider'])} providers")
    return config

def main():
    if len(sys.argv) != 3:
        print(f"Usage: {sys.argv[0]} <providers_json> <env_file>")
        sys.exit(1)
    
    providers_json = sys.argv[1]
    env_file = sys.argv[2]
    
    if not os.path.exists(providers_json):
        print(f"Error: {providers_json} not found")
        sys.exit(1)
    
    if not os.path.exists(env_file):
        print(f"Error: {env_file} not found")
        sys.exit(1)
    
    config = generate_proper_opencode(providers_json, env_file)
    
    # Output to stdout
    json.dump(config, sys.stdout, indent=2)
    print()  # Newline

if __name__ == "__main__":
    main()
