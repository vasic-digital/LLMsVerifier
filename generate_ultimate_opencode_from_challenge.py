#!/usr/bin/env python3
"""
Ultimate OpenCode Configuration Generator from Challenge Logs

This script monitors the ultimate challenge log file and generates a comprehensive
OpenCode JSON configuration with all discovered providers and models.
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
from dataclasses import dataclass
from collections import defaultdict

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

@dataclass
class ModelInfo:
    """Model information structure"""
    id: str
    name: str
    max_tokens: int = 4096
    cost_per_1m_in: float = 0.0
    cost_per_1m_out: float = 0.0
    supports_brotli: bool = False
    verified: bool = False
    features: List[str] = None
    
    def __post_init__(self):
        if self.features is None:
            self.features = []

@dataclass
class ProviderInfo:
    """Provider information structure"""
    id: str
    name: str
    api_key_env: str
    base_url: str
    models: Dict[str, ModelInfo]
    registered: bool = False
    verified: bool = False

class ChallengeLogParser:
    """Parser for challenge log files"""
    
    def __init__(self, log_file: str):
        self.log_file = log_file
        self.providers = {}
        self.models = defaultdict(list)
        self.discovered_providers = set()
        
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
            'sarvarm': 'SARVAM_API_KEY',
            'nlpcloud': 'NLP_API_KEY',
            'perplexity': 'PERPLEXITY_API_KEY',
            'together': 'TOGETHER_API_KEY'
        }
        
        # Provider base URLs from the API endpoints file
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
            'openrouter': 'https://openrouter.ai/api/v1'
        }

    def parse_log_file(self) -> Dict[str, Any]:
        """Parse the challenge log file and extract information"""
        if not os.path.exists(self.log_file):
            logger.error(f"Log file not found: {self.log_file}")
            return {}
        
        try:
            with open(self.log_file, 'r', encoding='utf-8') as f:
                content = f.read()
                
            logger.info(f"Parsing log file: {self.log_file} ({len(content)} bytes)")
            
            # Extract provider registration information
            self._extract_provider_registrations(content)
            
            # Extract model discovery information
            self._extract_model_discoveries(content)
            
            # Extract verification results
            self._extract_verification_results(content)
            
            return {
                'providers': self.providers,
                'models': dict(self.models),
                'discovered_providers': list(self.discovered_providers)
            }
            
        except Exception as e:
            logger.error(f"Error parsing log file: {e}")
            return {}

    def _extract_provider_registrations(self, content: str):
        """Extract provider registration information"""
        # Pattern for registered providers
        provider_pattern = r"Registered provider:\s*(\w+)"
        
        matches = re.findall(provider_pattern, content, re.IGNORECASE)
        for provider in matches:
            provider_id = provider.lower()
            self.discovered_providers.add(provider_id)
            
            # Create provider info if not exists
            if provider_id not in self.providers:
                self.providers[provider_id] = ProviderInfo(
                    id=provider_id,
                    name=provider.title(),
                    api_key_env=self.provider_env_mapping.get(provider_id, f"{provider.upper()}_API_KEY"),
                    base_url=self.provider_base_urls.get(provider_id, ""),
                    models={},
                    registered=True
                )
            else:
                self.providers[provider_id].registered = True
            
            logger.info(f"Found registered provider: {provider_id}")

    def _extract_model_discoveries(self, content: str):
        """Extract model discovery information"""
        # Pattern for model discoveries
        model_pattern = r"Found\s+(\d+)\s+.*models?.*(?:for|from)\s+(\w+)"
        
        matches = re.findall(model_pattern, content, re.IGNORECASE)
        for count_str, provider in matches:
            provider_id = provider.lower()
            count = int(count_str)
            
            logger.info(f"Found {count} models for provider: {provider_id}")
            
            # Create provider if not exists
            if provider_id not in self.providers:
                self.providers[provider_id] = ProviderInfo(
                    id=provider_id,
                    name=provider.title(),
                    api_key_env=self.provider_env_mapping.get(provider_id, f"{provider.upper()}_API_KEY"),
                    base_url=self.provider_base_urls.get(provider_id, ""),
                    models={}
                )

    def _extract_verification_results(self, content: str):
        """Extract verification results"""
        # Pattern for verification status
        verify_pattern = r"(\w+).*?verified.*?models?.*?(\d+)"
        
        matches = re.findall(verify_pattern, content, re.IGNORECASE)
        for provider, count in matches:
            provider_id = provider.lower()
            if provider_id in self.providers:
                self.providers[provider_id].verified = True
                logger.info(f"Provider {provider_id} verified with {count} models")

class OpenCodeConfigGenerator:
    """Generator for OpenCode configuration"""
    
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
        """Generate OpenCode configuration from challenge data"""
        providers = challenge_data.get('providers', {})
        
        config = {
            "$schema": "https://opencode.sh/schema.json",
            "username": "OpenCode AI Assistant",
            "provider": {},
            "metadata": {
                "generated_at": datetime.now().isoformat(),
                "total_providers": len(providers),
                "total_models": sum(len(p.models) for p in providers.values()),
                "challenge_based": True,
                "security_warning": "CONTAINS API KEYS - DO NOT COMMIT",
                "safe_to_commit": False
            }
        }
        
        # Generate provider configurations
        for provider_id, provider_info in providers.items():
            provider_config = self._generate_provider_config(provider_info)
            if provider_config:
                config["provider"][provider_id] = provider_config
        
        return config

    def _generate_provider_config(self, provider_info: ProviderInfo) -> Optional[Dict[str, Any]]:
        """Generate configuration for a single provider"""
        api_key = self.get_api_key(provider_info.api_key_env)
        
        if not api_key:
            logger.warning(f"No API key found for provider: {provider_info.id}")
            # Still include the provider structure for future key addition
            api_key = f"${{{provider_info.api_key_env}}}"
        
        provider_config = {
            "options": {
                "apiKey": api_key,
                "baseURL": provider_info.base_url
            },
            "models": {}
        }
        
        # Add models if available
        if provider_info.models:
            for model_id, model_info in provider_info.models.items():
                provider_config["models"][model_id] = {
                    "name": model_info.name,
                    "maxTokens": model_info.max_tokens,
                    "cost_per_1m_in": model_info.cost_per_1m_in,
                    "cost_per_1m_out": model_info.cost_per_1m_out,
                    "supports_brotli": model_info.supports_brotli,
                    "verified": model_info.verified,
                    "features": model_info.features
                }
        else:
            # Add default models based on provider type
            default_models = self._get_default_models(provider_info.id)
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
                    "verified": False
                },
                "gpt-4-turbo": {
                    "name": "GPT-4 Turbo",
                    "maxTokens": 128000,
                    "cost_per_1m_in": 10.0,
                    "cost_per_1m_out": 30.0,
                    "supports_brotli": True,
                    "verified": False
                }
            },
            'anthropic': {
                "claude-3-opus": {
                    "name": "Claude 3 Opus",
                    "maxTokens": 200000,
                    "cost_per_1m_in": 15.0,
                    "cost_per_1m_out": 75.0,
                    "supports_brotli": False,
                    "verified": False
                },
                "claude-3-sonnet": {
                    "name": "Claude 3 Sonnet",
                    "maxTokens": 200000,
                    "cost_per_1m_in": 3.0,
                    "cost_per_1m_out": 15.0,
                    "supports_brotli": False,
                    "verified": False
                }
            },
            'groq': {
                "llama2-70b": {
                    "name": "LLaMA 2 70B (Groq)",
                    "maxTokens": 4096,
                    "cost_per_1m_in": 0.0,
                    "cost_per_1m_out": 0.0,
                    "supports_brotli": True,
                    "verified": False
                }
            }
        }
        
        return default_models.get(provider_id, {})

def monitor_log_file(log_file: str, callback, interval: int = 10):
    """Monitor log file for changes and call callback when new content is added"""
    logger.info(f"Starting log file monitoring: {log_file}")
    
    if not os.path.exists(log_file):
        logger.warning(f"Log file does not exist yet: {log_file}")
        return
    
    last_size = os.path.getsize(log_file)
    
    try:
        while True:
            time.sleep(interval)
            
            if os.path.exists(log_file):
                current_size = os.path.getsize(log_file)
                
                if current_size > last_size:
                    logger.info(f"Log file updated: {current_size - last_size} new bytes")
                    callback()
                    last_size = current_size
                elif current_size < last_size:
                    # Log file was truncated or rotated
                    logger.info("Log file was truncated or rotated")
                    callback()
                    last_size = current_size
            else:
                logger.warning(f"Log file disappeared: {log_file}")
                
    except KeyboardInterrupt:
        logger.info("Log monitoring stopped by user")
    except Exception as e:
        logger.error(f"Error monitoring log file: {e}")

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
    
    parser = argparse.ArgumentParser(description="Generate Ultimate OpenCode Configuration from Challenge Logs")
    parser.add_argument("--log-file", default="ultimate_challenge_complete.log", 
                       help="Challenge log file to parse")
    parser.add_argument("--env-file", default=".env", 
                       help="Environment file with API keys")
    parser.add_argument("--output", default="ultimate_opencode_final.json", 
                       help="Output configuration file")
    parser.add_argument("--monitor", action="store_true", 
                       help="Continuously monitor log file for updates")
    parser.add_argument("--validate", action="store_true", 
                       help="Validate generated configuration")
    parser.add_argument("--pretty", action="store_true", 
                       help="Pretty print JSON output")
    
    args = parser.parse_args()
    
    def generate_config():
        """Generate configuration from current log state"""
        logger.info("Generating OpenCode configuration from challenge logs...")
        
        # Parse challenge logs
        parser = ChallengeLogParser(args.log_file)
        challenge_data = parser.parse_log_file()
        
        if not challenge_data:
            logger.error("No data extracted from challenge logs")
            return False
        
        # Generate configuration
        generator = OpenCodeConfigGenerator(args.env_file)
        config = generator.generate_config(challenge_data)
        
        # Validate if requested
        if args.validate and not validate_opencode_config(config):
            logger.error("Configuration validation failed")
            return False
        
        # Save configuration
        try:
            with open(args.output, 'w', encoding='utf-8') as f:
                if args.pretty:
                    json.dump(config, f, indent=2, sort_keys=True)
                else:
                    json.dump(config, f, sort_keys=True)
            
            # Set restrictive permissions
            os.chmod(args.output, 0o600)
            
            logger.info(f"Configuration saved to: {args.output}")
            logger.info(f"Total providers: {len(config.get('provider', {}))}")
            logger.info(f"Total models: {sum(len(p.get('models', {})) for p in config.get('provider', {}).values())}")
            
            return True
            
        except Exception as e:
            logger.error(f"Error saving configuration: {e}")
            return False
    
    # Generate initial configuration
    success = generate_config()
    
    if not success:
        logger.error("Failed to generate initial configuration")
        return 1
    
    # Monitor for updates if requested
    if args.monitor:
        logger.info("Starting continuous monitoring...")
        monitor_log_file(args.log_file, generate_config)
    
    return 0

if __name__ == "__main__":
    sys.exit(main())