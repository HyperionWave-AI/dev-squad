# Hyperion Add Tests to Existing Code

Add comprehensive unit and integration tests to existing implementation following Hyperion testing best practices.

## Instructions

You are coordinating a testing task for existing code. Follow the Hyperion Golden Path with parallel test agent coordination.

### Phase 1: Context Gathering & Code Analysis (Sequential)

1. **Query Testing Lessons**
   ```
   reflection_query_relevant_lessons({
     situation: "Adding tests for {{feature_description}}. Need to understand testing patterns, common pitfalls, and coverage requirements.",
     limit: 5
   })
   ```

2. **Find Existing Implementation**
   ```
   code_index_search({
     query: "{{feature_description}} implementation",
     retrieve: "chunk-l",
     fileTypes: [".go", ".ts", ".tsx"],
     minScore: 0.6
   })
   ```
   → Identify implementation files

3. **Find Existing Test Patterns**
   ```
   code_index_search({
     query: "test patterns for {{similar_feature}} with mocks and table-driven tests",
     retrieve: "chunk-m",
     fileTypes: ["_test.go", ".test.ts", ".spec.ts"],
     minScore: 0.5
   })
   ```
   → Learn existing test conventions

4. **Query Test Knowledge**
   ```
   coordinator_query_knowledge({
     collection: "technical-knowledge",
     query: "testing best practices unit integration mocking coverage",
     limit: 3
   })
   ```

5. **Read Implementation Files**
   - Read top 2-3 files from step 2
   - Analyze: functions, dependencies, edge cases, error paths

### Phase 2: Test Planning (Sequential)

6. **Create Human Task**
   ```
   coordinator_create_human_task({
     prompt: "Add comprehensive unit and integration tests for: {{verbatim_description}}"
   })
   ```
   → Save humanTaskId

7. **Analyze Test Requirements**
   - List all functions/handlers to test
   - Identify external dependencies (DB, APIs, services)
   - Map error scenarios
   - Determine coverage gaps

8. **Record Testing Decision**
   ```
   reflection_record_decision({
     chatId: "{{session_id}}",
     context: {
       userRequest: "Add tests for {{feature}}",
       availableInfo: "Implementation files: {{files}}, Dependencies: {{deps}}",
       uncertainty: "{{test_coverage_unknowns}}"
     },
     decision: {
       action: "Create unit tests ({{count}} test files) + integration tests ({{count}} scenarios)",
       reasoning: "{{why_this_test_strategy}}",
       alternatives: ["{{other_test_approaches}}"],
       confidence: {{0.0_to_1.0}}
     },
     predictions: {
       successProbability: {{0.0_to_1.0}},
       risks: ["{{test_brittleness}}", "{{mock_complexity}}"],
       timeEstimate: "{{estimate}}"
     }
   })
   ```
   → Save decisionId

### Phase 3: Agent Task Creation (Sequential - prepare for parallel execution)

9. **Create Unit Test Tasks** (one per major component)

   **For Go Backend:**
   ```
   coordinator_create_agent_task({
     humanTaskId: "{{from_step_6}}",
     agentName: "go-dev",
     role: "Write comprehensive unit tests for {{component_name}} with table-driven tests and mocks",
     contextSummary: "
       WHY: Ensure {{component}} reliability, prevent regressions, enable safe refactoring
       WHAT: Test {{functions_list}}
       HOW:
       - Table-driven tests for all public functions
       - Mock external dependencies (MongoDB, HTTP clients, NATS)
       - Test happy path + error paths
       - Edge cases: nil inputs, empty strings, boundary values
       CONSTRAINTS:
       - Use testify/assert for assertions
       - Use testify/mock or mockery for mocks
       - Achieve >80% coverage
       - Run: go test ./... -v -cover -race
       PATTERNS:
       {{test_patterns_from_step_3}}
       FILES_TO_TEST:
       {{implementation_files}}
     ",
     filesModified: ["{{test_file_paths}}"],
     qdrantCollections: ["technical-knowledge"],
     todos: [
       {
         description: "Write unit tests for {{function_name}}",
         filePath: "{{test_file_path}}",
         functionName: "Test{{FunctionName}}",
         contextHint: "
           Test scenarios:
           1. Happy path: {{expected_behavior}}
           2. Error: {{error_case_1}}
           3. Edge: {{edge_case_1}}
           Mock: {{dependencies_to_mock}}
           Pattern: See {{reference_test_file}}
         "
       }
     ]
   })
   ```

   **For React/TypeScript:**
   ```
   coordinator_create_agent_task({
     humanTaskId: "{{from_step_6}}",
     agentName: "ui-tester",
     role: "Write comprehensive unit tests for React components with RTL and mocked dependencies",
     contextSummary: "
       WHY: Ensure UI component reliability and user interaction correctness
       WHAT: Test {{component_names}}
       HOW:
       - React Testing Library for rendering
       - Mock API calls with MSW (Mock Service Worker)
       - Test user interactions (click, type, submit)
       - Test state changes and side effects
       CONSTRAINTS:
       - Use @testing-library/react
       - Use @testing-library/user-event for interactions
       - Mock fetch/axios with MSW
       - Run: npm test -- --coverage
       PATTERNS:
       {{test_patterns_from_step_3}}
       FILES_TO_TEST:
       {{component_files}}
     ",
     filesModified: ["{{test_file_paths}}"],
     todos: [
       {
         description: "Write unit tests for {{ComponentName}}",
         filePath: "{{test_file_path}}",
         contextHint: "
           Test scenarios:
           1. Renders correctly with props
           2. Handles user interactions ({{interactions}})
           3. Displays loading/error states
           4. Calls API correctly
           Pattern: See {{reference_test_file}}
         "
       }
     ]
   })
   ```
   → Save unitTestTaskIds[]

