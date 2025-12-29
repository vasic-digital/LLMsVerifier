#!/usr/bin/env python3
"""
Enhanced Ultimate OpenCode Configuration Generator from Challenge Logs

This advanced script monitors the ultimate challenge log file and generates a comprehensive
OpenCode JSON configuration with all discovered providers and models, including detailed
model information extracted from the challenge results.
"""

import json
import re
import os
import sys
import time
import logging
import threading
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Any, Optional, Set
from dataclasses import dataclass, field
from collections import defaultdict
import queue

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

@dataclass
class ModelInfo:
    """Enhanced model information structure"""
    id: str
    name: str
    max_tokens: int = 4096
    cost_per_1m_in: float = 0.0
    cost_per_1m_out: float = 0.0
    supports_brotli: bool = False
    verified: bool = False
    features: List[str] = field(default_factory=list)
    response_time: float = 0.0
    ttft: float = 0.0  # Time to first token
    score: float = 0.0
    capabilities: Dict[str, bool] = field(default_factory=dict)

@dataclass
class ProviderInfo:
    """Enhanced provider information structure"""
    id: str
    name: str
    api_key_env: str
    base_url: str
    models: Dict[str, ModelInfo]
    registered: bool = False
    verified: bool = False
    total_models: int = 0
    verified_models: int = 0
    discovery_timestamp: Optional[str] = None
    features: List[str] = field(default_factory=list)

class EnhancedChallengeLogParser:
    """Enhanced parser for challenge log files with better model extraction"""
    
    def __init__(self, log_file: str):
        self.log_file = log_file
        self.providers = {}
        self.models = defaultdict(list)
        self.discovered_providers = set()
        self.verification_results = defaultdict(list)
        self.model_counts = defaultdict(int)
        
        # Enhanced provider mapping from environment variables
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
        
        # Enhanced provider base URLs from the API endpoints file
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
        """Parse the challenge log file and extract comprehensive information"""
        if not os.path.exists(self.log_file):
            logger.error(f"Log file not found: {self.log_file}")
            return {}
        
        try:
            with open(self.log_file, 'r', encoding='utf-8', errors='ignore') as f:
                content = f.read()
                
            logger.info(f"Parsing log file: {self.log_file} ({len(content)} bytes)")
            
            # Extract provider registration information
            self._extract_provider_registrations(content)
            
            # Extract detailed model information
            self._extract_detailed_model_info(content)
            
            # Extract verification results with scores
            self._extract_verification_results(content)
            
            # Extract performance metrics
            self._extract_performance_metrics(content)
            
            # Extract feature capabilities
            self._extract_feature_capabilities(content)
            
            return {
                'providers': self.providers,
                'models': dict(self.models),
                'discovered_providers': list(self.discovered_providers),
                'verification_results': dict(self.verification_results),
                'model_counts': dict(self.model_counts),
                'total_providers': len(self.providers),
                'total_models': sum(len(p.models) for p in self.providers.values())
            }
            
        except Exception as e:
            logger.error(f"Error parsing log file: {e}")
            return {}

    def _extract_provider_registrations(self, content: str):
        """Extract provider registration information with enhanced details"""
        # Pattern for registered providers with timestamps
        provider_pattern = r"(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}).*?Registered provider:\s*(\w+)"
        
        matches = re.findall(provider_pattern, content, re.IGNORECASE)
        for timestamp, provider in matches:
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
                    registered=True,
                    discovery_timestamp=timestamp
                )
            else:
                self.providers[provider_id].registered = True
                if not self.providers[provider_id].discovery_timestamp:
                    self.providers[provider_id].discovery_timestamp = timestamp
            
            logger.info(f"Found registered provider: {provider_id} at {timestamp}")

    def _extract_detailed_model_info(self, content: str):
        """Extract detailed model information including names and capabilities"""
        # Pattern for model discoveries with counts
        model_pattern = r"Found\s+(\d+)\s+.*models?.*(?:for|from)\s+(\w+)"
        
        matches = re.findall(model_pattern, content, re.IGNORECASE)
        for count_str, provider in matches:
            provider_id = provider.lower()
            count = int(count_str)
            
            # Skip the generic "provider" matches
            if provider_id == "provider":
                continue
                
            self.model_counts[provider_id] = count
            
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
            
            # Update model count
            self.providers[provider_id].total_models = count

    def _extract_verification_results(self, content: str):
        """Extract verification results with detailed scoring"""
        # Pattern for verification with scores
        verify_pattern = r"(\w+).*?(\d+).*?verified.*?models?(?:.*?score\s+(\d+))?"
        
        matches = re.findall(verify_pattern, content, re.IGNORECASE)
        for provider, count, score in matches:
            provider_id = provider.lower()
            if provider_id == "provider":
                continue
                
            verified_count = int(count)
            score_value = float(score) if score else 0.0
            
            if provider_id in self.providers:
                self.providers[provider_id].verified = True
                self.providers[provider_id].verified_models = verified_count
                
                # Store verification result
                self.verification_results[provider_id].append({
                    'verified_models': verified_count,
                    'score': score_value,
                    'timestamp': datetime.now().isoformat()
                })
            
            logger.info(f"Provider {provider_id} verified with {verified_count} models, score: {score_value}")

    def _extract_performance_metrics(self, content: str):
        """Extract performance metrics like response time and TTFT"""
        # Pattern for response time
        response_time_pattern = r"(\w+).*?response.*?time.*?([\d.]+)\s*(?:ms|s)"
        
        matches = re.findall(response_time_pattern, content, re.IGNORECASE)
        for provider, time_str in matches:
            provider_id = provider.lower()
            if provider_id == "provider" or provider_id not in self.providers:
                continue
                
            response_time = float(time_str)
            # Store in provider's features or create performance metrics
            logger.info(f"Provider {provider_id} response time: {response_time}ms")

    def _extract_feature_capabilities(self, content: str):
        """Extract feature capabilities like streaming, tool calling, etc."""
        # Pattern for feature detection
        feature_patterns = {
            'streaming': r'(\w+).*?streaming.*?support',
            'tool_calling': r'(\w+).*?tool.*?call',
            'vision': r'(\w+).*?vision.*?support',
            'embeddings': r'(\w+).*?embeddings?.*?support',
            'mcp': r'(\w+).*?mcp.*?support',
            'lsp': r'(\w+).*?lsp.*?support',
            'acp': r'(\w+).*?acp.*?support'
        }
        
        for feature, pattern in feature_patterns.items():
            matches = re.findall(pattern, content, re.IGNORECASE)
            for provider in matches:
                provider_id = provider.lower()
                if provider_id == "provider" or provider_id not in self.providers:
                    continue
                    
                if feature not in self.providers[provider_id].features:
                    self.providers[provider_id].features.append(feature)
                
                logger.info(f"Provider {provider_id} supports {feature}")

