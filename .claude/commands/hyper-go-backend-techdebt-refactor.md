# Hyperion Go Backend Tech Debt & Refactor Task

Refactor Go backend services to meet Hyperion gold standards with improved code quality, test coverage, and compliance.

## Instructions

You are coordinating a Go backend refactoring/tech debt task. Focus on code quality, compliance, and test coverage.

### Phase 1: Service Audit (Sequential)

1. **Query Past Lessons on Refactoring**
   ```
   reflection_query_relevant_lessons({
     situation: "Refactoring {{service_name}} to meet gold standards",
     limit: 5
   })
   ```

2. **Analyze Current Service State**
   ```
   code_index_search({
     query: "{{service_name}} main handlers services",
     retrieve: "chunk-m",
     fileTypes: [".go"],
     minScore: 0.5
   })
   ```

3. **Query Known Tech Debt Patterns**
   ```
   coordinator_query_knowledge({
     collection: "hyperion_bugs",
     query: "tech debt refactoring {{service_name}}",
     limit: 3
   })
   ```

4. **Run Compliance Audit**
   Execute these checks to identify violations:
   - `make lint` - Check all linting rules
   - `go test ./... -cover` - Current test coverage
   - Search for forbidden patterns:
     - `c.Get("identity")` or `c.MustGet("identity")`
     - `models.NewSystemIdentity()`
     - `GetCompanyPublicKey()`
     - `http.NewServeMux()` or `gorilla/mux`
     - Static `JWT_SECRET` in configs

### Phase 2: Task Planning (Sequential)

5. **Create Human Task**
   ```
   coordinator_create_human_task({
     prompt: "Tech debt refactor: {{service_name}} - {{focus_areas}}"
   })
   ```
   → Save humanTaskId

6. **Record Refactoring Decision**
   ```
   reflection_record_decision({
     chatId: "{{session_id}}",
     context: {
       userRequest: "Refactor {{service_name}} to gold standards",
       availableInfo: "Audit findings: {{violations_found}}, coverage: {{current_coverage}}%",
       uncertainty: "{{breaking_change_risks}}"
     },
     decision: {
       action: "{{refactoring_approach}}",
       reasoning: "{{why_this_priority_order}}",
       alternatives: ["{{other_approaches}}"],
       confidence: {{0.0_to_1.0}}
     },
     predictions: {
       successProbability: {{0.0_to_1.0}},
       risks: ["{{potential_regressions}}", "{{api_breaking_changes}}"],
       timeEstimate: "{{estimate}}"
     }
   })
   ```
   → Save decisionId

### Phase 3: Agent Task Creation

7. **Create Security Compliance Task** (Priority 1 - Critical)
   ```
   coordinator_create_agent_task({
     humanTaskId: "{{from_step_5}}",
     agentName: "go-dev",
     role: "Fix security and compliance violations to meet Hyperion gold standards",
     contextSummary: "
       WHY: Security violations create vulnerabilities and break CI/CD
       WHAT: Fix all linting violations and security patterns
       VIOLATIONS FOUND:
       - {{list_violations_from_audit}}
       MANDATORY FIXES:
       - Replace c.Get('identity') → middleware.GetIdentityFromCentralizedGinContext(c)
       - Replace NewSystemIdentity() → HTTP/Gin identity provider
       - Replace GetCompanyPublicKey() → security-api /validate endpoint
       - Replace gorilla/mux → Gin framework
       - Remove static JWT_SECRET → MCPJWTMiddleware
       CONSTRAINTS:
       - Run make lint after EVERY change
       - NO functional changes, ONLY compliance fixes
       - Preserve existing behavior exactly
       TESTING: Run existing tests after each fix
     ",
     filesModified: ["{{files_with_violations}}"],
     qdrantCollections: ["technical-knowledge"],
     todos: [
       {
         description: "Fix identity access pattern violations",
         filePath: "{{service}}/internal/handlers/",
         contextHint: "Replace c.Get/MustGet('identity') with middleware.GetIdentityFromCentralizedGinContext(c)"
       },
       {
         description: "Fix JWT validation patterns",
         filePath: "{{service}}/internal/middleware/",
         contextHint: "Use security-api /validate, NOT GetCompanyPublicKey()"
       },
       {
         description: "Run make lint and fix remaining issues",
         contextHint: "Iterate until make lint passes with 0 errors"
       }
     ]
   })
   ```
   → Save securityTaskId

