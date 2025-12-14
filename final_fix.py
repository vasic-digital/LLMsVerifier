#!/usr/bin/env python3

# Read the file
with open('/Volumes/T7/Projects/AiTest/llm-verifier/tests/test_helpers.go', 'r') as f:
    content = f.read()

# Find and replace the broken section
old_section = '''\t\t// Simulate tool use if requested
\t\tif tools, ok := request["tools"].([]interface{}); ok && len(tools) > 0 {
\t\t\t\tif choices, ok := response["choices"].([]interface{}); ok && len(choices) > 0 {
\t\t\t\t\tif choice, ok := choices[0].(map[string]interface{}); ok {
\t\t\t\t\t\tif message, ok := choice["message"].(map[string]interface{}); ok {
\t\t\t\t\t\t\tmessage["tool_calls"] = []map[string]interface{}{
\t\t\t\t\t\t\t\t{
\t\t\t\t\t\t\t\t\t"id":       "call_test",
\t\t\t\t\t\t\t\t\t"type":     "function",
\t\t\t\t\t\t\t\t\t"function": map[string]interface{}{
\t\t\t\t\t\t\t\t\t\t"name":      "get_current_weather",
\t\t\t\t\t\t\t\t\t\t"arguments": `{"location": "New York, NY"}`,
\t\t\t\t\t\t\t\t\t},
\t\t\t\t\t\t\t\t},\n\t\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}'''

new_section = '''\t\t// Simulate tool use if requested
\t\tif tools, ok := request["tools"].([]interface{}); ok && len(tools) > 0 {
\t\t\tif choices, ok := response["choices"].([]interface{}); ok && len(choices) > 0 {
\t\t\t\tif choice, ok := choices[0].(map[string]interface{}); ok {
\t\t\t\t\tif message, ok := choice["message"].(map[string]interface{}); ok {
\t\t\t\t\t\tmessage["tool_calls"] = []map[string]interface{}{
\t\t\t\t\t\t\t{
\t\t\t\t\t\t\t\t"id":       "call_test",
\t\t\t\t\t\t\t\t"type":     "function",
\t\t\t\t\t\t\t\t"function": map[string]interface{}{
\t\t\t\t\t\t\t\t\t"name":      "get_current_weather",
\t\t\t\t\t\t\t\t\t"arguments": `{"location": "New York, NY"}`,
\t\t\t\t\t\t\t\t},
\t\t\t\t\t\t\t},
\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\t\t}'''

# Replace the broken section
new_content = content.replace(old_section, new_section)

# Write back
with open('/Volumes/T7/Projects/AiTest/llm-verifier/tests/test_helpers.go', 'w') as f:
    f.write(new_content)

print("Fixed test_helpers.go properly")