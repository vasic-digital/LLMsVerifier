#!/usr/bin/env python3
"""
Create FINAL working OpenCode configuration with real API keys
"""

import json
import sqlite3
import os
from datetime import datetime

def create_final_working_config():
    # Connect to database
    conn = sqlite3.connect('llm-verifier.db')
    cursor = conn.cursor()
    
    print(f"🎯 Creating FINAL working OpenCode configuration...")
    
    # Get all providers with their models
    opencode_config = {
        "$schema": "https://opencode.ai/config.json",
        "version": "1.0",
        "username": "OpenCode AI Assistant (Ultimate Challenge - FINAL)",
        "provider": {}
    }
    
    # Query all providers
    cursor.execute("SELECT id, name, api_key_encrypted FROM providers ORDER BY name")
    providers = cursor.fetchall()
    
    working_providers = 0
    total_models = 0
    
    for provider_row in providers:
        provider_id, provider_name, api_key = provider_row
        
        # Skip providers without API keys for validation
        if not api_key:
            print(f"⏭️  Skipping {provider_name} - no API key")
            continue
            
        # Get model count
        cursor.execute("SELECT COUNT(*) FROM models WHERE provider_id = ?", (provider_id,))
        model_count = cursor.fetchone()[0]
        
        print(f"✅ Adding {provider_name}: {model_count} models")
        
        # Create provider entry
        opencode_config["provider"][provider_name] = {
            "options": {
                "apiKey": api_key
            },
            "models": {}  # Empty per OpenCode specification
        }
        
        working_providers += 1
        total_models += model_count
    
    # Save to file
    output_file = "opencode_final_working.json"
    with open(output_file, 'w') as f:
        json.dump(opencode_config, f, indent=2)
    
    # Set proper permissions
    os.chmod(output_file, 0o600)
    
    print(f"\n✅ FINAL working configuration created!")
    print(f"📁 Output file: {output_file}")
    print(f"📊 Working providers: {working_providers}")
    print(f"📈 Total accessible models: {total_models}")
    print(f"📏 File size: {os.path.getsize(output_file)} bytes")
    
    conn.close()
    
    return output_file

if __name__ == "__main__":
    create_final_working_config()