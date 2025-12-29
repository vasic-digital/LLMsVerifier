#!/usr/bin/env python3
"""
Create COMPLETE OpenCode configuration with ALL models from llm-verifier binary
This shows every single model that was discovered and verified by the llm-verifier.
"""

import json
import sqlite3
import os
import subprocess
from datetime import datetime

def create_complete_models_config():
    print("🚀 Creating COMPLETE OpenCode configuration with ALL models from llm-verifier binary...")
    
    # Connect to llm-verifier database (sole source of truth)
    conn = sqlite3.connect('llm-verifier.db')
    cursor = conn.cursor()
    
    # Step 1: Get ALL models with their details from llm-verifier database
    print("📊 Fetching ALL models from llm-verifier database...")
    cursor.execute("""
        SELECT p.name as provider_name, p.api_key_encrypted, 
               m.model_id, m.name as model_name, m.verification_status
        FROM providers p
        JOIN models m ON p.id = m.provider_id
        WHERE p.api_key_encrypted != '' AND p.api_key_encrypted IS NOT NULL
        ORDER BY p.name, m.model_id
    """)
    
    all_models = cursor.fetchall()
    print(f"📈 Found {len(all_models)} total models in llm-verifier database")
    
    # Step 2: Organize by provider and show detailed model information
    providers_dict = {}
    verified_models = 0
    
    for provider_name, api_key, model_id, model_name, verification_status in all_models:
        if provider_name not in providers_dict:
            providers_dict[provider_name] = {
                "options": {
                    "apiKey": api_key
                },
                "models": {},
                "model_details": []
            }
        
        # Add the actual model with all its details
        providers_dict[provider_name]["models"][model_id] = {
            "name": model_name,
            "model_id": model_id,
            "verification_status": verification_status,
            "verified": verification_status == 'verified'
        }
        
        providers_dict[provider_name]["model_details"].append({
            "model_id": model_id,
            "name": model_name,
            "status": verification_status
        })
        
        if verification_status == 'verified':
            verified_models += 1
    
    # Step 3: Create comprehensive OpenCode configuration
    opencode_config = {
        "$schema": "https://opencode.ai/config.json",
        "version": "1.0",
        "username": "OpenCode AI Assistant (Complete Models - LLM-Verifier Binary)",
        "provider": {},
        "summary": {
            "total_models": len(all_models),
            "verified_models": verified_models,
            "providers": len(providers_dict),
            "source": "llm-verifier binary database - sole source of truth"
        }
    }
    
    # Step 4: Add each provider with ALL their models
    for provider_name, data in providers_dict.items():
        model_count = len(data["models"])
        print(f"🔌 {provider_name}: {model_count} models")
        
        # Show first few models as preview
        preview_models = list(data["models"].keys())[:3]
        for model in preview_models:
            print(f"   📋 {model}")
        if model_count > 3:
            print(f"   ... and {model_count - 3} more models")
        
        opencode_config["provider"][provider_name] = {
            "options": {
                "apiKey": data["options"]["apiKey"]
            },
            "models": data["models"]  # ALL models from llm-verifier binary
        }
    
    # Step 5: Save complete configuration
    output_file = "opencode_complete_models_llmverifier.json"
    with open(output_file, 'w') as f:
        json.dump(opencode_config, f, indent=2)
    
    # Set proper permissions
    os.chmod(output_file, 0o600)
    
    print(f"\n✅ COMPLETE configuration created with ALL models!")
    print(f"📁 Output file: {output_file}")
    print(f"📊 Total providers: {len(providers_dict)}")
    print(f"📈 Total models: {len(all_models)}")
    print(f"✅ Verified models: {verified_models}")
    print(f"📏 File size: {os.path.getsize(output_file)} bytes")
    
    # Step 6: Show detailed breakdown by provider
    print(f"\n📋 Complete breakdown by provider:")
    for provider_name, data in providers_dict.items():
        model_count = len(data["models"])
        verified_count = sum(1 for model in data["models"].values() if model["verified"])
        print(f"   ✅ {provider_name}: {model_count} models ({verified_count} verified)")
    
    # Step 7: Validate with llm-verifier binary
    print(f"\n🔍 Validating complete configuration with llm-verifier binary...")
    result = subprocess.run(['./bin/llm-verifier', 'ai-config', 'validate', output_file], 
                          capture_output=True, text=True)
    
    if result.returncode == 0:
        print("✅ llm-verifier binary validation PASSED!")
        
        # Copy to Downloads
        subprocess.run(['cp', output_file, '/home/milosvasic/Downloads/opencode.json'], 
                     capture_output=True)
        print("✅ Copied complete configuration to Downloads!")
        
        success = True
    else:
        print(f"❌ llm-verifier binary validation FAILED: {result.stderr}")
        success = False
    
    # Step 8: Show sample of actual models
    print(f"\n🎯 SAMPLE OF ACTUAL MODELS (first 10 models):")
    for i, (provider_name, data) in enumerate(list(providers_dict.items())[:3]):
        print(f"\n   📁 {provider_name}:")
        sample_models = list(data["models"].items())[:3]
        for model_id, model_info in sample_models:
            print(f"      • {model_id} - {model_info['name']} ({model_info['verification_status']})")
        if len(data["models"]) > 3:
            print(f"      ... and {len(data['models']) - 3} more models")
    
    conn.close()
    
    return len(providers_dict), len(all_models), verified_models, success

if __name__ == "__main__":
    provider_count, total_models, verified_models, is_valid = create_complete_models_config()
    
    if is_valid:
        print(f"\n🎉 ULTIMATE SUCCESS!")
        print(f"llm-verifier binary has created complete configuration with ALL {total_models} models!")
        print(f"{verified_models} models are verified and ready to use!")
        print(f"Complete configuration is ready in /home/milosvasic/Downloads/opencode.json")
    else:
        print(f"\n⚠️  Complete configuration created but llm-verifier binary validation failed")
        print(f"However, all {total_models} models from llm-verifier database are included")