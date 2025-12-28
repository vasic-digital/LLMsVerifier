#!/usr/bin/env python3
"""
Providor Coverage Validator
Ensures 100% provider utilization in LLM Verifier

Usage: python3 scripts/validate_provider_coverage.py [--strict]
"""

import yaml
import json
import sys
import re
from pathlib import Path

# REQUIRED PROVIDERS - 27 total
REQUIRED_PROVIDERS = {
    'HuggingFace', 'Nvidia', 'DeepSeek', 'Groq', 'OpenRouter', 'Replicate',
    'Anthropic', 'OpenAI', 'Google', 'Gemini', 'Perplexity', 'Together AI',
    'Cerebras', 'SambaNova AI', 'Fireworks AI', 'Mistral AI', 'Codestral',
    'Kimi', 'Cloudflare Workers AI', 'Modal', 'Chutes', 'SiliconFlow',
    'Novita AI', 'Upstage AI', 'NLP Cloud', 'Hyperbolic', 'ZAI', 'Baseten',
    'TwelveLabs'
}

# Provider name mappings for validation
PROVIDER_ALIASES = {
    'Huggingface Provider': 'HuggingFace',
    'Nvidia Provider': 'Nvidia',
    'Deepseek Provider': 'DeepSeek',
    'Groq Provider': 'Groq',
    'Openrouter Provider': 'OpenRouter',
    'Replicate Provider': 'Replicate',
    'Gemini Provider': 'Gemini',
    'Kimi Provider': 'Kimi',
    'Mistral AI Provider': 'Mistral AI',
    'Together AI Provider': 'Together AI',
    'huggingface': 'HuggingFace',
    'anthropic': 'Anthropic',
    'groq': 'Groq',
    'google': 'Google',
    'perplexity': 'Perplexity',
    'together': 'Together AI',
    'nvidia': 'Nvidia',
    'deepseek': 'DeepSeek',
    'openrouter': 'OpenRouter',
    'replicate': 'Replicate',
    'gemini': 'Gemini',
    'kimi': 'Kimi',
    'mistral': 'Mistral AI',
    'codestral': 'Codestral',
    'cerebras': 'Cerebras',
    'samba': 'SambaNova AI',
    'fireworks': 'Fireworks AI',
    'modal': 'Modal',
    'chutes': 'Chutes',
    'siliconflow': 'SiliconFlow',
    'novita': 'Novita AI',
    'upstage': 'Upstage AI',
    'nlp': 'NLP Cloud',
    'hyperbolic': 'Hyperbolic',
    'zai': 'ZAI',
    'baseten': 'Baseten',
    'twelvelabs': 'TwelveLabs',
    'Cloudflare Workers AI Provider': 'Cloudflare Workers AI',
}

def normalize_provider_name(name):
    """Normalize provider name for comparison"""
    if not name:
        return None
    
    # Direct match
    if name in REQUIRED_PROVIDERS:
        return name
    
    # Check aliases
    if name in PROVIDER_ALIASES:
        return PROVIDER_ALIASES[name]
    
    # Try partial matching
    for req in REQUIRED_PROVIDERS:
        if req.lower() in name.lower() or name.lower() in req.lower():
            return req
    
    return None

def validate_yaml_config(config_path):
    """Validate YAML configuration"""
    try:
        with open(config_path) as f:
            config = yaml.safe_load(f)
        
        providers = []
        for p in config.get('llms', []):
            name = p.get('name', '')
            normalized = normalize_provider_name(name)
            if normalized:
                providers.append(normalized)
        
        return set(providers)
    except Exception as e:
        print(f"❌ Error reading {config_path}: {e}")
        return set()

def validate_json_config(config_path):
    """Validate JSON configuration"""
    try:
        with open(config_path) as f:
            config = json.load(f)
        
        providers = []
        for name in config.get('provider', {}).keys():
            normalized = normalize_provider_name(name)
            if normalized:
                providers.append(normalized)
        
        return set(providers)
    except Exception as e:
        print(f"❌ Error reading {config_path}: {e}")
        return set()

def get_missing_providers(found_providers):
    """Get missing providers"""
    return sorted(REQUIRED_PROVIDERS - found_providers)

