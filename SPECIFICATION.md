We need simple program written in go lang. It will verify LLM. It must support OpenAI API fully. Based on configuration file it will check if: model exist, if is responsive, is it overloaded, what features does it support and five the final score of its real usability for the coding purposes. How much realistically it can pull. If configuration does not specify any particular model, just the enpoint(s), and api key, it will use api calls to obtain the list of all available models and check and bencark every single one of them! Working result will be written as human readable markdown report with full explanation of features and the possibilities. Sexond file will be json report which we can then use by other systems to use discovered models (by priority and quality). There should be lists of various criterias sortings of models - strength, speed, reliability etc. Make sure that we can provide one or more LLMs for checks or none (obtaining all models lists via api) All power features must be checked MCPs, LSPs, rerankings, beddings and all possible types organized into fully coding capable, just chat, with and without tooling or reasoning, generative (audio, video, imaging and others). Everything must be covered with the tests of the following types: unit, integration, e2e, full automation, security, performance and benchmark tests.

We need Scoring System, which means each LLM will be tested and benchmarked. Besides being classified to a proper category (which we have mentioned alrady) it will have score value of realistic usability / quality in percents 0-100%. The reports created will have the propery of integer for this value.

We must be able to determine limits of the LLMs. For example if there are imposed restrictions of some kind, like maximal number of requests per day, week, month. Then, how much requests we have left? And other types of limits which we must determine. All this must go into the report and be a part of sorting / filtering lists. 

We must obtain all available pricings and prices as well and make them one of criterias for sorting / filtering.

System has to create SQLite database with SQL Cipher where it will stor the information about processed providers and all its LLMs. Every next time we process them, data will be updated.

In top of this we must have several clients which will expose all supported commands and ways to access the database. Requests, commands and browsing of the database (quering) must be done via one of the clients: CLI (main implementation), TUI (all CLI offers but with possibility to filter and quiery database easily), REST API (goes on top of everything) - written using the GinGonic Go lang framework, Web (Angular), Desktops and all supported Mobile platforms + Harmony and Aurora OS-es.

All detected faulty LLMs will be documented about discovered issues, severity, ways to workaround and other relevant information.

All logs will be stored in proper log files and in separate database just for the pfs so it is easy to query them by various dimensions. Both databases will have proper indexes set so we can query the data by many dimensions.

Databases data about LLMs will be possible to export into the configuration files for OpenCode, Crush, Claude Code and other major AI CLI Coding Agents! We can export all providers with all its LLMs or just chosen providers with chosen LLMs. All this exposed with all of mentioned supported clients / flavors / platforms. All configurations generated will be verified.

Make sure we can rerun benchmarking and checks without configuratio  provides, which will recheck and rebenchmark database providers and its LLMs. It should be possible to do all existing, inly chosen providers or only chosen LLMs. If score or usability of some of LLMs changes compared to existing information about it in the datase proper events will be triggered. Registered parties will be notified via websockers, gRPS or notifcarion challes about these events. Notification channels that we support are Slack, Email, Telegram, Max and other popular messaging systems (WhatsApp and others). System events will be created and emotted for all other important situations in the system, for example - new testing started, stopped, compledted, on error, and so on. If no registered subscribers via websockets or gRPC or no notification mechanism is registered no events will be emitted. All events must be logged as well into our log files and log database. 

All client types and flavor on all platforms will support registration of event subscribers (and unregistratiln as well).

Configuration files and system will support turning on of the periodical re-tests (repeting benchmarks and verifications). It could be scheduled to repeat per hours, days, weeks, months. Unscheduling and rescheduling must be supported. Scheduling mexhanism mustbsupport multiple scheduling configurations. For example daily for all providers and all LLMs, and per hour for certain procider and chosen LLMs. There must be maximal flexibility and efficiency!

With proper flag set on (regenerate_configurations_on_score_changes) if score of provider or llm changes the configuration for (all cli agents or just chosen ones - for example: OpenCode, Crush) will be recreated. There is default path for generated cli agents configuration files, but it can be changed with configuration (global, per procider or llm).


---

## Model Verification System

### Overview
Validates that LLM models work correctly through actual API testing, not just configuration file parsing.

### Validation Requirements

Each model MUST pass the following tests to be considered verified:

#### 1. Existence Test
- **What**: Verify model is accessible on provider's API
- **Validation**:
  - HTTP HEAD or GET request to model endpoint returns 200 OK
  - GET /models endpoint includes model in available models list
  - Response includes valid model_id and model_name
- **Pass Criteria**: HTTP 200 response + valid model data in response

#### 2. Responsiveness Test  
- **What**: Verify model responds to requests within acceptable time limits
- **Validation**:
  - HTTP POST request with test prompt completes successfully
  - Time to First Token (TTFT) < 10 seconds
  - Total response time < 60 seconds
  - No timeout errors
- **Pass Criteria**: Request completes within time limits without errors

#### 3. Latency Test
- **What**: Measure actual response performance for performance tracking
- **Validation**:
  - TTFT is measured and recorded to database
  - Total response time is measured
  - Average latency calculated from multiple requests
- **Pass Criteria**: Latency data collected and within acceptable range (< 5 seconds preferred)

#### 4. Feature Tests

##### 4.1 Streaming
- **What**: Verify streaming capability works
- **Validation**:
  - At least one chunk is received in streaming response
  - Chunks are delivered in order
  - No connection drops
- **Pass Criteria**: Successfully receives streamed response

##### 4.2 Function Calling
- **What**: Verify tool/function calling capability works
- **Validation**:
  - Tool call is successfully parsed from response
  - Tool parameters match expected schema
  - Tool results are processed
- **Pass Criteria**: Tool definition executes successfully

##### 4.3 Vision
- **What**: Verify multimodal image/vision input works
- **Validation**:
  - Image input is accepted by model
  - Image is processed and analyzed
  - Vision-related output is returned
- **Pass Criteria**: Image successfully processed

##### 4.4 Embeddings
- **What**: Verify text embedding generation works
- **Validation**:
  - Embedding request is successful
  - Embedding vector is returned
- Vector dimension matches expected size
- **Pass Criteria**: Valid embedding vector returned

#### 5. Scoring & Coding Capability
- **What**: Evaluate model's effectiveness for development tasks
- **Validation**:
  - Coding benchmark test passes with score > 80%
  - Code quality test passes with score > 70%
  - Code speed test passes
  - Model can handle real-world coding tasks
- **Pass Criteria**: Model achieves minimum coding capability score

#### 6. Error Detection
- **What**: Identify and categorize API errors
- **Validation**:
  - HTTP errors (4xx, 5xx) are detected and logged
  - Rate limit errors (429) are identified
  - Authentication errors (401) are detected
  - Model not found errors (404) are detected
  - Connection errors are detected
- **Pass Criteria**: Errors are caught and properly categorized

### Implementation Requirements

- **Real API Calls**: Must make actual HTTP requests to provider APIs (not just config file parsing)
- **Database Storage**: Test results must be stored in verification_results table
- **Scoring System**: Models must have coding capability scores stored in verification_scores table
- **Rate Limit Detection**: Must detect and log HTTP 429 responses
- **Timeout Handling**: Must detect and log request timeouts

### Related Documentation

- [Model Verification Challenge](challenges/docs/model_verification_challenge.md)
- [Provider Integration Documentation](llm-verifier/docs/PROVIDER_INTEGRATION.md)
- [Database Schema Documentation](llm-verifier/database/schema.sql)

