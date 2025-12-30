# LLM Verifier - Complete Specification

<p align="center">
  <img src="docs/images/Logo.jpeg" alt="LLMsVerifier Logo" width="150" height="150">
</p>

<p align="center">
  <strong>Verify. Monitor. Optimize.</strong>
</p>

---

## Overview
LLM Verifier is a comprehensive tool to verify, test, and benchmark LLMs with full OpenAI API compatibility. The tool must support configuration-based operations and automatic discovery of all available models.

## Core Requirements

### 1. Model Discovery and Verification
- **Automatic Discovery**: When no models are specified in config, the tool must use API calls to discover all available models
- **Model Verification**: Check if each model exists, is responsive, is overloaded, and determine available features
- **Performance Assessment**: Determine real usability for coding purposes and practical capabilities

### 2. Feature Detection
The tool must detect support for:
- MCPs (Model Context Protocol)
- LSPs (Language Server Protocol)
- Reranking capabilities
- Embeddings functionality
- Tooling capabilities
- Reasoning capabilities
- Audio generation
- Video generation
- Image generation
- All possible feature types

### 3. Code Capability Assessment
- Language-specific tests (Python, JavaScript, TypeScript, Go, Java, C++, etc.)
- Code generation and completion abilities
- Debugging and optimization capabilities
- Code review and explanation abilities
- Test generation
- Documentation abilities
- Refactoring capabilities
- Architecture understanding
- Security assessment
- Pattern recognition

### 4. Scoring and Ranking
- Calculate realistic usability scores
- Rank models by various criteria:
  - Strength
  - Speed
  - Reliability
  - Coding capability
  - Feature richness
- Provide detailed breakdown of score calculations

### 5. Output Formats
- **Markdown Report**: Human-readable with full explanations of features and possibilities
- **JSON Report**: Machine-readable for other systems to use discovered models by priority and quality

### 6. Configuration Support
- Support for one or more LLMs specified in configuration
- Support for automatic discovery when no models are specified
- Support for endpoint(s) and API key specifications
- Concurrency and timeout configurations

### 7. Testing Requirements
Complete test coverage including:
- Unit tests
- Integration tests
- End-to-end tests
- Full automation tests
- Security tests
- Performance tests
- Benchmark tests

## Technical Implementation Details

### API Support
- Full OpenAI API compatibility
- Support for chat completions
- Support for embeddings
- Support for fine-tuning (if available)
- Support for models listing
- Support for image generation (if available)
- Support for audio processing (if available)

### Feature Detection Methods

#### MCPs (Model Context Protocol)
- Check for context window capabilities
- Test token handling and limits
- Evaluate conversation history management

#### LSPs (Language Server Protocol)
- Test for language analysis capabilities
- Check for real-time error detection
- Verify code completion within IDE contexts
- Evaluate symbol navigation

#### Reranking
- Test reranking API endpoints
- Verify relevance scoring
- Check ordering improvements

#### Embeddings
- Test embedding generation
- Verify vector space capabilities
- Check dimensional properties

#### Generative Capabilities
- Audio generation testing
- Video generation testing (if supported)
- Image generation testing (DALL-E or similar)

## Output Format Requirements

### Markdown Report
- Executive summary
- Detailed model analysis
- Feature-by-feature breakdown
- Performance metrics
- Reliability assessment
- Rankings by various criteria
- Recommendations

### JSON Report
- Comprehensive model data
- Scoring details
- Performance metrics
- Feature detection results
- Ranking information
- Metadata for automated systems

## Testing Strategy

### Unit Tests
- Individual function testing
- Algorithm validation
- Calculation verification
- Data structure validation

### Integration Tests
- Component interaction testing
- API integration
- Configuration loading
- Report generation

### End-to-End Tests
- Complete workflow testing
- API call sequences
- Report generation and validation
- Error handling

### Performance Tests
- Response time measurements
- Throughput testing
- Memory usage analysis
- Concurrency handling

### Security Tests
- API key handling
- Input validation
- Secure communication
- Data protection

### Benchmark Tests
- Performance baselines
- Load testing
- Stress testing
- Scalability validation

## Non-Functional Requirements

### Performance
- Efficient concurrent processing
- Minimal memory footprint
- Fast response time measurements
- Scalable to 100+ models

### Reliability
- Robust error handling
- Graceful degradation
- Timeout management
- Retry mechanisms

### Security
- Secure API key handling
- No sensitive data in logs
- Encrypted communication
- Input sanitization