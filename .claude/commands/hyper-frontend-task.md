# Hyperion React UI Feature Task

Implement a React/TypeScript UI feature following Hyperion best practices with parallel agent coordination and Atomic Design principles.

## Instructions

You are coordinating a React UI implementation task. Follow the Hyperion Golden Path with parallel execution, strict TypeScript typing, and Atomic Design architecture.

### Phase 1: Context Gathering (Sequential)

1. **Query Past Lessons**
   ```
   reflection_query_relevant_lessons({
     situation: "{{user_requirements}} - React TypeScript UI development with Atomic Design",
     limit: 5
   })
   ```
   Look for:
   - TypeScript antipatterns (avoid `any`)
   - Component composition patterns
   - State management decisions (Zustand vs local state)
   - Accessibility lessons
   - Performance optimizations

2. **Semantic Code Search**
   ```
   code_index_search({
     query: "{{semantic_description}} React component Atomic Design",
     retrieve: "chunk-m",
     fileTypes: [".tsx", ".ts"],
     minScore: 0.6,
     folderPath: "hyperion-ui/src/components"
   })
   ```
   Find similar components at the right Atomic Design level:
   - Atoms: Button, Input, Label, Avatar, Badge
   - Molecules: SearchBar, FormField, Card
   - Organisms: Header, Sidebar, DataTable
   - Templates: Page layouts
   - Pages: Full pages with data

