# Agent Genericization Report

## Executive Summary

This report documents the analysis and initial genericization of Claude Code agent definition files to remove project-specific "Hyperion" references and make them reusable across different projects.

## Analysis Results

### Files Analyzed: 17 agent definition files

1. `ai-integration-specialist.md` - AI/Claude/GPT integration expert
2. `backend-services-specialist.md` - Go microservices expert
3. `data-platform-specialist.md` - MongoDB/Qdrant database expert
4. `end-to-end-testing-coordinator.md` - Testing orchestration
5. `event-systems-specialist.md` - NATS/MCP protocol expert
6. `frontend-experience-specialist.md` - React/TypeScript UI expert
7. `go-dev.md` - General Go development (PARTIALLY GENERICIZED)
8. `go-mcp-dev.md` - Go with MCP tools development
9. `infrastructure-automation-specialist.md` - GKE/GitHub Actions expert
10. `k8s-deployment-expert.md` - Kubernetes deployment expert
11. `observability-specialist.md` - Monitoring/metrics expert
12. `real-time-systems-specialist.md` - WebSocket/real-time expert
13. `security-auth-specialist.md` - Security/JWT expert
14. `sre.md` - Deployment/SRE work (PARTIALLY GENERICIZED)
15. `ui-dev.md` - UI development work (PARTIALLY GENERICIZED)
16. `ui-tester.md` - UI testing with Playwright
17. `ui-testing-expert.md` - UI testing coordination

## Project-Specific References Found

### 1. **Project Name "Hyperion"**
- Appears in titles, descriptions, and throughout content
- Needs replacement with generic placeholders or "project"

### 2. **Specific Service Names**
- `tasks-api`, `staff-api`, `documents-api`, `chat-api`, `config-api`
- `hyperion-core`, `notification-service`, `report-api`
- Should be replaced with generic service type descriptions

### 3. **Technology Stack Assumptions**
- MongoDB with specific database names (`hyperion-tasks`, `hyperion-staff`)
- Qdrant vector database (called "coordinator knowledge")
- NATS JetStream for event streaming
- Go 1.25 specific version requirements
- React 18 with specific library versions

### 4. **Infrastructure Details**
- GKE cluster names: `hyperion-production`, `gke_production-471918_europe-west2_hyperion-production`
- Registry URLs: `registry.hyperionwave.com`, `europe-west2-docker.pkg.dev/production-471918/hyperion/`
- Kubernetes contexts: `docker-desktop`, `kind-hyperion-dev`
- Namespaces: `hyperion-dev`, `hyperion-prod`

### 5. **Authentication/Credentials**
- Hardcoded JWT tokens with 50-year expiration
- Specific email: `max@hyperionwave.com`
- Specific password: `Megadeth_123`
- Domain URLs: `ws://hyperion:9999`, `https://hyperion.spiritcurrent.com`

### 6. **File System Paths**
- `/Users/maxmednikov/MaxSpace/Hyperion/`
- `/Users/maxmednikov/MaxSpace/hyper/`
- Absolute paths to scripts and documentation

### 7. **MCP Tool Names**
- Prefixes like `mcp__hyper__coordinator_query_knowledge`
- Collection names: `hyperion_project`, `hyperion_bugs`, `hyperion_architecture`

### 8. **Squad/Team Structure**
- "AI & Experience Squad"
- "Backend Infrastructure Squad"
- "Platform & Security Squad"
- "Cross-Squad Coordination"

### 9. **Documentation References**
- `docs/04-development/HYPERION_SERVICE_GOLD_STANDARD.md`
- `docs/04-development/coordinator-search-rules.md`
- `.claude/schema-standards.md`

### 10. **Makefile Targets**
- Specific service names in deployment targets
- `make -f Makefile.production documents-api`
- Registry-specific commands

## Work Completed

### Files Partially Genericized:

#### 1. `go-dev.md` ✅ (Partially Complete)
**Changes Made:**
- Removed "Hyperion" from title
- Genericized project documentation references
- Removed specific service names (tasks-api, staff-api, etc.)
- Removed hardcoded JWT token and credentials
- Replaced specific file paths with generic descriptions
- Removed MCP tool name prefixes
- Genericized knowledge collection names
- Made reference implementation descriptions generic

**Remaining Work:**
- Still contains some technology-specific assumptions
- AI client framework references need review
- Some architecture patterns still mention specific services

#### 2. `ui-dev.md` ✅ (Partially Complete)
**Changes Made:**
- Changed title from "Hyperion Web UI" to "Web UI"
- Removed hardcoded JWT credentials
- Genericized design system references
- Removed specific API endpoint URLs
- Replaced knowledge collection names
- Made schema standards references generic

**Remaining Work:**
- Design system component names may still be project-specific
- Some React Query patterns reference specific services
- Component catalog may need further genericization

