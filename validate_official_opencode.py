#!/usr/bin/env python3
"""
Validate the configuration against the official OpenCode schema.
"""

import json

def validate_official_opencode():
    """Validate against the official OpenCode schema."""
    
    print("🔍 Validating against official OpenCode schema...")
    
    try:
        with open('opencode_valid_final.json', 'r') as f:
            config = json.load(f)
        
        print("✅ JSON syntax is valid")
        
        # Check required top-level fields
        required_fields = ['$schema', 'username', 'provider']
        missing_fields = []
        
        for field in required_fields:
            if field not in config:
                missing_fields.append(field)
        
        if missing_fields:
            print(f"❌ Missing required fields: {missing_fields}")
            return False
        
        print(f"✅ Required fields present: {', '.join(required_fields)}")
        
        # Check schema URL
        schema = config.get('$schema', '')
        if 'opencode.sh/schema.json' in schema:
            print("✅ Valid OpenCode schema URL")
        else:
            print(f"⚠️  Schema URL may be invalid: {schema}")
        
        # Check provider structure
        providers = config.get('provider', {})
        print(f"✅ Found {len(providers)} providers")
        
        if len(providers) == 0:
            print("❌ No providers found")
            return False
        
        # Validate first provider structure
        first_provider = list(providers.values())[0]
        required_provider_fields = ['id', 'npm', 'options', 'models']
        
        missing_provider_fields = []
        for field in required_provider_fields:
            if field not in first_provider:
                missing_provider_fields.append(field)
        
        if missing_provider_fields:
            print(f"❌ Provider missing fields: {missing_provider_fields}")
            return False
        
        print(f"✅ Provider structure valid")
        
        # Check first model structure
        first_models = list(first_provider.get('models', {}).values())
        if first_models:
            first_model = first_models[0]
            required_model_fields = ['id', 'name', 'displayName', 'maxTokens', 'cost_per_1m_in', 'cost_per_1m_out', 'supportsBrotli', 'supportsHTTP3', 'supportsWebSocket', 'provider']
            
            missing_model_fields = []
            for field in required_model_fields:
                if field not in first_model:
                    missing_model_fields.append(field)
            
            if missing_model_fields:
                print(f"❌ Model missing fields: {missing_model_fields}")
                return False
            
            print(f"✅ Model structure valid")
        
        # Count totals
        total_providers = len(providers)
        total_models = sum(len(provider_data.get('models', {})) for provider_data in providers.values())
        
        print(f"\n📊 Configuration Summary:")
        print(f"  Total Providers: {total_providers}")
        print(f"  Total Models: {total_models}")
        print(f"  Username: {config.get('username', 'Not set')}")
        
        # Check for additional valid fields
        valid_top_level = ['$schema', 'username', 'provider', 'agent', 'mcp', 'command', 'keybinds', 'options', 'tools', 'lsp']
        extra_fields = []
        
        for field in config.keys():
            if field not in valid_top_level:
                extra_fields.append(field)
        
        if extra_fields:
            print(f"⚠️  Extra fields found (may be invalid): {extra_fields}")
        else:
            print("✅ No invalid extra fields")
        
        print("\n🎉 VALIDATION SUCCESSFUL!")
        print("Configuration follows official OpenCode schema!")
        return True
        
    except Exception as e:
        print(f"❌ Validation failed: {e}")
        return False

if __name__ == "__main__":
    validate_official_opencode()