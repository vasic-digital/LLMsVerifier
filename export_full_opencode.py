#!/usr/bin/env python3
"""
Export FULL OpenCode configuration with all models from our challenge data
This creates a complete configuration with all 1016 models and API keys
"""

import json
import sqlite3
import os
from datetime import datetime

def export_full_opencode():
    # Connect to database
    conn = sqlite3.connect('llm-verifier.db')
    cursor = conn.cursor()
    
    # Load environment variables for API keys
    env_vars = {}
    if os.path.exists('.env'):
        with open('.env', 'r') as f:
            for line in f:
                line = line.strip()
                if line and '=' in line and not line.startswith('#'):
                    key, value = line.split('=', 1)
                    env_vars[key] = value
    
    print(f"🔄 Exporting FULL OpenCode configuration with all models...")
    
    # Get all providers with their models
    opencode_config = {
        "$schema": "https://opencode.ai/config.json",
        "version": "1.0",
        "username": "OpenCode AI Assistant (Ultimate Challenge - FULL Models)",
        "provider": {}
    }
    
    # Query all providers with their models
    cursor.execute("""
        SELECT p.id, p.name, p.api_key_encrypted, 
               m.model_id, m.name as model_name, m.verification_status
        FROM providers p
        JOIN models m ON p.id = m.provider_id
        ORDER BY p.name, m.model_id
    """)
    
    results = cursor.fetchall()
    print(f"📊 Found {len(results)} provider-model combinations")
    
    # Organize by provider
    providers_dict = {}
    for row in results:
        provider_id, provider_name, api_key, model_id, model_name, verification_status = row
        
        if provider_name not in providers_dict:
            providers_dict[provider_name] = {
                "options": {
                    "apiKey": api_key or ""
                },
                "models": {}
            }
        
        # Add model to provider's models dict
        providers_dict[provider_name]["models"][model_id] = {
            "name": model_name,
            "verification_status": verification_status,
            "verified": verification_status == 'verified'
        }
    
    # Add to final config
    opencode_config["provider"] = providers_dict
    
    # Save to file
    output_file = "opencode_ultimate_full_models.json"
    with open(output_file, 'w') as f:
        json.dump(opencode_config, f, indent=2)
    
    # Set proper permissions
    os.chmod(output_file, 0o600)
    
    conn.close()
    
    print(f"\n✅ FULL OpenCode configuration exported!")
    print(f"📁 Output file: {output_file}")
    print(f"📊 Total providers: {len(providers_dict)}")
    print(f"📈 Total models: {len(results)}")
    print(f"🔑 Providers with API keys: {sum(1 for p in providers_dict.values() if p['options']['apiKey'])}")
    
    # Show summary by provider
    print(f"\n📋 Provider summary:")
    for provider_name, provider_data in providers_dict.items():
        model_count = len(provider_data["models"])
        has_api_key = bool(provider_data["options"]["apiKey"])
        print(f"   {provider_name}: {model_count} models {'✅' if has_api_key else '❌'}")

if __name__ == "__main__":
    export_full_opencode()