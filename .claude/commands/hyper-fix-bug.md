# Hyperion Bug Fix Workflow with Mandatory Regression Tests

Fix bugs following Hyperion's bug investigation process with MANDATORY regression test requirement.

## ⚠️ CRITICAL REQUIREMENT

**Every bug fix MUST include a regression test**. No exceptions.

- ✅ Test MUST fail before fix
- ✅ Test MUST pass after fix
- ✅ Test MUST be in same commit as fix
- ✅ Test MUST reproduce exact bug scenario
- ✅ Test MUST be named `Test<BugName>_Regression`

## Instructions

You are coordinating a bug fix task. Follow the Hyperion Golden Path with mandatory regression testing.

### Phase 1: Investigation & Context Gathering (Sequential)

1. **Query Past Bug Lessons**
   ```
   reflection_query_relevant_lessons({
     situation: "Fixing bug: {{bug_description}}. Need to understand similar bugs, common pitfalls, and proven solutions.",
     limit: 5
   })
   ```

2. **Search Similar Bug Patterns**
   ```
   knowledge_find({
     collectionName: "hyperion_bugs",
     query: "{{bug_symptom_keywords}}",
     retrieveMode: "chunk",
     limit: 5
   })
   ```

3. **Locate Bug Code**
   ```
   code_index_search({
     query: "{{component_description}} {{error_keywords}}",
     retrieve: "chunk-l",
     fileTypes: [".go", ".ts", ".tsx"],
     minScore: 0.6
   })
   ```
   → Identify files containing bug

4. **Analyze Logs** (if applicable)
   ```
   loki_query({
     query: '{job="kubernetes-system"} |~ "{{service}}" |= "{{error_message}}" [15m]',
     limit: 50
   })
   ```

5. **Read Bug Code**
   - Read top 2-3 files from code search
   - Analyze: bug location, root cause, affected code paths

### Phase 2: Bug Fix Planning (Sequential)

6. **Create Human Task**
   ```
   coordinator_create_human_task({
     prompt: "Fix bug: {{verbatim_bug_description}}"
   })
   ```
   → Save humanTaskId

7. **Record Bug Fix Decision**
   ```
   reflection_record_decision({
     chatId: "{{session_id}}",
     context: {
       userRequest: "Fix bug: {{description}}",
       availableInfo: "Root cause: {{root_cause}}, Affected files: {{files}}",
       uncertainty: "{{unknowns_about_fix}}"
     },
     decision: {
       action: "{{fix_approach}}",
       reasoning: "{{why_this_fix}}",
       alternatives: ["{{other_approaches}}"],
       confidence: {{0.0_to_1.0}}
     },
     predictions: {
       successProbability: {{0.0_to_1.0}},
       risks: ["{{regression_risks}}", "{{side_effects}}"],
       timeEstimate: "{{estimate}}"
     }
   })
   ```
   → Save decisionId

### Phase 3: Create Agent Tasks (Sequential)

