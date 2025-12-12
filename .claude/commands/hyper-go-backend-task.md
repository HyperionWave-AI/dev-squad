# Hyperion Go Backend Task

Implement a Go backend feature/fix following Hyperion best practices with parallel agent coordination.

## Instructions

You are coordinating a Go backend implementation task. Follow the Hyperion Golden Path with parallel execution.

### Phase 1: Context Gathering (Sequential)

1. **Query Past Lessons**
   ```
   reflection_query_relevant_lessons({
     situation: "{{user_requirements}}",
     limit: 5
   })
   ```

2. **Semantic Code Search**
   ```
   code_index_search({
     query: "{{semantic_description}}",
     retrieve: "chunk-m",
     fileTypes: [".go"],
     minScore: 0.6
   })
   ```

3. **Query Knowledge Base**
   ```
   coordinator_query_knowledge({
     collection: "technical-knowledge",
     query: "{{pattern_description}}",
     limit: 3
   })
   ```

### Phase 2: Task Planning (Sequential)

4. **Create Human Task**
   ```
   coordinator_create_human_task({
     prompt: "{{verbatim_user_requirements}}"
   })
   ```
   → Save humanTaskId

5. **Record Decision** (if architectural/security change)
   ```
   reflection_record_decision({
     chatId: "{{session_id}}",
     context: {
       userRequest: "{{user_requirements}}",
       availableInfo: "{{lessons_and_patterns_found}}",
       uncertainty: "{{unknowns}}"
     },
     decision: {
       action: "{{approach}}",
       reasoning: "{{why_this_approach}}",
       alternatives: ["{{other_options}}"],
       confidence: {{0.0_to_1.0}}
     },
     predictions: {
       successProbability: {{0.0_to_1.0}},
       risks: ["{{potential_issues}}"],
       timeEstimate: "{{estimate}}"
     }
   })
   ```
   → Save decisionId (if recorded)

### Phase 3: Agent Task Creation (Sequential - but prepare for parallel execution)

6. **Create Implementation Task**
   ```
   coordinator_create_agent_task({
     humanTaskId: "{{from_step_4}}",
     agentName: "go-dev",
     role: "Implement Go backend feature with security and performance best practices",
     contextSummary: "
       WHY: {{business_context}}
       WHAT: {{requirements_and_acceptance_criteria}}
       HOW: {{approach_from_code_search_and_knowledge}}
       CONSTRAINTS:
       - Use database.NewSecureMongoClient (NO system service identities)
       - Use middleware.GetIdentityFromCentralizedGinContext (NOT c.Get('identity'))
       - Use middleware.MCPJWTMiddleware for MCP endpoints
       - Use middleware.GinJWTCentralized for REST APIs
       - Follow URI normalization (shared/uris package)
       - Run make lint before completion
       LESSONS: {{key_lessons_from_reflection}}
       TESTING: Unit tests required, integration tests in separate task
     ",
     filesModified: ["{{file_paths}}"],
     qdrantCollections: ["technical-knowledge"],
     todos: [
       {
         description: "{{specific_implementation_step}}",
         filePath: "{{exact_file_path}}",
         functionName: "{{function_name}}",
         contextHint: "{{how_to_implement_with_pattern_refs}}"
       }
     ]
   })
   ```
   → Save implementationTaskId

7. **Create Unit Test Task**
   ```
   coordinator_create_agent_task({
     humanTaskId: "{{from_step_4}}",
     agentName: "go-dev",
     role: "Write comprehensive unit tests for Go backend implementation",
     contextSummary: "
       Test coverage for: {{feature_name}}
       Implementation files: {{files_from_step_6}}
       Test requirements:
       - Table-driven tests for edge cases
       - Mock external dependencies
       - Achieve >80% coverage
       - Test error paths
       Dependencies: Wait for implementation task completion
     ",
     filesModified: ["{{test_file_paths}}"],
     todos: [
       {
         description: "Write unit tests for {{feature}}",
         filePath: "{{test_file_path}}",
         contextHint: "Use testify/assert, mock MongoDB with mockery"
       }
     ]
   })
   ```
   → Save unitTestTaskId

8. **Create Integration Test Task**
   ```
   coordinator_create_agent_task({
     humanTaskId: "{{from_step_4}}",
     agentName: "End-to-End Testing Coordinator",
     role: "Create integration tests for end-to-end feature validation",
     contextSummary: "
       Integration test for: {{feature_name}}
       Test scenarios:
       - Happy path with real MongoDB
       - Error handling (auth failures, DB errors)
       - Concurrent request handling
       Dependencies: Wait for implementation + unit tests
     ",
     filesModified: ["{{integration_test_paths}}"],
     todos: [
       {
         description: "Write integration test for {{feature}}",
         contextHint: "Use ai3_integration_suite patterns, test with real services"
       }
     ]
   })
   ```
   → Save integrationTestTaskId

### Phase 4: Parallel Execution

9. **Launch Implementation Agent**
   ```
   Task({
     subagent_type: "go-dev",
     description: "Implement {{feature_short_name}}",
     prompt: "
       Retrieve task: coordinator_get_agent_task('{{implementationTaskId}}')
       Read contextSummary (80% of needed info)
       Query qdrantCollections ONLY if needed (≤1 query)
       Read ≤3 files via code_index_search results
       Start coding ≤2 minutes
       Update coordinator_update_todo_status per TODO with file:line refs
       Run make lint before marking complete
       Store decisions via coordinator_upsert_knowledge
     "
   })
   ```

10. **Monitor Implementation Progress**
    - Poll coordinator_list_agent_tasks({ humanTaskId })
    - Wait for implementation status: "completed"
    - If blocked, add guidance via coordinator_add_task_prompt_notes

### Phase 5: Sequential Testing (After Implementation)

