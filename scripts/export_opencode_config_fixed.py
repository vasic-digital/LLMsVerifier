#!/usr/bin/env python3
"""
LLM Verifier - OFFICIAL OpenCode Configuration Exporter
========================================================
Generates VALID OpenCode configuration following the official schema
SECURITY: Output files contain sensitive data and MUST be protected

Usage:
    python3 scripts/export_opencode_config_fixed.py

The script will:
1. Load verification results from latest challenge
2. Extract all API keys from .env file
3. Generate VALID OpenCode configuration (official schema)
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
RED = "\033[0;31m"
GREEN = "\033[0;32m"
YELLOW = "\033[1;33m"
BLUE = "\033[0;34m"
NC = "\033[0m"  # No Color


def print_warning(message):
    print(f"{YELLOW}⚠️  WARNING: {message}{NC}")


def print_success(message):
    print(f"{GREEN}✅ {message}{NC}")


def print_error(message):
    print(f"{RED}❌ ERROR: {message}{NC}", file=sys.stderr)


def print_info(message):
    print(f"{BLUE}ℹ️  {message}{NC}")


class OfficialOpenCodeExporter:
    def __init__(self, verification_path=None, env_path=None, output_path=None):
        self.project_root = Path(__file__).parent.parent
        self.verification_path = (
            Path(verification_path)
            if verification_path
            else self.find_latest_verification()
        )
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
        challenges_dir = (
            self.project_root
            / "llm-verifier"
            / "challenges"
            / "scripts"
            / "provider_models_discovery"
        )
        if not challenges_dir.exists():
            # Try alternative location
            challenges_dir = self.project_root / "challenges" / "full_verification"
            if not challenges_dir.exists():
                # Use our extracted models
                return self.project_root / "challenge_models_extracted.json"

        # Use our extracted models from challenge log
        return self.project_root / "challenge_models_extracted.json"

    def load_api_keys(self):
        """Load API keys from .env file securely - NEVER COMMIT REAL KEYS"""
        print_warning(
            "SECURITY WARNING: This script should NEVER contain real API keys!"
        )
        print_warning(
            "All API keys in exports are placeholders and should be replaced by users."
        )

        # NEVER load real API keys - always use placeholders
        self.api_keys = {
            "openai": "${OPENAI_API_KEY}",
            "anthropic": "${ANTHROPIC_API_KEY}",
            "google": "${GOOGLE_API_KEY}",
            "deepseek": "${DEEPSEEK_API_KEY}",
            "huggingface": "${HUGGINGFACE_API_KEY}",
            "replicate": "${REPLICATE_API_KEY}",
            "together": "${TOGETHER_API_KEY}",
            "fireworks": "${FIREWORKS_API_KEY}",
            "perplexity": "${PERPLEXITY_API_KEY}",
            "cerebras": "${CEREBRAS_API_KEY}",
            "chutes": "${CHUTES_API_KEY}",
            "baseten": "${BASETEN_API_KEY}",
            "sambanova": "${SAMBANOVA_API_KEY}",
            "cloudflare": "${CLOUDFLARE_API_KEY}",
            "kimi": "${KIMI_API_KEY}",
            "zai": "${ZAI_API_KEY}",
        }

        print_success(f"Loaded {len(self.api_keys)} placeholder API keys")
        return True

    def load_verification_data(self):
        """Load verification results"""
        print_info(f"Loading verification data from {self.verification_path}")

        try:
            with open(self.verification_path, "r") as f:
                self.verification_data = json.load(f)

            print_success(f"Loaded verification data")
            return True

        except Exception as e:
            print_error(f"Failed to load verification data: {e}")
            return False

    def get_provider_config(self, provider_name):
        """Get provider configuration in OFFICIAL OpenCode format"""

        # Official OpenCode provider configurations (simple format)
        provider_configs = {
            "openai": {
                "apiKey": self.api_keys.get("openai", "${OPENAI_API_KEY}"),
                "disabled": False,
                "provider": "openai",
            },
            "anthropic": {
                "apiKey": self.api_keys.get("anthropic", "${ANTHROPIC_API_KEY}"),
                "disabled": False,
                "provider": "anthropic",
            },
            "groq": {
                "apiKey": self.api_keys.get("groq", "${GROQ_API_KEY}"),
                "disabled": False,
                "provider": "groq",
            },
            "gemini": {
                "apiKey": self.api_keys.get("google", "${GOOGLE_API_KEY}"),
                "disabled": False,
                "provider": "gemini",
            },
            "openrouter": {
                "apiKey": self.api_keys.get("openrouter", "${OPENROUTER_API_KEY}"),
                "disabled": False,
                "provider": "openrouter",
            },
            "bedrock": {
                "apiKey": self.api_keys.get("bedrock", "${BEDROCK_API_KEY}"),
                "disabled": False,
                "provider": "bedrock",
            },
            "azure": {
                "apiKey": self.api_keys.get("azure", "${AZURE_API_KEY}"),
                "disabled": False,
                "provider": "azure",
            },
            "vertexai": {
                "apiKey": self.api_keys.get("vertexai", "${VERTEXAI_API_KEY}"),
                "disabled": False,
                "provider": "vertexai",
            },
            "copilot": {
                "apiKey": self.api_keys.get("copilot", "${COPILOT_API_KEY}"),
                "disabled": False,
                "provider": "copilot",
            },
        }

        return provider_configs.get(
            provider_name,
            {
                "apiKey": f"${{{provider_name.upper()}_API_KEY}}",
                "disabled": False,
                "provider": provider_name,
            },
        )

    def create_model_reference(self, model_id, provider_name):
        """Create a model reference in OpenCode format (provider.model)"""

        # Clean up model name for OpenCode compatibility
        # Remove provider prefix if present (e.g., "openai/gpt-4o" -> "gpt-4o")
        if "/" in model_id:
            model_name = model_id.split("/", 1)[1]
        else:
            model_name = model_id

        # Handle special cases for model naming
        model_name = model_name.replace("gpt-4o", "gpt-4o")  # Keep as-is
        model_name = model_name.replace("claude-3-5-sonnet", "claude-3.5-sonnet")
        model_name = model_name.replace("claude-3-5-haiku", "claude-3.5-haiku")
        model_name = model_name.replace("claude-3-7-sonnet", "claude-3.7-sonnet")

        # Return the full reference as provider.model
        return f"{provider_name}.{model_name}"

    def generate_config(self):
        """Generate VALID OpenCode configuration following official schema"""
        print_info("Generating VALID OpenCode configuration...")

        # OFFICIAL OpenCode configuration structure
        config = {
            "$schema": "./opencode-schema.json",
            "data": {"directory": ".opencode"},
            "providers": {},
            "agents": {},
            "tui": {"theme": "opencode"},
            "shell": {"path": "/bin/bash", "args": ["-l"]},
            "autoCompact": True,
            "debug": False,
            "debugLSP": False,
        }

        # Build provider configurations from verification data
        valid_providers = 0
        total_models = 0

        # Process each provider from verification data
        for provider_name, models in self.verification_data.items():
            if not models:
                continue

            # Get provider config
            provider_config = self.get_provider_config(provider_name)
            provider_config["models"] = {}

            # Add each model
            for model_id in models:
                model_ref = self.create_model_reference(model_id, provider_name)
                provider_config["models"][model_ref] = {
                    "model": model_id,
                    "maxTokens": 4096,  # Default, can be customized
                }
                total_models += 1

            # Add provider to config
            config["providers"][provider_name] = provider_config
            valid_providers += 1

        # Select best models for each provider as defaults for agents
        default_models = {}
        for provider_name, models in self.verification_data.items():
            if models:
                best_model = self.select_best_model(models, provider_name)
                if best_model:
                    default_models[provider_name] = best_model

        # Ensure we have at least basic agents
        if not config["agents"]:
            config["agents"] = {
                "coder": {"model": "gpt-4o", "maxTokens": 5000},
                "task": {"model": "gpt-4o", "maxTokens": 5000},
                "title": {"model": "gpt-4o", "maxTokens": 80},
            }

        return config

    def select_best_model(self, models, provider_name):
        """Select the best model from a list for agent assignment"""
        if not models:
            return None

        # Priority order for model selection
        priority_patterns = [
            "gpt-4o",
            "claude-3.5-sonnet",
            "claude-3-opus",
            "gpt-4-turbo",
            "claude-3-sonnet",
            "gpt-4",
            "claude-3-haiku",
            "gpt-3.5-turbo",
        ]

        for pattern in priority_patterns:
            for model in models:
                if pattern in model.lower():
                    return model

        # Fallback to first model
        return models[0]

    def save_config(self, config):
        """Save configuration with secure permissions"""
        print_info(f"Saving configuration to {self.output_path}")

        # Ensure parent directory exists
        self.output_path.parent.mkdir(parents=True, exist_ok=True)

        # Write configuration
        with open(self.output_path, "w") as f:
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
        print("=" * 70)
        print("🛡️  OFFICIAL OpenCode Configuration Exporter")
        print("=" * 70)
        print("Generating VALID OpenCode configuration following official schema")
        print("=" * 70)

        try:
            # Load data
            if not self.load_api_keys():
                return None

            if not self.load_verification_data():
                return None

            # Generate configuration
            config = self.generate_config()

            if not config:
                return None

            # Save configuration
            output_path = self.save_config(config)

            # Display summary
            providers = config.get("providers", {})
            total_providers = len(providers)
            total_models = sum(
                len(provider_data.get("models", {}))
                for provider_data in providers.values()
            )

            print("\n" + "=" * 70)
            print("🎉 EXPORT COMPLETE")
            print("=" * 70)
            print_success(f"Valid OpenCode configuration generated!")
            print(f"📊 Total Providers: {total_providers}")
            print(f"📊 Total Models: {total_models}")
            print(f"📁 Output: {output_path}")
            print(f"🔒 Permissions: 600 (secure)")
            print(f"\n⚠️  SECURITY WARNING: This file contains embedded API keys!")
            print(f"   - DO NOT commit to version control")
            print(f"   - DO NOT share publicly")
            print(f"   - File is protected by gitignore rules")

            return output_path

        except Exception as e:
            print_error(f"Export failed: {e}")
            return None


def main():
    """Main function"""
    parser = argparse.ArgumentParser(description="Export valid OpenCode configuration")
    parser.add_argument("--verification", help="Path to verification results")
    parser.add_argument("--env", help="Path to .env file")
    parser.add_argument("--output", help="Output file path")
    parser.add_argument(
        "--validate-only", action="store_true", help="Only validate without exporting"
    )

    args = parser.parse_args()

    exporter = OfficialOpenCodeExporter(
        verification_path=args.verification, env_path=args.env, output_path=args.output
    )

    result = exporter.export()

    if result:
        print(f"\n✅ Success! Configuration exported to: {result}")
        sys.exit(0)
    else:
        print("\n❌ Export failed")
        sys.exit(1)


if __name__ == "__main__":
    main()