8. **Create Bug Fix Task**

   **For Go Backend Bugs:**
   ```
   coordinator_create_agent_task({
     humanTaskId: "{{from_step_6}}",
     agentName: "go-dev",
     role: "Fix {{bug_name}} and add mandatory regression test",
     contextSummary: "
       WHY: {{bug_impact_on_users}}

       WHAT - Bug Details:
       - Symptom: {{what_user_sees}}
       - Root Cause: {{technical_explanation}}
       - Affected Code: {{file_paths}}
       - Error Pattern: {{error_message_if_any}}

       HOW - Fix Approach:
       {{fix_strategy_from_investigation}}

       MANDATORY - Regression Test:
       - Test MUST reproduce exact bug scenario
       - Test MUST fail before fix (verify with git stash)
       - Test MUST pass after fix
       - Test naming: Test{{BugName}}_Regression
       - Test comments: Bug, Root Cause, Fix
       - Example: {{reference_to_similar_regression_test}}

       CONSTRAINTS:
       - Minimal code changes (surgical fix only)
       - No refactoring outside bug scope
       - Preserve existing behavior for non-bug cases
       - Add explanatory comments at fix location

       PATTERNS (from lessons):
       {{relevant_bug_patterns_from_step_2}}

       FILES_TO_MODIFY:
       {{file_list}}
     ",
     filesModified: ["{{implementation_files}}", "{{test_files}}"],
     qdrantCollections: ["hyperion_bugs"],
     todos: [
       {
         description: "Reproduce bug with failing test",
         contextHint: "
           Write Test{{BugName}}_Regression that:
           1. Sets up exact bug conditions
           2. Triggers buggy code path
           3. Asserts incorrect behavior (will fail before fix)
           Run: go test -v -run Test{{BugName}}_Regression
           Should FAIL initially (proving test catches bug)
         "
       },
       {
         description: "Implement minimal fix for {{bug_name}}",
         filePath: "{{bug_file_path}}",
         functionName: "{{buggy_function}}",
         contextHint: "
           Fix approach: {{specific_fix_strategy}}
           Change only: {{minimal_change_description}}
           Add comment: // FIX: {{bug_name}} - {{one_line_explanation}}
         "
       },
       {
         description: "Verify regression test passes after fix",
         contextHint: "
           Run: go test -v -run Test{{BugName}}_Regression
           Should PASS now (proving fix works)
           Verify test would catch bug: git stash, run test (should fail), git stash pop
         "
       },
       {
         description: "Run full test suite",
         contextHint: "
           Run: make test SERVICE={{service}}
           Ensure no regressions in other tests
           Check coverage includes bug code path
         "
       }
     ]
   })
   ```

   **For React/TypeScript Bugs:**
   ```
   coordinator_create_agent_task({
     humanTaskId: "{{from_step_6}}",
     agentName: "ui-dev",
     role: "Fix {{bug_name}} UI bug and add regression test",
     contextSummary: "
       WHY: {{user_experience_impact}}

       WHAT - Bug Details:
       - Symptom: {{UI_behavior_issue}}
       - Root Cause: {{state_or_render_issue}}
       - Affected Components: {{component_paths}}

       HOW - Fix Approach:
       {{fix_strategy}}

       MANDATORY - Regression Test:
       - Use React Testing Library
       - Test MUST reproduce bug scenario
       - Test naming: describe('{{BugName}} Regression', ...)
       - Verify fix with user interactions

       CONSTRAINTS:
       - Minimal component changes
       - No unnecessary refactoring
       - Preserve accessibility

       FILES_TO_MODIFY:
       {{component_and_test_files}}
     ",
     filesModified: ["{{component_files}}", "{{test_files}}"],
     todos: [...]
   })
   ```
   → Save bugFixTaskId

### Phase 4: Execute Bug Fix (Agent)

9. **Launch Bug Fix Agent**
   ```
   Task({
     subagent_type: "go-dev" | "ui-dev",
     description: "Fix {{bug_name}} with test",
     prompt: "
       Retrieve task: coordinator_get_agent_task('{{bugFixTaskId}}')

       CRITICAL: Follow MANDATORY regression test workflow

       Step 1: Write failing regression test
       - Test MUST reproduce exact bug conditions
       - Test MUST fail initially (proving it catches bug)
       - Name: Test{{BugName}}_Regression
       - Include header comments: Bug, Root Cause, Fix

       Step 2: Implement minimal fix
       - Change only code necessary to fix bug
       - Add explanatory comment at fix location
       - No refactoring outside bug scope

       Step 3: Verify regression test passes
       - Run regression test (should PASS now)
       - Verify test catches bug: git stash fix, run test (FAIL), restore

       Step 4: Run full test suite
       - Ensure no regressions: make test
       - Verify coverage: go test -cover

       Step 5: Update TODOs
       - coordinator_update_todo_status for each step
       - Include test verification notes

       Step 6: Store fix knowledge
       - coordinator_upsert_knowledge with bug pattern, fix, test details
     "
   })
   ```

10. **Monitor Progress**
    - Poll coordinator_list_agent_tasks({ humanTaskId })
    - Check regression test status
    - Verify fix completion

### Phase 5: Verification & Deployment

11. **Verify Fix Locally**
    ```bash
    # Verify regression test exists and passes
    make test SERVICE={{service}}

    # Verify test would have caught bug
    git stash push -m "Temporarily remove fix"
    make test SERVICE={{service}}  # Should FAIL
    git stash pop
    make test SERVICE={{service}}  # Should PASS

    # Deploy to dev
    make dev-deploy-{{service}}

    # Test in dev environment
    # Reproduce original bug steps
    # Verify bug no longer occurs
    ```

12. **Check Logs**
    ```
    loki_query({
      query: '{job="kubernetes-system"} |~ "{{service}}" |= "ERROR" [5m]',
      limit: 20
    })
    ```
    → Should NOT show original error

### Phase 6: Post-Flight Documentation (Coordinator - You)

13. **Update All TODOs**
    ```
    coordinator_update_todo_status({
      agentTaskId,
      todoId,
      status: "completed",
      notes: "
        Regression test: {{test_file}}:{{line}}
        Test verified: FAIL before fix, PASS after fix
        Fix applied: {{bug_file}}:{{line}}
        Full suite: {{pass_count}} tests pass
      "
    })
    ```

