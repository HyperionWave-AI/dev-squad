Hyperion Go Service Refactor Task

Refactor a Go service to Hyperion standards with parallel agent coordination.

Input:
Target Service: {{service_name}}
Service Root Package: {{service_package_path}} (e.g. internal/services/{{service_name}})

Goal

Bring {{service_name}} up to Hyperion standards by:
	•	Removing dead / unused code and legacy paths
	•	Extracting smaller, focused methods (readable, testable)
	•	Aligning with security & infrastructure standards
	•	Adding/expanding unit tests and integration tests
	•	Ensuring behavior is preserved and verified with tests

⸻

Phase 1: Context & Baseline (Sequential)
	1.	Query Past Refactoring Lessons

reflection_query_relevant_lessons({
  situation: "Refactor Go service {{service_name}}: remove dead code, improve design, add tests, enforce Hyperion standards",
  limit: 5
})


	2.	Semantic Code Search – Service Scope

code_index_search({
  query: "service {{service_name}} handlers, business logic, repository, tests, integration tests",
  retrieve: "chunk-m",
  fileTypes: [".go"],
  minScore: 0.6,
  pathPrefix: "{{service_package_path}}"
})


	3.	Dependency & Call Graph Context (if available)

coordinator_query_knowledge({
  collection: "technical-knowledge",
  query: "patterns for refactoring Go services with high coupling, large handlers, dead code detection, safe deprecations",
  limit: 3
})


	4.	Current Test & Coverage Snapshot
(Read-only, via tooling or existing knowledge)

coordinator_query_knowledge({
  collection: "ci-metrics",
  query: "go test coverage for {{service_name}}",
  limit: 3
})



⸻

Phase 2: Assessment & Refactor Plan (Sequential)
	5.	Create Human Task

coordinator_create_human_task({
  prompt: `
    Refactor Go service: {{service_name}}

    Goals:
    - Remove dead/unused code
    - Split large functions/methods into smaller, focused ones
    - Align with security & infra standards
    - Add/repair unit tests + integration tests
    - Preserve external behavior (API contracts, events, DB schema)

    Known constraints / business rules:
    {{verbatim_user_requirements}}
  `
})

→ Save humanTaskId

	6.	Architectural / Security Decision (if needed)
Only if you change public APIs, database schema, or auth behavior.

reflection_record_decision({
  chatId: "{{session_id}}",
  context: {
    userRequest: "Refactor {{service_name}} to Hyperion standards",
    availableInfo: "{{lessons_and_patterns_found}}",
    uncertainty: "{{unknowns}}"
  },
  decision: {
    action: "Refactor {{service_name}} with non-breaking behavior, internal structure changes only",
    reasoning: "Improve readability, testability, and security without changing external contracts.",
    alternatives: [
      "Broader redesign with breaking changes and versioned APIs"
    ],
    confidence: 0.8
  },
  predictions: {
    successProbability: 0.9,
    risks: [
      "Hidden dependencies on legacy paths",
      "Implicit contracts not covered by tests"
    ],
    timeEstimate: "1–3 working days depending on complexity"
  }
})

→ Save decisionId (if recorded)

	7.	Refactor Plan (High-Level) – Self-Generated
	•	Identify:
	•	God functions (too long, too many responsibilities)
	•	Duplicate logic across handlers/services
	•	Unused exported functions / dead code
	•	Legacy, unused feature flags
	•	Missing/weak tests for critical paths
	•	Produce a concise plan:
	•	refactorPlan.summary
	•	refactorPlan.steps[] (each step with file, function, rationale)
Store plan:

coordinator_upsert_knowledge({
  collection: "task:hyperion://task/human/{{humanTaskId}}",
  text: `
    SERVICE: {{service_name}}
    REFACTOR PLAN:
    {{refactor_plan_text}}
  `,
  taskId: "{{humanTaskId}}",
  metadata: { type: "refactor-plan" }
})



⸻

Phase 3: Agent Tasks (Sequential – Prepare Parallel Execution)
	8.	Create Refactor Implementation Task