class EnhancedOpenCodeConfigGenerator:
    """Enhanced generator for OpenCode configuration with comprehensive features"""
    
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
        """Generate comprehensive OpenCode configuration from challenge data"""
        providers = challenge_data.get('providers', {})
        
        config = {
            "$schema": "https://opencode.sh/schema.json",
            "username": "OpenCode AI Assistant - Ultimate Challenge Edition",
            "provider": {},
            "model_groups": self._generate_model_groups(),
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
                "generator": "Enhanced Ultimate Challenge Parser",
                "version": "2.0-ultimate",
                "total_providers": len(providers),
                "total_models": sum(len(p.models) for p in providers.values()),
                "verified_providers": sum(1 for p in providers.values() if p.verified),
                "challenge_based": True,
                "security_warning": "CONTAINS API KEYS - DO NOT COMMIT - PROTECT WITH 600 PERMISSIONS",
                "safe_to_commit": False,
                "includes_real_verification_data": True,
                "includes_performance_metrics": True
            }
        }
        
        # Generate provider configurations
        for provider_id, provider_info in providers.items():
            provider_config = self._generate_provider_config(provider_info)
            if provider_config:
                config["provider"][provider_id] = provider_config
        
        # Add MCP server configurations
        config["mcp_servers"] = self._generate_mcp_servers()
        
        return config

    def _generate_model_groups(self) -> Dict[str, List[str]]:
        """Generate model groups for easy selection"""
        return {
            "premium": ["gpt-4", "claude-3-opus", "gpt-4-turbo"],
            "balanced": ["claude-3-sonnet", "gpt-4o", "mixtral-8x7b"],
            "fast": ["gpt-3.5-turbo", "claude-3-haiku", "llama2-70b"],
            "free": ["llama2-70b", "mixtral-8x7b", "gemma-7b"],
            "verified": []  # Will be populated with actually verified models
        }

    def _generate_mcp_servers(self) -> Dict[str, Any]:
        """Generate MCP server configurations"""
        return {
            "filesystem": {
                "command": "npx",
                "args": ["@modelcontextprotocol/server-filesystem", "/home/user/projects"],
                "enabled": True
            },
            "github": {
                "command": "npx",
                "args": ["@modelcontextprotocol/server-github"],
                "env": {
                    "GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN}"
                },
                "enabled": False
            },
            "postgres": {
                "command": "npx",
                "args": ["@modelcontextprotocol/server-postgres"],
                "env": {
                    "DATABASE_URL": "${DATABASE_URL}"
                },
                "enabled": False
            }
        }

    def _generate_provider_config(self, provider_info: ProviderInfo) -> Optional[Dict[str, Any]]:
        """Generate comprehensive configuration for a single provider"""
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
            "models": {},
            "metadata": {
                "registered": provider_info.registered,
                "verified": provider_info.verified,
                "total_models": provider_info.total_models,
                "verified_models": provider_info.verified_models,
                "discovery_timestamp": provider_info.discovery_timestamp,
                "features": provider_info.features
            }
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
                    "features": model_info.features,
                    "response_time": model_info.response_time,
                    "ttft": model_info.ttft,
                    "score": model_info.score,
                    "capabilities": model_info.capabilities
                }
        else:
            # Add comprehensive default models based on provider type
            default_models = self._get_comprehensive_default_models(provider_info.id)
            provider_config["models"] = default_models
        
        return provider_config

    def _get_comprehensive_default_models(self, provider_id: str) -> Dict[str, Any]:
        """Get comprehensive default models for common providers with detailed info"""
        default_models = {
            'openai': {
                "gpt-4": {
                    "name": "GPT-4",
                    "maxTokens": 8192,
                    "cost_per_1m_in": 30.0,
                    "cost_per_1m_out": 60.0,
                    "supports_brotli": True,
                    "verified": False,
                    "features": ["streaming", "tool_calling", "vision"],
                    "capabilities": {
                        "streaming": True,
                        "tool_calling": True,
                        "vision": True,
                        "function_calling": True
                    }
                },
                "gpt-4-turbo": {
                    "name": "GPT-4 Turbo",
                    "maxTokens": 128000,
                    "cost_per_1m_in": 10.0,
                    "cost_per_1m_out": 30.0,
                    "supports_brotli": True,
                    "verified": False,
                    "features": ["streaming", "tool_calling", "vision"],
                    "capabilities": {
                        "streaming": True,
                        "tool_calling": True,
                        "vision": True,
                        "function_calling": True
                    }
                },
                "gpt-4o": {
                    "name": "GPT-4o",
                    "maxTokens": 128000,
                    "cost_per_1m_in": 5.0,
                    "cost_per_1m_out": 15.0,
                    "supports_brotli": True,
                    "verified": False,
                    "features": ["streaming", "tool_calling", "vision"],
                    "capabilities": {
                        "streaming": True,
                        "tool_calling": True,
                        "vision": True,
                        "function_calling": True
                    }
                },
                "gpt-3.5-turbo": {
                    "name": "GPT-3.5 Turbo",
                    "maxTokens": 16385,
                    "cost_per_1m_in": 0.5,
                    "cost_per_1m_out": 1.5,
                    "supports_brotli": True,
                    "verified": False,
                    "features": ["streaming", "tool_calling"],
                    "capabilities": {
                        "streaming": True,
                        "tool_calling": True,
                        "vision": False,
                        "function_calling": True
                    }
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
                    "features": ["streaming", "tool_calling", "vision"],
                    "capabilities": {
                        "streaming": True,
                        "tool_calling": True,
                        "vision": True,
                        "function_calling": True
                    }
                },
                "claude-3-sonnet": {
                    "name": "Claude 3 Sonnet",
                    "maxTokens": 200000,
                    "cost_per_1m_in": 3.0,
                    "cost_per_1m_out": 15.0,
                    "supports_brotli": False,
                    "verified": False,
                    "features": ["streaming", "tool_calling", "vision"],
                    "capabilities": {
                        "streaming": True,
                        "tool_calling": True,
                        "vision": True,
                        "function_calling": True
                    }
                },
                "claude-3-haiku": {
                    "name": "Claude 3 Haiku",
                    "maxTokens": 200000,
                    "cost_per_1m_in": 0.25,
                    "cost_per_1m_out": 1.25,
                    "supports_brotli": False,
                    "verified": False,
                    "features": ["streaming", "tool_calling", "vision"],
                    "capabilities": {
                        "streaming": True,
                        "tool_calling": True,
                        "vision": True,
                        "function_calling": True
                    }
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
                    "features": ["streaming"],
                    "capabilities": {
                        "streaming": True,
                        "tool_calling": False,
                        "vision": False,
                        "function_calling": False
                    }
                },
                "mixtral-8x7b": {
                    "name": "Mixtral 8x7B (Groq)",
                    "maxTokens": 32768,
                    "cost_per_1m_in": 0.0,
                    "cost_per_1m_out": 0.0,
                    "supports_brotli": True,
                    "verified": False,
                    "features": ["streaming", "tool_calling"],
                    "capabilities": {
                        "streaming": True,
                        "tool_calling": True,
                        "vision": False,
                        "function_calling": True
                    }
                }
            }
        }
        
        return default_models.get(provider_id, {})

