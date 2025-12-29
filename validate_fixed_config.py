#!/usr/bin/env python3
"""
Validate the fixed OpenCode configuration
"""

import json
import sys
import os
from final_opencode_comparison import OpenCodeValidator

def main():
    # Load the fixed configuration
    with open("/media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier/opencode_fixed_structure.json", 'r') as f:
        config = json.load(f)
    
    print("🔍 Validating fixed OpenCode configuration...")
    print("=" * 60)
    
    # Initialize validator
    validator = OpenCodeValidator()
    
    # Validate configuration
    result = validator.validate_configuration(config)
    
    print("📊 VALIDATION RESULTS")
    print("=" * 60)
    
    if result["valid"]:
        print("✅ Configuration is structurally valid!")
    else:
        print("❌ Configuration has structural issues")
    
    print(f"📈 Summary:")
    print(f"  - Total Errors: {result['summary']['total_errors']}")
    print(f"  - Total Warnings: {result['summary']['total_warnings']}")
    print(f"  - Total Fixes: {result['summary']['total_fixes']}")
    
    if result["errors"]:
        print("\n🔴 ERRORS:")
        for error in result["errors"]:
            print(f"  - {error}")
    
    if result["warnings"]:
        print("\n🟡 WARNINGS:")
        for warning in result["warnings"]:
            print(f"  - {warning}")
    
    # Final validation with the ultimate-challenge binary
    print("\n🔍 RUNNING ULTIMATE VALIDATION")
    print("=" * 60)
    
    # Copy to the location expected by ultimate-challenge
    import shutil
    shutil.copy("/media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier/opencode_fixed_structure.json", "/tmp/opencode.json")
    
    # Run the ultimate-challenge validation
    validation_output = os.popen("cd /media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier/llm-verifier/cmd/ultimate-challenge && timeout 10 ./ultimate-challenge-test 2>&1 | grep -E '(✅|❌|⚠️|🔍)' | head -10").read()
    print(validation_output)
    
    print("\n📋 FINAL VALIDATION SUMMARY")
    print("=" * 60)
    
    if result["valid"] and "✅" in validation_output:
        print("🎉 SUCCESS: Configuration is fully OpenCode compatible!")
        return 0
    elif result["valid"]:
        print("✅ SUCCESS: Configuration is structurally valid!")
        return 0
    else:
        print("❌ FAILURE: Configuration has unresolved issues")
        return 1

if __name__ == "__main__":
    sys.exit(main())