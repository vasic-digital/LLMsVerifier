# Research: Extend LLM Providers Support

## Groq Provider Research

**Decision**: Implement Groq provider using their official API at https://console.groq.com/docs

**Rationale**: Groq offers fast inference with Llama models. API is well-documented and follows standard patterns.

**Alternatives considered**: No alternatives - official API is the only option.

**API Details**:
- Base URL: https://api.groq.com/openai/v1
- Authentication: Bearer token
- Models: llama2-70b-4096, llama2-7b-2048, mixtral-8x7b-32768
- Rate limits: 30 requests per minute

## Together AI Provider Research

**Decision**: Implement Together AI provider using https://docs.together.ai/reference

**Rationale**: Together AI provides access to many open-source models with clear API documentation.

**Alternatives considered**: No alternatives needed.

**API Details**:
- Base URL: https://api.together.xyz/v1
- Authentication: Bearer token
- Models: Multiple Llama, Falcon, and other open-source models
- Rate limits: Varies by model, generally 10-100 req/min

## Fireworks AI Provider Research

**Decision**: Implement Fireworks AI using https://readme.fireworks.ai

**Rationale**: Fireworks AI offers optimized inference for various models.

**Alternatives considered**: None.

**API Details**:
- Base URL: https://api.fireworks.ai/inference/v1
- Authentication: Bearer token
- Models: Llama, Mistral, and custom models
- Rate limits: Model-specific, up to 1000 req/min

## Poe Provider Research

**Decision**: Implement Poe using OpenAI-compatible API at https://api.poe.com/v1

**Rationale**: Poe provides OpenAI-compatible endpoint for their models.

**Alternatives considered**: None.

**API Details**:
- Base URL: https://api.poe.com/v1
- Authentication: API key
- Models: Various models available through Poe
- Rate limits: Standard OpenAI-compatible limits

## NaviGator AI Provider Research

**Decision**: Implement NaviGator AI using https://docs.ai.it.ufl.edu

**Rationale**: NaviGator AI provides research-focused models with documented API.

**Alternatives considered**: None.

**API Details**:
- Base URL: https://api.ai.it.ufl.edu/v1
- Authentication: API key
- Models: Mistral-small-3.1 and others
- Rate limits: Research-oriented, moderate limits

## Integration Patterns Research

**Decision**: Follow existing provider implementation patterns in LLM Verifier

**Rationale**: Maintain consistency with current architecture.

**Alternatives considered**: No alternatives - must follow existing patterns.

**Patterns**:
- Provider struct in providers/ directory
- Integration with pricing, limits, failover modules
- Config export support for OpenCode, Crush, Claude Code
- Comprehensive test coverage

## Testing Strategy Research

**Decision**: Apply 100% test coverage across all 6 test types for each provider

**Rationale**: Constitution requires 100% coverage per test type.

**Alternatives considered**: None - mandatory requirement.

**Coverage Requirements**:
- Unit tests for provider logic
- Integration tests for API calls
- End-to-end tests for full workflows
- Automation tests for scheduled verification
- Security tests for API key handling
- Performance tests for load handling