#### 3. `sre.md` ✅ (Partially Complete)
**Changes Made:**
- Removed "Hyperion" from title
- Genericized deployment architecture descriptions
- Removed specific GKE cluster names and GCP project IDs
- Removed hardcoded JWT tokens
- Removed registry URLs and domain names
- Genericized Kubernetes context references
- Made knowledge collection names generic

**Remaining Work:**
- Makefile references still specific
- Hot-reload patterns mention specific tools
- Some deployment workflow steps reference specific services

## Genericization Strategy

### Recommended Approach:

#### Phase 1: Template Variables (RECOMMENDED)
Create a configuration-driven approach where projects define their specifics:

```markdown
# Project Configuration Section
Define these variables for your project:
- PROJECT_NAME: Your project name
- SERVICE_REGISTRY: Your container registry URL
- K8S_CONTEXT_DEV: Your dev Kubernetes context
- K8S_CONTEXT_PROD: Your prod Kubernetes context
- JWT_SECRET: Your JWT secret key
- API_BASE_URL: Your API base URL
```

**Benefits:**
- Agents remain generic
- Easy to customize per project
- Clear separation of generic vs project-specific

#### Phase 2: Remove Technology Lock-in
Make technology choices optional:
- Database: "MongoDB, PostgreSQL, or your preferred database"
- Message Queue: "NATS, RabbitMQ, Kafka, or your event system"
- Container Orchestration: "Kubernetes, Docker Swarm, or your orchestrator"

#### Phase 3: Example-Based Documentation
Replace specific implementations with:
```markdown
## Example Service Architecture
Your project might have services like:
- **API Services**: Handle business logic (e.g., user-api, order-api)
- **Core Services**: Orchestration and coordination
- **Integration Services**: External system integrations
```

## Files Needing Genericization

### Priority 1 (Core Development Agents):
- [x] `go-dev.md` - PARTIAL
- [x] `ui-dev.md` - PARTIAL
- [x] `sre.md` - PARTIAL
- [ ] `backend-services-specialist.md`
- [ ] `frontend-experience-specialist.md`
- [ ] `data-platform-specialist.md`

### Priority 2 (Specialized Agents):
- [ ] `ai-integration-specialist.md`
- [ ] `event-systems-specialist.md`
- [ ] `security-auth-specialist.md`
- [ ] `observability-specialist.md`
- [ ] `k8s-deployment-expert.md`

### Priority 3 (Testing & Infrastructure):
- [ ] `end-to-end-testing-coordinator.md`
- [ ] `ui-tester.md`
- [ ] `ui-testing-expert.md`
- [ ] `infrastructure-automation-specialist.md`
- [ ] `real-time-systems-specialist.md`
- [ ] `go-mcp-dev.md`

## Recommendations

### Immediate Actions:

1. **Create Project Configuration Template**
   - File: `.claude/agents/PROJECT_CONFIG.template.md`
   - Contains all project-specific variables
   - Used by all agent files via reference

2. **Complete Genericization of Priority 1 Files**
   - Finish go-dev.md, ui-dev.md, sre.md
   - Remove all remaining hardcoded values
   - Replace with template variable references

3. **Create Generic Examples**
   - Replace Hyperion-specific examples with generic ones
   - Use placeholder names like "ProjectName", "ServiceA", "ServiceB"
   - Provide clear "customize this" markers

4. **Documentation Structure**
   - Create a "Customization Guide" for new projects
   - Document all variables that need to be set
   - Provide migration guide from Hyperion-specific to generic

### Long-term Strategy:

1. **Multi-Project Support**
   - Allow multiple project configurations
   - Switch between projects easily
   - Maintain project-specific customizations separately

2. **Template Generation**
   - Create CLI tool to generate customized agents
   - Input: project configuration
   - Output: fully customized agent definitions

3. **Community Templates**
   - Create generic templates for common architectures
   - Microservices template
   - Monolith template
   - Serverless template

## Estimated Effort

- **Complete Priority 1**: 4-6 hours
- **Complete Priority 2**: 6-8 hours
- **Complete Priority 3**: 4-6 hours
- **Create Configuration System**: 2-3 hours
- **Documentation & Testing**: 3-4 hours

**Total**: 19-27 hours for complete genericization

## Next Steps

1. Review this report with project stakeholders
2. Decide on genericization strategy (template variables recommended)
3. Complete Priority 1 file genericization
4. Create project configuration template system
5. Test genericized agents with a different project
6. Create migration guide for existing Hyperion users

## Conclusion

The agent definition files are currently tightly coupled to the Hyperion project. Initial genericization work has been completed on the three core development agents (go-dev, ui-dev, sre). The recommended approach is to create a configuration-driven system with template variables, allowing projects to easily customize the generic agents without modifying the core agent logic.

---

Generated: 2025-01-20
Author: Claude Code Agent Analysis
Status: Initial Analysis Complete - Genericization In Progress
