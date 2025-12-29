#!/usr/bin/env python3
"""
Generate a VALID OpenCode configuration following the official schema.
This version removes the custom extensions and follows the exact OpenCode format.
"""

import json
import os
from datetime import datetime

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

def load_extracted_models():
    """Load the extracted models from challenge log."""
    try:
        with open('challenge_models_extracted.json', 'r') as f:
            return json.load(f)
    except FileNotFoundError:
        print("❌ challenge_models_extracted.json not found. Run parse_challenge_log_fixed.py first.")
        return {}

def get_provider_config(provider_name, api_keys):
    """Get provider configuration in valid OpenCode format."""
    
    # Map provider names to their proper configurations
    provider_configs = {
        'openai': {
            'id': 'openai',
            'npm': '@opencode/openai-provider',
            'options': {
                'apiKey': api_keys.get('openai', '${OPENAI_API_KEY}'),
                'baseURL': 'https://api.openai.com/v1'
            }
        },
        'anthropic': {
            'id': 'anthropic',
            'npm': '@opencode/anthropic-provider',
            'options': {
                'apiKey': api_keys.get('anthropic', '${ANTHROPIC_API_KEY}'),
                'baseURL': 'https://api.anthropic.com/v1'
            }
        },
        'groq': {
            'id': 'groq',
            'npm': '@opencode/groq-provider',
            'options': {
                'apiKey': api_keys.get('groq', '${GROQ_API_KEY}'),
                'baseURL': 'https://api.groq.com/openai/v1'
            }
        },
        'huggingface': {
            'id': 'huggingface',
            'npm': '@opencode/huggingface-provider',
            'options': {
                'apiKey': api_keys.get('huggingface', '${HUGGINGFACE_API_KEY}'),
                'baseURL': 'https://api-inference.huggingface.co'
            }
        },
        'gemini': {
            'id': 'gemini',
            'npm': '@opencode/gemini-provider',
            'options': {
                'apiKey': api_keys.get('gemini', '${GEMINI_API_KEY}'),
                'baseURL': 'https://generativelanguage.googleapis.com/v1'
            }
        },
        'deepseek': {
            'id': 'deepseek',
            'npm': '@opencode/deepseek-provider',
            'options': {
                'apiKey': api_keys.get('deepseek', '${DEEPSEEK_API_KEY}'),
                'baseURL': 'https://api.deepseek.com'
            }
        },
        'openrouter': {
            'id': 'openrouter',
            'npm': '@opencode/openrouter-provider',
            'options': {
                'apiKey': api_keys.get('openrouter', '${OPENROUTER_API_KEY}'),
                'baseURL': 'https://openrouter.ai/api/v1'
            }
        },
        'perplexity': {
            'id': 'perplexity',
            'npm': '@opencode/perplexity-provider',
            'options': {
                'apiKey': api_keys.get('perplexity', '${PERPLEXITY_API_KEY}'),
                'baseURL': 'https://api.perplexity.ai'
            }
        },
        'together': {
            'id': 'together',
            'npm': '@opencode/together-provider',
            'options': {
                'apiKey': api_keys.get('together', '${TOGETHER_API_KEY}'),
                'baseURL': 'https://api.together.xyz/v1'
            }
        },
        'mistral': {
            'id': 'mistral',
            'npm': '@opencode/mistral-provider',
            'options': {
                'apiKey': api_keys.get('mistral', '${MISTRAL_API_KEY}'),
                'baseURL': 'https://api.mistral.ai/v1'
            }
        },
        'fireworks': {
            'id': 'fireworks',
            'npm': '@opencode/fireworks-provider',
            'options': {
                'apiKey': api_keys.get('fireworks', '${FIREWORKS_API_KEY}'),
                'baseURL': 'https://api.fireworks.ai/inference/v1'
            }
        },
        'nvidia': {
            'id': 'nvidia',
            'npm': '@opencode/nvidia-provider',
            'options': {
                'apiKey': api_keys.get('nvidia', '${NVIDIA_API_KEY}'),
                'baseURL': 'https://integrate.api.nvidia.com/v1'
            }
        },
        'cerebras': {
            'id': 'cerebras',
            'npm': '@opencode/cerebras-provider',
            'options': {
                'apiKey': api_keys.get('cerebras', '${CEREBRAS_API_KEY}'),
                'baseURL': 'https://api.cerebras.ai/v1'
            }
        },
        'hyperbolic': {
            'id': 'hyperbolic',
            'npm': '@opencode/hyperbolic-provider',
            'options': {
                'apiKey': api_keys.get('hyperbolic', '${HYPERBOLIC_API_KEY}'),
                'baseURL': 'https://api.hyperbolic.xyz/v1'
            }
        },
        'huggingface': {
            'id': 'huggingface',
            'npm': '@opencode/huggingface-provider',
            'options': {
                'apiKey': api_keys.get('huggingface', '${HUGGINGFACE_API_KEY}'),
                'baseURL': 'https://api-inference.huggingface.co'
            }
        },
        'inference': {
            'id': 'inference',
            'npm': '@opencode/inference-provider',
            'options': {
                'apiKey': api_keys.get('inference', '${INFERENCE_API_KEY}'),
                'baseURL': 'https://api.inference.net/v1'
            }
        },
        'vercel': {
            'id': 'vercel',
            'npm': '@opencode/vercel-provider',
            'options': {
                'apiKey': api_keys.get('vercel', '${VERCEL_API_KEY}'),
                'baseURL': 'https://api.vercel.com/v1'
            }
        },
        'baseten': {
            'id': 'baseten',
            'npm': '@opencode/baseten-provider',
            'options': {
                'apiKey': api_keys.get('baseten', '${BASETEN_API_KEY}'),
                'baseURL': 'https://inference.baseten.co/v1'
            }
        },
        'novita': {
            'id': 'novita',
            'npm': '@opencode/novita-provider',
            'options': {
                'apiKey': api_keys.get('novita', '${NOVITA_API_KEY}'),
                'baseURL': 'https://api.novita.ai/v3/openai'
            }
        },
        'upstage': {
            'id': 'upstage',
            'npm': '@opencode/upstage-provider',
            'options': {
                'apiKey': api_keys.get('upstage', '${UPSTAGE_API_KEY}'),
                'baseURL': 'https://api.upstage.ai/v1'
            }
        },
        'nlpcloud': {
            'id': 'nlpcloud',
            'npm': '@opencode/nlpcloud-provider',
            'options': {
                'apiKey': api_keys.get('nlpcloud', '${NLPCLOUD_API_KEY}'),
                'baseURL': 'https://api.nlpcloud.com/v1'
            }
        },
        'modal': {
            'id': 'modal',
            'npm': '@opencode/modal-provider',
            'options': {
                'apiKey': api_keys.get('modal', '${MODAL_API_KEY}'),
                'baseURL': 'https://api.modal.com/v1'
            }
        },
        'chutes': {
            'id': 'chutes',
            'npm': '@opencode/chutes-provider',
            'options': {
                'apiKey': api_keys.get('chutes', '${CHUTES_API_KEY}'),
                'baseURL': 'https://api.chutes.ai/v1'
            }
        },
        'cloudflare': {
            'id': 'cloudflare',
            'npm': '@opencode/cloudflare-provider',
            'options': {
                'apiKey': api_keys.get('cloudflare', '${CLOUDFLARE_API_KEY}'),
                'baseURL': 'https://api.cloudflare.com/client/v4/ai/inference'
            }
        },
        'siliconflow': {
            'id': 'siliconflow',
            'npm': '@opencode/siliconflow-provider',
            'options': {
                'apiKey': api_keys.get('siliconflow', '${SILICONFLOW_API_KEY}'),
                'baseURL': 'https://api.siliconflow.cn/v1'
            }
        },
        'kimi': {
            'id': 'kimi',
            'npm': '@opencode/kimi-provider',
            'options': {
                'apiKey': api_keys.get('kimi', '${KIMI_API_KEY}'),
                'baseURL': 'https://api.moonshot.cn/v1'
            }
        },
        'zai': {
            'id': 'zai',
            'npm': '@opencode/zai-provider',
            'options': {
                'apiKey': api_keys.get('zai', '${ZAI_API_KEY}'),
                'baseURL': 'https://api.z.ai/v1'
            }
        },
        'sambanova': {
            'id': 'sambanova',
            'npm': '@opencode/sambanova-provider',
            'options': {
                'apiKey': api_keys.get('sambanova', '${SAMBANOVA_API_KEY}'),
                'baseURL': 'https://api.sambanova.ai/v1'
            }
        },
        'replicate': {
            'id': 'replicate',
            'npm': '@opencode/replicate-provider',
            'options': {
                'apiKey': api_keys.get('replicate', '${REPLICATE_API_KEY}'),
                'baseURL': 'https://api.replicate.com/v1'
            }
        },
        'sarvam': {
            'id': 'sarvam',
            'npm': '@opencode/sarvam-provider',
            'options': {
                'apiKey': api_keys.get('sarvam', '${SARVAM_API_KEY}'),
                'baseURL': 'https://api.sarvam.ai'
            }
        },
        'vulavula': {
            'id': 'vulavula',
            'npm': '@opencode/vulavula-provider',
            'options': {
                'apiKey': api_keys.get('vulavula', '${VULAVULA_API_KEY}'),
                'baseURL': 'https://api.lelapa.ai'
            }
        },
        'twelvelabs': {
            'id': 'twelvelabs',
            'npm': '@opencode/twelvelabs-provider',
            'options': {
                'apiKey': api_keys.get('twelvelabs', '${TWELVELABS_API_KEY}'),
                'baseURL': 'https://api.twelvelabs.io/v1'
            }
        },
        'codestral': {
            'id': 'codestral',
            'npm': '@opencode/codestral-provider',
            'options': {
                'apiKey': api_keys.get('codestral', '${CODESTRAL_API_KEY}'),
                'baseURL': 'https://codestral.mistral.ai/v1'
            }
        }
    }
    
    return provider_configs.get(provider_name, {
        'id': provider_name,
        'npm': f'@opencode/{provider_name}-provider',
        'options': {
            'apiKey': f'${{{provider_name.upper()}_API_KEY}}',
            'baseURL': f'https://api.{provider_name}.com/v1'
        }
    })