10. **Create Integration Test Task**
    ```
    coordinator_create_agent_task({
      humanTaskId: "{{from_step_6}}",
      agentName: "End-to-End Testing Coordinator",
      role: "Create end-to-end integration tests validating full feature workflow with real services",
      contextSummary: "
        WHY: Validate feature works end-to-end with real dependencies
        WHAT: Test {{feature_name}} complete workflow
        HOW:
        - Use ai3_integration_suite for Go services
        - Use Playwright for UI flows
        - Test with real MongoDB, NATS, services
        - Test authentication flow
        - Test error handling (DB down, auth failure)
        SCENARIOS:
        1. Happy path: {{full_workflow}}
        2. Auth failure: {{auth_error_handling}}
        3. DB error: {{db_error_handling}}
        4. Concurrent requests: {{concurrency_test}}
        CONSTRAINTS:
        - Run against hyperion-dev environment
        - MANDATORY: 100% pass rate
        - Clean up test data after run
        DEPENDENCIES:
        - Requires unit tests complete first
        FILES_TO_TEST:
        {{implementation_files}}
      ",
      filesModified: ["{{integration_test_paths}}"],
      qdrantCollections: ["technical-knowledge"],
      todos: [
        {
          description: "Write integration test for {{scenario}}",
          contextHint: "
            Setup: {{test_environment_setup}}
            Steps:
            1. {{step_1}}
            2. {{step_2}}
            3. {{step_3}}
            Assertions: {{expected_results}}
            Cleanup: {{cleanup_actions}}
            Pattern: See ai3_integration_suite/internal/testcases/
          "
        }
      ]
    })
    ```
    → Save integrationTestTaskId

### Phase 4: Parallel Unit Test Execution

11. **Launch Unit Test Agents** (in parallel)
    ```
    # Launch multiple unit test agents simultaneously
    FOR EACH unitTestTaskId:
      Task({
        subagent_type: "go-dev" | "ui-tester",
        description: "Unit tests for {{component}}",
        prompt: "
          Retrieve task: coordinator_get_agent_task('{{unitTestTaskId}}')
          Read contextSummary (80% of needed info)
          Read implementation files (≤3 files)
          Write table-driven tests
          Mock external dependencies
          Test happy path + errors + edge cases
          Run tests: go test -v -cover -race OR npm test -- --coverage
          Target: >80% coverage
          Update coordinator_update_todo_status per TODO with coverage stats
          Store gotchas via coordinator_upsert_knowledge
        "
      })
    ```

12. **Monitor Unit Test Progress**
    - Poll coordinator_list_agent_tasks({ humanTaskId })
    - Wait for ALL unit test tasks status: "completed"
    - If any blocked, add guidance via coordinator_add_task_prompt_notes

### Phase 5: Sequential Integration Test Execution

13. **Launch Integration Test Agent** (after ALL unit tests pass)
    ```
    Task({
      subagent_type: "End-to-End Testing Coordinator",
      description: "Integration tests for {{feature}}",
      prompt: "
        Retrieve task: coordinator_get_agent_task('{{integrationTestTaskId}}')
        Review unit test results: {{check_coverage}}
        Write end-to-end test scenarios
        Test with real services (MongoDB, NATS, security-api)
        Verify authentication flow
        Test error handling (service failures)
        Test concurrent requests (if applicable)
        Run: make test-integration OR npm run test:e2e
        MANDATORY: 100% pass rate (no flaky tests)
        Update coordinator_update_todo_status per scenario
        Store integration patterns via coordinator_upsert_knowledge
      "
    })
    ```

