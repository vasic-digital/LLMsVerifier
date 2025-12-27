# ACP (AI Coding Protocol) Research Summary

## Overview
ACP (AI Coding Protocol) is an open protocol that standardizes communication between code editors and AI coding agents. It's designed to enable AI coding assistants to work seamlessly across different editors and IDEs through a standardized JSON-RPC over stdio communication protocol.

## Key Features
- **JSON-RPC Communication**: Uses JSON-RPC protocol over stdio for communication
- **Editor Integration**: Works with editors like Zed, JetBrains IDEs, Avante.nvim, CodeCompanion.nvim
- **Tool Support**: Supports built-in tools, custom tools, and slash commands
- **MCP Compatibility**: Works with MCP servers configured in OpenCode config
- **Project Rules**: Supports project-specific rules from AGENTS.md files
- **Formatters/Linters**: Supports custom formatters and linters
- **Permissions System**: Includes agents and permissions system

## Protocol Specifications
Based on the OpenCode implementation, ACP appears to:
1. Start as a subprocess that communicates via JSON-RPC
2. Support stdio-based communication between editor and AI agent
3. Handle tool calling and resource management
4. Support context management and conversation history
5. Enable real-time code assistance and generation

## Integration Points for LLM Verifier

### 1. Capability Detection
- Test if LLMs can understand and respond to ACP-style requests
- Check for JSON-RPC protocol comprehension
- Test tool calling capabilities
- Verify context management across conversation turns

### 2. Configuration Support
- Add ACP configuration options to provider configs
- Support ACP-specific settings and parameters
- Enable ACP feature flags per provider/model

### 3. Feature Testing
- Test ACP protocol compliance
- Validate JSON-RPC message handling
- Test tool integration capabilities
- Check conversation context management

### 4. Scoring Integration
- Include ACP support in feature richness scoring
- Weight ACP capabilities in overall model assessment
- Track ACP-specific performance metrics

## Implementation Strategy

### Phase 1: Basic ACP Detection
1. Implement ACP capability detection function
2. Add ACP fields to data models
3. Update database schema
4. Basic ACP testing logic

### Phase 2: Advanced ACP Features
1. Comprehensive ACP protocol testing
2. Tool calling validation
3. Context management testing
4. JSON-RPC compliance verification

### Phase 3: Integration & Documentation
1. Provider configuration updates
2. API validation updates
3. Scoring system integration
4. Documentation and guides

### Phase 4: Testing & Automation
1. Unit tests for ACP components
2. Integration tests
3. End-to-end tests
4. Performance and security tests
5. Full automation workflows

## ACP Test Scenarios

### 1. Protocol Comprehension
- Test understanding of JSON-RPC format
- Validate request/response handling
- Check error handling capabilities

### 2. Tool Integration
- Test tool calling functionality
- Validate parameter passing
- Check result processing

### 3. Context Management
- Test conversation history retention
- Validate context window handling
- Check multi-turn conversations

### 4. Code Assistance
- Test code generation capabilities
- Validate code completion
- Check error detection and fixing

## Success Criteria
- ACP support detection for all tested LLMs
- Comprehensive ACP capability assessment
- Integration with existing scoring system
- Full test coverage across all test types
- Complete documentation and examples
- Video course materials
- Website updates