def create_model_entry(model_id, provider_name):
    """Create a model entry following OpenCode schema."""
    
    # Extract base model name for display
    display_name = model_id.split('/')[-1].replace('-', ' ').title()
    
    # Determine capabilities based on name patterns
    supports_brotli = True  # Most modern models support Brotli
    supports_http3 = True   # Most support HTTP/3
    supports_websocket = True  # Most support WebSocket
    
    # Estimate max tokens based on model type
    if '32k' in model_id or '128k' in model_id:
        max_tokens = 128000
    elif '8k' in model_id:
        max_tokens = 8192
    elif 'gpt-4' in model_id:
        max_tokens = 8192
    elif 'claude' in model_id:
        max_tokens = 200000
    else:
        max_tokens = 4096
    
    # Estimate costs (simplified)
    cost_in = 0.5
    cost_out = 1.5
    
    if 'gpt-4' in model_id:
        cost_in = 30.0
        cost_out = 60.0
    elif 'claude-3-opus' in model_id:
        cost_in = 15.0
        cost_out = 75.0
    elif 'claude-3-sonnet' in model_id:
        cost_in = 3.0
        cost_out = 15.0
    elif 'claude-3-haiku' in model_id:
        cost_in = 0.25
        cost_out = 1.25
    
    return {
        'id': model_id,
        'name': f'{display_name} (Verified)',
        'displayName': f'{display_name} (Challenge Verified)',
        'maxTokens': max_tokens,
        'cost_per_1m_in': cost_in,
        'cost_per_1m_out': cost_out,
        'supportsBrotli': supports_brotli,
        'supportsHTTP3': supports_http3,
        'supportsWebSocket': supports_websocket,
        'provider': {
            'id': provider_name,
            'npm': f'@opencode/{provider_name}-provider'
        }
    }

