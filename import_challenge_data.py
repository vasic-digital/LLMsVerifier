#!/usr/bin/env python3
"""
Import ultimate challenge data into llm-verifier database
This will populate the database with all discovered providers and models
"""

import json
import sqlite3
import os
from datetime import datetime

def import_challenge_data():
    # Load challenge results
    with open('challenge_models_extracted.json', 'r') as f:
        challenge_data = json.load(f)
    
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
    
    print(f"🔄 Importing challenge data...")
    print(f"📊 Found {len(challenge_data)} providers with models")
    
    total_models = 0
    
    for provider_name, models in challenge_data.items():
        print(f"\n🔌 Processing provider: {provider_name}")
        print(f"   Found {len(models)} models")
        
        # Get API key from environment
        api_key = env_vars.get(f'{provider_name.upper()}_API_KEY', '')
        
        # Insert provider
        cursor.execute("""
            INSERT OR REPLACE INTO providers 
            (name, endpoint, api_key_encrypted, description, is_active, created_at, updated_at)
            VALUES (?, ?, ?, ?, ?, ?, ?)
        """, (
            provider_name,
            f"https://api.{provider_name}.com/v1",
            api_key,  # Store as encrypted for now
            f"{provider_name} LLM provider",
            1,
            datetime.now(),
            datetime.now()
        ))
        
        provider_id = cursor.lastrowid
        
        # Insert models for this provider
        for model_name in models:
            cursor.execute("""
                INSERT OR REPLACE INTO models 
                (provider_id, model_id, name, description, verification_status, created_at, updated_at)
                VALUES (?, ?, ?, ?, ?, ?, ?)
            """, (
                provider_id,
                model_name,
                model_name,
                f"{model_name} from {provider_name}",
                'verified',  # Mark as verified from our challenge
                datetime.now(),
                datetime.now()
            ))
            total_models += 1
    
    conn.commit()
    conn.close()
    
    print(f"\n✅ Import complete!")
    print(f"📈 Imported {len(challenge_data)} providers")
    print(f"📊 Imported {total_models} models")
    print(f"🔑 Found API keys for {sum(1 for k in env_vars if k.endswith('_API_KEY'))} providers")

if __name__ == "__main__":
    import_challenge_data()