14. **Monitor Integration Test Progress**
    - Poll coordinator_list_agent_tasks({ humanTaskId })
    - Wait for integration test status: "completed"
    - Verify 100% pass rate

### Phase 6: Coverage Analysis & Validation

15. **Analyze Coverage Results**
    - Go: Parse `go test -cover` output
    - TypeScript: Parse coverage/lcov-report
    - Identify gaps (functions with <80% coverage)
    - Check critical paths are tested

16. **Create Gap-Filling Tasks** (if coverage <80%)
    ```
    IF coverage < 80%:
      coordinator_create_agent_task({
        humanTaskId,
        agentName: "go-dev" | "ui-tester",
        role: "Fill test coverage gaps to reach >80%",
        contextSummary: "
          Current coverage: {{current_coverage}}%
          Missing coverage:
          {{uncovered_functions}}
          Add tests for edge cases and error paths
        ",
        todos: [...]
      })
    ```

### Phase 7: Post-Flight (Coordinator - You)

17. **Update All TODO Statuses**
    - Mark each TODO as completed with coverage stats

18. **Store Test Knowledge**
    ```
    coordinator_upsert_knowledge({
      collection: "task:hyperion://task/human/{{humanTaskId}}",
      text: "
        TESTING STRATEGY: {{approach_taken}}
        UNIT TESTS:
        - Files: {{test_files}}
        - Coverage: {{coverage_percentage}}%
        - Patterns used: {{patterns}}
        INTEGRATION TESTS:
        - Scenarios: {{scenario_list}}
        - Pass rate: {{pass_rate}}
        - Run time: {{execution_time}}
        GOTCHAS:
        - {{mock_complexity_issues}}
        - {{flaky_test_fixes}}
        - {{environment_dependencies}}
        HANDOFF: {{how_to_run_tests_maintain_them}}
      ",
      taskId: "{{unitTestTaskIds[0]}}",
      metadata: {
        type: "testing_completion",
        coverage: {{percentage}},
        files: [...],
        pass_rate: 1.0
      }
    })
    ```

19. **Record Outcome**
    ```
    reflection_record_outcome({
      decisionId: "{{from_step_8}}",
      outcome: {
        success: {{true_if_all_tests_pass}},
        actualResult: "
          Unit tests: {{count}} files, {{coverage}}% coverage
          Integration tests: {{count}} scenarios, {{pass_rate}}% pass rate
          Total execution time: {{time}}
        ",
        userFeedback: "{{feedback}}"
      },
      analysis: {
        predictionAccuracy: {{actual_vs_predicted}},
        confidenceCalibration: "{{well-calibrated|overconfident|underconfident}}",
        missedSignals: ["{{what_was_missed_in_test_planning}}"]
      }
    })
    ```

20. **Extract Testing Lesson** (if novel pattern or gotcha)
    ```
    reflection_extract_lesson({
      patternName: "{{test_pattern_name}}",
      problem: "{{testing_challenge_encountered}}",
      solution: "{{how_it_was_solved}}",
      antipattern: "{{what_not_to_do_in_tests}}",
      context: "Testing {{feature_type}} with {{dependencies}}",
      applicableTo: ["testing", "{{language}}", "{{framework}}"],
      confidence: {{0.0_to_1.0}}
    })
    ```

21. **Store Reusable Test Patterns**
    ```
    knowledge_store({
      collectionName: "technical-knowledge",
      information: "
        PATTERN: {{test_pattern_name}}
        USE CASE: {{when_to_use}}
        IMPLEMENTATION:
        {{code_snippet_or_description}}
        BENEFITS: {{why_this_pattern_works}}
        GOTCHAS: {{common_mistakes}}
      ",
      metadata: {
        tags: ["testing", "{{language}}", "pattern"],
        pattern_type: "test_pattern",
        confidence: {{score}}
      }
    })
    ```

22. **Mark All Tasks Complete**
    ```
    FOR EACH unitTestTaskId:
      coordinator_update_task_status({
        taskId: unitTestTaskId,
        status: "completed",
        notes: "Unit tests complete. Coverage: {{coverage}}%."
      })

    coordinator_update_task_status({
      taskId: integrationTestTaskId,
      status: "completed",
      notes: "Integration tests complete. Pass rate: 100%."
    })

    coordinator_update_task_status({
      taskId: humanTaskId,
      status: "completed",
      notes: "All tests complete. Coverage >80%. Integration tests 100% pass rate."
    })
    ```