11. **Launch Unit Test Agent** (after implementation complete)
    ```
    Task({
      subagent_type: "go-dev",
      description: "Unit tests for {{feature_short_name}}",
      prompt: "
        Retrieve task: coordinator_get_agent_task('{{unitTestTaskId}}')
        Review implementation files: {{files_from_step_6}}
        Write comprehensive table-driven tests
        Mock external dependencies
        Run: go test ./... -v -cover
        Target: >80% coverage
        Update coordinator_update_todo_status per TODO
      "
    })
    ```

12. **Launch Integration Test Agent** (after unit tests pass)
    ```
    Task({
      subagent_type: "End-to-End Testing Coordinator",
      description: "Integration tests for {{feature_short_name}}",
      prompt: "
        Retrieve task: coordinator_get_agent_task('{{integrationTestTaskId}}')
        Write end-to-end test scenarios
        Test with real MongoDB/services
        Verify happy path + error handling
        Run: make test-integration
        MANDATORY: 100% pass rate
        Update coordinator_update_todo_status per TODO
      "
    })
    ```

### Phase 6: Post-Flight (Coordinator - You)

13. **Update All TODO Statuses**
    - Mark each TODO as completed with notes (file:line refs)

14. **Store Knowledge**
    ```
    coordinator_upsert_knowledge({
      collection: "task:hyperion://task/human/{{humanTaskId}}",
      text: "
        DECISION: {{what_was_decided}}
        IMPLEMENTATION: {{key_changes_with_file_line_refs}}
        TESTING: {{test_results}}
        GOTCHAS: {{edge_cases_and_issues}}
        HANDOFF: {{what_next_maintainer_needs_to_know}}
      ",
      taskId: "{{implementationTaskId}}",
      metadata: { type: "completion", files: [...] }
    })
    ```

15. **Record Outcome** (if decision was recorded in step 5)
    ```
    reflection_record_outcome({
      decisionId: "{{from_step_5}}",
      outcome: {
        success: {{true_or_false}},
        actualResult: "{{what_happened}}",
        userFeedback: "{{feedback}}"
      },
      analysis: {
        predictionAccuracy: {{actual_vs_predicted}},
        confidenceCalibration: "{{well-calibrated|overconfident|underconfident}}",
        missedSignals: ["{{what_was_missed}}"]
      }
    })
    ```

16. **Extract Lesson** (if novel/reusable pattern)
    ```
    reflection_extract_lesson({
      patternName: "{{descriptive_pattern_name}}",
      problem: "{{problem_encountered}}",
      solution: "{{solution_applied}}",
      antipattern: "{{what_not_to_do}}",
      applicableTo: ["golang", "backend", "{{domain}}"],
      confidence: {{0.0_to_1.0}}
    })
    ```

17. **Store Global Knowledge** (if pattern is reusable)
    ```
    knowledge_store({
      collectionName: "technical-knowledge",
      information: "{{concise_reusable_pattern}}",
      metadata: { tags: [...], pattern_type: "{{type}}", confidence: {{score}} }
    })
    ```

18. **Mark All Tasks Complete**
    ```
    coordinator_update_task_status({
      taskId: "{{implementationTaskId}}",
      status: "completed",
      notes: "Implementation complete. Lint passed. Tests written."
    })
    coordinator_update_task_status({
      taskId: "{{unitTestTaskId}}",
      status: "completed",
      notes: "Unit tests pass. Coverage >80%."
    })
    coordinator_update_task_status({
      taskId: "{{integrationTestTaskId}}",
      status: "completed",
      notes: "Integration tests pass. 100% success rate."
    })
    coordinator_update_task_status({
      taskId: "{{humanTaskId}}",
      status: "completed",
      notes: "All agents complete. Knowledge stored. Lessons extracted."
    })
    ```

### Pre-Flight Checklist
- [ ] Lessons queried (reflection_query_relevant_lessons)
- [ ] Code searched semantically (code_index_search)
- [ ] Knowledge queried (coordinator_query_knowledge)
- [ ] Human task created
- [ ] Decision recorded (if architectural)
- [ ] Agent tasks created (implementation + tests)
- [ ] Sub-agents launched
- [ ] No direct implementation by coordinator

### Post-Flight Checklist
- [ ] All TODOs updated with file:line refs
- [ ] Knowledge stored (coordinator_upsert_knowledge)
- [ ] Outcome recorded (if decision made)
- [ ] Lesson extracted (if novel pattern)
- [ ] Global knowledge stored (if reusable)
- [ ] All tasks marked complete
- [ ] make lint passed
- [ ] All tests passed (100% success rate)

### Security Checklist (MANDATORY)
- [ ] Using database.NewSecureMongoClient (NOT system service identity)
- [ ] Using middleware.GetIdentityFromCentralizedGinContext (NOT c.Get("identity"))
- [ ] Using middleware.MCPJWTMiddleware for MCP endpoints
- [ ] Using middleware.GinJWTCentralized for REST APIs
- [ ] JWT validation via security-api (NOT GetCompanyPublicKey)
- [ ] URI normalization with shared/uris package
- [ ] No static JWT_SECRET in configs
- [ ] Gin framework only (no gorilla/mux, chi, echo)

### Execution Summary

Present the user with:
1. **Context Found**: Key lessons, patterns, code locations
2. **Approach**: Decision reasoning, alternatives considered
3. **Tasks Created**: humanTaskId, implementationTaskId, unitTestTaskId, integrationTestTaskId
4. **Agents Launched**: Status of each agent
5. **Progress**: Real-time updates as agents work
6. **Results**: Final status, knowledge stored, lessons learned
7. **Next Steps**: Any follow-up work or monitoring needed

---

**User Requirements**: {{paste_requirements_here}}
