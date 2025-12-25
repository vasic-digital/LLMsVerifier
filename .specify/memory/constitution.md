<!--
Sync Impact Report:
- Version change: N/A → 1.0.0
- Added sections: Core Principles (Test-First, Complete Documentation, Zero Broken Features, Manual CI/CD Only, Modular Architecture), Production Deployment Policies, Development Workflow, Governance
- Removed sections: None
- Modified principles: All new
- Templates requiring updates: None
- Follow-up TODOs: Determine ratification date from project history
-->
# LLM Verifier Constitution

## Core Principles

### Test-First (NON-NEGOTIABLE)
All code must be covered by comprehensive tests across all 6 test types (Unit, Integration, End-to-End, Automation, Security, Performance) with 100% coverage per type. Tests written before implementation, red-green-refactor cycle strictly enforced.

### Complete Documentation
Every component, feature, and API must be fully documented with user guides, manuals, and API docs. Website must accurately reflect 100% of project state.

### Zero Broken Features
No feature, module, or test can remain disabled or broken. All functionality must be operational and validated.

### Manual CI/CD Only
CI/CD pipelines exist only as manually triggerable mechanisms. No automatic GitHub actions, hooks, or continuous deployment.

### Modular Architecture
Every feature starts as a standalone, self-contained library that can be independently tested and documented. Clear purpose required - no organizational-only libraries.

## Production Deployment Policies

Technology stack: Go 1.21+, SQLite with SQL Cipher, Docker, Kubernetes. Compliance: Enterprise-grade security, monitoring, and scalability. Deployment: Manual triggers only, no automatic pipelines.

## Development Workflow

Code review mandatory for all changes. Testing gates require 100% coverage across all test types. Deployment approval requires manual verification and documentation updates.

## Governance

Constitution supersedes all other practices; Amendments require documentation, approval, migration plan. Compliance verified in all code reviews.

**Version**: 1.0.0 | **Ratified**: TODO(RATIFICATION_DATE): Original adoption date unknown, to be determined from project history. | **Last Amended**: 2025-12-25
