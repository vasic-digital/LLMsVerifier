#!/usr/bin/env python3
"""
Validate OpenCode configuration follows official schema
"""

import json
import sys

# Official OpenCode schema - only these top-level keys are allowed
VALID_TOP_LEVEL_KEYS = {
    "$schema", "plugin", "enterprise", "instructions", "provider", 
    "mcp", "tools", "agent", "command", "keybinds", "username", 
    "share", "permission", "compaction", "sse", "mode", "autoshare"
}

def validate_config(config_path):
    """Validate an OpenCode configuration file"""
    try:
        with open(config_path, 'r') as f:
            config = json.load(f)
    except Exception as e:
        return False, [f"Failed to parse JSON: {e}"]
    
    errors = []
    
    # Check for invalid top-level keys
    invalid_keys = []
    for key in config.keys():
        if key not in VALID_TOP_LEVEL_KEYS:
            invalid_keys.append(key)
    
    if invalid_keys:
        errors.append(f"Invalid top-level keys: {', '.join(invalid_keys)}")
        errors.append(f"Valid keys are: {', '.join(sorted(VALID_TOP_LEVEL_KEYS))}")
    
    # Must have provider
    if "provider" not in config:
        errors.append("Missing required 'provider' key at top level")
    elif not isinstance(config["provider"], dict):
        errors.append("'provider' must be an object")
    
    # Check provider structure
    if "provider" in config and isinstance(config["provider"], dict):
        for provider_key, provider_config in config["provider"].items():
            if not isinstance(provider_config, dict):
                errors.append(f"Provider '{provider_key}' must be an object")
                continue
            
            # Check provider has options
            if "options" not in provider_config:
                errors.append(f"Provider '{provider_key}' missing 'options'")
            elif not isinstance(provider_config["options"], dict):
                errors.append(f"Provider '{provider_key}' options must be an object")
            
            # Check models are nested under provider, not top-level
            if "models" in config:
                errors.append("'models' should not be at top level - move it inside each provider")
    
    return len(errors) == 0, errors

if __name__ == "__main__":
    config_path = "/home/milosvasic/Downloads/opencode.json"
    
    is_valid, errors = validate_config(config_path)
    
    print("="*70)
    print("OPENCODE CONFIGURATION VALIDATION")
    print("="*70)
    print()
    
    if is_valid:
        print("✅ CONFIGURATION IS VALID")
        print()
        with open(config_path, 'r') as f:
            config = json.load(f)
        print(f"Providers configured: {len(config.get('provider', {}))}")
    else:
        print("❌ CONFIGURATION HAS ERRORS:")
        print()
        for error in errors:
            print(f"  - {error}")
        sys.exit(1)
    
    print("="*70)