8. **Create Code Quality Refactor Task** (Priority 2)
   ```
   coordinator_create_agent_task({
     humanTaskId: "{{from_step_5}}",
     agentName: "go-dev",
     role: "Improve code quality, reduce complexity, apply DRY principles",
     contextSummary: "
       WHY: Improve maintainability and reduce tech debt
       WHAT: Refactor code following Go best practices
       TARGETS:
       - Functions >50 lines → Extract helpers
       - Duplicate code → Create shared utilities
       - Complex conditionals → Simplify or extract
       - Magic numbers/strings → Named constants
       - Missing error wrapping → Add context with fmt.Errorf
       CONSTRAINTS:
       - Preserve API contracts (no breaking changes)
       - Keep commits atomic (one refactor per commit)
       - Add comments for complex logic
       - Follow Go naming conventions
       DEPENDENCIES: Wait for security compliance task
     ",
     filesModified: ["{{target_files}}"],
     todos: [
       {
         description: "Identify and extract duplicate code patterns",
         contextHint: "Search for similar code blocks, create shared functions in internal/utils/"
       },
       {
         description: "Simplify complex functions (>50 lines)",
         contextHint: "Extract helper functions, use early returns, reduce nesting"
       },
       {
         description: "Add error context wrapping",
         contextHint: "Use fmt.Errorf('operation failed: %w', err) pattern"
       },
       {
         description: "Replace magic values with constants",
         contextHint: "Create constants.go for service-specific values"
       }
     ]
   })
   ```
   → Save codeQualityTaskId

9. **Create Test Coverage Task** (Priority 3)
   ```
   coordinator_create_agent_task({
     humanTaskId: "{{from_step_5}}",
     agentName: "go-dev",
     role: "Increase unit test coverage to >80% with comprehensive edge case testing",
     contextSummary: "
       WHY: Low test coverage leads to regressions
       CURRENT COVERAGE: {{coverage_percentage}}%
       TARGET: >80% coverage
       UNCOVERED FILES:
       - {{list_uncovered_files}}
       TEST REQUIREMENTS:
       - Table-driven tests for all public functions
       - Edge cases: nil inputs, empty strings, boundary values
       - Error path testing for every error return
       - Mock external dependencies (MongoDB, HTTP clients)
       PATTERNS:
       - Use testify/assert and testify/require
       - Use mockery for interface mocks
       - Follow *_test.go naming convention
       DEPENDENCIES: Wait for code quality refactor
     ",
     filesModified: ["{{test_file_paths}}"],
     todos: [
       {
         description: "Add tests for {{uncovered_handler}}",
         filePath: "{{handler_test_file}}",
         contextHint: "Table-driven tests: success, auth failure, validation error, DB error"
       },
       {
         description: "Add tests for {{uncovered_service}}",
         filePath: "{{service_test_file}}",
         contextHint: "Mock dependencies, test business logic isolation"
       },
       {
         description: "Add edge case tests",
         contextHint: "nil checks, empty inputs, concurrent access, timeout handling"
       },
       {
         description: "Verify >80% coverage",
         contextHint: "Run: go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out"
       }
     ]
   })
   ```
   → Save testCoverageTaskId

10. **Create Documentation & Comments Task** (Priority 4)
    ```
    coordinator_create_agent_task({
      humanTaskId: "{{from_step_5}}",
      agentName: "go-dev",
      role: "Add missing documentation and improve code comments",
      contextSummary: "
        WHY: Documentation improves maintainability
        WHAT: Add godoc comments and inline documentation
        REQUIREMENTS:
        - All exported functions need godoc comments
        - Complex algorithms need inline explanations
        - Package-level doc.go for each package
        - Update README.md with architecture overview
        CONSTRAINTS:
        - Follow godoc conventions
        - Comments explain WHY, not WHAT
        - No obvious comments ('// increment i')
        DEPENDENCIES: Wait for test coverage task
      ",
      filesModified: ["{{target_files}}"],
      todos: [
        {
          description: "Add godoc comments to exported functions",
          contextHint: "Format: // FunctionName does X. It returns Y when Z."
        },
        {
          description: "Add package documentation",
          filePath: "{{package}}/doc.go",
          contextHint: "Package overview, usage examples, key types"
        },
        {
          description: "Document complex business logic",
          contextHint: "Explain WHY decisions were made, not obvious WHAT"
        }
      ]
    })
    ```
    → Save documentationTaskId