14. **Store Bug Pattern Knowledge**
    ```
    knowledge_store({
      collectionName: "hyperion_bugs",
      information: "
        **Bug Pattern**: {{concise_pattern_name}}

        **Symptom**: {{what_user_sees}}

        **Root Cause**: {{technical_explanation}}

        **Detection**: {{how_to_identify}}

        **Fix**: {{code_change_summary}}
        ```{{language}}
        {{code_example}}
        ```

        **Regression Test**: {{test_file}}::Test{{BugName}}_Regression

        **Test Verification**:
        - Test fails before fix: ✓ (verified with git stash)
        - Test passes after fix: ✓
        - Test location: {{test_file}}:{{line}}

        **Prevention**: {{how_to_avoid}}

        **References**:
        - Bug fix: {{bug_file}}:{{line}}
        - Regression test: {{test_file}}:{{line}}
        - Logs: {{loki_query_if_applicable}}
      ",
      metadata: {
        tags: ["{{service}}", "{{component}}", "bug-fix", "regression-test"],
        severity: "{{P0|P1|P2|P3}}",
        fixed_date: "{{date}}",
        has_regression_test: true,
        test_file: "{{test_file_path}}"
      }
    })
    ```

15. **Record Outcome**
    ```
    reflection_record_outcome({
      decisionId: "{{from_step_7}}",
      outcome: {
        success: {{true_if_bug_fixed}},
        actualResult: "
          Bug fixed: {{bug_description}}
          Regression test: {{test_file}}:{{line}}
          Test verification: FAIL before, PASS after
          Deployment: {{environment}}
          User verification: {{user_feedback}}
        ",
        userFeedback: "{{user_confirmation}}"
      },
      analysis: {
        predictionAccuracy: {{actual_vs_predicted}},
        confidenceCalibration: "{{well-calibrated|overconfident|underconfident}}",
        missedSignals: ["{{what_was_unexpected}}"]
      }
    })
    ```

16. **Extract Lesson** (if novel pattern)
    ```
    reflection_extract_lesson({
      patternName: "{{bug_pattern_name}}",
      problem: "{{bug_that_occurred}}",
      solution: "{{how_it_was_fixed}}",
      antipattern: "{{what_caused_bug}}",
      context: "{{when_this_applies}}",
      applicableTo: ["{{service}}", "{{technology}}", "bug-fix"],
      confidence: {{0.0_to_1.0}}
    })
    ```

17. **Mark Tasks Complete**
    ```
    coordinator_update_task_status({
      taskId: bugFixTaskId,
      status: "completed",
      notes: "
        Bug fixed: {{description}}
        Regression test: ✅ {{test_file}}
        Test verified: ✅ FAIL before, PASS after
        Deployed to: dev
        Knowledge stored: hyperion_bugs collection
      "
    })

    coordinator_update_task_status({
      taskId: humanTaskId,
      status: "completed",
      notes: "Bug fix complete with regression test"
    })
    ```

### Pre-Flight Checklist
- [ ] Bug lessons queried
- [ ] Similar bugs researched
- [ ] Bug code located (code_index_search)
- [ ] Root cause identified
- [ ] Human task created
- [ ] Bug fix decision recorded
- [ ] Bug fix task created with REGRESSION TEST requirement
- [ ] Agent launched

### Post-Flight Checklist
- [ ] **MANDATORY: Regression test exists**
- [ ] **MANDATORY: Test verified (FAIL before, PASS after)**
- [ ] **MANDATORY: Test in same commit as fix**
- [ ] All TODOs updated
- [ ] Bug pattern stored in knowledge base
- [ ] Outcome recorded
- [ ] Lesson extracted (if novel)
- [ ] All tasks marked complete
- [ ] Fix deployed to dev
- [ ] Bug verified fixed in dev

## Regression Test Examples

### Example 1: NilObjectID Security Violation

```go
// TestNilObjectIDSecurityViolation_Regression verifies fix for NilObjectID triggering security violation
// Bug: Optional ObjectID fields without omitempty caused "SECURITY VIOLATION: Cross-company ObjectID reference"
// Root Cause: Zero-value ObjectIDs (000000000000000000000000) passed to SecureMongoClient validation
// Fix: Added omitempty BSON tag to optional ObjectID fields
func TestNilObjectIDSecurityViolation_Regression(t *testing.T) {
    // Arrange: Create entity with optional ProcessID field set to NilObjectID
    agent := &Agent{
        ID:        primitive.NewObjectID(),
        Name:      "Test Agent",
        ProcessID: primitive.NilObjectID, // This should NOT trigger security violation
    }

    // Act: Marshal to BSON (would have included process_id: NilObjectID before fix)
    data, err := bson.Marshal(agent)
    require.NoError(t, err)

    // Assert: NilObjectID should be omitted from BSON
    var doc bson.M
    err = bson.Unmarshal(data, &doc)
    require.NoError(t, err)

    // With fix: process_id should NOT be in BSON document
    _, hasProcessID := doc["process_id"]
    assert.False(t, hasProcessID, "NilObjectID should be omitted with omitempty tag")
}
```

