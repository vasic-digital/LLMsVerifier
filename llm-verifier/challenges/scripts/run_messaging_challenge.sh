#!/bin/bash

# Messaging Integration Challenge Script
# Tests RabbitMQ and Kafka broker implementations

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Project root (LLMsVerifier directory)
PROJECT_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
cd "$PROJECT_ROOT"

echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}   Messaging Integration Challenge${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""

# Helper functions
pass_test() {
    PASSED_TESTS=$((PASSED_TESTS + 1))
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -e "${GREEN}✓ PASS:${NC} $1"
}

fail_test() {
    FAILED_TESTS=$((FAILED_TESTS + 1))
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -e "${RED}✗ FAIL:${NC} $1"
    if [ -n "$2" ]; then
        echo -e "  ${YELLOW}Reason:${NC} $2"
    fi
}

skip_test() {
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    echo -e "${YELLOW}⊘ SKIP:${NC} $1"
    if [ -n "$2" ]; then
        echo -e "  ${YELLOW}Reason:${NC} $2"
    fi
}

section() {
    echo ""
    echo -e "${BLUE}--- $1 ---${NC}"
}

# Phase 1: Core Package Tests
section "Phase 1: Core Messaging Package"

echo "Testing messaging package build..."
if go build ./internal/messaging/... 2>/dev/null; then
    pass_test "Messaging package builds successfully"
else
    fail_test "Messaging package build failed"
fi

echo "Testing in-memory broker build..."
if go build ./internal/messaging/inmemory/... 2>/dev/null; then
    pass_test "In-memory broker package builds"
else
    fail_test "In-memory broker package build failed"
fi

echo "Testing RabbitMQ broker build..."
if go build ./internal/messaging/rabbitmq/... 2>/dev/null; then
    pass_test "RabbitMQ broker package builds"
else
    fail_test "RabbitMQ broker package build failed"
fi

echo "Testing Kafka broker build..."
if go build ./internal/messaging/kafka/... 2>/dev/null; then
    pass_test "Kafka broker package builds"
else
    fail_test "Kafka broker package build failed"
fi

# Phase 2: Unit Tests
section "Phase 2: Unit Tests"

echo "Running messaging unit tests..."
if go test ./internal/messaging/ 2>&1 | grep -q "ok"; then
    pass_test "Messaging unit tests pass"
else
    fail_test "Messaging unit tests failed"
fi

echo "Running in-memory broker tests..."
if go test ./internal/messaging/inmemory/ 2>&1 | grep -q "ok"; then
    pass_test "In-memory broker tests pass"
else
    fail_test "In-memory broker tests failed"
fi

echo "Running RabbitMQ broker tests..."
if go test ./internal/messaging/rabbitmq/ 2>&1 | grep -q "ok"; then
    pass_test "RabbitMQ broker tests pass"
else
    fail_test "RabbitMQ broker tests failed"
fi

echo "Running Kafka broker tests..."
if go test ./internal/messaging/kafka/ 2>&1 | grep -q "ok"; then
    pass_test "Kafka broker tests pass"
else
    fail_test "Kafka broker tests failed"
fi

# Phase 3: Interface Compliance
section "Phase 3: Interface Compliance"

echo "Checking MessageBroker interface compliance..."
if grep -q "type MessageBroker interface" ./internal/messaging/broker.go; then
    pass_test "MessageBroker interface defined"
else
    fail_test "MessageBroker interface not found"
fi

echo "Checking TaskQueueBroker interface compliance..."
if grep -q "type TaskQueueBroker interface" ./internal/messaging/task_queue.go; then
    pass_test "TaskQueueBroker interface defined"
else
    fail_test "TaskQueueBroker interface not found"
fi

echo "Checking EventStreamBroker interface compliance..."
if grep -q "type EventStreamBroker interface" ./internal/messaging/event_stream.go; then
    pass_test "EventStreamBroker interface defined"
else
    fail_test "EventStreamBroker interface not found"
fi

# Phase 4: Configuration Files
section "Phase 4: Configuration"

echo "Checking messaging configuration file..."
if [ -f "./configs/messaging.yaml" ]; then
    pass_test "Messaging configuration file exists"

    # Check RabbitMQ config
    if grep -q "rabbitmq:" ./configs/messaging.yaml; then
        pass_test "RabbitMQ configuration section present"
    else
        fail_test "RabbitMQ configuration section missing"
    fi

    # Check Kafka config
    if grep -q "kafka:" ./configs/messaging.yaml; then
        pass_test "Kafka configuration section present"
    else
        fail_test "Kafka configuration section missing"
    fi

    # Check circuit breaker config
    if grep -q "circuit_breaker:" ./configs/messaging.yaml; then
        pass_test "Circuit breaker configuration present"
    else
        fail_test "Circuit breaker configuration missing"
    fi
else
    fail_test "Messaging configuration file not found"
fi

# Phase 5: Docker Compose
section "Phase 5: Docker Infrastructure"

echo "Checking Docker Compose messaging file..."
if [ -f "./docker-compose.messaging.yml" ]; then
    pass_test "Docker Compose messaging file exists"

    # Check RabbitMQ service
    if grep -q "rabbitmq:" ./docker-compose.messaging.yml; then
        pass_test "RabbitMQ service defined"
    else
        fail_test "RabbitMQ service not defined"
    fi

    # Check Kafka service
    if grep -q "kafka:" ./docker-compose.messaging.yml; then
        pass_test "Kafka service defined"
    else
        fail_test "Kafka service not defined"
    fi

    # Check Zookeeper service
    if grep -q "zookeeper:" ./docker-compose.messaging.yml; then
        pass_test "Zookeeper service defined"
    else
        fail_test "Zookeeper service not defined"
    fi

    # Check Schema Registry
    if grep -q "schema-registry:" ./docker-compose.messaging.yml; then
        pass_test "Schema Registry service defined"
    else
        fail_test "Schema Registry service not defined"
    fi
else
    fail_test "Docker Compose messaging file not found"
fi

# Phase 6: Makefile Targets
section "Phase 6: Makefile Targets"

echo "Checking Makefile messaging targets..."
if [ -f "./Makefile" ]; then
    if grep -q "messaging-start:" ./Makefile; then
        pass_test "messaging-start target exists"
    else
        fail_test "messaging-start target missing"
    fi

    if grep -q "messaging-stop:" ./Makefile; then
        pass_test "messaging-stop target exists"
    else
        fail_test "messaging-stop target missing"
    fi

    if grep -q "messaging-health:" ./Makefile; then
        pass_test "messaging-health target exists"
    else
        fail_test "messaging-health target missing"
    fi

    if grep -q "test-messaging:" ./Makefile; then
        pass_test "test-messaging target exists"
    else
        fail_test "test-messaging target missing"
    fi
else
    fail_test "Makefile not found"
fi

# Summary
echo ""
echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}   Challenge Summary${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""
echo "Total Tests: $TOTAL_TESTS"
echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
echo -e "${RED}Failed: $FAILED_TESTS${NC}"
echo ""

if [ $TOTAL_TESTS -gt 0 ]; then
    PASS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))
    echo "Pass Rate: ${PASS_RATE}%"
fi
echo ""

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}✓ All challenges passed!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some challenges failed.${NC}"
    exit 1
fi
