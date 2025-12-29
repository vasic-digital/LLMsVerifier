#!/usr/bin/env python3
"""
Parse the ultimate challenge log to extract model and provider information.
"""

import re
import json

def parse_challenge_log(log_file):
    """Parse the challenge log and extract model information."""
    
    with open(log_file, 'r') as f:
        log_content = f.read()
    
    # Remove ANSI color codes
    log_content = re.sub(r'\x1b\[[0-9;]*m', '', log_content)
    
    # Find all model verification completions using a simpler approach
    # Look for the pattern: model_name: status=verified ... "model_id":"value" ... "provider_id":"value"
    model_pattern = r'Mandatory verification completed for model ([^:]+): .*?"model_id":"([^"]+)".*?"provider_id":"([^"]+)"'
    
    models = []
    for line in log_content.split('\n'):
        if 'Mandatory verification completed for model' in line:
            match = re.search(model_pattern, line)
            if match:
                model_name, model_id, provider_id = match.groups()
                models.append((model_name.strip(), model_id.strip(), provider_id.strip()))
    
    print(f'Found {len(models)} models from line-by-line parsing')
    
    # Group by provider
    provider_models = {}
    for model_name, model_id, provider_id in models:
        if provider_id not in provider_models:
            provider_models[provider_id] = []
        if model_id not in provider_models[provider_id]:  # Avoid duplicates
            provider_models[provider_id].append(model_id)
    
    print(f'Providers with models: {len(provider_models)}')
    total_models = 0
    for provider, models in sorted(provider_models.items()):
        model_count = len(models)
        total_models += model_count
        print(f'{provider}: {model_count} models')
        if model_count > 0 and model_count <= 5:
            print(f'  All models: {models}')
        elif model_count > 5:
            print(f'  Sample models: {models[:5]}...')
    
    print(f'Total models across all providers: {total_models}')
    
    return provider_models

if __name__ == "__main__":
    log_file = "ultimate_challenge_complete.log"
    provider_models = parse_challenge_log(log_file)
    
    # Save to JSON for analysis
    with open('challenge_models_extracted.json', 'w') as f:
        json.dump(provider_models, f, indent=2, sort_keys=True)
    
    print(f"\nExtracted model data saved to challenge_models_extracted.json")