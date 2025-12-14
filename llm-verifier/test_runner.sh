#!/bin/bash

# Test runner script for LLM Verifier

echo "LLM Verifier - Test Suite Runner"
echo "==============================="

echo "Running unit tests..."
go test ./tests/unit_test.go -v

echo -e "\nRunning integration tests..."
go test ./tests/integration_test.go -v

echo -e "\nRunning end-to-end tests..."
go test ./tests/e2e_test.go -v

echo -e "\nRunning performance tests..."
go test ./tests/performance_test.go -bench=.

echo -e "\nRunning security tests..."
go test ./tests/security_test.go -v

echo -e "\nRunning automation tests..."
go test ./tests/automation_test.go -v

echo -e "\nRunning all tests together..."
go test ./tests/... -v

echo -e "\nBuilding the application..."
go build -o llm-verifier cmd/main.go

echo -e "\nLLM Verifier built successfully!"
echo "Run './llm-verifier' to start the verification process"