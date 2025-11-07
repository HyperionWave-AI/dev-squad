# Repository Organization

This document describes the organization of files and directories in the Hyperion repository.

## Root Directory

The root directory contains only mandatory project files:

- **CLAUDE.md** - Project instructions and coordinator system prompt
- **README.md** - Main project documentation
- **Makefile** - Build automation and commands
- **.env*** - Environment configuration files
- **docker-compose*.yml** - Docker orchestration files
- **.gitignore** - Git exclusion rules

## Directory Structure

### `/hyper`
Go backend services and core application code.

### `/ui`
Current React-based web UI.

### `/docs`
All project documentation organized by category:

- **`/docs/setup`** - Installation, initialization, and quick start guides
  - ANTHROPIC_SETUP.md
  - INSTALLATION.md
  - HYPER_INIT_GUIDE.md
  - HYPER_INIT_WITH_PROVIDER.md
  - QUICK_REFERENCE.md
  - QUICK_START_MCP_TOOLS.md
  - CLEAN_INSTALL_GUIDE.md
  - UPDATE_SYSTEM_PROMPT_README.md

- **`/docs/guides`** - Technical guides and reference documentation
  - MAKEFILE_AND_DOCKER_GUIDE.md
  - DOCKER.md
  - HOT_RELOAD_GUIDE.md
  - README-HYPER.md
  - README-NATIVE.md
  - MCP_AGENT_CONFIG.md
  - MCP_CHEAT_SHEET.md
  - HYPERION_COORDINATOR_MCP_REFERENCE.md
  - HYPERION_COORDINATOR_TECHNICAL_GUIDE.md
  - LOGGING_STANDARDS.md
  - TOOL_RESULT_FLOW_DIAGRAM.md
  - TOOL_RESULT_QUICK_REFERENCE.md
  - DEV_HOT_QUICK_REFERENCE.md

- **`/docs/reports`** - Implementation summaries, test reports, and analysis
  - Various *_REPORT.md files
  - Various *_SUMMARY.md files
  - Various *_ANALYSIS.md files
  - Various *_COMPLETE.md files
  - Various *_FIX.md files
  - Various *_IMPLEMENTATION.md files

- **`/docs/archive`** - Historical documentation (ready for future use)

### `/scripts`
All shell scripts organized by purpose:

- **`/scripts/deployment`** - Installation and deployment scripts
  - install.sh
  - clean-install.sh
  - hyper-manager.sh
  - scale.k8s.zero.sh

- **`/scripts/utils`** - Utility and helper scripts
  - build-native.sh
  - configure-claude-native.sh
  - download-embedding-model.sh
  - run-native.sh
  - claude.sh
  - update_prompt_via_api.sh
  - update_system_prompt.js
  - update_system_prompt.py

- **`/scripts/tests`** - Test scripts
  - test_godotenv.go
  - (Note: most test scripts were already in scripts/ and remain organized there)

### `/bin`
Compiled binaries.

### `/models`
ML models and embeddings.

### `/logs`
Runtime logs.

### `/tmp`
Temporary files and test artifacts.

### `/test-reports`
Test execution reports and results.

## Notes

- All environment configuration files (.env*) remain at root for proper loading
- Docker compose files remain at root for standard Docker workflow
- The scripts/ directory already contained many organized scripts; new categorization added
- Test artifacts moved to tmp/ to keep root clean
- Old/backup files moved to tmp/ or archive as appropriate

## Finding Documentation

**For setup/installation:**
```bash
ls docs/setup/
```

**For technical guides:**
```bash
ls docs/guides/
```

**For implementation reports:**
```bash
ls docs/reports/
```

**For scripts:**
```bash
ls scripts/deployment/  # Deployment scripts
ls scripts/utils/       # Utility scripts
ls scripts/tests/       # Test scripts
```