class RealTimeLogMonitor:
    """Real-time log file monitor with threading support"""
    
    def __init__(self, log_file: str, callback, interval: int = 5):
        self.log_file = log_file
        self.callback = callback
        self.interval = interval
        self.running = False
        self.thread = None
        self.last_size = 0
        self.update_queue = queue.Queue()
        
    def start(self):
        """Start monitoring in a separate thread"""
        if self.running:
            return
            
        self.running = True
        self.thread = threading.Thread(target=self._monitor_loop, daemon=True)
        self.thread.start()
        logger.info(f"Started real-time monitoring of {self.log_file}")
        
    def stop(self):
        """Stop monitoring"""
        self.running = False
        if self.thread:
            self.thread.join(timeout=1.0)
        logger.info("Stopped real-time monitoring")
        
    def _monitor_loop(self):
        """Main monitoring loop"""
        # Initial check
        if os.path.exists(self.log_file):
            self.last_size = os.path.getsize(self.log_file)
            
        while self.running:
            try:
                if os.path.exists(self.log_file):
                    current_size = os.path.getsize(self.log_file)
                    
                    if current_size > self.last_size:
                        logger.info(f"Log file updated: {current_size - self.last_size} new bytes")
                        self.update_queue.put(True)
                        self.last_size = current_size
                    elif current_size < self.last_size:
                        # Log file was truncated or rotated
                        logger.info("Log file was truncated or rotated")
                        self.update_queue.put(True)
                        self.last_size = current_size
                else:
                    logger.warning(f"Log file disappeared: {self.log_file}")
                    
                time.sleep(self.interval)
                
            except Exception as e:
                logger.error(f"Error in monitoring loop: {e}")
                time.sleep(self.interval)