coordinator_create_agent_task({
  humanTaskId: "{{humanTaskId}}",
  agentName: "go-dev",
  role: "Refactor Go service {{service_name}} to Hyperion standards",
  contextSummary: `
    SERVICE: {{service_name}}
    PACKAGE ROOT: {{service_package_path}}

    WHY:
    - Improve maintainability, readability, and testability
    - Remove dead code and legacy branches
    - Enforce Hyperion security and infra standards

    WHAT:
    - Apply refactorPlan
    - Consolidate duplicated logic
    - Split large handlers/functions into smaller, cohesive units
    - Remove/mark deprecated code paths safely

    HOW:
    - Use patterns and lessons from previous refactors
    - Keep public APIs and DB schemas backward compatible unless explicitly allowed
    - Maintain clear separation between transport (HTTP/MCP), business logic, and persistence

    CONSTRAINTS:
    - Use database.NewSecureMongoClient (NO system service identities)
    - Use middleware.GetIdentityFromCentralizedGinContext (NOT c.Get("identity"))
    - Use middleware.MCPJWTMiddleware for MCP endpoints
    - Use middleware.GinJWTCentralized for REST APIs
    - JWT validation via security-api (NOT GetCompanyPublicKey)
    - URI normalization with shared/uris package
    - No static JWT_SECRET in configs
    - Gin framework only (no gorilla/mux, chi, echo)
    - Run 'make lint' before completion

    LESSONS:
    {{key_lessons_from_reflection}}

    TESTING:
    - Do not delete any code without confirming it's unused (via search/tests/usage)
    - If behavior changes intentionally, document it explicitly in the task notes
  `,
  filesModified: [
    "{{service_package_path}}",
    // add more file paths when known
  ],
  qdrantCollections: ["technical-knowledge"],
  todos: [
    {
      description: "Identify and remove dead/unused code in {{service_name}}",
      filePath: "{{service_package_path}}",
      functionName: "",
      contextHint: "Use semantic search and global references to confirm non-usage; mark risky deletions with comments and tests."
    },
    {
      description: "Split large handlers and business functions into smaller methods with clear responsibilities",
      filePath: "{{service_package_path}}",
      functionName: "",
      contextHint: "Apply single-responsibility and clear input/output contracts; move shared logic to internal helper packages."
    }
  ]
})

→ Save refactorTaskId

	9.	Create Unit Test Task

coordinator_create_agent_task({
  humanTaskId: "{{humanTaskId}}",
  agentName: "go-dev",
  role: "Write / update unit tests for {{service_name}}",
  contextSummary: `
    SERVICE: {{service_name}}
    TARGET: Unit tests for refactored code

    Implementation files:
    - {{service_package_path}} and subpackages

    Test requirements:
    - Table-driven tests for major public functions
    - Edge-case coverage for error paths
    - Mocks for external dependencies (DB, HTTP clients, queues)
    - Coverage target: >80% for {{service_package_path}}
  `,
  filesModified: [
    "{{service_package_path}}",
    "{{service_package_path}}/../{{service_name}}_test.go"
  ],
  todos: [
    {
      description: "Write table-driven tests for main business logic in {{service_name}}",
      filePath: "{{service_package_path}}/service_test.go",
      contextHint: "Use testify/assert; mock MongoDB and external clients where needed."
    }
  ]
})

→ Save unitTestTaskId

	10.	Create Integration Test Task

coordinator_create_agent_task({
  humanTaskId: "{{humanTaskId}}",
  agentName: "End-to-End Testing Coordinator",
  role: "Create integration tests for {{service_name}} end-to-end behavior",
  contextSummary: `
    SERVICE: {{service_name}}
    TARGET: Integration-level verification

    Scenarios:
    - Happy path for main operations (CRUD / workflows)
    - Error handling (auth failures, DB failures, invalid inputs)
    - Idempotency and concurrency where applicable

    Dependencies:
    - Wait for refactorTaskId + unitTestTaskId to be completed
  `,
  filesModified: [
    "test/integration/{{service_name}}_integration_test.go"
  ],
  todos: [
    {
      description: "Write integration test scenarios for {{service_name}}",
      contextHint: "Follow ai3_integration_suite style; test with real MongoDB or test containers; validate logs/metrics if available."
    }
  ]
})

→ Save integrationTestTaskId

⸻

Phase 4: Parallel Execution
	11.	Launch Refactor Implementation Agent

Task({
  subagent_type: "go-dev",
  description: "Refactor service {{service_name}}",
  prompt: `
    Retrieve task: coordinator_get_agent_task('{{refactorTaskId}}')

    Steps:
    1. Read contextSummary and todos
    2. Use code_index_search to find all references for candidates for deletion/changes
    3. Keep external behavior stable; if you must break it, document and flag in notes
    4. Refactor in small, reviewable commits:
       - Extract helper methods
       - Consolidate duplicated logic
       - Remove dead code only after confirming non-usage
    5. Run: make lint
    6. Update coordinator_update_todo_status for each TODO with file:line refs
    7. Store key refactor decisions via coordinator_upsert_knowledge
  `
})

	12.	Monitor Refactor Progress

	•	Poll:

coordinator_list_agent_tasks({ humanTaskId: "{{humanTaskId}}" })


	•	Wait for refactorTaskId status: "completed"
	•	If blocked, add guidance:

coordinator_add_task_prompt_notes({ taskId: "{{refactorTaskId}}", notes: "{{guidance}}" })



⸻

Phase 5: Testing Sequence
	13.	Launch Unit Test Agent (after refactor complete)

Task({
  subagent_type: "go-dev",
  description: "Unit tests for {{service_name}}",
  prompt: `
    Retrieve task: coordinator_get_agent_task('{{unitTestTaskId}}')
    Review refactored implementation files under {{service_package_path}}

    Requirements:
    - Table-driven tests for core functions
    - Mock external dependencies
    - Cover happy + error paths
    - Run: go test {{service_package_path}}/... -v -cover
    - Target: >80% coverage

    Update coordinator_update_todo_status per TODO with file:line refs.
  `
})

	14.	Launch Integration Test Agent (after unit tests pass)

