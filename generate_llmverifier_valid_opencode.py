#!/usr/bin/env python3
"""
Generate VALID OpenCode configuration for llm-verifier binary.
Based on the exact schema expected by llm-verifier from Go types.
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

def get_provider_config_llmverifier(provider_name, api_keys):
    """Get provider configuration for llm-verifier (based on Go types)."""
    
    provider_configs = {
        'openai': {
            'options': {
                'apiKey': api_keys.get('openai', '${OPENAI_API_KEY}'),
                'baseURL': 'https://api.openai.com/v1'
            }
        },
        'anthropic': {
            'options': {
                'apiKey': api_keys.get('anthropic', '${ANTHROPIC_API_KEY}'),
                'baseURL': 'https://api.anthropic.com/v1'
            }
        },
        'groq': {
            'options': {
                'apiKey': api_keys.get('groq', '${GROQ_API_KEY}'),
                'baseURL': 'https://api.groq.com/openai/v1'
            }
        },
        'huggingface': {
            'options': {
                'apiKey': api_keys.get('huggingface', '${HUGGINGFACE_API_KEY}'),
                'baseURL': 'https://api-inference.huggingface.co'
            }
        },
        'gemini': {
            'options': {
                'apiKey': api_keys.get('gemini', '${GEMINI_API_KEY}'),
                'baseURL': 'https://generativelanguage.googleapis.com/v1'
            }
        },
        'deepseek': {
            'options': {
                'apiKey': api_keys.get('deepseek', '${DEEPSEEK_API_KEY}'),
                'baseURL': 'https://api.deepseek.com'
            }
        },
        'openrouter': {
            'options': {
                'apiKey': api_keys.get('openrouter', '${OPENROUTER_API_KEY}'),
                'baseURL': 'https://openrouter.ai/api/v1'
            }
        },
        'perplexity': {
            'options': {
                'apiKey': api_keys.get('perplexity', '${PERPLEXITY_API_KEY}'),
                'baseURL': 'https://api.perplexity.ai'
            }
        },
        'together': {
            'options': {
                'apiKey': api_keys.get('together', '${TOGETHER_API_KEY}'),
                'baseURL': 'https://api.together.xyz/v1'
            }
        },
        'mistral': {
            'options': {
                'apiKey': api_keys.get('mistral', '${MISTRAL_API_KEY}'),
                'baseURL': 'https://api.mistral.ai/v1'
            }
        },
        'fireworks': {
            'options': {
                'apiKey': api_keys.get('fireworks', '${FIREWORKS_API_KEY}'),
                'baseURL': 'https://api.fireworks.ai/inference/v1'
            }
        },
        'nvidia': {
            'options': {
                'apiKey': api_keys.get('nvidia', '${NVIDIA_API_KEY}'),
                'baseURL': 'https://integrate.api.nvidia.com/v1'
            }
        },
        'cerebras': {
            'options': {
                'apiKey': api_keys.get('cerebras', '${CEREBRAS_API_KEY}'),
                'baseURL': 'https://api.cerebras.ai/v1'
            }
        },
        'hyperbolic': {
            'options': {
                'apiKey': api_keys.get('hyperbolic', '${HYPERBOLIC_API_KEY}'),
                'baseURL': 'https://api.hyperbolic.xyz/v1'
            }
        },
        'inference': {
            'options': {
                'apiKey': api_keys.get('inference', '${INFERENCE_API_KEY}'),
                'baseURL': 'https://api.inference.net/v1'
            }
        },
        'vercel': {
            'options': {
                'apiKey': api_keys.get('vercel', '${VERCEL_API_KEY}'),
                'baseURL': 'https://api.vercel.com/v1'
            }
        },
        'baseten': {
            'options': {
                'apiKey': api_keys.get('baseten', '${BASETEN_API_KEY}'),
                'baseURL': 'https://inference.baseten.co/v1'
            }
        },
        'novita': {
            'options': {
                'apiKey': api_keys.get('novita', '${NOVITA_API_KEY}'),
                'baseURL': 'https://api.novita.ai/v3/openai'
            }
        },
        'upstage': {
            'options': {
                'apiKey': api_keys.get('upstage', '${UPSTAGE_API_KEY}'),
                'baseURL': 'https://api.upstage.ai/v1'
            }
        },
        'nlpcloud': {
            'options': {
                'apiKey': api_keys.get('nlpcloud', '${NLPCLOUD_API_KEY}'),
                'baseURL': 'https://api.nlpcloud.com/v1'
            }
        },
        'modal': {
            'options': {
                'apiKey': api_keys.get('modal', '${MODAL_API_KEY}'),
                'baseURL': 'https://api.modal.com/v1'
            }
        },
        'chutes': {
            'options': {
                'apiKey': api_keys.get('chutes', '${CHUTES_API_KEY}'),
                'baseURL': 'https://api.chutes.ai/v1'
            }
        },
        'cloudflare': {
            'options': {
                'apiKey': api_keys.get('cloudflare', '${CLOUDFLARE_API_KEY}'),
                'baseURL': 'https://api.cloudflare.com/client/v4/ai/inference'
            }
        },
        'siliconflow': {
            'options': {
                'apiKey': api_keys.get('siliconflow', '${SILICONFLOW_API_KEY}'),
                'baseURL': 'https://api.siliconflow.cn/v1'
            }
        },
        'kimi': {
            'options': {
                'apiKey': api_keys.get('kimi', '${KIMI_API_KEY}'),
                'baseURL': 'https://api.moonshot.cn/v1'
            }
        },
        'zai': {
            'options': {
                'apiKey': api_keys.get('zai', '${ZAI_API_KEY}'),
                'baseURL': 'https://api.z.ai/v1'
            }
        },
        'sambanova': {
            'options': {
                'apiKey': api_keys.get('sambanova', '${SAMBANOVA_API_KEY}'),
                'baseURL': 'https://api.sambanova.ai/v1'
            }
        },
        'replicate': {
            'options': {
                'apiKey': api_keys.get('replicate', '${REPLICATE_API_KEY}'),
                'baseURL': 'https://api.replicate.com/v1'
            }
        },
        'sarvam': {
            'options': {
                'apiKey': api_keys.get('sarvam', '${SARVAM_API_KEY}'),
                'baseURL': 'https://api.sarvam.ai'
            }
        },
        'vulavula': {
            'options': {
                'apiKey': api_keys.get('vulavula', '${VULAVULA_API_KEY}'),
                'baseURL': 'https://api.lelapa.ai'
            }
        },
        'twelvelabs': {
            'options': {
                'apiKey': api_keys.get('twelvelabs', '${TWELVELABS_API_KEY}'),
                'baseURL': 'https://api.twelvelabs.io/v1'
            }
        },
        'codestral': {
            'options': {
                'apiKey': api_keys.get('codestral', '${CODESTRAL_API_KEY}'),
                'baseURL': 'https://codestral.mistral.ai/v1'
            }
        }
    }
    
    return provider_configs.get(provider_name, {
        'options': {
            'apiKey': f'${{{provider_name.upper()}_API_KEY}}',
            'baseURL': f'https://api.{provider_name}.com/v1'
        }
    })

def generate_llmverifier_valid_config():
    """Generate VALID OpenCode configuration for llm-verifier binary."""
    
    print("🎯 Generating VALID OpenCode configuration for llm-verifier binary...")
    print("=" * 60)
    
    # Load extracted models
    provider_models = load_extracted_models()
    if not provider_models:
        return None
    
    print(f"✅ Loaded {sum(len(models) for models in provider_models.values())} models from challenge data")
    
    # Load API keys
    api_keys = load_env_api_keys()
    print(f"✅ Loaded {len(api_keys)} API keys from .env")
    
    # Build configuration following llm-verifier's exact schema
    config = {
        # Use the schema URL that llm-verifier expects
        '$schema': 'https://opencode.ai/config.json',
        'version': '1.0',
        'username': 'OpenCode AI Assistant (Ultimate Challenge)',
        'provider': {}
    }
    
    # Build provider configurations
    valid_providers = 0
    total_models = 0
    
    for provider_name, models in provider_models.items():
        if not models:  # Skip providers with no models
            continue
                
        provider_config = get_provider_config_llmverifier(provider_name, api_keys)
        provider_config['models'] = {}
        
        for model_id in models:
            # For llm-verifier, we need to determine if this model should be the primary model
            # Let's use the first model as the primary model for each provider
            if 'model' not in provider_config:
                provider_config['model'] = model_id
            
            # According to llm-verifier, models field should be empty object per OpenCode spec
            # So we just set it to empty object and track the count
            provider_config['models'] = {}
            total_models += 1
        
        config['provider'][provider_name] = provider_config
        valid_providers += 1
    
    return config

def main():
    """Main function to generate and save the configuration."""
    
    print("🔧 Generating VALID OpenCode Configuration for llm-verifier")
    print("=" * 60)
    print("Following llm-verifier's exact Go types schema")
    
    # Generate configuration
    config = generate_llmverifier_valid_config()
    
    if not config:
        return
    
    # Save configuration
    output_file = 'opencode_llmverifier_valid.json'
    
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
    
    print(f"✅ Valid configuration saved to: {output_file}")
    print(f"🔒 Set file permissions to 600 (owner read/write only)")
    
    # Display summary
    providers = config.get('provider', {})
    total_providers = len(providers)
    total_models = sum(len(provider_data.get('models', {})) for provider_data in providers.values())
    
    print("\n📊 LLM-VERIFIER VALID CONFIGURATION SUMMARY")
    print("=" * 60)
    print(f"Total Providers: {total_providers}")
    print(f"Total Models: {total_models}")
    print(f"Schema: {config.get('$schema')}")
    print(f"Username: {config.get('username')}")
    
    # Show provider breakdown
    print(f"\n🔍 Provider Breakdown:")
    for provider_name, provider_data in sorted(providers.items()):
        model_count = len(provider_data.get('models', {}))
        primary_model = provider_data.get('model', 'None')
        print(f"  {provider_name}: {model_count} models (primary: {primary_model})")
    
    print(f"\n⚠️  SECURITY NOTICE: This file contains API keys!")
    print(f"📋 Use this configuration with llm-verifier - it follows their exact schema")
    
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