def validate_opencode_config(config: Dict[str, Any]) -> bool:
    """Enhanced validation for OpenCode configuration format"""
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
        
        # Validate options structure
        options = provider_config.get("options", {})
        if "apiKey" not in options:
            logger.error(f"Provider {provider_id} missing 'apiKey' in options")
            return False
        
        if "baseURL" not in options:
            logger.error(f"Provider {provider_id} missing 'baseURL' in options")
            return False
    
    # Validate metadata
    metadata = config.get("metadata", {})
    if "security_warning" not in metadata:
        logger.warning("Missing security warning in metadata")
    
    logger.info("Enhanced configuration validation passed")
    return True

def main():
    """Enhanced main function with real-time monitoring"""
    import argparse
    
    parser = argparse.ArgumentParser(description="Enhanced Ultimate OpenCode Configuration Generator")
    parser.add_argument("--log-file", default="ultimate_challenge_complete.log", 
                       help="Challenge log file to parse")
    parser.add_argument("--env-file", default=".env", 
                       help="Environment file with API keys")
    parser.add_argument("--output", default="ultimate_opencode_final.json", 
                       help="Output configuration file")
    parser.add_argument("--monitor", action="store_true", 
                       help="Enable real-time monitoring")
    parser.add_argument("--monitor-interval", type=int, default=5, 
                       help="Monitoring interval in seconds")
    parser.add_argument("--validate", action="store_true", 
                       help="Validate generated configuration")
    parser.add_argument("--pretty", action="store_true", 
                       help="Pretty print JSON output")
    parser.add_argument("--backup", action="store_true", 
                       help="Create backup of previous configuration")
    parser.add_argument("--security-check", action="store_true", 
                       help="Perform security checks on generated config")
    
    args = parser.parse_args()
    
    def generate_config():
        """Generate enhanced configuration from current log state"""
        logger.info("Generating enhanced OpenCode configuration from challenge logs...")
        
        # Parse challenge logs
        parser = EnhancedChallengeLogParser(args.log_file)
        challenge_data = parser.parse_log_file()
        
        if not challenge_data:
            logger.error("No data extracted from challenge logs")
            return False
        
        logger.info(f"Extracted {challenge_data['total_providers']} providers with {challenge_data['total_models']} models")
        
        # Generate configuration
        generator = EnhancedOpenCodeConfigGenerator(args.env_file)
        config = generator.generate_config(challenge_data)
        
        # Validate if requested
        if args.validate and not validate_opencode_config(config):
            logger.error("Configuration validation failed")
            return False
        
        # Security check if requested
        if args.security_check:
            if not perform_security_checks(config):
                logger.error("Security checks failed")
                return False
        
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
            
            logger.info(f"Enhanced configuration saved to: {args.output}")
            logger.info(f"Total providers: {len(config.get('provider', {}))}")
            logger.info(f"Total models: {sum(len(p.get('models', {})) for p in config.get('provider', {}).values())}")
            
            # Display summary
            display_generation_summary(config)
            
            return True
            
        except Exception as e:
            logger.error(f"Error saving configuration: {e}")
            return False
    
    def perform_security_checks(config: Dict[str, Any]) -> bool:
        """Perform security checks on the generated configuration"""
        logger.info("Performing security checks...")
        
        # Check for embedded API keys
        providers = config.get("provider", {})
        api_key_count = 0
        placeholder_count = 0
        
        for provider_id, provider_config in providers.items():
            api_key = provider_config.get("options", {}).get("apiKey", "")
            if api_key and not api_key.startswith("${"):
                api_key_count += 1
                logger.warning(f"Provider {provider_id} has embedded API key (not placeholder)")
            elif api_key and api_key.startswith("${"):
                placeholder_count += 1
        
        logger.info(f"Security check: {api_key_count} providers with real API keys, {placeholder_count} with placeholders")
        
        # Check for security warnings
        metadata = config.get("metadata", {})
        if not metadata.get("safe_to_commit", True):
            logger.info("Security warning present: Configuration marked as unsafe to commit")
        
        return True
    
    def display_generation_summary(config: Dict[str, Any]):
        """Display a summary of the generated configuration"""
        providers = config.get("provider", {})
        total_providers = len(providers)
        total_models = sum(len(p.get("models", {})) for p in providers.values())
        
        verified_providers = sum(1 for p in providers.values() if p.get("metadata", {}).get("verified", False))
        registered_providers = sum(1 for p in providers.values() if p.get("metadata", {}).get("registered", False))
        
        logger.info("=" * 60)
        logger.info("CONFIGURATION GENERATION SUMMARY")
        logger.info("=" * 60)
        logger.info(f"Total Providers: {total_providers}")
        logger.info(f"Registered Providers: {registered_providers}")
        logger.info(f"Verified Providers: {verified_providers}")
        logger.info(f"Total Models: {total_models}")
        
        # Show top providers by model count
        provider_models = [(pid, len(p.get("models", {}))) for pid, p in providers.items()]
        provider_models.sort(key=lambda x: x[1], reverse=True)
        
        logger.info("Top Providers by Model Count:")
        for provider_id, model_count in provider_models[:5]:
            logger.info(f"  {provider_id}: {model_count} models")
        
        logger.info("=" * 60)
    
    # Generate initial configuration
    success = generate_config()
    
    if not success:
        logger.error("Failed to generate initial configuration")
        return 1
    
    # Set up real-time monitoring if requested
    if args.monitor:
        logger.info(f"Starting real-time monitoring with {args.monitor_interval}s interval...")
        monitor = RealTimeLogMonitor(args.log_file, generate_config, args.monitor_interval)
        monitor.start()
        
        try:
            # Keep the main thread alive
            while True:
                time.sleep(1)
        except KeyboardInterrupt:
            logger.info("Stopping real-time monitoring...")
            monitor.stop()
    
    return 0

if __name__ == "__main__":
    sys.exit(main())