def generate_valid_opencode_config():
    """Generate a valid OpenCode configuration following the official schema."""
    
    print("🎯 Generating VALID OpenCode configuration...")
    print("=" * 60)
    
    # Load extracted models
    provider_models = load_extracted_models()
    if not provider_models:
        return None
    
    print(f"✅ Loaded {sum(len(models) for models in provider_models.values())} models from challenge data")
    
    # Load API keys
    api_keys = load_env_api_keys()
    print(f"✅ Loaded {len(api_keys)} API keys from .env")
    
    # Build valid OpenCode configuration
    config = {
        '$schema': 'https://opencode.sh/schema.json',
        'username': 'OpenCode AI Assistant (Ultimate Challenge)',
        'provider': {}
    }
    
    # Build provider configurations
    valid_providers = 0
    total_models = 0
    
    for provider_name, models in provider_models.items():
        if not models:  # Skip providers with no models
            continue
            
        provider_config = get_provider_config(provider_name, api_keys)
        provider_config['models'] = {}
        
        for model_id in models:
            model_config = create_model_entry(model_id, provider_name)
            provider_config['models'][model_id] = model_config
            total_models += 1
        
        config['provider'][provider_name] = provider_config
        valid_providers += 1
    
    # Add other standard OpenCode sections (minimal required)
    config.update({
        'agent': {
            'name': 'OpenCode AI Assistant',
            'version': '1.0.0'
        },
        'mcp': {
            'servers': []
        },
        'command': {
            'timeout': 30000
        },
        'keybinds': {
            'toggleSidebar': 'ctrl+b',
            'quickCommand': 'ctrl+shift+p'
        },
        'options': {
            'theme': 'dark',
            'autoSave': True,
            'enableChallengeVerification': True
        },
        'tools': {
            'enabled': True,
            'maxTools': 10
        },
        'lsp': {
            'enabled': True,
            'servers': []
        }
    })
    
    return config

def main():
    """Main function to generate and save the valid configuration."""
    
    print("🔧 Generating VALID OpenCode Configuration")
    print("=" * 60)
    print("Following official OpenCode schema - removing custom extensions")
    
    # Generate configuration
    config = generate_valid_opencode_config()
    
    if not config:
        return
    
    # Save configuration
    output_file = 'opencode_valid_final.json'
    
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
    
    print("\n📊 VALID CONFIGURATION SUMMARY")
    print("=" * 60)
    print(f"Total Providers: {total_providers}")
    print(f"Total Models: {total_models}")
    print(f"Schema: {config.get('$schema')}")
    print(f"Username: {config.get('username')}")
    
    # Show provider breakdown
    print(f"\n🔍 Provider Breakdown:")
    for provider_name, provider_data in sorted(providers.items()):
        model_count = len(provider_data.get('models', {}))
        print(f"  {provider_name}: {model_count} models")
    
    print(f"\n⚠️  SECURITY NOTICE: This file contains API keys!")
    print(f"📋 Use this configuration with OpenCode - it follows the official schema")

if __name__ == "__main__":
    main()