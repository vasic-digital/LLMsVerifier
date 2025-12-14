#!/usr/bin/env python3

# Read the file
with open('/Volumes/T7/Projects/AiTest/llm-verifier/tests/test_helpers.go', 'r') as f:
    content = f.read()

# Find and replace the corrupted section
old_section = '''		// Simulate tool use if requested
		if tools, ok := request["tools"].([]interface{}); ok && len(tools) > 0 {
				if choices, ok := response["choices"].([]interface{}); ok && len(choices) > 0 {
					if choice, ok := choices[0].(map[string]interface{}); ok {
					"id":       "call_test",
					"type":     "function",
					"function": map[string]interface{}{
						"name":      "get_current_weather",
						"arguments": `{"location": "New York, NY"}`,
					},
				},
			}
		}'''

new_section = '''		// Simulate tool use if requested
		if tools, ok := request["tools"].([]interface{}); ok && len(tools) > 0 {
			if choices, ok := response["choices"].([]interface{}); ok && len(choices) > 0 {
				if choice, ok := choices[0].(map[string]interface{}); ok {
					if message, ok := choice["message"].(map[string]interface{}); ok {
						message["tool_calls"] = []map[string]interface{}{
							{
								"id":       "call_test",
								"type":     "function",
								"function": map[string]interface{}{
									"name":      "get_current_weather",
									"arguments": `{"location": "New York, NY"}`,
								},
							},
						}
					}
				}
			}
		}'''

# Replace the corrupted section
new_content = content.replace(old_section, new_section)

# Write back
with open('/Volumes/T7/Projects/AiTest/llm-verifier/tests/test_helpers.go', 'w') as f:
    f.write(new_content)

print("Fixed test_helpers.go")