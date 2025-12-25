---

description: "Task list template for feature implementation"
---

# Tasks: Extend LLM Providers Support

**Input**: Design documents from `specs/001-extend-llm-providers/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Comprehensive test coverage required per constitution (100% across all 6 test types: Unit, Integration, End-to-End, Automation, Security, Performance)

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: `llm-verifier/` at repository root
- Paths relative to repository root

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Verify research and prepare for implementation

- [ ] T001 Review research.md findings for all 5 providers
- [ ] T002 Validate provider API endpoints from research
- [ ] T003 Confirm API key availability for testing
- [ ] T004 Update Go module dependencies if needed for new providers

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Update shared infrastructure required by all provider implementations

**⚠️ CRITICAL**: Complete this phase before any user story work

- [ ] T005 Update enhanced/validation/schema.go to include new provider names
- [ ] T006 Update enhanced/limits.go with rate limit detection for new providers
- [ ] T007 Update enhanced/pricing.go with pricing detection for new providers
- [ ] T008 Update client/http_client.go with new provider endpoints
- [ ] T009 Update client/http_client_test.go with new provider test cases
- [ ] T010 Update llmverifier/config_export.go to support new providers in export formats
- [ ] T011 Update failover/failover_manager.go with new provider logic
- [ ] T012 Update database models to support extended provider metadata
- [ ] T013 Update config/production_config.go with new provider configurations
- [ ] T014 Update api/schema_validator.go for new provider validation rules

**Checkpoint**: Foundational infrastructure ready - user story implementation can now begin

## Phase 3: User Story 1 - Add Groq Provider Support (Priority: P1) 🎯 MVP

**Goal**: Fully implement Groq provider support with complete testing and documentation

**Independent Test**: Configure Groq provider, verify models, generate reports, export configs without affecting other providers

### Implementation for User Story 1

- [ ] T015 [US1] Create llm-verifier/providers/groq.go with provider struct and interface
- [ ] T016 [US1] Implement Groq API client methods in groq.go
- [ ] T017 [US1] Add Groq model discovery logic in groq.go
- [ ] T018 [US1] Implement Groq verification workflow in groq.go
- [ ] T019 [US1] Update enhanced/limits.go with Groq-specific rate limits
- [ ] T020 [US1] Update enhanced/pricing.go with Groq pricing detection
- [ ] T021 [US1] Update llmverifier/config_export.go for Groq OpenCode export
- [ ] T022 [US1] Update llmverifier/config_export.go for Groq Crush export
- [ ] T023 [US1] Update llmverifier/config_export.go for Groq Claude Code export
- [ ] T024 [US1] Update client/http_client.go with Groq endpoint mappings
- [ ] T025 [US1] Update enhanced/validation/schema.go to validate Groq configs
- [ ] T026 [US1] Update failover/failover_manager.go with Groq failover logic
- [ ] T027 [US1] Update config/production_config.go with Groq config template
- [ ] T028 [US1] Add Groq provider to database seeding scripts

### Tests for User Story 1

- [ ] T029 [P] [US1] Create unit tests for Groq provider in llm-verifier/providers/groq_test.go
- [ ] T030 [P] [US1] Add integration tests for Groq API calls in llm-verifier/tests/integration/groq_test.go
- [ ] T031 [P] [US1] Add end-to-end tests for Groq verification workflow in llm-verifier/tests/e2e/groq_test.go
- [ ] T032 [P] [US1] Add automation tests for Groq scheduling in llm-verifier/tests/automation/groq_test.go
- [ ] T033 [P] [US1] Add security tests for Groq API key handling in llm-verifier/tests/security/groq_test.go
- [ ] T034 [P] [US1] Add performance tests for Groq load handling in llm-verifier/tests/performance/groq_test.go

### Documentation for User Story 1

- [ ] T035 [US1] Update llm-verifier/README.md with Groq provider information
- [ ] T036 [US1] Update docs/ with Groq setup instructions
- [ ] T037 [US1] Update specs/001-extend-llm-providers/quickstart.md with Groq examples

**Checkpoint**: Groq provider fully implemented and testable independently

## Phase 4: User Story 2 - Add Together AI Provider Support (Priority: P1)

**Goal**: Fully implement Together AI provider support with complete testing and documentation

**Independent Test**: Configure Together AI provider, verify models, generate reports, export configs independently

### Implementation for User Story 2

- [ ] T038 [US2] Create llm-verifier/providers/togetherai.go with provider struct and interface
- [ ] T039 [US2] Implement Together AI API client methods in togetherai.go
- [ ] T040 [US2] Add Together AI model discovery logic in togetherai.go
- [ ] T041 [US2] Implement Together AI verification workflow in togetherai.go
- [ ] T042 [US2] Update enhanced/limits.go with Together AI-specific rate limits
- [ ] T043 [US2] Update enhanced/pricing.go with Together AI pricing detection
- [ ] T044 [US2] Update llmverifier/config_export.go for Together AI OpenCode export
- [ ] T045 [US2] Update llmverifier/config_export.go for Together AI Crush export
- [ ] T046 [US2] Update llmverifier/config_export.go for Together AI Claude Code export
- [ ] T047 [US2] Update client/http_client.go with Together AI endpoint mappings
- [ ] T048 [US2] Update enhanced/validation/schema.go to validate Together AI configs
- [ ] T049 [US2] Update failover/failover_manager.go with Together AI failover logic
- [ ] T050 [US2] Update config/production_config.go with Together AI config template
- [ ] T051 [US2] Add Together AI provider to database seeding scripts

### Tests for User Story 2

- [ ] T052 [P] [US2] Create unit tests for Together AI provider in llm-verifier/providers/togetherai_test.go
- [ ] T053 [P] [US2] Add integration tests for Together AI API calls in llm-verifier/tests/integration/togetherai_test.go
- [ ] T054 [P] [US2] Add end-to-end tests for Together AI verification workflow in llm-verifier/tests/e2e/togetherai_test.go
- [ ] T055 [P] [US2] Add automation tests for Together AI scheduling in llm-verifier/tests/automation/togetherai_test.go
- [ ] T056 [P] [US2] Add security tests for Together AI API key handling in llm-verifier/tests/security/togetherai_test.go
- [ ] T057 [P] [US2] Add performance tests for Together AI load handling in llm-verifier/tests/performance/togetherai_test.go

### Documentation for User Story 2

- [ ] T058 [US2] Update llm-verifier/README.md with Together AI provider information
- [ ] T059 [US2] Update docs/ with Together AI setup instructions
- [ ] T060 [US2] Update specs/001-extend-llm-providers/quickstart.md with Together AI examples

**Checkpoint**: Together AI provider fully implemented and testable independently

## Phase 5: User Story 3 - Add Fireworks AI Provider Support (Priority: P2)

**Goal**: Fully implement Fireworks AI provider support with complete testing and documentation

**Independent Test**: Configure Fireworks AI provider, verify models, generate reports, export configs independently

### Implementation for User Story 3

- [ ] T061 [US3] Create llm-verifier/providers/fireworks.go with provider struct and interface
- [ ] T062 [US3] Implement Fireworks AI API client methods in fireworks.go
- [ ] T063 [US3] Add Fireworks AI model discovery logic in fireworks.go
- [ ] T064 [US3] Implement Fireworks AI verification workflow in fireworks.go
- [ ] T065 [US3] Update enhanced/limits.go with Fireworks AI-specific rate limits
- [ ] T066 [US3] Update enhanced/pricing.go with Fireworks AI pricing detection
- [ ] T067 [US3] Update llmverifier/config_export.go for Fireworks AI OpenCode export
- [ ] T068 [US3] Update llmverifier/config_export.go for Fireworks AI Crush export
- [ ] T069 [US3] Update llmverifier/config_export.go for Fireworks AI Claude Code export
- [ ] T070 [US3] Update client/http_client.go with Fireworks AI endpoint mappings
- [ ] T071 [US3] Update enhanced/validation/schema.go to validate Fireworks AI configs
- [ ] T072 [US3] Update failover/failover_manager.go with Fireworks AI failover logic
- [ ] T073 [US3] Update config/production_config.go with Fireworks AI config template
- [ ] T074 [US3] Add Fireworks AI provider to database seeding scripts

### Tests for User Story 3

- [ ] T075 [P] [US3] Create unit tests for Fireworks AI provider in llm-verifier/providers/fireworks_test.go
- [ ] T076 [P] [US3] Add integration tests for Fireworks AI API calls in llm-verifier/tests/integration/fireworks_test.go
- [ ] T077 [P] [US3] Add end-to-end tests for Fireworks AI verification workflow in llm-verifier/tests/e2e/fireworks_test.go
- [ ] T078 [P] [US3] Add automation tests for Fireworks AI scheduling in llm-verifier/tests/automation/fireworks_test.go
- [ ] T079 [P] [US3] Add security tests for Fireworks AI API key handling in llm-verifier/tests/security/fireworks_test.go
- [ ] T080 [P] [US3] Add performance tests for Fireworks AI load handling in llm-verifier/tests/performance/fireworks_test.go

### Documentation for User Story 3

- [ ] T081 [US3] Update llm-verifier/README.md with Fireworks AI provider information
- [ ] T082 [US3] Update docs/ with Fireworks AI setup instructions
- [ ] T083 [US3] Update specs/001-extend-llm-providers/quickstart.md with Fireworks AI examples

**Checkpoint**: Fireworks AI provider fully implemented and testable independently

## Phase 6: User Story 4 - Add Poe Provider Support (Priority: P2)

**Goal**: Fully implement Poe provider support with complete testing and documentation

**Independent Test**: Configure Poe provider, verify models, generate reports, export configs independently

### Implementation for User Story 4

- [ ] T084 [US4] Create llm-verifier/providers/poe.go with provider struct and interface
- [ ] T085 [US4] Implement Poe API client methods in poe.go
- [ ] T086 [US4] Add Poe model discovery logic in poe.go
- [ ] T087 [US4] Implement Poe verification workflow in poe.go
- [ ] T088 [US4] Update enhanced/limits.go with Poe-specific rate limits
- [ ] T089 [US4] Update enhanced/pricing.go with Poe pricing detection
- [ ] T090 [US4] Update llmverifier/config_export.go for Poe OpenCode export
- [ ] T091 [US4] Update llmverifier/config_export.go for Poe Crush export
- [ ] T092 [US4] Update llmverifier/config_export.go for Poe Claude Code export
- [ ] T093 [US4] Update client/http_client.go with Poe endpoint mappings
- [ ] T094 [US4] Update enhanced/validation/schema.go to validate Poe configs
- [ ] T095 [US4] Update failover/failover_manager.go with Poe failover logic
- [ ] T096 [US4] Update config/production_config.go with Poe config template
- [ ] T097 [US4] Add Poe provider to database seeding scripts

### Tests for User Story 4

- [ ] T098 [P] [US4] Create unit tests for Poe provider in llm-verifier/providers/poe_test.go
- [ ] T099 [P] [US4] Add integration tests for Poe API calls in llm-verifier/tests/integration/poe_test.go
- [ ] T100 [P] [US4] Add end-to-end tests for Poe verification workflow in llm-verifier/tests/e2e/poe_test.go
- [ ] T101 [P] [US4] Add automation tests for Poe scheduling in llm-verifier/tests/automation/poe_test.go
- [ ] T102 [P] [US4] Add security tests for Poe API key handling in llm-verifier/tests/security/poe_test.go
- [ ] T103 [P] [US4] Add performance tests for Poe load handling in llm-verifier/tests/performance/poe_test.go

### Documentation for User Story 4

- [ ] T104 [US4] Update llm-verifier/README.md with Poe provider information
- [ ] T105 [US4] Update docs/ with Poe setup instructions
- [ ] T106 [US4] Update specs/001-extend-llm-providers/quickstart.md with Poe examples

**Checkpoint**: Poe provider fully implemented and testable independently

## Phase 7: User Story 5 - Add NaviGator AI Provider Support (Priority: P3)

**Goal**: Fully implement NaviGator AI provider support with complete testing and documentation

**Independent Test**: Configure NaviGator AI provider, verify models, generate reports, export configs independently

### Implementation for User Story 5

- [ ] T107 [US5] Create llm-verifier/providers/navigator.go with provider struct and interface
- [ ] T108 [US5] Implement NaviGator AI API client methods in navigator.go
- [ ] T109 [US5] Add NaviGator AI model discovery logic in navigator.go
- [ ] T110 [US5] Implement NaviGator AI verification workflow in navigator.go
- [ ] T111 [US5] Update enhanced/limits.go with NaviGator AI-specific rate limits
- [ ] T112 [US5] Update enhanced/pricing.go with NaviGator AI pricing detection
- [ ] T113 [US5] Update llmverifier/config_export.go for NaviGator AI OpenCode export
- [ ] T114 [US5] Update llmverifier/config_export.go for NaviGator AI Crush export
- [ ] T115 [US5] Update llmverifier/config_export.go for NaviGator AI Claude Code export
- [ ] T116 [US5] Update client/http_client.go with NaviGator AI endpoint mappings
- [ ] T117 [US5] Update enhanced/validation/schema.go to validate NaviGator AI configs
- [ ] T118 [US5] Update failover/failover_manager.go with NaviGator AI failover logic
- [ ] T119 [US5] Update config/production_config.go with NaviGator AI config template
- [ ] T120 [US5] Add NaviGator AI provider to database seeding scripts

### Tests for User Story 5

- [ ] T121 [P] [US5] Create unit tests for NaviGator AI provider in llm-verifier/providers/navigator_test.go
- [ ] T122 [P] [US5] Add integration tests for NaviGator AI API calls in llm-verifier/tests/integration/navigator_test.go
- [ ] T123 [P] [US5] Add end-to-end tests for NaviGator AI verification workflow in llm-verifier/tests/e2e/navigator_test.go
- [ ] T124 [P] [US5] Add automation tests for NaviGator AI scheduling in llm-verifier/tests/automation/navigator_test.go
- [ ] T125 [P] [US5] Add security tests for NaviGator AI API key handling in llm-verifier/tests/security/navigator_test.go
- [ ] T126 [P] [US5] Add performance tests for NaviGator AI load handling in llm-verifier/tests/performance/navigator_test.go

### Documentation for User Story 5

- [ ] T127 [US5] Update llm-verifier/README.md with NaviGator AI provider information
- [ ] T128 [US5] Update docs/ with NaviGator AI setup instructions
- [ ] T129 [US5] Update specs/001-extend-llm-providers/quickstart.md with NaviGator AI examples

**Checkpoint**: NaviGator AI provider fully implemented and testable independently

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final integration, testing, and documentation updates

- [ ] T130 Run full test suite across all providers and test types
- [ ] T131 Update main README.md with new provider count and features
- [ ] T132 Update CHANGELOG.md with new provider additions
- [ ] T133 Validate all export formats work for new providers
- [ ] T134 Run performance benchmarks comparing old vs new provider count
- [ ] T135 Update deployment documentation for new providers
- [ ] T136 Create migration guide for existing installations
- [ ] T137 Validate constitution compliance (100% coverage, documentation, no broken features)

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies
- **Foundational (Phase 2)**: Depends on Setup completion
- **User Stories (Phase 3-7)**: All depend on Foundational phase completion
  - User stories can be implemented in parallel after foundational work
  - Priority order: P1 stories first, then P2, then P3
- **Polish (Phase 8)**: Depends on all user stories being complete

### User Story Dependencies

- **US1 (Groq, P1)**: Independent after foundational
- **US2 (Together AI, P1)**: Independent after foundational
- **US3 (Fireworks AI, P2)**: Independent after foundational
- **US4 (Poe, P2)**: Independent after foundational
- **US5 (NaviGator AI, P3)**: Independent after foundational

### Within Each User Story

- Implementation tasks: Sequential within provider (create file → implement methods → update integrations)
- Test tasks: Can run in parallel after implementation tasks complete
- Documentation tasks: Sequential after implementation

### Parallel Opportunities

- All foundational tasks can run in parallel
- All user stories can run in parallel after foundational
- Within each story: Test tasks can run in parallel
- Documentation updates can run in parallel across stories

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: US1 (Groq)
4. **STOP and VALIDATE**: Test Groq provider independently
5. Deploy/demo with 1 new provider

### Incremental Delivery

1. Complete Setup + Foundational → Infrastructure ready
2. Add US1 (Groq) → Test independently → Deploy/Demo (14 total providers)
3. Add US2 (Together AI) → Test independently → Deploy/Demo
4. Add US3 + US4 (Fireworks + Poe) → Test independently → Deploy/Demo
5. Add US5 (NaviGator) → Test independently → Deploy/Demo
6. Each addition delivers immediate value without breaking existing functionality

### Parallel Team Strategy

With multiple developers:

1. One developer completes Setup + Foundational
2. Once foundational ready:
   - Developer A: US1 + US2 (P1 providers)
   - Developer B: US3 + US4 (P2 providers)
   - Developer C: US5 (P3 provider)
3. Merge and integrate completed stories
4. Run final polish and testing

## Notes

- [P] tasks = different files, no dependencies
- [US#] label maps task to specific user story
- Each user story is independently completable and testable
- Constitution requires 100% test coverage across all test types
- All tasks include exact file paths for immediate execution
- Nano-sized tasks enable precise progress tracking and parallel work