### Example 2: React State Update Bug

```typescript
// Regression test for message hiding bug
// Bug: Messages with type='tool_execution' hidden from DOM in non-debug mode
// Root Cause: Component returned null, removing element from DOM entirely
// Fix: Always render element, control visibility with CSS classes
describe('MessageHiding Regression', () => {
  it('should keep tool execution messages in DOM but hidden when debug mode off', () => {
    // Arrange: Tool execution message with debug mode OFF
    const message = {
      id: '1',
      type: 'tool_execution',
      content: 'Tool output'
    };

    // Act: Render component with debug mode off
    render(<ChatMessage message={message} debugMode={false} />);

    // Assert: Element exists in DOM but is hidden
    const element = screen.getByTestId('message-1');
    expect(element).toBeInTheDocument(); // Fix: Element exists
    expect(element).toHaveClass('hidden'); // Fix: Hidden via CSS

    // Before fix: element would be null (returned null from component)
  });
});
```

### Example 3: Authentication Propagation Bug

```go
// TestMCPAuthenticationPropagation_Regression verifies fix for identity not propagating to MCP handlers
// Bug: Write operations failed with "authentication required" despite valid JWT
// Root Cause: Identity not extracted from HTTP context and passed to MCP tool handler context
// Fix: Extract identity in middleware and inject into handler context
func TestMCPAuthenticationPropagation_Regression(t *testing.T) {
    // Arrange: Create HTTP request with valid JWT
    token := generateTestJWT(t, "user-123", "company-123")
    req := httptest.NewRequest("POST", "/mcp", nil)
    req.Header.Set("Authorization", "Bearer "+token)

    // Act: Process through MCP middleware and handler
    rr := httptest.NewRecorder()
    handler := mcpJWT.HTTPMiddleware(mcpHandler)
    handler.ServeHTTP(rr, req)

    // Assert: Handler should receive identity in context
    assert.Equal(t, 200, rr.Code, "Should succeed with authentication")

    // Verify identity was available to handler (mock would have recorded this)
    assert.True(t, mockHandler.ReceivedIdentity, "Handler should receive identity")
    assert.Equal(t, "user-123", mockHandler.Identity.ID)

    // Before fix: mockHandler.ReceivedIdentity would be false
}
```

## Common Regression Test Patterns

### Pattern 1: Error That Should Not Occur
```go
func TestBugName_Regression(t *testing.T) {
    // Setup exact bug conditions
    result, err := functionThatHadBug(bugTriggeringInput)

    // Should NOT error (but did before fix)
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

### Pattern 2: Incorrect Value That Should Be Correct
```go
func TestBugName_Regression(t *testing.T) {
    // Setup exact bug conditions
    result := functionThatReturnedWrongValue(input)

    // Should return correct value (returned wrong value before fix)
    assert.Equal(t, expectedCorrectValue, result)
}
```

### Pattern 3: Missing Data That Should Exist
```go
func TestBugName_Regression(t *testing.T) {
    // Setup exact bug conditions
    data := queryThatMissedData(query)

    // Should include expected data (was missing before fix)
    assert.Len(t, data, expectedCount)
    assert.Contains(t, data, expectedItem)
}
```

### Pattern 4: Security Violation That Should Not Trigger
```go
func TestBugName_Regression(t *testing.T) {
    // Setup exact conditions that triggered violation
    err := operationThatTriggeredSecurityCheck(input)

    // Should NOT trigger security violation (did before fix)
    assert.NoError(t, err)
    assert.NotContains(t, logs, "SECURITY VIOLATION")
}
```

## Verification Commands

```bash
# Verify regression test exists
find . -name "*_test.go" -exec grep -l "Test.*_Regression" {} \;

# Run only regression tests
go test ./... -run "_Regression" -v

# Verify test fails without fix (dangerous - use with care)
git stash push -m "Temporarily stash fix to verify test"
go test ./... -run "Test{{BugName}}_Regression" -v  # Should FAIL
git stash pop

# Run full test suite
make test SERVICE={{service}}

# Check coverage includes bug code path
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep {{buggy_file}}
```

---

**Describe the bug to fix**: {{paste_bug_description_error_logs_or_reproduction_steps}}
