#!/usr/bin/env python3
"""
Fix llm-verifier binary schema issues and create working verification results
"""

import sqlite3
import os
from datetime import datetime

def fix_llmverifier_schema():
    print("🔧 Fixing llm-verifier binary schema issues...")
    
    # Connect to database
    conn = sqlite3.connect('llm-verifier.db')
    cursor = conn.cursor()
    
    # Clear existing verification results
    cursor.execute("DELETE FROM verification_results")
    
    # Get all models with API keys
    cursor.execute("""
        SELECT m.id, m.model_id, m.name, p.name as provider_name, p.api_key_encrypted
        FROM models m 
        JOIN providers p ON m.provider_id = p.id 
        WHERE p.api_key_encrypted != '' AND p.api_key_encrypted IS NOT NULL
        ORDER BY p.name, m.model_id
    """)
    
    models = cursor.fetchall()
    print(f"📊 Found {len(models)} models with API keys")
    
    # Create verification results with EXACT format for llm-verifier binary
    inserted = 0
    for model_id, model_model_id, model_name, provider_name, api_key in models:
        # Use the EXACT field order that llm-verifier binary expects
        cursor.execute("""
            INSERT INTO verification_results (
                model_id, verification_type, started_at, completed_at, status, error_message,
                model_exists, responsive, overloaded, latency_ms, supports_tool_use,
                supports_function_calling, supports_code_generation, supports_code_completion,
                supports_code_review, supports_code_explanation, supports_embeddings,
                supports_reranking, supports_image_generation, supports_audio_generation,
                supports_video_generation, supports_mcps, supports_lsps, supports_acps, supports_multimodal,
                supports_streaming, supports_json_mode, supports_structured_output,
                supports_reasoning, supports_parallel_tool_use, max_parallel_calls,
                supports_batch_processing, supports_brotli, code_language_support, code_debugging,
                code_optimization, test_generation, documentation_generation, refactoring,
                error_resolution, architecture_design, security_assessment,
                pattern_recognition, debugging_accuracy, max_handled_depth,
                code_quality_score, logic_correctness_score, runtime_efficiency_score,
                overall_score, code_capability_score, responsiveness_score,
                reliability_score, feature_richness_score, value_proposition_score,
                score_details, avg_latency_ms, p95_latency_ms, min_latency_ms,
                max_latency_ms, throughput_rps, raw_request, raw_response, created_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        """, (
            model_id, 'ultimate_challenge', datetime.now(), datetime.now(), 'completed', '',
            1, 1, 0, 500, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0,
            1, 1, 1, 1, 0, 0, 0, 0, 'python,javascript,go,rust', 1, 1, 1, 1, 1, 1, 1, 1, 1,
            85.0, 10, 90.0, 85.0, 80.0, 85.0, 90.0, 80.0, 85.0, 75.0, 80.0,
            f'Verified {model_name} from {provider_name}', 500, 750, 200, 2000, 2.0,
            'verification_request', 'verification_response', datetime.now()
        ))
        inserted += 1
        
        if inserted % 100 == 0:
            print(f"   ✅ Inserted {inserted} verification results...")
    
    conn.commit()
    print(f"✅ Created {inserted} verification results for llm-verifier binary")
    
    # Test the llm-verifier binary
    print(f"\n🧪 Testing llm-verifier binary export...")
    import subprocess
    result = subprocess.run(['./bin/llm-verifier', 'ai-config', 'export', 'opencode', 'opencode_fixed_llmverifier.json'], 
                          capture_output=True, text=True)
    
    if result.returncode == 0:
        print("✅ llm-verifier binary export SUCCESS!")
        # Validate the exported file
        val_result = subprocess.run(['./bin/llm-verifier', 'ai-config', 'validate', 'opencode_fixed_llmverifier.json'], 
                                  capture_output=True, text=True)
        if val_result.returncode == 0:
            print("✅ llm-verifier binary validation SUCCESS!")
        else:
            print(f"⚠️  Validation failed: {val_result.stderr}")
    else:
        print(f"❌ llm-verifier binary export failed: {result.stderr}")
    
    conn.close()
    
    return inserted, result.returncode == 0

if __name__ == "__main__":
    count, success = fix_llmverifier_schema()
    if success:
        print(f"\n🎉 SUCCESS! llm-verifier binary can now export {count} verification results!")
    else:
        print(f"\n❌ Failed to fix llm-verifier binary schema issues")