3. **Query Knowledge Base**
   ```
   coordinator_query_knowledge({
     collection: "hyperion_ui_architecture",
     query: "{{pattern_description}} TypeScript React patterns",
     limit: 3
   })
   ```
   AND
   ```
   knowledge_find({
     collectionName: "technical-knowledge",
     query: "React hooks TypeScript best practices",
     retrieveMode: "chunk",
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
   → Save humanTaskId (status is automatically "pending")

5. **Record Decision** (if architectural choice needed)
   ```
   reflection_record_decision({
     chatId: "{{session_id}}",
     context: {
       userRequest: "{{user_requirements}}",
       availableInfo: "Found similar patterns: {{patterns_from_code_search}}.
                       Atomic Design level: {{atoms|molecules|organisms|templates|pages}}.
                       State management: {{zustand|react-query|local}}",
       uncertainty: "{{unknowns}} - e.g., proper TypeScript generics, API integration pattern"
     },
     decision: {
       action: "Create {{level}} component using {{approach}}",
       reasoning: "{{why_this_approach}} - follows existing patterns, type-safe, accessible",
       alternatives: ["{{other_options}} - monolithic component, different state solution"],
       confidence: {{0.7_to_0.95}}
     },
     predictions: {
       successProbability: {{0.8_to_0.95}},
       risks: ["Type complexity", "Performance with large datasets", "Browser compatibility"],
       timeEstimate: "{{1-4_hours}}"
     }
   })
   ```
   � Save decisionId (if recorded)

### Phase 3: Agent Task Creation (Sequential - prepare for parallel execution)

6. **Create Implementation Task**
   ```
   coordinator_create_agent_task({
     humanTaskId: "{{from_step_4}}",
     agentName: "ui-dev",
     role: "Implement React TypeScript UI feature with Atomic Design, strict typing, and accessibility",
     contextSummary: "
       WHY: {{business_context_and_user_need}}

       WHAT: {{specific_requirements}}
       - User story: {{acceptance_criteria}}
       - UI/UX: {{design_specs_or_description}}
       - Interactions: {{user_interactions}}
       - Data flow: {{props_down_events_up}}

       HOW: {{technical_approach}}
       - Atomic Design level: {{atoms|molecules|organisms|templates|pages}}
       - Similar components: {{file_paths_from_code_search}}
       - Component structure: {{composition_pattern}}
       - State management: {{zustand|react-query|useState}}
       - API integration: {{react_query_hooks_if_needed}}

       CONSTRAINTS:
       - L NEVER use 'any' type (use 'unknown' + type guards, or proper types)
       -  All props via TypeScript interfaces (not inline types)
       -  Event handlers typed: React.MouseEvent<HTMLButtonElement>, etc.
       -  Use class-variance-authority (cva) for variant props
       -  Forward refs for atoms: React.forwardRef<HTMLElement, Props>
       -  TailwindCSS for styling (no inline styles)
       -  Radix UI primitives when available (@radix-ui/react-*)
       -  Accessibility: ARIA labels, keyboard navigation, semantic HTML
       -  React 18 patterns: functional components, hooks
       -  Error boundaries for error handling
       -  Lazy loading for large components (React.lazy + Suspense)

       NAMING CONVENTIONS:
       - Component folder: PascalCase (e.g., StatusBadge/, SearchBar/, UserProfileCard/)
       - Component file: ComponentName.tsx
       - Test file: ComponentName.test.tsx
       - Props interface: ComponentNameProps
       - No kebab-case, no snake_case, no Hungarian notation

       PATTERNS:
       {{key_patterns_from_knowledge_base}}
       Example: Button atom (components/atoms/Button/Button.tsx)
       - Uses cva for variants
       - VariantProps<typeof variants> for type inference
       - React.forwardRef for ref forwarding
       - Radix Slot for asChild prop

       LESSONS: {{key_lessons_from_reflection}}

       TESTING: Unit tests with Vitest + React Testing Library (separate task)
     ",
     filesModified: [
       "hyperion-ui/src/components/{{level}}/{{ComponentName}}/{{ComponentName}}.tsx"
     ],
     qdrantCollections: ["hyperion_ui_architecture", "technical-knowledge"],
     todos: [
       {
         description: "Create {{ComponentName}} component interface and types",
         filePath: "hyperion-ui/src/components/{{level}}/{{ComponentName}}/{{ComponentName}}.tsx",
         functionName: "{{ComponentName}}",
         contextHint: "
           - Define interface {{ComponentName}}Props extends React.ComponentProps<'{{element}}'>
           - NO 'any' types - use specific types or 'unknown' with type guards
           - Use VariantProps<typeof variants> if using cva
           - Example: components/atoms/Button/Button.tsx lines 37-42
         "
       },
       {
         description: "Implement {{ComponentName}} component logic",
         filePath: "hyperion-ui/src/components/{{level}}/{{ComponentName}}/{{ComponentName}}.tsx",
         contextHint: "
           - Use React.forwardRef if atom-level component
           - Use cva for style variants (class-variance-authority)
           - Compose with Radix UI primitives if available
           - Handle edge cases: loading, error, empty states
           - Example pattern: components/atoms/Input/Input.tsx
         "
       },
       {
         description: "Add accessibility features",
         filePath: "hyperion-ui/src/components/{{level}}/{{ComponentName}}/{{ComponentName}}.tsx",
         contextHint: "
           - ARIA labels: aria-label, aria-labelledby, aria-describedby
           - Keyboard navigation: onKeyDown for Enter/Escape/Arrows
           - Focus management: autoFocus, tabIndex, focus-visible styles
           - Semantic HTML: <button> not <div>, <nav> not <div>
         "
       },
       {
         description: "Add TypeScript exports and documentation",
         filePath: "hyperion-ui/src/components/{{level}}/{{ComponentName}}/{{ComponentName}}.tsx",
         contextHint: "
           - Export component: export { {{ComponentName}} }
           - Export types: export type { {{ComponentName}}Props }
           - Add JSDoc comments for complex props
           - Example: /** @description Button variant for destructive actions */
         "
       }
     ]
   })
   ```
   � Save implementationTaskId

7. **Create Unit Test Task**
   ```
   coordinator_create_agent_task({
     humanTaskId: "{{from_step_4}}",
     agentName: "ui-dev",
     role: "Write comprehensive Vitest unit tests for React component",
     contextSummary: "
       Test coverage for: {{ComponentName}} component
       Implementation file: {{file_from_step_6}}

       Test requirements:
       - Vitest + React Testing Library (@testing-library/react)
       - Test rendering with different props/variants
       - Test user interactions (clicks, keyboard, hover)
       - Test edge cases: empty data, loading states, errors
       - Test accessibility: ARIA attributes, keyboard navigation
       - Mock external dependencies (API calls, Zustand stores)
       - Achieve >80% coverage

       Testing patterns:
       - describe() blocks for grouping
       - it() / test() for individual tests
       - render() from @testing-library/react
       - screen queries: getByRole, getByText, getByLabelText
       - userEvent for interactions
       - waitFor() for async behavior

       Dependencies: Wait for implementation task completion (step 6)
     ",
     filesModified: [
       "hyperion-ui/src/components/{{level}}/{{ComponentName}}/{{ComponentName}}.test.tsx"
     ],
     todos: [
       {
         description: "Write unit tests for {{ComponentName}} rendering",
         filePath: "hyperion-ui/src/components/{{level}}/{{ComponentName}}/{{ComponentName}}.test.tsx",
         contextHint: "
           - Test default render
           - Test all prop variants
           - Test with different data shapes
           - Use screen.getByRole for accessibility-first queries
         "
       },
       {
         description: "Write interaction tests for {{ComponentName}}",
         filePath: "hyperion-ui/src/components/{{level}}/{{ComponentName}}/{{ComponentName}}.test.tsx",
         contextHint: "
           - Test onClick, onChange, onSubmit handlers
           - Use userEvent.click(), userEvent.type()
           - Test keyboard interactions (Enter, Escape, Tab)
           - Assert callbacks called with correct arguments
         "
       },
       {
         description: "Write edge case tests",
         filePath: "hyperion-ui/src/components/{{level}}/{{ComponentName}}/{{ComponentName}}.test.tsx",
         contextHint: "
           - Test with null/undefined props
           - Test loading states
           - Test error states
           - Test empty data arrays
         "
       }
     ]
   })
   ```
   � Save unitTestTaskId

8. **Skip E2E Test Task** (OPTIONAL - Not required for this project)
   ```
   # E2E tests are optional for this project
   # Only unit tests are mandatory
   # e2eTestTaskId = null (not created)
   ```

### Phase 4: Implementation Execution (MANDATORY)

**⚠️ CHECKPOINT: DO NOT SKIP THIS PHASE**

**8a. Update Human Task Status to "in_progress"** (MANDATORY - before starting implementation)
   ```
   coordinator_update_task_status({
     taskId: "{{humanTaskId}}",
     status: "in_progress",
     notes: "Starting implementation. Agent tasks created."
   })
   ```

9. **Launch Implementation Agent** (MANDATORY - cannot proceed without this)

   **First, mark agent task as in_progress:**
   ```
   coordinator_update_task_status({
     taskId: "{{implementationTaskId}}",
     status: "in_progress",
     notes: "Agent started working on implementation."
   })
   ```

   **Then launch the agent:**
   ```
   Task({
     subagent_type: "ui-dev",
     description: "Implement {{ComponentName}} UI component",
     prompt: "
       Retrieve task: coordinator_get_agent_task('{{implementationTaskId}}')

       Read contextSummary (80% of needed info is there)
       Query qdrantCollections ONLY if you need deeper patterns (≤1 query total)
       Read ≤3 existing component files from code_index_search results

       Start coding ≤2 minutes after reading context

       As you work:
       - Update coordinator_update_todo_status for each TODO with file:line refs
       - Store key decisions via coordinator_upsert_knowledge
       - NO 'any' types (use 'unknown' + type guards or proper types)
       - Run 'npm lint' to check TypeScript errors
       - Test component locally with 'npm dev'

       When complete:
       - Verify all TODOs marked completed
       - Run 'npm type-check' (must pass)
       - Mark task complete via coordinator_update_task_status
     "
   })
   ```

10. **Monitor Implementation Progress** (MANDATORY - wait for completion)
    ```
    Poll: coordinator_list_agent_tasks({ humanTaskId: "{{from_step_4}}" })
    Check: status field for implementationTaskId

    If status === "blocked":
      coordinator_add_task_prompt_notes({
        agentTaskId: "{{implementationTaskId}}",
        promptNotes: "{{guidance_to_unblock}}"
      })

    MANDATORY: Wait for status === "completed"
    DO NOT PROCEED to Phase 5 until implementationTaskId status is "completed"
    ```

**✅ PHASE 4 COMPLETION CHECKLIST:**
- [ ] Implementation agent launched (step 9)
- [ ] Implementation task status = "completed"
- [ ] Component file created at correct Atomic Design level
- [ ] All TODOs marked completed with file:line references
- [ ] TypeScript compilation passes (npm type-check)
- [ ] Zero 'any' types confirmed

### Phase 5: Testing Execution (MANDATORY - NO EXCEPTIONS)

**⚠️ CRITICAL CHECKPOINT: TESTING IS NOT OPTIONAL**

**🚨 RULE: Feature is NOT complete without tests. Even 99% pass rate = TOTAL FAILURE.**

11. **Launch Unit Test Agent** (MANDATORY - after Phase 4 complete)

    **PRE-FLIGHT CHECK:**
    - ✅ Verify implementationTaskId status === "completed"
    - ✅ Verify component file exists
    - ✅ DO NOT SKIP - tests are MANDATORY

    **First, mark unit test task as in_progress:**
    ```
    coordinator_update_task_status({
      taskId: "{{unitTestTaskId}}",
      status: "in_progress",
      notes: "Agent started working on unit tests."
    })
    ```

    **Then launch the agent:**
    ```
    Task({
      subagent_type: "ui-dev",
      description: "Unit tests for {{ComponentName}}",
      prompt: "
        🚨 CRITICAL: This task is MANDATORY. Tests are NOT optional.

        Retrieve task: coordinator_get_agent_task('{{unitTestTaskId}}')

        Review implementation file: {{file_from_step_6}}
        Understand component props, variants, interactions

        Write comprehensive tests:
        - Rendering: all variants, edge cases
        - Interactions: clicks, keyboard, events
        - Accessibility: ARIA, focus, semantic HTML
        - Mock: Zustand stores, React Query hooks, external APIs

        Run tests:
        - npm test {{ComponentName}}.test.tsx
        - npm test:coverage (check >80% coverage)

        MANDATORY REQUIREMENTS:
        - 100% of tests MUST pass (not 99%, not 99.9% - 100%)
        - Coverage MUST be >80%
        - Fix ALL failing tests immediately
        - DO NOT mark task complete until ALL tests pass

        Update coordinator_update_todo_status per TODO
        Mark task complete ONLY when 100% tests pass with >80% coverage
      "
    })
    ```

12. **Monitor Unit Test Progress** (MANDATORY - wait for completion)
    ```
    Poll: coordinator_list_agent_tasks({ humanTaskId: "{{from_step_4}}" })
    Check: status field for unitTestTaskId

    MANDATORY: Wait for status === "completed"
    VERIFY: Test file exists at expected location
    VERIFY: All tests pass (100% pass rate)
    VERIFY: Coverage >80%

    If status === "blocked" OR tests failing:
      coordinator_add_task_prompt_notes({
        agentTaskId: "{{unitTestTaskId}}",
        promptNotes: "{{guidance_to_fix_tests}}"
      })
      DO NOT PROCEED until tests pass

    DO NOT PROCEED to step 13 until unitTestTaskId status is "completed"
    ```

**✅ UNIT TEST COMPLETION CHECKLIST:**
- [ ] Unit test agent launched (step 11)
- [ ] Unit test task status = "completed"
- [ ] Test file created: {{ComponentName}}.test.tsx
- [ ] All tests pass (100% pass rate - NO EXCEPTIONS)
- [ ] Coverage >80% confirmed
- [ ] All TODOs marked completed

**✅ PHASE 5 COMPLETION CHECKLIST:**
- [ ] Unit tests: 100% pass, >80% coverage ✅
- [ ] Test task marked "completed" ✅
- [ ] Test file exists and contains comprehensive test cases ✅

### Phase 6: Post-Flight & Quality Verification (Coordinator - You)

**⚠️ CRITICAL CHECKPOINT: Verify ALL phases completed before proceeding**

**PRE-FLIGHT CHECK FOR PHASE 6:**
- ✅ Phase 4 complete: Implementation task status = "completed"
- ✅ Phase 5 complete: Unit tests 100% pass
- ✅ Test file exists and contains tests
- ✅ DO NOT PROCEED if any phase is incomplete

15. **Update All TODO Statuses** (MANDATORY)
    - Mark each TODO as completed with notes (file:line refs)
    - Example: "Created Button interface at Button.tsx:37. Used VariantProps pattern from existing atoms."
    - VERIFY: All TODOs from implementationTaskId and unitTestTaskId marked "completed"

16. **Store Task Knowledge** (MANDATORY)
    ```
    coordinator_upsert_knowledge({
      collection: "task:hyperion://task/human/{{humanTaskId}}",
      text: "
        FEATURE: {{ComponentName}} component at {{atomic_design_level}}

        DECISION: {{architectural_choice}}
        - Reasoning: {{why_this_approach}}
        - Alternatives considered: {{rejected_options}}

        IMPLEMENTATION:
        - Component: {{file_path}}:{{line_number}}
        - Props interface: {{file_path}}:{{line_number}}
        - Key patterns used: {{cva_variants|forwardRef|radix_composition}}
        - State management: {{zustand|react_query|local_state}}

        TYPESCRIPT:
        - Zero 'any' types 
        - All props typed via interface 
        - Event handlers typed 
        - Generics used: {{if_applicable}}

        TESTING:
        - Unit tests: {{file_path}} (100% pass MANDATORY, {{coverage}}% coverage)
        - 🚨 Unit tests are NOT optional - feature incomplete without tests

        ACCESSIBILITY:
        - ARIA labels: {{attributes_added}}
        - Keyboard navigation: {{keys_supported}}
        - Semantic HTML: {{elements_used}}

        GOTCHAS:
        - {{edge_cases_or_tricky_parts}}
        - {{performance_considerations}}
        - {{browser_compatibility_notes}}

        HANDOFF:
        - Next maintainer should know: {{important_context}}
        - Future improvements: {{potential_enhancements}}
      ",
      taskId: "{{implementationTaskId}}",
      metadata: {
        type: "completion",
        files: ["{{all_modified_files}}"],
        componentLevel: "{{atoms|molecules|organisms|templates|pages}}",
        testsIncluded: true,
        unitTestsPassed: true,
        timestamp: "{{iso_timestamp}}"
      }
    })
    ```

17. **Record Outcome** (MANDATORY if decision was recorded in step 5)
    ```
    reflection_record_outcome({
      decisionId: "{{from_step_5}}",
      outcome: {
        success: {{true_if_all_tests_pass}},
        actualResult: "
          Component implemented at {{file_path}}.
          Unit tests: {{X}}/{{X}} passed ({{Y}}% coverage).
          TypeScript: Zero 'any' types.
          Accessibility: Full keyboard nav + ARIA.
        ",
        userFeedback: "{{user_reaction_if_available}}"
      },
      analysis: {
        predictionAccuracy: {{compare_predicted_vs_actual_time}},
        confidenceCalibration: "{{well-calibrated|overconfident|underconfident}}",
        missedSignals: [
          "{{what_was_not_anticipated}}"
        ]
      }
    })
    ```

18. **Extract Lesson** (if novel/reusable pattern discovered)
    ```
    reflection_extract_lesson({
      patternName: "{{descriptive_pattern_name}}",
      problem: "{{challenge_encountered}}",
      solution: "{{how_it_was_solved}}",
      antipattern: "{{what_not_to_do}}",
      context: "React TypeScript UI development, Atomic Design level: {{level}}",
      applicableTo: ["react", "typescript", "ui", "atomic-design", "{{domain}}"],
      confidence: {{0.7_to_1.0}}
    })
    ```

19. **Store Global Knowledge** (if pattern is project-wide reusable)
    ```
    knowledge_store({
      collectionName: "hyperion_ui_architecture",
      information: "{{concise_pattern_with_code_example}}",
      metadata: {
        tags: ["react", "typescript", "atomic-design"],
        pattern_type: "component",
        confidence: {{0.8_to_1.0}}
      }
    })
    ```

20. **Mark All Tasks Complete** (MANDATORY)
    ```
    coordinator_update_task_status({
      taskId: "{{implementationTaskId}}",
      status: "completed",
      notes: "Component implemented. All TODOs complete. TypeScript passes."
    })

    coordinator_update_task_status({
      taskId: "{{unitTestTaskId}}",
      status: "completed",
      notes: "Unit tests complete. 100% pass rate. {{coverage}}% coverage."
    })

    coordinator_update_task_status({
      taskId: "{{humanTaskId}}",
      status: "completed",
      notes: "All agent tasks complete. Knowledge stored. Lessons extracted. Quality verified."
    })
    ```

**✅ PHASE 6 COMPLETION CHECKLIST:**
- [ ] All TODOs marked "completed" (step 15)
- [ ] Task knowledge stored (step 16)
- [ ] Outcome recorded (step 17, if decision made)
- [ ] Lesson extracted (step 18, if applicable)
- [ ] Global knowledge stored (step 19, if applicable)
- [ ] All agent tasks marked "completed" (step 20)
- [ ] Human task marked "completed" (step 20)

21. **Trigger Quality Check** (MANDATORY - automatic for newly created files)

    **⚠️ CRITICAL: This step is MANDATORY. Quality check CANNOT be skipped.**

    **PRE-FLIGHT CHECK:**
    - ✅ All files created (component, unit tests)
    - ✅ All tasks marked "completed"
    - ✅ DO NOT SKIP - quality check is MANDATORY

    **Automatically run quality check on ALL newly created files:**

    ```
    Execute: /fix-all-issues files={{comma_separated_list_of_ALL_created_files}}

    Example (INCLUDE ALL FILES):
    /fix-all-issues files=hyperion-ui/src/components/atoms/StatusBadge/StatusBadge.tsx,hyperion-ui/src/components/atoms/StatusBadge/StatusBadge.test.tsx

    🚨 CRITICAL: Do NOT skip any files. Include:
    - Component file (.tsx)
    - Unit test file (.test.tsx)
    - Any other modified files (App.tsx, routes, etc.)
    ```

    This will:
    - Run **TARGETED_CHECK** mode (only new files, not entire codebase)
    - Verify TypeScript quality (zero `any` types)
    - Verify naming conventions (PascalCase, correct structure)
    - Verify Atomic Design compliance (correct level, hierarchy)
    - Verify accessibility (ARIA, keyboard nav, semantic HTML)
    - Verify testing coverage (tests exist, >80% coverage)
    - Verify best practices (TailwindCSS, cva, Radix, React Query)

    **Quality check will:**
    1. Scan newly created files (2-5 minutes)
    2. Report violations if found (CRITICAL, HIGH, MEDIUM, LOW)
    3. Auto-create fix tasks for CRITICAL/HIGH violations
    4. Launch ui-dev agent to fix violations
    5. Re-run quality check after fixes
    6. Report final status to user

    **🚨 MANDATORY: Wait for quality check to complete before final summary.**

    If violations found and fixed:
    - Include fix summary in final report
    - Update knowledge base with violation patterns

    If all checks passed:
    - Include "✅ Quality verified" in final summary

**✅ QUALITY CHECK COMPLETION CHECKLIST:**
- [ ] Quality check executed (step 21)
- [ ] ALL created files included in check
- [ ] Quality check status: {{ALL_PASSED | VIOLATIONS_FIXED | VIOLATIONS_REMAIN}}
- [ ] If violations: fixes applied and verified

---

## 🚨 MANDATORY WORKFLOW EXECUTION GUARANTEES

**CRITICAL RULES - NO EXCEPTIONS:**

1. **ALL PHASES MUST BE COMPLETED** - You CANNOT skip any phase:
   - ✅ Phase 1: Context Gathering (steps 1-3)
   - ✅ Phase 2: Task Planning (steps 4-5)
   - ✅ Phase 3: Agent Task Creation (steps 6-7)
   - ✅ Phase 4: Implementation Execution (steps 9-10)
   - ✅ Phase 5: Testing Execution (steps 11-12) - **MANDATORY, NOT OPTIONAL**
   - ✅ Phase 6: Post-Flight & Quality Verification (steps 15-21)

2. **UNIT TESTS ARE MANDATORY** - Feature is NOT complete without:
   - ✅ Unit tests (100% pass rate, >80% coverage)
   - ❌ NO EXCEPTIONS - Even 99% pass rate = TOTAL FAILURE

3. **QUALITY CHECK IS MANDATORY** - Step 21 CANNOT be skipped:
   - ✅ Run /fix-all-issues on ALL created files
   - ✅ Include component, unit tests, modified files
   - ✅ Wait for quality check completion before final summary

4. **CHECKPOINTS MUST BE VERIFIED** - Before proceeding to next phase:
   - ✅ Verify previous phase completion checklist
   - ✅ Verify all task statuses = "completed"
   - ✅ Verify test file exists and tests pass
   - ❌ DO NOT PROCEED if any checkpoint fails

5. **FAILURE MODES - What constitutes TOTAL FAILURE:**
   - ❌ Skipping any phase (especially Phase 5 - Testing)
   - ❌ Tests not written (no test file created)
   - ❌ Tests failing (any % less than 100%)
   - ❌ Quality check skipped (step 21 not executed)
   - ❌ Files missing (component exists but no tests)

---

## Execution Summary Template

**🚨 IMPORTANT: Do NOT provide this summary until ALL phases complete (including quality check)**

Present the user with a comprehensive summary including:

1. **Context Found**: Lessons, similar components, patterns
2. **Approach**: Decision reasoning, alternatives, confidence
3. **Tasks Created**: humanTaskId, implementationTaskId, unitTestTaskId
4. **Agents Launched**: Status of each agent (all must be "completed")
5. **Implementation Progress**: Files created, key details
6. **Test Results**:
   - Unit tests: {{X}}/{{X}} passed (100% MANDATORY), {{coverage}}% coverage
   - 🚨 If less than 100%: **FEATURE INCOMPLETE - TESTS FAILED**
7. **TypeScript Quality**: Zero `any`, all typed, checks passed
8. **Accessibility**: ARIA, keyboard, semantic HTML
9. **Knowledge Stored**: Task knowledge, lessons, patterns
10. **Quality Check Results**: (MANDATORY - from step 21)
    - Mode: TARGETED_CHECK
    - Files checked: {{list_of_ALL_new_files_including_tests}}
    - Status: {{ALL_PASSED | VIOLATIONS_FIXED | VIOLATIONS_REMAIN}}
    - Violations: {{count_by_severity}}
    - Fixes applied: {{fix_summary_if_any}}
11. **Completion Status**:
    - ✅ All phases complete
    - ✅ Unit tests pass (100%)
    - ✅ Quality check complete
    - ✅ Feature ready for review
12. **Next Steps**: Review, manual testing, deployment

**⚠️ If ANY phase incomplete or tests failing:**
- 🚨 **FEATURE NOT COMPLETE**
- ❌ Do NOT mark as "done"
- 🔧 Continue work until 100% complete

---

**User Requirements**: {{paste_requirements_here}}
