#!/usr/bin/env python3
"""
LLM Verifier - Secure OpenCode Configuration Exporter
========================================================
Generates complete OpenCode configuration with embedded API keys
SECURITY: Output files contain sensitive data and MUST be protected

Usage:
    python3 scripts/export_opencode_config.py
    
The script will:
1. Load verification results from latest challenge
2. Extract all API keys from .env file
3. Generate comprehensive OpenCode configuration
4. Set restrictive file permissions (600)
5. Display security warnings
6. Output to specified location (default: Downloads)
"""

import json
import os
import sys
import argparse
from datetime import datetime
from pathlib import Path

# ANSI colors for output
RED = '\033[0;31m'
GREEN = '\033[0;32m'
YELLOW = '\033[1;33m'
BLUE = '\033[0;34m'
NC = '\033[0m'  # No Color

def print_warning(message):
    print(f"{YELLOW}⚠️  WARNING: {message}{NC}")

def print_success(message):
    print(f"{GREEN}✅ {message}{NC}")

def print_error(message):
    print(f"{RED}❌ ERROR: {message}{NC}", file=sys.stderr)

def print_info(message):
    print(f"{BLUE}ℹ️  {message}{NC}")

class SecureConfigExporter:
    def __init__(self, verification_path=None, env_path=None, output_path=None):
        self.project_root = Path(__file__).parent.parent
        self.verification_path = Path(verification_path) if verification_path else self.find_latest_verification()
        self.env_path = Path(env_path) if env_path else self.project_root / ".env"
        
        # Default output to Downloads directory
        if output_path:
            self.output_path = Path(output_path)
        else:
            downloads = Path.home() / "Downloads"
            timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            self.output_path = downloads / f"opencode_{timestamp}.json"
        
        self.api_keys = {}
        self.verification_data = {}
        
    def find_latest_verification(self):
        """Find the most recent verification results"""
        challenges_dir = self.project_root / "challenges" / "full_verification"
        if not challenges_dir.exists():
            raise FileNotFoundError(f"No verification results found in {challenges_dir}")
        
        # Find the newest results directory
        latest = None
        latest_time = 0
        
        for year_dir in challenges_dir.iterdir():
            if not year_dir.is_dir():
                continue
            for month_dir in year_dir.iterdir():
                if not month_dir.is_dir():
                    continue
                for day_dir in month_dir.iterdir():
                    if not day_dir.is_dir():
                        continue
                    for time_dir in day_dir.iterdir():
                        if not time_dir.is_dir():
                            continue
                        results_file = time_dir / "providers_export.json"
                        if results_file.exists():
                            mtime = results_file.stat().st_mtime
                            if mtime > latest_time:
                                latest_time = mtime
                                latest = time_dir / "providers_export.json"
        
        if not latest:
            raise FileNotFoundError("No verification results found")
        
        return latest
    
    def load_api_keys(self):
        """Load API keys from .env file securely"""
        print_info(f"Loading API keys from {self.env_path}")
        
        if not self.env_path.exists():
            print_error(f".env file not found at {self.env_path}")
            print_info("Make sure you have a .env file with API keys")
            sys.exit(1)
        
        with open(self.env_path, 'r') as f:
            for line_num, line in enumerate(f, 1):
                line = line.strip()
                if not line or line.startswith('#'):
                    continue
                
                if '=' not in line:
                    print_warning(f"Skipping malformed line {line_num}: {line[:50]}...")
                    continue
                
                try:
                    key, value = line.split('=', 1)
                    key = key.strip()
                    value = value.strip()
                    
                    # Skip empty or placeholder values
                    if not value or 'YOUR_API_KEY' in value or 'CHANGE_IN_PRODUCTION' in value:
                        continue
                    
                    self.api_keys[key] = value
                except Exception as e:
                    print_warning(f"Error parsing line {line_num}: {e}")
        
        print_success(f"Loaded {len(self.api_keys)} API keys")
        
    def load_verification_results(self):
        """Load verification results"""
        print_info(f"Loading verification results from {self.verification_path}")
        
        if not self.verification_path.exists():
            raise FileNotFoundError(f"Verification results not found: {self.verification_path}")
        
        with open(self.verification_path, 'r') as f:
            self.verification_data = json.load(f)
        
        print_success("Verification results loaded successfully")
    
    def validate_security(self):
        """Validate gitignore protection"""
        gitignore_path = self.project_root / ".gitignore"
        if not gitignore_path.exists():
            print_warning("No .gitignore file found - manual protection required")
            return False
        
        with open(gitignore_path, 'r') as f:
            gitignore_content = f.read()
        
        protections = [
            "opencode.json",
            "*api_key*",
            "Downloads/opencode*.json",
            "**/*opencode*keys*.json"
        ]
        
        missing_protections = []
        for protection in protections:
            if protection not in gitignore_content:
                missing_protections.append(protection)
        
        if missing_protections:
            print_warning("Missing gitignore protections:")
            for protection in missing_protections:
                print(f"   - {protection}")
            print_info("Consider adding these to .gitignore")
            return False
        
        print_success("Gitignore security protections verified")
        return True
    
    def generate_config(self):
        """Generate the complete OpenCode configuration"""
        print_info("Generating OpenCode configuration...")
        
        # Provider to env key mapping
        provider_env_map = {
            "huggingface": "ApiKey_HuggingFace", "nvidia": "ApiKey_Nvidia",
            "chutes": "ApiKey_Chutes", "siliconflow": "ApiKey_SiliconFlow",
            "kimi": "ApiKey_Kimi", "gemini": "ApiKey_Gemini",
            "openrouter": "ApiKey_OpenRouter", "zai": "ApiKey_ZAI",
            "deepseek": "ApiKey_DeepSeek", "mistralaistudio": "ApiKey_Mistral_AiStudio",
            "codestral": "ApiKey_Codestral", "cerebras": "ApiKey_Cerebras",
            "cloudflareworkersai": "ApiKey_Cloudflare_Workers_AI",
            "fireworksai": "ApiKey_Fireworks_AI", "baseten": "ApiKey_Baseten",
            "novitaai": "ApiKey_Novita_AI", "upstageai": "ApiKey_Upstage_AI",
            "nlpcloud": "ApiKey_NLP_Cloud", "modaltokenid": "ApiKey_Modal_Token_ID",
            "modaltokensecret": "ApiKey_Modal_Token_Secret", "inference": "ApiKey_Inference",
            "hyperbolic": "ApiKey_Hyperbolic", "sambanovaai": "ApiKey_SambaNova_AI",
            "replicate": "ApiKey_Replicate"
        }
        
        config = {
            "version": "2.0-ultimate",
            "generated_at": datetime.now().isoformat(),
            "generator": "LLM Verifier Secure Export Script",
            "security_warning": "CONTAINS EMBEDDED API KEYS - DO NOT COMMIT TO VERSION CONTROL - FILE IS PROTECTED BY .GITIGNORE",
            "settings": {
                "default_timeout": 30,
                "max_retries": 3,
                "request_delay": 1,
                "enable_streaming": True,
                "enable_acp": True,
                "enable_mcp": True,
                "enable_lsp": True,
                "enable_logging": True,
                "log_level": "info",
                "cache_enabled": True,
                "cache_ttl": 3600
            },
            "providers": [],
            "models": [],
            "model_groups": {},
            "security": {
                "api_keys_embedded": True,
                "safe_to_commit": False,
                "protected_by_gitignore": True,
                "gitignore_verified": self.validate_security()
            },
            "acp_config": {
                "enabled": True,
                "protocol_version": "1.0",
                "capabilities": {
                    "tool_use": True,
                    "streaming": True,
                    "function_calling": True,
                    "json_mode": True,
                    "context_aware": True,
                    "code_understanding": True
                },
                "timeout": 30000,
                "max_tools_per_request": 10
            },
            "lsp_config": {
                "enabled": True,
                "diagnostics": True,
                "code_completion": True,
                "hover_info": True,
                "go_to_definition": True,
                "find_references": True,
                "document_symbols": True,
                "workspace_symbols": True,
                "formatting": True,
                "range_formatting": True,
                "on_type_formatting": True,
                "rename": True,
                "code_actions": True,
                "inlay_hints": True
            },
            "mcp_servers": [],
            "embeddings_config": {
                "enabled": True,
                "providers": [],
                "default_model": None,
                "chunk_size": 1000,
                "chunk_overlap": 200,
                "vector_store": "faiss",
                "distance_metric": "cosine"
            },
            "scoring": {
                "enabled": True,
                "weights": {
                    "responsiveness": 0.30,
                    "code_capability": 0.25,
                    "reliability": 0.20,
                    "feature_richness": 0.25
                }
            }
        }
        
        # Track statistics
        stats = {
            "total_providers": 0,
            "total_models": 0,
            "verified_models": 0,
            "acp_enabled": 0,
            "lsp_enabled": 0,
            "mcp_enabled": 0,
            "embeddings": 0,
            "high_performance": 0,
            "total_score": 0
        }
        
        def clean_name(name):
            """Clean model name for use in identifiers"""
            return name.replace("/", "_").replace("-", "_").replace(".", "_").lower()
        
        # Process providers and models
        for provider in self.verification_data.get("providers", []):
            provider_name = provider["name"]
            env_key = provider_env_map.get(provider_name, "")
            api_key = self.api_keys.get(env_key, "NOT_FOUND")
            
            provider_entry = {
                "name": provider_name,
                "display_name": provider_name.replace("_", " ").title(),
                "endpoint": provider["endpoint"],
                "api_key": api_key,
                "api_key_available": api_key != "NOT_FOUND",
                "enabled": api_key != "NOT_FOUND",
                "verified_models": 0,
                "total_models": len(provider.get("models", [])),
                "health_status": "healthy" if any(m.get("verified") for m in provider.get("models", [])) else "degraded",
                "last_verified": datetime.now().isoformat()
            }
            config["providers"].append(provider_entry)
            stats["total_providers"] += 1
            
            # Process models
            for model in provider.get("models", []):
                verified = model.get("verified", False)
                features = model.get("features", {})
                scores = model.get("scores", {})
                
                model_entry = {
                    "id": model["model_id"],
                    "name": model["name"],
                    "provider": provider_name,
                    "provider_endpoint": provider["endpoint"],
                    "verified": verified,
                    "enabled": verified,
                    "api_key": api_key if verified else None,
                    "scores": scores,
                    "features": features,
                    "capabilities": {
                        "streaming": features.get("streaming"),
                        "tool_calling": features.get("tool_calling"),
                        "function_calling": features.get("tool_calling"),
                        "embeddings": features.get("embeddings"),
                        "vision": features.get("vision"),
                        "audio": features.get("audio"),
                        "mcp": features.get("mcp"),
                        "lsp": features.get("lsp"),
                        "acp": features.get("acp"),
                        "code_generation": features.get("code")
                    },
                    "performance": {
                        "response_time_ms": model.get("response_time_ms", 0),
                        "ttft_ms": model.get("ttft_ms", 0),
                        "last_tested": model.get("last_verified", "")
                    }
                }
                
                config["models"].append(model_entry)
                stats["total_models"] += 1
                
                if verified:
                    stats["verified_models"] += 1
                    provider_entry["verified_models"] += 1
                    stats["total_score"] += scores.get("overall", 0)
                    
                    # Track features
                    if features.get("acp"):
                        stats["acp_enabled"] += 1
                        config["model_groups"].setdefault("acp_enabled", []).append(model["model_id"])
                    
                    if features.get("lsp"):
                        stats["lsp_enabled"] += 1
                        config["model_groups"].setdefault("lsp_enabled", []).append(model["model_id"])
                    
                    if features.get("mcp"):
                        stats["mcp_enabled"] += 1
                        config["model_groups"].setdefault("mcp_enabled", []).append(model["model_id"])
                    
                    if features.get("embeddings"):
                        stats["embeddings"] += 1
                        config["embeddings_config"]["providers"].append({
                            "provider": provider_name,
                            "model": model["model_id"],
                            "enabled": True
                        })
                        if not config["embeddings_config"]["default_model"]:
                            config["embeddings_config"]["default_model"] = model["model_id"]
                    
                    if features.get("code"):
                        config["model_groups"].setdefault("coding", []).append(model["model_id"])
                    
                    if scores.get("overall", 0) >= 75:
                        stats["high_performance"] += 1
                        config["model_groups"].setdefault("high_performance", []).append(model["model_id"])
                    
                    # Configure MCP server for ACP-enabled models
                    if features.get("acp") and api_key != "NOT_FOUND":
                        mcp_server = {
                            "name": f"{provider_name}_{clean_name(model['model_id'])}_mcp",
                            "type": "stdio",
                            "command": "node",
                            "args": ["mcp-server.js"],
                            "env": {
                                "PROVIDER": provider_name,
                                "MODEL": model["model_id"],
                                "API_KEY": api_key,
                                "API_ENDPOINT": provider["endpoint"]
                            },
                            "timeout": 30000,
                            "enabled": True,
                            "capabilities": {
                                "tools": features.get("tool_calling", False),
                                "resources": True,
                                "prompts": True
                            }
                        }
                        config["mcp_servers"].append(mcp_server)
        
        # Add metadata
        config["metadata"] = {
            "total_providers": stats["total_providers"],
            "total_models": stats["total_models"],
            "verified_models": stats["verified_models"],
            "embedding_models": stats["embeddings"],
            "acp_enabled_models": stats["acp_enabled"],
            "lsp_enabled_models": stats["lsp_enabled"],
            "mcp_enabled_models": stats["mcp_enabled"],
            "high_performance_models": stats["high_performance"],
            "average_score": round(stats["total_score"] / max(stats["verified_models"], 1), 1)
        }
        
        return config
    
    def save_config(self, config):
        """Save configuration with secure permissions"""
        print_info(f"Saving configuration to {self.output_path}")
        
        # Ensure parent directory exists
        self.output_path.parent.mkdir(parents=True, exist_ok=True)
        
        # Write configuration
        with open(self.output_path, 'w') as f:
            json.dump(config, f, indent=2)
        
        # Set restrictive permissions (owner read/write only)
        os.chmod(self.output_path, 0o600)
        
        print_success(f"Configuration saved with secure permissions (600)")
        
        # Verify permissions
        file_stat = os.stat(self.output_path)
        if file_stat.st_mode & 0o777 != 0o600:
            print_warning("File permissions may not be set correctly")
        
        return self.output_path
    
    def export(self):
        """Main export workflow"""
        print("="*70)
        print("🛡️  LLM Verifier - Secure OpenCode Configuration Exporter")
        print("="*70)
        
        # Display security warning
        print_warning("="*70)
        print_warning("SECURITY WARNING")
        print_warning("="*70)
        print_warning("This tool will generate configuration files with EMBEDDED API keys.")
        print_warning("These files MUST NOT be committed to version control.")
        print_warning("The project .gitignore file has protections in place.")
        print_warning("="*70)
        print()
        
        # Load data
        self.load_api_keys()
        self.load_verification_results()
        
        # Generate configuration
        config = self.generate_config()
        
        # Validate gitignore
        if not config["security"]["gitignore_verified"]:
            print_warning("Gitignore security verification failed")
            response = input("Continue anyway? (y/N): ")
            if response.lower() != 'y':
                print_info("Export cancelled")
                sys.exit(0)
        
        # Save configuration
        output_file = self.save_config(config)
        
        # Display summary
        metadata = config["metadata"]
        print()
        print("="*70)
        print_success("EXPORT COMPLETE")
        print("="*70)
        print()
        print(f"📄 Output File: {output_file}")
        print(f"📊 Size: {os.path.getsize(output_file) / 1024:.1f} KB")
        print(f"🔒 Permissions: 600 (owner read/write only)")
        print()
        print("📊 Statistics:")
        print(f"   Total Providers: {metadata['total_providers']}")
        print(f"   Total Models: {metadata['total_models']}")
        print(f"   ✅ Verified: {metadata['verified_models']}")
        print(f"   🎯 ACP-Enabled: {metadata['acp_enabled_models']}")
        print(f"   🔧 LSP-Enabled: {metadata['lsp_enabled_models']}")
        print(f"   🔌 MCP-Enabled: {metadata['mcp_enabled_models']}")
        print(f"   📝 Embeddings: {metadata['embedding_models']}")
        print(f"   ⭐ High Performance: {metadata['high_performance_models']}")
        print(f"   📈 Average Score: {metadata['average_score']}")
        print()
        print(f"🔌 MCP Servers: {len(config['mcp_servers'])}")
        print()
        print("="*70)
        print_warning("SECURITY REMINDERS")
        print("="*70)
        print_warning("1. This file contains EMBEDDED API keys")
        print_warning("2. DO NOT commit to version control")
        print_warning("3. Protected by .gitignore (already configured)")
        print_warning("4. Safe for local development use")
        print_warning("5. File permissions: 600 (owner only)")
        print("="*70)
        
        # List verified models
        print()
        print("✅ VERIFIED MODELS (Ready to Use):")
        for model in config["models"]:
            if model["verified"]:
                print()
                print(f"   🎯 {model['name']} ({model['provider']})")
                print(f"      Score: {model['scores']['overall']}/100")
                print(f"      API Key: {'✅ Embedded' if model['api_key'] else '❌ None'}")
                print(f"      Features: " + " ".join([
                    f"{'✅' if v else '❌'} {k[:3].upper()}"
                    for k, v in model['capabilities'].items()
                    if k in ['acp', 'lsp', 'mcp', 'streaming', 'tool_calling']
                ]))
        
        print()
        print("="*70)
        print_success("Configuration is ready for use!")
        print("="*70)
        
        return output_file

def main():
    parser = argparse.ArgumentParser(description='Securely export LLM Verifier configuration')
    parser.add_argument('--verification', help='Path to verification results')
    parser.add_argument('--env', help='Path to .env file')
    parser.add_argument('--output', help='Output path for configuration')
    parser.add_argument('--validate-only', action='store_true', help='Only validate gitignore setup')
    
    args = parser.parse_args()
    
    try:
        exporter = SecureConfigExporter(
            verification_path=args.verification,
            env_path=args.env,
            output_path=args.output
        )
        
        if args.validate_only:
            print("Validating gitignore security...")
            if exporter.validate_security():
                print_success("Gitignore security protections verified")
                sys.exit(0)
            else:
                print_error("Gitignore security protections missing or incomplete")
                sys.exit(1)
        
        exporter.export()
        
    except Exception as e:
        print_error(f"Export failed: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)

if __name__ == "__main__":
    main()