### Phase 4: Sequential Execution (Dependencies Matter)

11. **Launch Security Compliance Agent** (First - Blocking)
    ```
    Task({
      subagent_type: "go-dev",
      description: "Security compliance fixes for {{service_name}}",
      prompt: "
        Retrieve task: coordinator_get_agent_task('{{securityTaskId}}')
        CRITICAL: This is security compliance work. Follow patterns EXACTLY.

        For each violation:
        1. Read the file containing violation
        2. Apply the EXACT fix pattern from contextHint
        3. Run make lint
        4. If lint fails, fix and retry
        5. Run existing tests to prevent regressions

        Update coordinator_update_todo_status per TODO with:
        - Exact file:line changes
        - Before/after snippets
        - make lint result

        Store findings via coordinator_upsert_knowledge:
        - Violation pattern found
        - Fix applied
        - Files modified
      "
    })
    ```

12. **Monitor and Wait for Security Task**
    - Poll coordinator_list_agent_tasks({ humanTaskId })
    - Wait for securityTaskId status: "completed"
    - Verify make lint passes

13. **Launch Code Quality Agent** (After security complete)
    ```
    Task({
      subagent_type: "go-dev",
      description: "Code quality refactor for {{service_name}}",
      prompt: "
        Retrieve task: coordinator_get_agent_task('{{codeQualityTaskId}}')
        Review security fixes from previous task.

        Refactoring guidelines:
        1. One logical change per commit
        2. Run tests after each refactor
        3. Preserve API contracts
        4. Extract, don't rewrite

        Update coordinator_update_todo_status with:
        - Refactoring applied
        - Complexity metrics (before/after)
        - Test results
      "
    })
    ```

14. **Launch Test Coverage Agent** (After code quality complete)
    ```
    Task({
      subagent_type: "go-dev",
      description: "Test coverage for {{service_name}}",
      prompt: "
        Retrieve task: coordinator_get_agent_task('{{testCoverageTaskId}}')

        Test writing guidelines:
        1. Run coverage first: go test ./... -cover
        2. Identify lowest coverage files
        3. Write tests targeting uncovered paths
        4. Use table-driven tests
        5. Mock external dependencies
        6. Test error paths explicitly

        Target: >80% coverage

        Update coordinator_update_todo_status with:
        - Coverage before/after per file
        - Test count added
        - Edge cases covered
      "
    })
    ```

15. **Launch Documentation Agent** (After tests complete)
    ```
    Task({
      subagent_type: "go-dev",
      description: "Documentation for {{service_name}}",
      prompt: "
        Retrieve task: coordinator_get_agent_task('{{documentationTaskId}}')

        Documentation standards:
        1. godoc format for all exports
        2. Package doc.go files
        3. Complex logic explanations
        4. No obvious comments

        Update coordinator_update_todo_status with:
        - Functions documented
        - Packages with doc.go
      "
    })
    ```

### Phase 5: Post-Flight (Coordinator - You)

16. **Verify All Changes**
    - Run `make lint` - Must pass
    - Run `go test ./... -cover` - Must be >80%
    - Review git diff for breaking changes

17. **Update All TODO Statuses**
    - Mark each TODO as completed with file:line refs
    - Include metrics: coverage %, lint errors fixed

18. **Store Refactoring Knowledge**
    ```
    coordinator_upsert_knowledge({
      collection: "task:hyperion://task/human/{{humanTaskId}}",
      text: "
        SERVICE: {{service_name}}
        TECH DEBT ADDRESSED:
        - Security violations fixed: {{count}}
        - Code quality improvements: {{summary}}
        - Test coverage: {{before}}% → {{after}}%
        - Documentation added: {{files_count}}

        BREAKING CHANGES: {{none_or_list}}

        PATTERNS DISCOVERED:
        - {{reusable_patterns}}

        REMAINING TECH DEBT:
        - {{items_for_future}}

        HANDOFF:
        - {{what_next_maintainer_needs}}
      ",
      taskId: "{{securityTaskId}}",
      metadata: { type: "refactoring", service: "{{service_name}}" }
    })
    ```

