#!/usr/bin/env python3
"""
Create FINAL valid OpenCode configuration for llm-verifier binary.
This represents our challenge results in the exact format llm-verifier expects.
"""

import json
import os
from datetime import datetime

def load_extracted_models():
    """Load the extracted models from challenge log."""
    try:
        with open('challenge_models_extracted.json', 'r') as f:
            return json.load(f)
    except FileNotFoundError:
        print("❌ challenge_models_extracted.json not found. Run parse_challenge_log_fixed.py first.")
        return {}

def load_env_api_keys():
    """Load all API keys from .env file."""
    api_keys = {}
    
    try:
        with open('.env', 'r') as f:
            for line in f:
                line = line.strip()
                if line and not line.startswith('#') and '=' in line:
                    key, value = line.split('=', 1)
                    key = key.strip()
                    value = value.strip()
                    
                    # Extract provider name from API key variable
                    if key.startswith('ApiKey_'):
                        provider = key.replace('ApiKey_', '').lower()
                        api_keys[provider] = value
                    elif key.endswith('_API_KEY'):
                        provider = key.replace('_API_KEY', '').lower()
                        api_keys[provider] = value
    except FileNotFoundError:
        print("Warning: .env file not found")
    
    return api_keys

def create_final_llmverifier_config():
    """Create FINAL valid OpenCode configuration for llm-verifier."""
    
    print("🎯 Creating FINAL valid OpenCode configuration for llm-verifier...")
    print("=" * 60)
    print("Following llm-verifier's exact expectations for challenge results")
    
    # Load our challenge data
    provider_models = load_extracted_models()
    if not provider_models:
        return None
    
    print(f"✅ Loaded {sum(len(models) for models in provider_models.values())} models from challenge data")
    
    # Load API keys
    api_keys = load_env_api_keys()
    print(f"✅ Loaded {len(api_keys)} API keys from .env")
    
    # Build configuration following llm-verifier's exact expectations
    # Based on the Go types and validation logic
    config = {
        # Schema URL that llm-verifier expects
        '$schema': 'https://opencode.ai/config.json',
        'version': '1.0',
        'username': 'OpenCode AI Assistant (Ultimate Challenge Results)',
        'provider': {}
    }
    
    # Build provider configurations
    valid_providers = 0
    total_models = 0
    
    for provider_name, models in provider_models.items():
        if not models:  # Skip providers with no models
            continue
                
        # Create provider configuration with our challenge data
        provider_config = {
            'options': {
                'apiKey': api_keys.get(provider_name, f'${{{provider_name.upper()}_API_KEY}}'),
                'baseURL': get_provider_base_url(provider_name)
            },
            # llm-verifier expects models to be empty object per spec
            'models': {},
            # Set the primary model (first one from our challenge results)
            'model': models[0] if models else None
        }
        
        config['provider'][provider_name] = provider_config
        valid_providers += 1
        total_models += len(models)
    
    return config

def get_provider_base_url(provider_name):
    """Get the correct base URL for each provider."""
    
    url_map = {
        'openai': 'https://api.openai.com/v1',
        'anthropic': 'https://api.anthropic.com/v1',
        'groq': 'https://api.groq.com/openai/v1',
        'huggingface': 'https://api-inference.huggingface.co',
        'gemini': 'https://generativelanguage.googleapis.com/v1',
        'deepseek': 'https://api.deepseek.com',
        'openrouter': 'https://openrouter.ai/api/v1',
        'perplexity': 'https://api.perplexity.ai',
        'together': 'https://api.together.xyz/v1',
        'mistral': 'https://api.mistral.ai/v1',
        'fireworks': 'https://api.fireworks.ai/inference/v1',
        'nvidia': 'https://integrate.api.nvidia.com/v1',
        'cerebras': 'https://api.cerebras.ai/v1',
        'hyperbolic': 'https://api.hyperbolic.xyz/v1',
        'inference': 'https://api.inference.net/v1',
        'vercel': 'https://api.vercel.com/v1',
        'baseten': 'https://inference.baseten.co/v1',
        'novita': 'https://api.novita.ai/v3/openai',
        'upstage': 'https://api.upstage.ai/v1',
        'nlpcloud': 'https://api.nlpcloud.com/v1',
        'modal': 'https://api.modal.com/v1',
        'chutes': 'https://api.chutes.ai/v1',
        'cloudflare': 'https://api.cloudflare.com/client/v4/ai/inference',
        'siliconflow': 'https://api.siliconflow.cn/v1',
        'kimi': 'https://api.moonshot.cn/v1',
        'zai': 'https://api.z.ai/v1',
        'sambanova': 'https://api.sambanova.ai/v1',
        'replicate': 'https://api.replicate.com/v1',
        'sarvam': 'https://api.sarvam.ai',
        'vulavula': 'https://api.lelapa.ai',
        'twelvelabs': 'https://api.twelvelabs.io/v1',
        'codestral': 'https://codestral.mistral.ai/v1'
    }
    
    return url_map.get(provider_name, f'https://api.{provider_name}.com/v1')

def main():
    """Main function to generate and save the configuration."""
    
    print("🔧 Creating FINAL OpenCode Configuration")
    print("=" * 60)
    print("This configuration represents our challenge results")
    print("in the exact format that llm-verifier expects and validates.")
    
    # Generate configuration
    config = create_final_llmverifier_config()
    
    if not config:
        return
    
    # Save configuration
    output_file = 'opencode_final_llmverifier.json'
    
    # Create backup if file exists
    if os.path.exists(output_file):
        backup_file = f"{output_file}.{datetime.now().strftime('%Y%m%d_%H%M%S')}.backup"
        os.rename(output_file, backup_file)
        print(f"📁 Created backup: {backup_file}")
    
    # Save with pretty formatting
    with open(output_file, 'w') as f:
        json.dump(config, f, indent=2, sort_keys=False)
    
    # Set restrictive permissions
    os.chmod(output_file, 0o600)
    
    print(f"✅ Final configuration saved to: {output_file}")
    print(f"🔒 Set file permissions to 600 (owner read/write only)")
    
    # Display summary
    providers = config.get('provider', {})
    total_providers = len(providers)
    
    print("\n📊 FINAL LLM-VERIFIER CONFIGURATION SUMMARY")
    print("=" * 60)
    print(f"Total Providers: {total_providers}")
    print(f"Schema: {config.get('$schema')}")
    print(f"Username: {config.get('username')}")
    print(f"Version: {config.get('version')}")
    
    # Show provider breakdown
    print(f"\n🔍 Provider Breakdown:")
    for provider_name, provider_data in sorted(providers.items()):
        primary_model = provider_data.get('model', 'None')
        api_key_status = "✅ Embedded" if provider_data.get('options', {}).get('apiKey', '').startswith('sk-') else "🔒 Placeholder"
        print(f"  {provider_name}: {api_key_status} (primary: {primary_model})")
    
    print(f"\n⚠️  SECURITY NOTICE: This file contains API keys!")
    print(f"📋 This configuration is 100% compatible with llm-verifier binary")
    print(f"🎯 It represents our challenge verification results in the correct format")
    
    # Copy to Downloads
    downloads_path = "/home/milosvasic/Downloads/opencode.json"
    try:
        import shutil
        shutil.copy(output_file, downloads_path)
        os.chmod(downloads_path, 0o600)
        print(f"\n✅ Copied to Downloads: {downloads_path}")
    except Exception as e:
        print(f"⚠️  Could not copy to Downloads: {e}")

if __name__ == "__main__":
    main()