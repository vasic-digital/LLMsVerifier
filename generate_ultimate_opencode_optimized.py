#!/usr/bin/env python3
"""
Optimized Ultimate OpenCode Configuration Generator from Challenge Logs

This optimized script efficiently processes large challenge log files and generates
a comprehensive OpenCode JSON configuration with all discovered providers and models.
"""

import json
import re
import os
import sys
import time
import logging
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Any, Optional
from collections import defaultdict

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

class OptimizedChallengeLogParser:
    """Optimized parser for large challenge log files"""
    
    def __init__(self, log_file: str):
        self.log_file = log_file
        self.providers = {}
        self.discovered_providers = set()
        self.model_counts = defaultdict(int)
        
        # Provider mapping from environment variables
        self.provider_env_mapping = {
            'openai': 'OPENAI_API_KEY',
            'anthropic': 'ANTHROPIC_API_KEY',
            'groq': 'GROQ_API_KEY',
            'huggingface': 'HUGGINGFACE_API_KEY',
            'gemini': 'GEMINI_API_KEY',
            'deepseek': 'DEEPSEEK_API_KEY',
            'openrouter': 'OPENROUTER_API_KEY',
            'fireworks': 'FIREWORKS_API_KEY',
            'cerebras': 'CEREBRAS_API_KEY',
            'cloudflare': 'CLOUDFLARE_API_KEY',
            'mistral': 'MISTRAL_API_KEY',
            'codestral': 'CODESTRAL_API_KEY',
            'novita': 'NOVITA_API_KEY',
            'upstage': 'UPSTAGE_API_KEY',
            'hyperbolic': 'HYPERBOLIC_API_KEY',
            'sambanova': 'SAMBANOVA_API_KEY',
            'replicate': 'REPLICATE_API_KEY',
            'modal': 'MODAL_API_KEY',
            'inference': 'INFERENCE_API_KEY',
            'baseten': 'BASETEN_API_KEY',
            'nvidia': 'NVIDIA_API_KEY',
            'chutes': 'CHUTES_API_KEY',
            'siliconflow': 'SILICONFLOW_API_KEY',
            'kimi': 'KIMI_API_KEY',
            'vercel': 'VERCEL_API_KEY',
            'zai': 'ZAI_API_KEY',
            'twelvelabs': 'TWELVE_LABS_API_KEY',
            'vulavula': 'VULAVULA_API_KEY',
            'sarvam': 'SARVAM_API_KEY',
            'nlpcloud': 'NLP_API_KEY',
            'perplexity': 'PERPLEXITY_API_KEY',
            'together': 'TOGETHER_API_KEY'
        }
        
        # Provider base URLs
        self.provider_base_urls = {
            'fireworks': 'https://api.fireworks.ai/v1',
            'chutes': 'https://api.chutes.ai/v1',
            'siliconflow': 'https://api.siliconflow.cn/v1',
            'kimi': 'https://api.moonshot.cn/v1',
            'gemini': 'https://generativelanguage.googleapis.com/v1beta',
            'hyperbolic': 'https://api.hyperbolic.xyz/v1',
            'baseten': 'https://api.baseten.co/v1',
            'novita': 'https://api.novita.ai/v1',
            'upstage': 'https://api.upstage.ai/v1',
            'inference': 'https://api.inference.net/v1',
            'replicate': 'https://api.replicate.com/v1',
            'nvidia': 'https://integrate.api.nvidia.com/v1',
            'cerebras': 'https://api.cerebras.ai/v1',
            'cloudflare': 'https://api.cloudflare.com/client/v4/accounts/{account_id}/ai',
            'mistral': 'https://api.mistral.ai/v1',
            'codestral': 'https://api.mistral.ai/v1',
            'zai': 'https://api.z.ai/v1',
            'modal': 'https://api.modal.com/v1',
            'sambanova': 'https://api.sambanova.ai/v1',
            'openai': 'https://api.openai.com/v1',
            'anthropic': 'https://api.anthropic.com/v1',
            'groq': 'https://api.groq.com/openai/v1',
            'huggingface': 'https://huggingface.co/api',
            'deepseek': 'https://api.deepseek.com/v1',
            'openrouter': 'https://openrouter.ai/api/v1',
            'vercel': 'https://api.vercel.com/v1',
            'twelvelabs': 'https://api.twelvelabs.io/v1',
            'vulavula': 'https://api.lelapa.ai/v1',
            'sarvam': 'https://api.sarvam.ai/v1',
            'nlpcloud': 'https://api.nlpcloud.io/v1',
            'perplexity': 'https://api.perplexity.ai/v1',
            'together': 'https://api.together.xyz/v1'
        }

    def parse_log_file(self) -> Dict[str, Any]:
        """Parse the challenge log file efficiently"""
        if not os.path.exists(self.log_file):
            logger.error(f"Log file not found: {self.log_file}")
            return {}
        
        try:
            file_size = os.path.getsize(self.log_file)
            logger.info(f"Parsing log file: {self.log_file} ({file_size} bytes)")
            
            # Read file in chunks for large files
            if file_size > 10 * 1024 * 1024:  # 10MB
                logger.info("Large file detected, using chunked processing")
                return self._parse_large_log_file()
            else:
                return self._parse_small_log_file()
                
        except Exception as e:
            logger.error(f"Error parsing log file: {e}")
            return {}

    def _parse_small_log_file(self) -> Dict[str, Any]:
        """Parse small log files normally"""
        with open(self.log_file, 'r', encoding='utf-8', errors='ignore') as f:
            content = f.read()
        return self._extract_information(content)

    def _parse_large_log_file(self) -> Dict[str, Any]:
        """Parse large log files in chunks"""
        chunk_size = 1024 * 1024  # 1MB chunks
        
        with open(self.log_file, 'r', encoding='utf-8', errors='ignore') as f:
            while True:
                chunk = f.read(chunk_size)
                if not chunk:
                    break
                
                # Process this chunk
                self._extract_information(chunk)
                
                # Show progress
                current_pos = f.tell()
                progress = (current_pos / os.path.getsize(self.log_file)) * 100
                if int(progress) % 10 == 0:
                    logger.info(f"Processing progress: {progress:.1f}%")
        
        return {
            'providers': self.providers,
            'discovered_providers': list(self.discovered_providers),
            'model_counts': dict(self.model_counts),
            'total_providers': len(self.providers),
            'total_models': sum(self.model_counts.values())
        }

    def _extract_information(self, content: str) -> Dict[str, Any]:
        """Extract information from content"""
        # Extract provider registrations (fast regex)
        self._extract_provider_registrations(content)
        
        # Extract model counts (fast regex)
        self._extract_model_counts(content)
        
        return {
            'providers': self.providers,
            'discovered_providers': list(self.discovered_providers),
            'model_counts': dict(self.model_counts),
            'total_providers': len(self.providers),
            'total_models': sum(self.model_counts.values())
        }

    def _extract_provider_registrations(self, content: str):
        """Extract provider registration information"""
        # Fast regex for provider registrations
        provider_pattern = r"Registered provider:\s*(\w+)"
        
        for match in re.finditer(provider_pattern, content, re.IGNORECASE):
            provider = match.group(1).lower()
            self.discovered_providers.add(provider)
            
            # Create provider info if not exists
            if provider not in self.providers:
                self.providers[provider] = {
                    'id': provider,
                    'name': provider.title(),
                    'api_key_env': self.provider_env_mapping.get(provider, f"{provider.upper()}_API_KEY"),
                    'base_url': self.provider_base_urls.get(provider, ""),
                    'registered': True,
                    'verified': False,
                    'total_models': 0,
                    'verified_models': 0
                }
            else:
                self.providers[provider]['registered'] = True

    def _extract_model_counts(self, content: str):
        """Extract model count information"""
        # Fast regex for model counts
        model_pattern = r"Found\s+(\d+)\s+.*models?.*(?:for|from)\s+(\w+)"
        
        for match in re.finditer(model_pattern, content, re.IGNORECASE):
            count = int(match.group(1))
            provider = match.group(2).lower()
            
            # Skip generic "provider" matches
            if provider == "provider":
                continue
                
            self.model_counts[provider] = count
            
            # Create provider if not exists
            if provider not in self.providers:
                self.providers[provider] = {
                    'id': provider,
                    'name': provider.title(),
                    'api_key_env': self.provider_env_mapping.get(provider, f"{provider.upper()}_API_KEY"),
                    'base_url': self.provider_base_urls.get(provider, ""),
                    'registered': False,
                    'verified': False,
                    'total_models': count,
                    'verified_models': 0
                }
            else:
                self.providers[provider]['total_models'] = count

