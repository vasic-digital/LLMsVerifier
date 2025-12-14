#!/usr/bin/env python3

import re

# Read the file
with open('/Volumes/T7/Projects/AiTest/llm-verifier/database/crud.go', 'r') as f:
    content = f.read()

# Replace verificationResult with result in the specific ranges
lines = content.split('\n')

# Fix lines 960-968
for i in range(959, 968):  # 0-indexed, so 959 = line 960
    if i < len(lines):
        lines[i] = lines[i].replace('verificationResult.', 'result.')
        lines[i] = lines[i].replace('scanNullableTime(', 'scanNullableTimeFromString(')

# Fix lines 1085-1093  
for i in range(1084, 1093):  # 0-indexed, so 1084 = line 1085
    if i < len(lines):
        lines[i] = lines[i].replace('verificationResult.', 'result.')
        lines[i] = lines[i].replace('scanNullableTime(', 'scanNullableTimeFromString(')

# Write back
with open('/Volumes/T7/Projects/AiTest/llm-verifier/database/crud.go', 'w') as f:
    f.write('\n'.join(lines))

print("Fixed verificationResult references in ListVerificationResults function")