Task({
  subagent_type: "End-to-End Testing Coordinator",
  description: "Integration tests for {{service_name}}",
  prompt: `
    Retrieve task: coordinator_get_agent_task('{{integrationTestTaskId}}')

    Requirements:
    - Use real MongoDB / test containers as per project pattern
    - Validate full request → business logic → persistence flow
    - Test happy path + main error scenarios
    - Run: make test-integration
    - MANDATORY: 100% pass rate

    Update coordinator_update_todo_status per TODO with file:line refs.
  `
})


⸻

Phase 6: Post-Flight (Coordinator – You)
	15.	Update All TODO Statuses

	•	Ensure each TODO from all tasks has:
	•	status: completed
	•	Notes with file.go:line references

	16.	Store Refactor Knowledge

coordinator_upsert_knowledge({
  collection: "task:hyperion://task/human/{{humanTaskId}}",
  text: `
    SERVICE: {{service_name}}
    REFACTOR SUMMARY:
    - Key structural changes (with file:line references)
    - Deleted/merged components and why
    - New helper abstractions

    TESTING:
    - Unit test coverage summary
    - Integration scenarios and any notable cases

    GOTCHAS:
    - Hidden dependencies discovered
    - Behavior quirks preserved intentionally

    HANDOFF:
    - How to extend {{service_name}} safely after refactor
    - Recommended patterns to follow
  `,
  taskId: "{{refactorTaskId}}",
  metadata: { type: "completion", files: ["{{service_package_path}}"] }
})

	17.	Record Outcome (if decision recorded)

reflection_record_outcome({
  decisionId: "{{decisionId}}",
  outcome: {
    success: true,
    actualResult: "Service {{service_name}} refactored successfully and tests pass.",
    userFeedback: "{{feedback_if_any}}"
  },
  analysis: {
    predictionAccuracy: 0.9,
    confidenceCalibration: "well-calibrated",
    missedSignals: []
  }
})

	18.	Extract & Store Global Refactor Lesson

reflection_extract_lesson({
  patternName: "Refactor Go service {{service_name}} while preserving behavior",
  problem: "Legacy service with dead code, low test coverage, and weak structure",
  solution: "Incremental refactor with semantic search, safe deletions, and expanded tests",
  antipattern: "Big-bang rewrite without tests or usage analysis",
  applicableTo: ["golang", "backend", "service-refactor"],
  confidence: 0.85
})

knowledge_store({
  collectionName: "technical-knowledge",
  information: "How to refactor a Go service ({{service_name}}-style) safely while increasing test coverage and enforcing Hyperion security standards.",
  metadata: { tags: ["go", "refactor", "service", "testing"], pattern_type: "refactor", confidence: 0.85 }
})

	19.	Mark All Tasks Complete

coordinator_update_task_status({
  taskId: "{{refactorTaskId}}",
  status: "completed",
  notes: "Refactor complete. Lint passed."
})
coordinator_update_task_status({
  taskId: "{{unitTestTaskId}}",
  status: "completed",
  notes: "Unit tests pass. Coverage >80% for {{service_name}}."
})
coordinator_update_task_status({
  taskId: "{{integrationTestTaskId}}",
  status: "completed",
  notes: "Integration tests pass. 100% success rate."
})
coordinator_update_task_status({
  taskId: "{{humanTaskId}}",
  status: "completed",
  notes: "Service {{service_name}} refactored to standard; tests and knowledge updated."
})


⸻

Pre-Flight Checklist (Refactor)
	•	Lessons queried (reflection_query_relevant_lessons)
	•	Code searched for service (code_index_search scoped to {{service_package_path}})
	•	Knowledge queried (coordinator_query_knowledge)
	•	Human task created
	•	Decision recorded (if architectural/security change)
	•	Refactor + test tasks created
	•	Sub-agents launched
	•	No direct implementation by coordinator

Post-Flight Checklist (Refactor)
	•	All TODOs updated with file:line refs
	•	Knowledge stored (coordinator_upsert_knowledge)
	•	Outcome recorded (if decision made)
	•	Lesson extracted and stored globally
	•	All tasks marked complete
	•	make lint passed
	•	Unit tests passed with >80% coverage for service
	•	Integration tests passed (100% success rate)
	•	Behavior preserved or intentional changes documented

Security & Quality Checklist (MANDATORY)
	•	Using database.NewSecureMongoClient
	•	Using middleware.GetIdentityFromCentralizedGinContext
	•	Using middleware.MCPJWTMiddleware for MCP endpoints
	•	Using middleware.GinJWTCentralized for REST APIs
	•	JWT validation via security-api (no GetCompanyPublicKey)
	•	URI normalization with shared/uris
	•	No static JWT_SECRET in configs
	•	Gin only (no other HTTP frameworks)
	•	No dead code / unused exported functions in {{service_name}}
	•	Large methods split into small, cohesive functions
	•	Critical paths protected by tests