class OptimizedOpenCodeConfigGenerator:
    """Optimized generator for OpenCode configuration"""
    
    def __init__(self, env_file: str = ".env"):
        self.env_file = env_file
        self.api_keys = {}
        self.load_api_keys()

    def load_api_keys(self):
        """Load API keys from environment file"""
        if not os.path.exists(self.env_file):
            logger.warning(f"Environment file not found: {self.env_file}")
            return
        
        try:
            with open(self.env_file, 'r') as f:
                for line in f:
                    line = line.strip()
                    if line and '=' in line and not line.startswith('#'):
                        key, value = line.split('=', 1)
                        self.api_keys[key.strip()] = value.strip()
            
            logger.info(f"Loaded {len(self.api_keys)} API keys from {self.env_file}")
            
        except Exception as e:
            logger.error(f"Error loading API keys: {e}")

    def get_api_key(self, env_var: str) -> str:
        """Get API key from environment or .env file"""
        # First check environment
        env_value = os.getenv(env_var)
        if env_value:
            return env_value
        
        # Then check .env file
        return self.api_keys.get(env_var, "")

    def generate_config(self, challenge_data: Dict[str, Any]) -> Dict[str, Any]:
        """Generate optimized OpenCode configuration from challenge data"""
        providers = challenge_data.get('providers', {})
        
        config = {
            "$schema": "https://opencode.sh/schema.json",
            "username": "OpenCode AI Assistant - Ultimate Challenge Edition",
            "provider": {},
            "model_groups": {
                "premium": ["gpt-4", "claude-3-opus", "gpt-4-turbo"],
                "balanced": ["claude-3-sonnet", "gpt-4o", "mixtral-8x7b"],
                "fast": ["gpt-3.5-turbo", "claude-3-haiku", "llama2-70b"],
                "free": ["llama2-70b", "mixtral-8x7b", "gemma-7b"],
                "verified": []
            },
            "features": {
                "streaming": True,
                "tool_calling": True,
                "vision": True,
                "embeddings": True,
                "mcp": True,
                "lsp": True,
                "acp": True,
                "verification": True,
                "challenge_based": True
            },
            "metadata": {
                "generated_at": datetime.now().isoformat(),
                "generator": "Optimized Ultimate Challenge Parser",
                "version": "2.0-optimized",
                "total_providers": len(providers),
                "total_models": sum(p.get('total_models', 0) for p in providers.values()),
                "verified_providers": sum(1 for p in providers.values() if p.get('verified', False)),
                "registered_providers": sum(1 for p in providers.values() if p.get('registered', False)),
                "challenge_based": True,
                "security_warning": "CONTAINS API KEYS - DO NOT COMMIT - PROTECT WITH 600 PERMISSIONS",
                "safe_to_commit": False,
                "includes_real_verification_data": True
            }
        }
        
        # Generate provider configurations
        for provider_id, provider_info in providers.items():
            provider_config = self._generate_provider_config(provider_info)
            if provider_config:
                config["provider"][provider_id] = provider_config
        
        return config

    def _generate_provider_config(self, provider_info: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """Generate configuration for a single provider"""
        api_key_env = provider_info.get('api_key_env', '')
        api_key = self.get_api_key(api_key_env)
        
        if not api_key:
            logger.debug(f"No API key found for provider: {provider_info['id']}")
            # Still include the provider structure for future key addition
            api_key = f"${{{api_key_env}}}"
        
        provider_config = {
            "options": {
                "apiKey": api_key,
                "baseURL": provider_info.get('base_url', '')
            },
            "models": {},
            "metadata": {
                "registered": provider_info.get('registered', False),
                "verified": provider_info.get('verified', False),
                "total_models": provider_info.get('total_models', 0),
                "verified_models": provider_info.get('verified_models', 0),
                "discovery_timestamp": provider_info.get('discovery_timestamp')
            }
        }
        
        # Add models based on provider type
        default_models = self._get_default_models(provider_info['id'])
        provider_config["models"] = default_models
        
        return provider_config

    def _get_default_models(self, provider_id: str) -> Dict[str, Any]:
        """Get default models for common providers"""
        default_models = {
            'openai': {
                "gpt-4": {
                    "name": "GPT-4",
                    "maxTokens": 8192,
                    "cost_per_1m_in": 30.0,
                    "cost_per_1m_out": 60.0,
                    "supports_brotli": True,
                    "verified": False,
                    "features": ["streaming", "tool_calling", "vision"]
                },
                "gpt-4-turbo": {
                    "name": "GPT-4 Turbo",
                    "maxTokens": 128000,
                    "cost_per_1m_in": 10.0,
                    "cost_per_1m_out": 30.0,
                    "supports_brotli": True,
                    "verified": False,
                    "features": ["streaming", "tool_calling", "vision"]
                },
                "gpt-4o": {
                    "name": "GPT-4o",
                    "maxTokens": 128000,
                    "cost_per_1m_in": 5.0,
                    "cost_per_1m_out": 15.0,
                    "supports_brotli": True,
                    "verified": False,
                    "features": ["streaming", "tool_calling", "vision"]
                },
                "gpt-3.5-turbo": {
                    "name": "GPT-3.5 Turbo",
                    "maxTokens": 16385,
                    "cost_per_1m_in": 0.5,
                    "cost_per_1m_out": 1.5,
                    "supports_brotli": True,
                    "verified": False,
                    "features": ["streaming", "tool_calling"]
                }
            },
            'anthropic': {
                "claude-3-opus": {
                    "name": "Claude 3 Opus",
                    "maxTokens": 200000,
                    "cost_per_1m_in": 15.0,
                    "cost_per_1m_out": 75.0,
                    "supports_brotli": False,
                    "verified": False,
                    "features": ["streaming", "tool_calling", "vision"]
                },
                "claude-3-sonnet": {
                    "name": "Claude 3 Sonnet",
                    "maxTokens": 200000,
                    "cost_per_1m_in": 3.0,
                    "cost_per_1m_out": 15.0,
                    "supports_brotli": False,
                    "verified": False,
                    "features": ["streaming", "tool_calling", "vision"]
                },
                "claude-3-haiku": {
                    "name": "Claude 3 Haiku",
                    "maxTokens": 200000,
                    "cost_per_1m_in": 0.25,
                    "cost_per_1m_out": 1.25,
                    "supports_brotli": False,
                    "verified": False,
                    "features": ["streaming", "tool_calling", "vision"]
                }
            },
            'groq': {
                "llama2-70b": {
                    "name": "LLaMA 2 70B (Groq)",
                    "maxTokens": 4096,
                    "cost_per_1m_in": 0.0,
                    "cost_per_1m_out": 0.0,
                    "supports_brotli": True,
                    "verified": False,
                    "features": ["streaming"]
                },
                "mixtral-8x7b": {
                    "name": "Mixtral 8x7B (Groq)",
                    "maxTokens": 32768,
                    "cost_per_1m_in": 0.0,
                    "cost_per_1m_out": 0.0,
                    "supports_brotli": True,
                    "verified": False,
                    "features": ["streaming", "tool_calling"]
                }
            }
        }
        
        return default_models.get(provider_id, {})

def validate_opencode_config(config: Dict[str, Any]) -> bool:
    """Validate OpenCode configuration format"""
    required_fields = ["$schema", "username", "provider"]
    
    for field in required_fields:
        if field not in config:
            logger.error(f"Missing required field: {field}")
            return False
    
    # Validate provider structure
    providers = config.get("provider", {})
    if not isinstance(providers, dict):
        logger.error("Provider section must be a dictionary")
        return False
    
    for provider_id, provider_config in providers.items():
        if not isinstance(provider_config, dict):
            logger.error(f"Provider {provider_id} configuration must be a dictionary")
            return False
        
        if "options" not in provider_config:
            logger.error(f"Provider {provider_id} missing 'options' section")
            return False
        
        if "models" not in provider_config:
            logger.error(f"Provider {provider_id} missing 'models' section")
            return False
    
    logger.info("Configuration validation passed")
    return True

def main():
    """Main function"""
    import argparse
    
    parser = argparse.ArgumentParser(description="Optimized Ultimate OpenCode Configuration Generator")
    parser.add_argument("--log-file", default="ultimate_challenge_complete.log", 
                       help="Challenge log file to parse")
    parser.add_argument("--env-file", default=".env", 
                       help="Environment file with API keys")
    parser.add_argument("--output", default="ultimate_opencode_final.json", 
                       help="Output configuration file")
    parser.add_argument("--validate", action="store_true", 
                       help="Validate generated configuration")
    parser.add_argument("--pretty", action="store_true", 
                       help="Pretty print JSON output")
    parser.add_argument("--backup", action="store_true", 
                       help="Create backup of previous configuration")
    parser.add_argument("--progress", action="store_true", 
                       help="Show progress for large files")
    
    args = parser.parse_args()
    
    logger.info("Starting optimized OpenCode configuration generation...")
    
    # Parse challenge logs
    parser = OptimizedChallengeLogParser(args.log_file)
    challenge_data = parser.parse_log_file()
    
    if not challenge_data:
        logger.error("No data extracted from challenge logs")
        return 1
    
    logger.info(f"Extracted {challenge_data['total_providers']} providers with {challenge_data['total_models']} models")
    
    # Generate configuration
    generator = OptimizedOpenCodeConfigGenerator(args.env_file)
    config = generator.generate_config(challenge_data)
    
    # Validate if requested
    if args.validate and not validate_opencode_config(config):
        logger.error("Configuration validation failed")
        return 1
    
    # Backup previous configuration if requested
    if args.backup and os.path.exists(args.output):
        backup_file = f"{args.output}.{datetime.now().strftime('%Y%m%d_%H%M%S')}.backup"
        try:
            os.rename(args.output, backup_file)
            logger.info(f"Created backup: {backup_file}")
        except Exception as e:
            logger.warning(f"Could not create backup: {e}")
    
    # Save configuration
    try:
        with open(args.output, 'w', encoding='utf-8') as f:
            if args.pretty:
                json.dump(config, f, indent=2, sort_keys=True)
            else:
                json.dump(config, f, sort_keys=True)
        
        # Set restrictive permissions (600 = owner read/write only)
        os.chmod(args.output, 0o600)
        
        logger.info(f"Configuration saved to: {args.output}")
        logger.info(f"Total providers: {len(config.get('provider', {}))}")
        logger.info(f"Total models: {sum(len(p.get('models', {})) for p in config.get('provider', {}).values())}")
        
        # Display summary
        display_generation_summary(config, challenge_data)
        
        return 0
        
    except Exception as e:
        logger.error(f"Error saving configuration: {e}")
        return 1

def display_generation_summary(config: Dict[str, Any], challenge_data: Dict[str, Any]):
    """Display a summary of the generated configuration"""
    providers = config.get("provider", {})
    total_providers = len(providers)
    total_models = sum(len(p.get("models", {})) for p in providers.values())
    
    metadata = config.get("metadata", {})
    verified_providers = metadata.get("verified_providers", 0)
    registered_providers = metadata.get("registered_providers", 0)
    
    logger.info("=" * 60)
    logger.info("OPTIMIZED CONFIGURATION GENERATION SUMMARY")
    logger.info("=" * 60)
    logger.info(f"Total Providers: {total_providers}")
    logger.info(f"Registered Providers: {registered_providers}")
    logger.info(f"Verified Providers: {verified_providers}")
    logger.info(f"Total Models: {total_models}")
    logger.info(f"Challenge-based Discovery: {metadata.get('challenge_based', False)}")
    logger.info(f"Configuration Version: {metadata.get('version', 'unknown')}")
    
    # Show discovered providers
    discovered = challenge_data.get('discovered_providers', [])
    logger.info(f"Discovered Providers ({len(discovered)}): {', '.join(sorted(discovered)[:10])}{'...' if len(discovered) > 10 else ''}")
    
    # Show model count distribution
    model_counts = challenge_data.get('model_counts', {})
    if model_counts:
        sorted_counts = sorted(model_counts.items(), key=lambda x: x[1], reverse=True)
        logger.info("Top Providers by Model Count:")
        for provider_id, count in sorted_counts[:5]:
            logger.info(f"  {provider_id}: {count} models")
    
    logger.info("=" * 60)

if __name__ == "__main__":
    sys.exit(main())