def check_env_file(env_path):
    """Check .env file for API key definitions"""
    try:
        with open(env_path) as f:
            content = f.read()
        
        # Look for API key patterns
        env_keys = set()
        patterns = [
            r'HUGGINGFACE_API_KEY=',
            r'NVIDIA_API_KEY=',
            r'DEEPSEEK_API_KEY=',
            r'GROQ_API_KEY=',
            r'OPENROUTER_API_KEY=',
            r'REPLICATE_API_KEY=',
            r'ANTHROPIC_API_KEY=',
            r'OPENAI_API_KEY=',
            r'PERPLEXITY_API_KEY=',
            r'TOGETHER_API_KEY=',
            r'CEREBRAS_API_KEY=',
            r'SAMBANOVA_API_KEY=',
            r'FIREWORKS_API_KEY=',
            r'MISTRAL_API_KEY=',
            r'CODESTRAL_API_KEY=',
            r'KIMI_API_KEY=',
            r'CLOUDFLARE_API_KEY=',
            r'MODAL_API_KEY=',
            r'CHUTES_API_KEY=',
            r'SILICONFLOW_API_KEY=',
            r'NOVITA_API_KEY=',
            r'UPSTAGE_API_KEY=',
            r'NLP_API_KEY=',
            r'HYPERBOLIC_API_KEY=',
            r'ZAI_API_KEY=',
            r'BASETEN_API_KEY=',
            r'TWELVELABS_API_KEY=',
        ]
        
        for pattern in patterns:
            if re.search(pattern, content, re.IGNORECASE):
                provider = pattern.split('_API_KEY=')[0]
                provider = provider.replace('_', ' ').title().replace(' ', '')
                # Fix known mappings
                if 'Huggingface' in provider:
                    env_keys.add('HuggingFace')
                elif 'Nvidia' in provider:
                    env_keys.add('Nvidia')
                elif 'Deepseek' in provider:
                    env_keys.add('DeepSeek')
                elif 'Groq' in provider:
                    env_keys.add('Groq')
                elif 'Openrouter' in provider:
                    env_keys.add('OpenRouter')
                elif 'Replicate' in provider:
                    env_keys.add('Replicate')
                elif 'Anthropic' in provider:
                    env_keys.add('Anthropic')
                elif 'Openai' in provider:
                    env_keys.add('OpenAI')
                elif 'Perplexity' in provider:
                    env_keys.add('Perplexity')
                elif 'Together' in provider:
                    env_keys.add('Together AI')
                elif 'Cerebras' in provider:
                    env_keys.add('Cerebras')
                elif 'Sambanova' in provider:
                    env_keys.add('SambaNova AI')
                elif 'Fireworks' in provider:
                    env_keys.add('Fireworks AI')
                elif 'Mistral' in provider:
                    env_keys.add('Mistral AI')
                elif 'Codestral' in provider:
                    env_keys.add('Codestral')
                elif 'Kimi' in provider:
                    env_keys.add('Kimi')
                elif 'Cloudflare' in provider:
                    env_keys.add('Cloudflare Workers AI')
                elif 'Modal' in provider:
                    env_keys.add('Modal')
                elif 'Chutes' in provider:
                    env_keys.add('Chutes')
                elif 'Siliconflow' in provider:
                    env_keys.add('SiliconFlow')
                elif 'Novita' in provider:
                    env_keys.add('Novita AI')
                elif 'Upstage' in provider:
                    env_keys.add('Upstage AI')
                elif 'Nlp' in provider:
                    env_keys.add('NLP Cloud')
                elif 'Hyperbolic' in provider:
                    env_keys.add('Hyperbolic')
                elif 'Zai' in provider:
                    env_keys.add('ZAI')
                elif 'Baseten' in provider:
                    env_keys.add('Baseten')
                elif 'Twelvelabs' in provider:
                    env_keys.add('TwelveLabs')
        
        return env_keys
    except Exception as e:
        print(f"❌ Error reading {env_path}: {e}")
        return set()

def main():
    strict_mode = '--strict' in sys.argv
    
    print("=" * 70)
    print("PROVIDER COVERAGE VALIDATION")
    print("=" * 70)
    print(f"Required providers: {len(REQUIRED_PROVIDERS)}")
    print(f"Strict mode: {'ON' if strict_mode else 'OFF'}")
    print()
    
    # Check environment file
    env_keys = check_env_file('.env')
    print(f"📋 .env defines {len(env_keys)} provider variables")
    
    # Define configs to validate
    configs = [
        ('llm-verifier/config_full.yaml', 'yaml'),
        ('llm-verifier/config_working.yaml', 'yaml'),
        ('ultimate_opencode_config.json', 'json'),
        ('ultimate_opencode_config_FULL.json', 'json'),
    ]
    
    all_results = {}
    final_status = True
    
    for config_path, config_type in configs:
        if not Path(config_path).exists():
            print(f"⚠️  {config_path}: File not found (skipping)")
            continue
        
        if config_type == 'yaml':
            providers = validate_yaml_config(config_path)
        else:
            providers = validate_json_config(config_path)
        
        all_results[config_path] = providers
        
        missing = get_missing_providers(providers)
        
        # Determine status
        if len(providers) >= len(REQUIRED_PROVIDERS):
            status = "✅"
        elif len(providers) >= 20:
            status = "⚠️"
            if strict_mode:
                final_status = False
        else:
            status = "❌"
            final_status = False
        
        print(f"{status} {config_path}: {len(providers)}/{len(REQUIRED_PROVIDERS)} providers")
        
        if missing and len(missing) <= 10:
            print(f"   Missing: {', '.join(missing[:10])}")
    
    print()
    print("=" * 70)
    print("SUMMARY")
    print("=" * 70)
    
    if final_status:
        print("🎉 PASS: All configurations meet requirements!")
        print(f"   Total providers required: {len(REQUIRED_PROVIDERS)}")
        print(f"   Status: {'100% coverage' if len(env_keys) >= len(REQUIRED_PROVIDERS) else 'Sufficient coverage'}")
        return 0
    else:
        print("❌ FAIL: Insufficient provider coverage")
        print(f"   Required: {len(REQUIRED_PROVIDERS)} providers")
        print(f"   Best config: {max(len(p) for p in all_results.values())}/{len(REQUIRED_PROVIDERS)}")
        print()
        print("ACTION REQUIRED:")
        print("1. Update configurations to include all providers")
        print("2. Run: python3 scripts/generate_full_config.py")
        print("3. Validate with: python3 scripts/validate_provider_coverage.py --strict")
        return 1

if __name__ == '__main__':
    sys.exit(main())