### Pre-Flight Checklist
- [ ] Testing lessons queried
- [ ] Implementation files found (code_index_search)
- [ ] Test patterns researched (code_index_search)
- [ ] Test knowledge queried (coordinator_query_knowledge)
- [ ] Human task created
- [ ] Testing decision recorded
- [ ] Unit test tasks created (one per component)
- [ ] Integration test task created
- [ ] Test agents launched (unit in parallel, integration after)

### Post-Flight Checklist
- [ ] All TODOs updated with coverage stats
- [ ] Test knowledge stored (patterns, gotchas, handoff)
- [ ] Outcome recorded (predicted vs actual)
- [ ] Lesson extracted (if novel pattern)
- [ ] Reusable patterns stored (if applicable)
- [ ] All tasks marked complete
- [ ] Coverage >80% achieved
- [ ] Integration tests 100% pass rate

### Testing Best Practices (MANDATORY)

**Go Unit Tests:**
- ✅ Table-driven tests ([]struct{name, input, expected})
- ✅ testify/assert for assertions
- ✅ testify/mock or mockery for mocks
- ✅ Test error paths (not just happy path)
- ✅ Test edge cases (nil, empty, boundary values)
- ✅ Run with race detector: `go test -race`
- ✅ >80% coverage minimum
- ✅ **REGRESSION TESTS: Test functions named `Test<BugName>_Regression` for all bug fixes**
- ❌ No global state in tests
- ❌ No test interdependencies (each test isolated)

**Regression Test Requirements (For Bug Fixes)**:
- ✅ MUST reproduce exact bug scenario
- ✅ MUST fail before fix applied
- ✅ MUST pass after fix applied
- ✅ MUST be named `Test<BugName>_Regression`
- ✅ MUST include comments: Bug, Root Cause, Fix
- ✅ MUST be in same commit as bug fix
- ❌ No generic "this might break" tests - test SPECIFIC bug

**React/TypeScript Tests:**
- ✅ React Testing Library (not Enzyme)
- ✅ User-event for interactions
- ✅ MSW (Mock Service Worker) for API mocking
- ✅ Test user perspective (not implementation details)
- ✅ Test accessibility (screen readers)
- ✅ Async handling (waitFor, findBy)
- ❌ No direct state access (test via UI)
- ❌ No implementation detail testing (internal functions)

**Integration Tests:**
- ✅ Test with real services (MongoDB, NATS, APIs)
- ✅ Test authentication flows
- ✅ Test error handling (service failures)
- ✅ Test concurrent requests (race conditions)
- ✅ Clean up test data after run
- ✅ 100% pass rate (no flaky tests)
- ❌ No hardcoded test data (generate or use fixtures)
- ❌ No dependencies between test cases

**Coverage Requirements:**
- Unit tests: >80% line coverage
- Critical paths: 100% coverage (auth, payment, security)
- Integration tests: All major workflows
- Error paths: All error scenarios tested

### Test Organization

**Go:**
```
service/
├── handlers/
│   ├── user_handler.go
│   └── user_handler_test.go     # Unit tests
├── services/
│   ├── user_service.go
│   └── user_service_test.go     # Unit tests
└── integration/
    └── user_flow_test.go         # Integration tests
```

**TypeScript/React:**
```
components/
├── UserProfile/
│   ├── UserProfile.tsx
│   ├── UserProfile.test.tsx     # Unit tests
│   └── UserProfile.spec.tsx     # Integration tests (optional)
└── __tests__/
    └── user-flow.test.tsx        # E2E tests
```

### Execution Summary

Present the user with:
1. **Implementation Analysis**: Files found, dependencies identified
2. **Test Strategy**: Unit tests ({{count}}), Integration tests ({{count}})
3. **Tasks Created**: humanTaskId, unitTestTaskIds[], integrationTestTaskId
4. **Agents Launched**: Status of each test agent
5. **Coverage Results**: Line coverage %, critical paths covered
6. **Pass Rate**: Unit tests pass rate, Integration tests pass rate (must be 100%)
7. **Lessons Learned**: Testing patterns, gotchas, improvements
8. **Next Steps**: Maintenance, CI/CD integration

---

**Describe the existing implementation to test**: {{paste_feature_description_or_file_paths_here}}
