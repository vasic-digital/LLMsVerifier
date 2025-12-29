#!/usr/bin/env python3
"""
Fix OpenCode configuration to add required fields and (llmsvd) suffix
"""
import json
import sys

def fix_opencode_config(input_file, output_file):
    """Fix OpenCode configuration with required fields and suffixes"""
    
    with open(input_file, 'r') as f:
        config = json.load(f)
    
    # Fix provider structure - add id and npm fields to each provider
    if 'provider' in config:
        for provider_key, provider_data in config['provider'].items():
            # Add required provider fields
            if 'id' not in provider_data:
                provider_data['id'] = provider_key
            if 'npm' not in provider_data:
                # Add npm package name with llmsvd suffix
                provider_data['npm'] = f"@llmsvd/{provider_key}-provider"
            
            # Fix baseURL formats for specific providers
            if 'options' in provider_data and 'baseURL' in provider_data['options']:
                base_url = provider_data['options']['baseURL']
                
                # Fix Perplexity baseURL - must contain /v1
                if provider_key == 'perplexity' and '/v1' not in base_url:
                    if base_url.endswith('/'):
                        provider_data['options']['baseURL'] = base_url + 'v1'
                    else:
                        provider_data['options']['baseURL'] = base_url + '/v1'
            
            # Fix models - add id field and (llmsvd) suffix to names
            if 'models' in provider_data:
                for model_key, model_data in provider_data['models'].items():
                    # Add required id field
                    if 'id' not in model_data:
                        model_data['id'] = model_key
                    
                    # Add required displayName field
                    if 'displayName' not in model_data:
                        model_data['displayName'] = model_data.get('name', model_key)
                    
                    # Add required provider field (as object with id and npm)
                    if 'provider' not in model_data:
                        model_data['provider'] = {
                            "id": provider_key,
                            "npm": f"@llmsvd/{provider_key}-provider"
                        }
                    
                    # Add required supportsHTTP3 field (default to true)
                    if 'supportsHTTP3' not in model_data:
                        model_data['supportsHTTP3'] = True
                    
                    # Add required supportsWebSocket field (default to true)
                    if 'supportsWebSocket' not in model_data:
                        model_data['supportsWebSocket'] = True
                    
                    # Add (llmsvd) suffix to model names if not already present
                    if 'name' in model_data and '(llmsvd)' not in model_data['name']:
                        model_data['name'] = f"{model_data['name']} (llmsvd)"
                    
                    # Also add (llmsvd) to displayName if not present
                    if 'displayName' in model_data and '(llmsvd)' not in model_data['displayName']:
                        model_data['displayName'] = f"{model_data['displayName']} (llmsvd)"
    
    # Fix agent models to use the correct format with llmsvd suffix
    if 'agent' in config:
        for agent_key, agent_data in config['agent'].items():
            if 'model' in agent_data:
                # Ensure the model reference uses the correct format
                model_ref = agent_data['model']
                if '/' in model_ref:
                    provider, model = model_ref.split('/', 1)
                    # The model name in the config already has (llmsvd) suffix
                    # so we just need to ensure the reference is correct
                    agent_data['model'] = f"{provider}/{model}"
    
    # Write the fixed configuration
    with open(output_file, 'w') as f:
        json.dump(config, f, indent=2)
    
    print(f"✅ Fixed OpenCode configuration saved to {output_file}")
    
    # Print statistics
    provider_count = len(config.get('provider', {}))
    total_models = sum(len(provider.get('models', {})) for provider in config.get('provider', {}).values())
    agent_count = len(config.get('agent', {}))
    
    print(f"📊 Configuration Statistics:")
    print(f"   • Providers: {provider_count}")
    print(f"   • Total Models: {total_models}")
    print(f"   • Agents: {agent_count}")
    print(f"   • MCP Servers: {len(config.get('mcp', {}))}")
    print(f"   • Commands: {len(config.get('command', {}))}")
    print(f"   • LSP Servers: {len(config.get('lsp', {}))}")
    
    return config

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: python fix_opencode_final.py <input_file> <output_file>")
        sys.exit(1)
    
    input_file = sys.argv[1]
    output_file = sys.argv[2]
    
    try:
        config = fix_opencode_config(input_file, output_file)
        print("✅ Configuration fixed successfully!")
    except Exception as e:
        print(f"❌ Error fixing configuration: {e}")
        sys.exit(1)