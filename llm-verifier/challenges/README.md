# LLM Verifier Challenges

This directory contains the complete challenge framework for testing the LLM Verifier system using ONLY production binaries.

## Directory Organization

### 📁 Structure Overview

```
challenges/
├── codebase/          # Implementation code and execution scripts
│   ├── challenge_runners/    # Individual challenge runner scripts
│   └── go_files/            # Go implementation files
├── data/              # Data files and challenge registry
│   └── challenges_bank.json # Complete challenge definitions
├── docs/              # Documentation and guides
├── results/           # Versioned challenge execution results
│   └── [challenge_name]/    # Results organized by challenge type
├── scripts/           # Utility and execution scripts
└── README.md          # This file
```

### 🎯 Key Directories

- **`codebase/`** - All implementation code, separated from results and data
- **`results/`** - Challenge execution outputs, organized by challenge type and timestamp
- **`docs/`** - Complete documentation suite for the challenge framework
- **`scripts/`** - Executable scripts for running challenges
- **`data/`** - Configuration and registry files

### 📋 Challenge Results Structure

Results are stored in: `results/[challenge_name]/[year]/[month]/[day]/[timestamp]/`

Each execution contains:
- `config.yaml` - Configuration used for the challenge
- `logs/` - Complete execution logs (commands, API calls, errors)
- `results/` - Challenge outputs (JSON files, reports)

### 🚀 Quick Start

1. Review available challenges: `data/challenges_bank.json`
2. Run a specific challenge: `scripts/run_provider_binary_challenge.sh`
3. Check results in `results/` directory
4. Review logs for troubleshooting

### 📚 Documentation

See `docs/` directory for complete guides:
- Challenge framework overview
- Individual challenge specifications
- Execution procedures
- Troubleshooting guides

## Verification

All challenges log complete execution details, including:
- All commands executed
- API requests/responses
- Configuration parameters
- Success/failure status
- Performance metrics

Results are versioned by timestamp and git-tracked for auditability.