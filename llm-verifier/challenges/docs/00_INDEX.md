# Challenge Framework Documentation

**Complete Documentation Suite for LLM Verifier Challenge Framework**

---

## 📚 Documentation Index

### Getting Started
1. [Introduction](01_INTRODUCTION.md) - Framework overview and concepts
2. [Quick Start](02_QUICK_START.md) - Get started in 5 minutes
3. [Installation](03_INSTALLATION.md) - Setup and configuration

### User Guides
4. [Challenge Runner Guide](04_CHALLENGE_RUNNER_GUIDE.md) - Using the generic runner
5. [Creating Challenges](05_CREATING_CHALLENGES.md) - Author your own challenges
6. [Configuring Platforms](06_PLATFORM_CONFIG.md) - Platform-specific setup

### Step-by-Step Tutorials
7. [Tutorial: Provider Discovery](TUTORIAL_01_PROVIDER_DISCOVERY.md) - Discover providers and models
8. [Tutorial: Model Verification](TUTORIAL_02_MODEL_VERIFICATION.md) - Verify model capabilities
9. [Tutorial: Feature Testing](TUTORIAL_03_FEATURE_TESTING.md) - Test model features

### Advanced Tutorials
10. [Tutorial: Multi-Platform Testing](TUTORIAL_04_MULTIPLATFORM.md) - Run on all platforms
11. [Tutorial: Custom Challenges](TUTORIAL_05_CUSTOM_CHALLENGES.md) - Create custom challenges
12. [Tutorial: Automation](TUTORIAL_06_AUTOMATION.md) - Automate challenge runs

### Technical Reference
13. [Challenge Bank Format](10_CHALLENGE_BANK_FORMAT.md) - JSON format reference
14. [Directory Structure](11_DIRECTORY_STRUCTURE.md) - File organization
15. [Configuration Reference](12_CONFIG_REFERENCE.md) - Config file options

### API Reference
16. [Platform Binaries](13_PLATFORM_BINARIES.md) - Available binaries and commands
17. [Challenge Scripting](14_CHALLENGE_SCRIPTING.md) - Writing challenge scripts

### Troubleshooting
18. [Common Issues](15_TROUBLESHOOTING.md) - Solve common problems
19. [Debug Guide](16_DEBUG_GUIDE.md) - Debug challenge execution

### Best Practices
20. [Challenge Design](17_BEST_PRACTICES.md) - Write better challenges
21. [Security](18_SECURITY.md) - Secure API keys and logs

### Examples
22. [Example: Simple Challenge](EXAMPLE_01_SIMPLE.md) - Basic challenge example
23. [Example: Complex Challenge](EXAMPLE_02_COMPLEX.md) - Advanced challenge example

---

## 📖 Reading Order

### For New Users
1. Start with [Introduction](01_INTRODUCTION.md)
2. Follow [Quick Start](02_QUICK_START.md)
3. Complete [Tutorial 1](TUTORIAL_01_PROVIDER_DISCOVERY.md)

### For Challenge Authors
1. Read [Creating Challenges](05_CREATING_CHALLENGES.md)
2. Study [Challenge Bank Format](10_CHALLENGE_BANK_FORMAT.md)
3. Follow [Challenge Design](17_BEST_PRACTICES.md)
4. Review [Example Challenges](EXAMPLE_01_SIMPLE.md)

### For Advanced Users
1. Read all Tutorials (7-12)
2. Study [Challenge Scripting](14_CHALLENGE_SCRIPTING.md)
3. Practice [Custom Challenges](TUTORIAL_05_CUSTOM_CHALLENGES.md)
4. Learn [Automation](TUTORIAL_06_AUTOMATION.md)

---

## 🎯 Key Concepts

### Challenge
A **challenge** is a test or verification of the LLM Verifier system using ONLY production binaries (CLI, REST API, TUI, Desktop, Mobile, Web).

### Challenge Bank
A **challenge bank** is a registry (JSON) of all available challenges, their configurations, dependencies, and metadata.

### Challenge Runner
A **challenge runner** is a generic script that can execute any challenge from the bank, supporting multiple platforms.

### Platform
A **platform** is one of the built applications:
- **CLI**: `llm-verifier` - Command Line Interface
- **REST API**: `llm-verifier-api` - REST API Server
- **TUI**: `llm-verifier-tui` - Terminal User Interface
- **Desktop**: `llm-verifier-desktop` - Desktop Application
- **Mobile**: `llm-verifier-mobile` - Mobile Application
- **Web**: `llm-verifier-web` - Web Application

### Challenge Result
Challenge results are stored in: `challenges/<name>/YYYY/MM/DD/timestamp/`
- `logs/` - Verbose execution logs and command logs
- `results/` - JSON result files (opencode.json, crush.json)
- `config.yaml` - Challenge configuration

---

## 📋 Quick Links

- [Run Your First Challenge](#run-your-first-challenge)
- [List All Challenges](#list-all-challenges)
- [Create Custom Challenge](#create-custom-challenge)
- [Troubleshoot Issues](#troubleshoot-issues)

---

**Last Updated**: 2025-12-23  
**Version**: 1.0.0  
**Framework Version**: 3.0 (Binary-Only)
