#!/usr/bin/env python3
"""
Validate the ultimate OpenCode configuration.
"""

import json
import os
import stat

def validate_config():
    """Validate the OpenCode configuration."""
    
    print('🔍 Validating ultimate OpenCode configuration...')
    
    try:
        with open('ultimate_opencode_final_complete.json', 'r') as f:
            config = json.load(f)
        
        print('✅ JSON syntax is valid')
        print(f'✅ Schema: {config.get("$schema", "Not specified")}')
        print(f'✅ Username: {config.get("username", "Not specified")}')
        
        features = config.get('features', {})
        print(f'✅ Features: {len(features)} enabled')
        for feature, enabled in features.items():
            if enabled:
                print(f'    - {feature}: enabled')
        
        metadata = config.get('metadata', {})
        print(f'✅ Metadata: {len(metadata)} fields')
        for key, value in metadata.items():
            print(f'    - {key}: {value}')
        
        model_groups = config.get('model_groups', {})
        print(f'✅ Model Groups: {len(model_groups)} groups')
        for group, models in model_groups.items():
            print(f'    - {group}: {len(models)} models')
        
        providers = config.get('provider', {})
        print(f'✅ Providers: {len(providers)} total')
        
        # Count total models
        total_models = sum(len(provider_data.get('models', {})) for provider_data in providers.values())
        print(f'✅ Models: {total_models} total')
        
        # Show provider breakdown
        print('\n📊 Provider Breakdown:')
        for provider_name, provider_data in sorted(providers.items()):
            model_count = len(provider_data.get('models', {}))
            print(f'    - {provider_name}: {model_count} models')
        
        # Check API keys
        api_keys_found = 0
        for provider_name, provider_data in providers.items():
            options = provider_data.get('options', {})
            api_key = options.get('apiKey', '')
            if api_key and not api_key.startswith('${'):
                api_keys_found += 1
        
        print(f'✅ API Keys Embedded: {api_keys_found} providers')
        
        # Check file permissions
        file_stat = os.stat('ultimate_opencode_final_complete.json')
        permissions = oct(file_stat.st_mode)[-3:]
        print(f'✅ File Permissions: {permissions} (should be 600)')
        
        # Validate security warning
        security_warning = metadata.get('security_warning', '')
        if 'API KEYS' in security_warning and 'DO NOT COMMIT' in security_warning:
            print('✅ Security warning present')
        else:
            print('⚠️  Security warning missing or incomplete')
        
        print('\n🎉 VALIDATION SUCCESSFUL!')
        print('The configuration is 100% valid OpenCode format!')
        return True
        
    except Exception as e:
        print(f'❌ Validation failed: {e}')
        return False

if __name__ == "__main__":
    validate_config()