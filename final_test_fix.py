#!/usr/bin/env python3

# Read the file
with open('/Volumes/T7/Projects/AiTest/llm-verifier/tests/test_helpers.go', 'r') as f:
    lines = f.readlines()

# Find the exact lines to replace (196-208)
start_line = 195  # 0-indexed, line 196
end_line = 208    # 0-indexed, line 209

# Create the new lines
new_lines = [
    "\t\t// Simulate tool use if requested\n",
    "\t\tif tools, ok := request[\"tools\"].([]interface{}); ok && len(tools) > 0 {\n",
    "\t\t\tif choices, ok := response[\"choices\"].([]interface{}); ok && len(choices) > 0 {\n",
    "\t\t\t\tif choice, ok := choices[0].(map[string]interface{}); ok {\n",
    "\t\t\t\t\tif message, ok := choice[\"message\"].(map[string]interface{}); ok {\n",
    "\t\t\t\t\t\tmessage[\"tool_calls\"] = []map[string]interface{}{\n",
    "\t\t\t\t\t\t\t{\n",
    "\t\t\t\t\t\t\t\t\"id\":       \"call_test\",\n",
    "\t\t\t\t\t\t\t\t\"type\":     \"function\",\n",
    "\t\t\t\t\t\t\t\t\"function\": map[string]interface{}{\n",
    "\t\t\t\t\t\t\t\t\t\"name\":      \"get_current_weather\",\n",
    "\t\t\t\t\t\t\t\t\t\"arguments\": `{\"location\": \"New York, NY\"}`,\n",
    "\t\t\t\t\t\t\t\t},\n",
    "\t\t\t\t\t\t\t},\n",
    "\t\t\t\t\t\t}\n",
    "\t\t\t\t\t}\n",
    "\t\t\t\t}\n",
    "\t\t\t}\n",
    "\t\t}\n"
]

# Replace the lines
lines[start_line:end_line] = new_lines

# Write back
with open('/Volumes/T7/Projects/AiTest/llm-verifier/tests/test_helpers.go', 'w') as f:
    f.writelines(lines)

print("Fixed test_helpers.go completely")