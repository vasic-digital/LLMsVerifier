#!/usr/bin/env python3
"""
Final OpenCode Configuration Comparison and Validation Tool

This script performs a comprehensive comparison between our generated configurations
and OpenCode standards, identifying any discrepancies and providing fixes.
"""

import json
import os
import sys
import re
from datetime import datetime
from typing import Dict, List, Any, Optional

class OpenCodeValidator:
    """Validates configurations against OpenCode standards"""
    
    def __init__(self):
        self.errors = []
        self.warnings = []
        self.schema_url = "https://opencode.sh/schema.json"
        
        # OpenCode standard field mappings
        self.required_fields = {
            "top_level": ["$schema", "username", "provider"],
            "provider": ["options", "models"],
            "options": ["apiKey", "baseURL"],
            "model": ["id", "name", "displayName", "provider", "maxTokens"]
        }
        
        # Expected field formats
        self.field_formats = {
            "apiKey": r"^\$\{[A-Z_]+\}$|^sk-[a-zA-Z0-9]+$",
            "baseURL": r"^https?://.+/v1/?$",
            "provider_id": r"^[a-z][a-zA-Z0-9]*$",
            "model_id": r"^[a-zA-Z0-9._-]+$"
        }
    
    def validate_schema(self, config: Dict[str, Any]) -> bool:
        """Validate top-level schema requirements"""
        if "$schema" not in config:
            self.errors.append("Missing required field: $schema")
            return False
            
        if config["$schema"] != self.schema_url:
            self.warnings.append(f"Schema URL mismatch: expected {self.schema_url}, got {config['$schema']}")
        
        return True
    
    def validate_provider_structure(self, providers: Dict[str, Any]) -> bool:
        """Validate provider section structure"""
        if not providers:
            self.errors.append("No providers found in configuration")
            return False
        
        for provider_id, provider_data in providers.items():
            # Check provider ID format (should be camelCase)
            if not re.match(self.field_formats["provider_id"], provider_id):
                self.warnings.append(f"Provider ID '{provider_id}' should be camelCase")
            
            # Check required provider fields
            for field in self.required_fields["provider"]:
                if field not in provider_data:
                    self.errors.append(f"Provider '{provider_id}' missing required field: {field}")
                    continue
            
            # Validate options structure
            if "options" in provider_data:
                self.validate_options(provider_id, provider_data["options"])
            
            # Validate models structure
            if "models" in provider_data:
                self.validate_models(provider_id, provider_data["models"])
        
        return len(self.errors) == 0
    
    def validate_options(self, provider_id: str, options: Dict[str, Any]) -> None:
        """Validate provider options"""
        for field in self.required_fields["options"]:
            if field not in options:
                self.errors.append(f"Provider '{provider_id}' options missing required field: {field}")
                continue
            
            # Validate field formats
            if field == "apiKey":
                if not re.match(self.field_formats["apiKey"], str(options[field])):
                    self.warnings.append(f"Provider '{provider_id}' apiKey format may be invalid")
            
            elif field == "baseURL":
                if not re.match(self.field_formats["baseURL"], str(options[field])):
                    self.warnings.append(f"Provider '{provider_id}' baseURL should end with /v1")
    
    def validate_models(self, provider_id: str, models: Dict[str, Any]) -> None:
        """Validate model definitions"""
        if not models:
            self.warnings.append(f"Provider '{provider_id}' has no models")
            return
        
        for model_id, model_data in models.items():
            # Check model ID format
            if not re.match(self.field_formats["model_id"], str(model_id)):
                self.warnings.append(f"Model ID '{model_id}' for provider '{provider_id}' has unusual format")
            
            # Check required model fields
            for field in self.required_fields["model"]:
                if field not in model_data:
                    self.errors.append(f"Model '{model_id}' for provider '{provider_id}' missing required field: {field}")
                    continue
            
            # Validate provider structure within model
            if "provider" in model_data:
                provider_info = model_data["provider"]
                if not isinstance(provider_info, dict):
                    self.errors.append(f"Model '{model_id}' provider field should be an object")
                else:
                    if "id" not in provider_info:
                        self.errors.append(f"Model '{model_id}' provider missing 'id' field")
                    if "npm" not in provider_info:
                        self.warnings.append(f"Model '{model_id}' provider missing 'npm' field")
    
    def validate_casing_consistency(self, config: Dict[str, Any]) -> None:
        """Check for consistent casing (camelCase vs snake_case)"""
        def check_dict_casing(data: Dict[str, Any], path: str = "") -> None:
            for key, value in data.items():
                current_path = f"{path}.{key}" if path else key
                
                # Check for snake_case in model fields (should be camelCase)
                if path.endswith(".models") and "_" in key and key not in ["cost_per_1m_in", "cost_per_1m_out"]:
                    self.warnings.append(f"Field '{current_path}' uses snake_case, consider camelCase")
                
                if isinstance(value, dict):
                    check_dict_casing(value, current_path)
        
        check_dict_casing(config)
    
    def compare_with_reference(self, our_config: Dict[str, Any], reference_config: Dict[str, Any]) -> Dict[str, Any]:
        """Compare our configuration with a reference OpenCode configuration"""
        comparison = {
            "missing_providers": [],
            "extra_providers": [],
            "provider_differences": {},
            "structural_differences": []
        }
        
        our_providers = set(our_config.get("provider", {}).keys())
        ref_providers = set(reference_config.get("provider", {}).keys())
        
        comparison["missing_providers"] = list(ref_providers - our_providers)
        comparison["extra_providers"] = list(our_providers - ref_providers)
        
        # Compare common providers
        common_providers = our_providers & ref_providers
        for provider_id in common_providers:
            our_provider = our_config["provider"][provider_id]
            ref_provider = reference_config["provider"][provider_id]
            
            differences = {}
            
            # Compare models
            our_models = set(our_provider.get("models", {}).keys())
            ref_models = set(ref_provider.get("models", {}).keys())
            
            missing_models = ref_models - our_models
            extra_models = our_models - ref_models
            
            if missing_models:
                differences["missing_models"] = list(missing_models)
            if extra_models:
                differences["extra_models"] = list(extra_models)
            
            # Compare structure
            our_keys = set(our_provider.keys())
            ref_keys = set(ref_provider.keys())
            
            missing_keys = ref_keys - our_keys
            extra_keys = our_keys - ref_keys
            
            if missing_keys:
                differences["missing_keys"] = list(missing_keys)
            if extra_keys:
                differences["extra_keys"] = list(extra_keys)
            
            if differences:
                comparison["provider_differences"][provider_id] = differences
        
        return comparison
    
    def generate_fixes(self, config: Dict[str, Any]) -> List[str]:
        """Generate suggested fixes for identified issues"""
        fixes = []
        
        for error in self.errors:
            if "missing required field" in error:
                field = error.split("missing required field: ")[1]
                if "$schema" in error:
                    fixes.append(f"Add: '$schema': '{self.schema_url}' at top level")
                elif "username" in error:
                    fixes.append("Add: 'username': 'Your Name' at top level")
                elif "provider" in error:
                    fixes.append("Add provider configuration section")
        
        for warning in self.warnings:
            if "snake_case" in warning and "consider camelCase" in warning:
                field = warning.split("Field '")[1].split("'")[0]
                camel_case = re.sub(r'_([a-z])', lambda x: x.group(1).upper(), field.split(".")[-1])
                fixes.append(f"Change '{field}' to use camelCase: '{camel_case}'")
        
        return fixes
    
    def validate_configuration(self, config: Dict[str, Any]) -> Dict[str, Any]:
        """Perform complete validation of configuration"""
        self.errors = []
        self.warnings = []
        
        print("🔍 Validating OpenCode configuration...")
        
        # Basic schema validation
        self.validate_schema(config)
        
        # Provider structure validation
        if "provider" in config:
            self.validate_provider_structure(config["provider"])
        
        # Casing consistency check
        self.validate_casing_consistency(config)
        
        # Generate fixes
        fixes = self.generate_fixes(config)
        
        return {
            "valid": len(self.errors) == 0,
            "errors": self.errors,
            "warnings": self.warnings,
            "fixes": fixes,
            "summary": {
                "total_errors": len(self.errors),
                "total_warnings": len(self.warnings),
                "total_fixes": len(fixes)
            }
        }