19. **Record Outcome**
    ```
    reflection_record_outcome({
      decisionId: "{{from_step_6}}",
      outcome: {
        success: {{true_or_false}},
        actualResult: "Security: {{fixed}}, Coverage: {{before}}%→{{after}}%",
        userFeedback: "{{feedback}}"
      },
      analysis: {
        predictionAccuracy: {{actual_vs_predicted}},
        confidenceCalibration: "{{calibration}}",
        missedSignals: ["{{missed}}"]
      }
    })
    ```

20. **Extract Refactoring Lessons**
    ```
    reflection_extract_lesson({
      patternName: "{{service_name}}-refactoring-pattern",
      problem: "Tech debt accumulation in {{service_name}}",
      solution: "{{approach_that_worked}}",
      antipattern: "{{what_to_avoid}}",
      applicableTo: ["golang", "refactoring", "tech-debt", "{{service_type}}"],
      confidence: {{0.0_to_1.0}}
    })
    ```

21. **Store Global Knowledge** (if pattern is reusable)
    ```
    knowledge_store({
      collectionName: "technical-knowledge",
      information: "{{concise_refactoring_pattern}}",
      metadata: {
        tags: ["refactoring", "go", "tech-debt"],
        pattern_type: "refactoring",
        service: "{{service_name}}"
      }
    })
    ```

22. **Mark All Tasks Complete**
    ```
    coordinator_update_task_status({ taskId: "{{securityTaskId}}", status: "completed", notes: "Lint passes. No violations." })
    coordinator_update_task_status({ taskId: "{{codeQualityTaskId}}", status: "completed", notes: "Refactoring complete." })
    coordinator_update_task_status({ taskId: "{{testCoverageTaskId}}", status: "completed", notes: "Coverage >80%." })
    coordinator_update_task_status({ taskId: "{{documentationTaskId}}", status: "completed", notes: "Docs complete." })
    coordinator_update_task_status({ taskId: "{{humanTaskId}}", status: "completed", notes: "Tech debt addressed." })
    ```

### Pre-Flight Checklist
- [ ] Lessons queried (refactoring patterns)
- [ ] Service code analyzed (code_index_search)
- [ ] Known tech debt queried (hyperion_bugs collection)
- [ ] Compliance audit run (make lint, coverage check)
- [ ] Human task created
- [ ] Refactoring decision recorded
- [ ] Agent tasks created (security → quality → tests → docs)
- [ ] Dependencies respected (sequential execution)

### Post-Flight Checklist
- [ ] make lint passes with 0 errors
- [ ] Test coverage >80%
- [ ] All TODOs updated with metrics
- [ ] Knowledge stored (tech debt findings)
- [ ] Outcome recorded
- [ ] Lessons extracted
- [ ] All tasks marked complete
- [ ] No breaking API changes (or documented)

### Security Compliance Checklist (MANDATORY)
After refactoring, verify ALL of these:
- [ ] NO `c.Get("identity")` or `c.MustGet("identity")` anywhere
- [ ] NO `models.NewSystemIdentity()` anywhere
- [ ] NO `GetCompanyPublicKey()` anywhere
- [ ] NO `http.NewServeMux()` or gorilla/mux imports
- [ ] NO static `JWT_SECRET` in configs
- [ ] ALL identity access via `middleware.GetIdentityFromCentralizedGinContext(c)`
- [ ] ALL JWT validation via security-api
- [ ] ALL MCP endpoints use `middleware.MCPJWTMiddleware`
- [ ] ALL REST endpoints use `middleware.GinJWTCentralized`
- [ ] ALL MongoDB access via `database.NewSecureMongoClient`

### Coverage Targets
| Component | Minimum | Target |
|-----------|---------|--------|
| Handlers  | 70%     | 85%    |
| Services  | 80%     | 90%    |
| Utils     | 90%     | 95%    |
| Overall   | 80%     | 85%    |

### Execution Summary

Present the user with:
1. **Audit Results**: Violations found, current coverage, tech debt items
2. **Refactoring Plan**: Priority order, dependencies, estimated scope
3. **Tasks Created**: All task IDs with roles
4. **Progress**: Real-time updates (security → quality → tests → docs)
5. **Metrics**: Before/after comparison (lint errors, coverage %)
6. **Results**: Final compliance status, knowledge stored
7. **Remaining Work**: Any tech debt items for future

---

**Service to Refactor**: {{paste_service_name_here}}
**Focus Areas**: {{compliance|coverage|quality|documentation|all}}