def load_json_file(filepath: str) -> Optional[Dict[str, Any]]:
    """Load JSON file with error handling"""
    try:
        with open(filepath, 'r') as f:
            return json.load(f)
    except Exception as e:
        print(f"❌ Error loading {filepath}: {e}")
        return None

def main():
    """Main comparison and validation function"""
    print("🚀 Starting Final OpenCode Configuration Comparison")
    print("=" * 60)
    
    # Load configurations
    print("📁 Loading configurations...")
    
    # Our current configuration
    our_config = load_json_file("/media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier/ultimate_opencode_config.json")
    if not our_config:
        print("❌ Failed to load our configuration")
        return 1
    
    # Reference configuration (from tests)
    reference_config = {
        "$schema": "https://opencode.sh/schema.json",
        "username": "OpenCode AI Assistant",
        "provider": {
            "openai": {
                "options": {
                    "apiKey": "${OPENAI_API_KEY}",
                    "baseURL": "https://api.openai.com/v1"
                },
                "models": {
                    "gpt-4": {
                        "id": "gpt-4",
                        "name": "GPT-4",
                        "displayName": "GPT-4",
                        "provider": {
                            "id": "openai",
                            "npm": "@openai/sdk"
                        },
                        "maxTokens": 8192,
                        "supportsHTTP3": True
                    }
                }
            }
        },
        "agent": {
            "code": {
                "model": "openai/gpt-4",
                "prompt": "You are a senior software engineer...",
                "tools": {
                    "bash": True,
                    "docker": True,
                    "git": True,
                    "lsp": True,
                    "webfetch": True
                },
                "temperature": 0.2,
                "maxSteps": 10
            }
        },
        "mcp": {
            "servers": []
        }
    }
    
    # Initialize validator
    validator = OpenCodeValidator()
    
    # Validate our configuration
    validation_result = validator.validate_configuration(our_config)
    
    # Compare with reference
    comparison_result = validator.compare_with_reference(our_config, reference_config)
    
    # Generate report
    print("\n📊 VALIDATION RESULTS")
    print("=" * 60)
    
    if validation_result["valid"]:
        print("✅ Configuration is structurally valid")
    else:
        print("❌ Configuration has structural issues")
    
    print(f"📈 Summary:")
    print(f"  - Total Errors: {validation_result['summary']['total_errors']}")
    print(f"  - Total Warnings: {validation_result['summary']['total_warnings']}")
    print(f"  - Total Fixes: {validation_result['summary']['total_fixes']}")
    
    if validation_result["errors"]:
        print("\n🔴 ERRORS:")
        for error in validation_result["errors"]:
            print(f"  - {error}")
    
    if validation_result["warnings"]:
        print("\n🟡 WARNINGS:")
        for warning in validation_result["warnings"]:
            print(f"  - {warning}")
    
    if validation_result["fixes"]:
        print("\n🔧 SUGGESTED FIXES:")
        for fix in validation_result["fixes"]:
            print(f"  - {fix}")
    
    print("\n🔍 COMPARISON WITH REFERENCE")
    print("=" * 60)
    
    if comparison_result["missing_providers"]:
        print(f"📉 Missing Providers: {len(comparison_result['missing_providers'])}")
        for provider in comparison_result["missing_providers"]:
            print(f"  - {provider}")
    
    if comparison_result["extra_providers"]:
        print(f"📈 Extra Providers: {len(comparison_result['extra_providers'])}")
        for provider in comparison_result["extra_providers"]:
            print(f"  - {provider}")
    
    if comparison_result["provider_differences"]:
        print(f"⚖️  Provider Differences: {len(comparison_result['provider_differences'])}")
        for provider, diffs in comparison_result["provider_differences"].items():
            print(f"  - {provider}:")
            for diff_type, diff_list in diffs.items():
                print(f"    * {diff_type}: {', '.join(diff_list)}")
    
    # Generate fixed configuration
    print("\n🔧 GENERATING FIXED CONFIGURATION")
    print("=" * 60)
    
    fixed_config = generate_fixed_configuration(our_config, validation_result["fixes"])
    
    # Save fixed configuration
    fixed_path = "/media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier/opencode_fixed_final.json"
    with open(fixed_path, 'w') as f:
        json.dump(fixed_config, f, indent=2)
    
    print(f"✅ Fixed configuration saved to: {fixed_path}")
    
    # Validate the fixed configuration
    print("\n🔍 VALIDATING FIXED CONFIGURATION")
    print("=" * 60)
    
    fixed_validation = validator.validate_configuration(fixed_config)
    
    if fixed_validation["valid"]:
        print("✅ Fixed configuration is valid!")
    else:
        print("❌ Fixed configuration still has issues")
        for error in fixed_validation["errors"]:
            print(f"  - {error}")
    
    # Final summary
    print("\n📋 FINAL SUMMARY")
    print("=" * 60)
    
    if validation_result["valid"] and fixed_validation["valid"]:
        print("🎉 SUCCESS: Configuration is fully OpenCode compatible!")
        return 0
    elif fixed_validation["valid"]:
        print("✅ SUCCESS: Issues were fixed, configuration is now compatible!")
        return 0
    else:
        print("❌ FAILURE: Configuration has unresolved issues")
        return 1

def generate_fixed_configuration(config: Dict[str, Any], fixes: List[str]) -> Dict[str, Any]:
    """Apply suggested fixes to configuration"""
    fixed = config.copy()
    
    # Apply basic fixes
    if "$schema" not in fixed:
        fixed["$schema"] = "https://opencode.sh/schema.json"
    
    if "username" not in fixed:
        fixed["username"] = "OpenCode AI Assistant"
    
    # Fix casing issues
    def fix_casing_in_dict(data: Dict[str, Any]) -> Dict[str, Any]:
        fixed_data = {}
        for key, value in data.items():
            # Convert snake_case to camelCase for model fields
            if "_" in key and key not in ["cost_per_1m_in", "cost_per_1m_out"]:
                new_key = re.sub(r'_([a-z])', lambda x: x.group(1).upper(), key)
            else:
                new_key = key
            
            if isinstance(value, dict):
                fixed_data[new_key] = fix_casing_in_dict(value)
            else:
                fixed_data[new_key] = value
        return fixed_data
    
    if "provider" in fixed:
        fixed["provider"] = fix_casing_in_dict(fixed["provider"])
    
    return fixed

if __name__ == "__main__":
    sys.exit(main())