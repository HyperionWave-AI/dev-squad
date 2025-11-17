# Subchat & Main Chat Backend Re-implementation Brief

**Date:** November 5, 2025
**Target Branch:** `megha/individual-work`
**Source Branch:** `megha/knowledge-browser`
**Scope:** Backend only (Go files)

---

## 📋 EXECUTIVE SUMMARY

Re-implement the chat and subchat backend from `megha/knowledge-browser` into `megha/individual-work` to bring in 25 critical missing features while preserving the confirmation step (Step 3) already implemented in individual-work.

**Context:**
- **Subchat** = Subchat created by `execute_subagent` tool call from main chat
- **Main Chat** = Primary chat session where user interacts
- Current branch has confirmation step; source branch has critical infrastructure improvements

---

## 🎯 OBJECTIVES

### Primary Goals:
1. ✅ Bring all 10 critical backend features from knowledge-browser
2. ✅ Preserve the confirmation step (Step 3) already in individual-work
3. ✅ Ensure subchats (via execute_subagent) work with all improvements
4. ✅ Maintain production stability and data integrity

### Success Criteria:
- [ ] User interruption works in both main chat and subchats
- [ ] MongoDB transactions protect data integrity
- [ ] Async file indexing prevents UI blocking
- [ ] Workflow enforcement prevents infinite loops
- [ ] Multi-tenant fix (no hardcoded "dev-company")
- [ ] Panic recovery prevents crashes
- [ ] Confirmation step (Step 3) still works
- [ ] All existing tests pass

---

## 📁 FILES TO RE-IMPLEMENT

### 🔴 CRITICAL FILES (Must Re-implement)

#### 1. `hyper/internal/handlers/chat_websocket.go` (1520 lines in KB)
**Current Status:** 1708 lines in IW (has confirmation step)
**Action:** Selective merge - take KB version + preserve Step 3 from IW

**Key Changes to Implement:**
- ✅ Prioritized interrupt handling (commit b64ffa2)
  - Two-select statement event loop
  - Non-blocking interrupt priority check
  - Process interrupts before AI events

- ✅ Async file indexing integration (commit 2a9ee79)
  - Non-blocking file indexing
  - Immediate interrupt processing during indexing

- ✅ Channel lifecycle management (commit ad4c011)
  - Proper channel cleanup
  - Prevents memory leaks
  - Graceful WebSocket shutdown

- ✅ User interruption in subchats (commits e43dc47, 61d61f8)
  - Interrupt handling for execute_subagent sessions
  - Database as source of truth
  - System prompt preservation
  - Graceful stream drain
  - Seamless resume

- ✅ Intelligent interrupt categorization (commit 7082676)
  - Analyze user intent during execution
  - Categorize interrupt types

- ✅ Subchat interrupt fix (commit 90568ec)
  - Prevent cascading task creation on interrupt
  - Critical bug fix

- ✅ Multi-tenant fix (commit d737d48)
  - Remove hardcoded "dev-company"
  - Use actual companyID from context

- ✅ Prometheus metrics (commit e8fd95f)
  - Add metrics collection
  - Track chat performance

**⚠️ PRESERVE FROM IW:**
- Step 3: Analyze & Present Implementation Options (lines 66-137 in IW)
- This is the confirmation step we just implemented

**Implementation Strategy:**
```
1. Copy KB version to a temp file
2. Extract Step 3 from IW version (lines 66-137)
3. Insert Step 3 into KB version after Step 2
4. Renumber subsequent steps (3→4, 4→5, 5→6)
5. Update step references throughout
6. Review and test
```

---

#### 2. `hyper/internal/services/chat_service.go` (558 lines)
**Current Status:** Identical structure in both branches
**Action:** Take KB version entirely

**Key Changes to Implement:**
- ✅ MongoDB transactions (commit d86fabf)
  - Wrap multi-step operations in transactions
  - Rollback on errors
  - Ensure data consistency

- ✅ Message content size validation (commit 53897c1)
  - Max 1MB message size
  - Prevents abuse and memory issues

- ✅ SessionId validation (commit 5856f49)
  - Validate after subchat creation
  - Descriptive error messages
  - Prevent silent failures

**Implementation Strategy:**
```
1. Read KB version
2. Replace IW version entirely
3. Verify imports and dependencies
4. Test transaction logic
```

---

#### 3. `hyper/internal/handlers/subchat_handler.go` (314 lines)
**Current Status:** Identical structure in both branches
**Action:** Take KB version entirely

**Key Changes to Implement:**
- ✅ Subchat deletion API (commit 76edca8)
  - Soft delete pattern
  - DELETE /api/v1/subchats/:id endpoint

- ✅ Rate limiting for subchat creation (commit 2747e6e)
  - Prevent excessive subchat creation
  - Resource protection

- ✅ Multi-tenant fix (commit d737d48)
  - Remove hardcoded "dev-company"
  - Use actual companyID

**Implementation Strategy:**
```
1. Read KB version
2. Replace IW version entirely
3. Verify API endpoints
4. Test deletion and rate limiting
```

---

### 🟡 IMPORTANT FILES (Should Re-implement)

#### 4. `hyper/internal/models/chat.go`
**Action:** Take KB version if different

**Key Changes:**
- Message model updates
- Subchat model updates
- Any new fields or methods

---

#### 5. `hyper/internal/mcp/storage/subchat_storage.go`
**Action:** Take KB version if different

**Key Changes:**
- Subchat storage methods
- Deletion support
- Query optimizations

---

### 🔵 SUPPORTING FILES (Check if needed)

#### 6. File Watcher / Indexing (if exists)
**Location:** `hyper/internal/services/` or similar
**Action:** Implement async file indexing

**Key Changes:**
- Non-blocking file indexing
- Async worker pool
- Graceful shutdown

---

#### 7. Panic Recovery Wrapper
**Location:** `hyper/internal/middleware/` or similar
**Action:** Add panic recovery

**Key Changes:**
- Catch panics in goroutines
- Cleanup guarantee
- Log and recover

---

## 🔄 IMPLEMENTATION SEQUENCE

### Phase 1: Foundation (2-3 hours)
**Order matters - follow this sequence!**

1. **Panic Recovery & Infrastructure** (30 mins)
   - File: Middleware or utils
   - Commit: 95b0e98
   - Why first: Protects everything else
   - Test: Trigger panic, verify recovery

2. **MongoDB Transactions** (45 mins)
   - File: `chat_service.go`
   - Commit: d86fabf
   - Why second: Data integrity foundation
   - Test: Concurrent operations, verify consistency

3. **Channel Lifecycle Management** (30 mins)
   - File: `chat_websocket.go`
   - Commit: ad4c011
   - Why third: Prevents memory leaks
   - Test: Long-running sessions, check memory

4. **Multi-tenant Fix** (15 mins)
   - Files: `chat_websocket.go`, `subchat_handler.go`
   - Commit: d737d48
   - Why fourth: Critical correctness
   - Test: Different company IDs

---

### Phase 2: Performance & Responsiveness (2 hours)

5. **Async File Indexing** (60 mins)
   - File: File watcher service + `chat_websocket.go`
   - Commit: 2a9ee79
   - Why: UI responsiveness during indexing
   - Test: Trigger indexing, verify non-blocking

6. **Prioritized Interrupt Handling** (45 mins)
   - File: `chat_websocket.go`
   - Commit: b64ffa2
   - Why: User interrupts processed immediately
   - Test: Send interrupt during execution
   - Implementation:
     ```go
     // Event loop structure:
     for {
         // First select: Priority check for interrupts (non-blocking)
         select {
         case interrupt := <-userInterruptChan:
             // Handle interrupt immediately
         default:
             // Continue to normal processing
         }

         // Second select: Normal AI event processing
         select {
         case event := <-aiStreamChan:
             // Process AI events
         case interrupt := <-userInterruptChan:
             // Handle interrupt
         }
     }
     ```

7. **Message Size Validation** (15 mins)
   - File: `chat_service.go`
   - Commit: 53897c1
   - Why: Prevent abuse
   - Test: Send large message, verify rejection

---

### Phase 3: User Experience (2-3 hours)

8. **User Interruption in Subchats** (90 mins)
   - File: `chat_websocket.go`
   - Commits: e43dc47, 61d61f8
   - Why: Critical UX feature
   - Test: Interrupt subchat (from execute_subagent), verify stop/resume
   - Implementation:
     ```go
     // Key components:
     // 1. Interrupt channel monitoring
     // 2. Database-backed message persistence
     // 3. System prompt preservation
     // 4. Graceful stream cleanup
     // 5. Resume from DB state
     ```

9. **Intelligent Interrupt Categorization** (45 mins)
   - File: `chat_websocket.go`
   - Commit: 7082676
   - Why: Better interrupt handling
   - Test: Different interrupt types

10. **Subchat Interrupt Fix** (30 mins)
    - File: `chat_websocket.go`
    - Commit: 90568ec
    - Why: Prevents infinite cascading
    - Test: Interrupt, verify no new subchats created

---

### Phase 4: API & Stability (1-2 hours)

11. **Subchat Deletion API** (30 mins)
    - File: `subchat_handler.go`
    - Commit: 76edca8
    - Why: Data cleanup capability
    - Test: Delete subchat, verify soft delete

12. **Rate Limiting** (30 mins)
    - File: `subchat_handler.go`
    - Commit: 2747e6e
    - Why: Resource protection
    - Test: Rapid subchat creation, verify limits

13. **SessionId Validation** (20 mins)
    - File: `chat_service.go`
    - Commit: 5856f49
    - Why: Error handling
    - Test: Missing sessionId, verify error

14. **Prometheus Metrics** (30 mins)
    - File: `chat_websocket.go` + metrics setup
    - Commit: e8fd95f
    - Why: Observability
    - Test: Check metrics endpoint

---

### Phase 5: Integration & Testing (2 hours)

15. **Preserve Confirmation Step** (30 mins)
    - File: `chat_websocket.go`
    - Action: Merge Step 3 from IW into KB version
    - Test: Verify options presentation works

16. **Comprehensive Testing** (90 mins)
    - Test all 14 features
    - Test execute_subagent flow end-to-end
    - Test interrupt during subagent execution
    - Test multiple concurrent sessions
    - Load testing

---

## 🧪 TESTING STRATEGY

### Test 1: Execute Subagent Basic Flow
```
1. User: "Create a new feature X"
2. Main chat presents options (Step 3)
3. User selects option
4. Main chat calls execute_subagent
5. Subchat is created via execute_subagent
6. Subchat executes the task
7. Subchat completes
8. Main chat receives result
✅ PASS: Complete flow works
```

### Test 2: Interrupt Subagent
```
1. User: "Create a new feature X"
2. Main chat calls execute_subagent
3. Subchat starts executing
4. User sends "stop" message
5. Subchat gracefully stops
6. User sends "continue with approach Y instead"
7. Subchat resumes with new approach
✅ PASS: Interrupt and resume works
```

### Test 3: Multi-tenant Isolation
```
1. Create session with companyID="company-A"
2. Main chat calls execute_subagent
3. Verify subchat uses companyID="company-A"
4. Verify no hardcoded "dev-company"
✅ PASS: Multi-tenant works
```

### Test 4: Cascading Interrupt Fix
```
1. User: "Create feature X"
2. Main chat calls execute_subagent
3. During subchat execution, user interrupts
4. Verify subchat stops
5. Verify no new subchats created
6. Verify no new tasks created
✅ PASS: No cascading
```

### Test 5: Data Integrity
```
1. Trigger concurrent chat operations
2. Simulate error mid-operation
3. Verify database consistency
4. Verify transaction rollback
✅ PASS: Transactions work
```

### Test 6: UI Responsiveness
```
1. Trigger file indexing (3-4 seconds)
2. Send message during indexing
3. Verify UI responds immediately
4. Verify message processed
✅ PASS: Async indexing works
```

### Test 7: Panic Recovery
```
1. Trigger panic in goroutine
2. Verify server doesn't crash
3. Verify cleanup happens
4. Verify error logged
✅ PASS: Panic recovery works
```

### Test 8: Message Size Limit
```
1. Send message > 1MB
2. Verify rejection
3. Verify error message
✅ PASS: Size validation works
```

### Test 9: Rate Limiting
```
1. Create 10 subchats rapidly
2. Verify rate limit kicks in
3. Verify error message
✅ PASS: Rate limiting works
```

### Test 10: Subchat Deletion
```
1. Create subchat via execute_subagent
2. Call DELETE /api/v1/subchats/:id
3. Verify soft delete
4. Verify data preserved
✅ PASS: Deletion works
```

---

## ⚠️ CRITICAL CONSIDERATIONS

### 1. Preserve Confirmation Step
**Issue:** IW has Step 3 (confirmation), KB doesn't
**Solution:** Merge Step 3 from IW into KB version
**Location:** Lines 66-137 in IW `chat_websocket.go`
**Steps:**
```
1. Extract Step 3 from IW
2. Insert after Step 2 in KB version
3. Renumber steps: 3→4, 4→5, 5→6
4. Update all step references
5. Update workflow title: "5-STEP" → "6-STEP"
```

### 2. Database Schema
**Issue:** May need new fields for interrupts/metrics
**Solution:** Check models, add migrations if needed
**Files to check:**
- `hyper/internal/models/chat.go`
- Migration files

### 3. Dependencies
**Issue:** May need new Go packages
**Solution:** Check imports, run `go mod tidy`
**Likely new packages:**
- Prometheus client (if metrics added)
- Additional MongoDB transaction helpers

### 4. Configuration
**Issue:** May need new env vars
**Solution:** Check `.env.hyper`, update if needed
**Possible new vars:**
- RATE_LIMIT_SUBCHAT_CREATION
- PROMETHEUS_METRICS_ENABLED
- MAX_MESSAGE_SIZE_BYTES

### 5. Backward Compatibility
**Issue:** Existing subchats in DB may not have new fields
**Solution:** Handle nil/missing fields gracefully
**Strategy:**
```go
// Example: Handle missing sessionId
if subchat.SessionID == nil {
    // Handle legacy subchat
    // Maybe create session on-demand
}
```

---

## 📊 RISK ASSESSMENT

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Breaking existing subchats | Medium | High | Test with existing DB data |
| Merge conflicts | Low | Medium | Selective file replacement |
| Lost confirmation step | Low | High | Careful merge of Step 3 |
| Performance regression | Low | Medium | Load testing |
| Data corruption | Low | Critical | Test transactions thoroughly |
| Memory leaks | Low | High | Monitor memory in long tests |
| Race conditions | Medium | High | Test concurrent operations |

---

## 🎯 SUCCESS METRICS

### Functional Metrics:
- ✅ All 10 critical features working
- ✅ Subchats (via execute_subagent) work end-to-end
- ✅ User interruption works in main chat and subchats
- ✅ Confirmation step preserved and working
- ✅ All existing tests pass
- ✅ No regressions in existing functionality

### Performance Metrics:
- ✅ UI responsive during file indexing (<100ms delay)
- ✅ Interrupts processed immediately (<50ms)
- ✅ No memory leaks (stable memory over 1-hour test)
- ✅ No goroutine leaks
- ✅ Database operations consistent (transactions)

### Stability Metrics:
- ✅ No server crashes on panic
- ✅ Graceful handling of large messages
- ✅ Rate limiting prevents abuse
- ✅ Multi-tenant isolation works

---

## 📝 IMPLEMENTATION CHECKLIST

### Pre-Implementation:
- [ ] Backup current `megha/individual-work` branch
- [ ] Create new branch: `megha/subchat-backend-reimpl`
- [ ] Document current behavior (baseline)
- [ ] Review all 14 feature commits in KB

### Phase 1: Foundation
- [ ] Implement panic recovery
- [ ] Implement MongoDB transactions
- [ ] Implement channel lifecycle management
- [ ] Fix multi-tenant hardcoded value
- [ ] Test Phase 1 features

### Phase 2: Performance
- [ ] Implement async file indexing
- [ ] Implement prioritized interrupt handling
- [ ] Implement message size validation
- [ ] Test Phase 2 features

### Phase 3: User Experience
- [ ] Implement user interruption in subchats
- [ ] Implement intelligent interrupt categorization
- [ ] Implement subchat interrupt fix
- [ ] Test Phase 3 features

### Phase 4: API & Stability
- [ ] Implement subchat deletion API
- [ ] Implement rate limiting
- [ ] Implement sessionId validation
- [ ] Implement Prometheus metrics
- [ ] Test Phase 4 features

### Phase 5: Integration
- [ ] Merge confirmation step (Step 3) from IW
- [ ] Renumber workflow steps
- [ ] Update all step references
- [ ] Comprehensive testing

### Post-Implementation:
- [ ] All 10 tests passing
- [ ] Performance benchmarks met
- [ ] No regressions found
- [ ] Code review completed
- [ ] Documentation updated
- [ ] Merge to `megha/individual-work`

---

## 📚 REFERENCE COMMITS

### Critical Commits (from megha/knowledge-browser):
1. **b64ffa2** - Prioritized interrupt handling
2. **2a9ee79** - Async file indexing
3. **ad4c011** - Channel lifecycle management
4. **d86fabf** - MongoDB transactions
5. **e43dc47, 61d61f8** - User interruption in subchats
6. **7082676** - Intelligent interrupt categorization
7. **90568ec** - Subchat interrupt fix
8. **d737d48** - Multi-tenant fix
9. **95b0e98** - Panic recovery
10. **76edca8** - Subchat deletion API
11. **2747e6e** - Rate limiting
12. **5856f49** - SessionId validation
13. **53897c1** - Message size validation
14. **e8fd95f** - Prometheus metrics

---

## 🚀 ESTIMATED TIMELINE

| Phase | Duration | Files | Complexity |
|-------|----------|-------|------------|
| Phase 1: Foundation | 2-3 hours | 3 files | Medium |
| Phase 2: Performance | 2 hours | 2 files | High |
| Phase 3: User Experience | 2-3 hours | 1 file | High |
| Phase 4: API & Stability | 1-2 hours | 2 files | Medium |
| Phase 5: Integration & Testing | 2 hours | All files | Medium |

**Total Estimated Time:** 9-12 hours

**Recommended Approach:** Split across 2-3 days with thorough testing between phases

---

## 🎓 KEY LEARNINGS TO APPLY

### From Analysis:
1. **Interrupts are critical** - Users need control over long-running tasks
2. **Async is essential** - Blocking operations destroy UX
3. **Transactions protect data** - Multi-step operations need atomicity
4. **Multi-tenancy must be enforced** - Hardcoded values break production
5. **Panic recovery is mandatory** - Server stability is non-negotiable

### Best Practices:
1. Test each phase independently before moving on
2. Keep confirmation step (Step 3) working throughout
3. Test with real database data, not just new records
4. Monitor memory during long tests
5. Test concurrent operations explicitly
6. Document any new configuration needed

---

## 📞 SUPPORT & RESOURCES

### Documentation:
- `CHAT_COMPARISON_REPORT.md` - Full feature analysis
- `CHAT_BEHAVIOR_IMPROVEMENT_PHASE1.md` - Recent prompt changes
- `docs/subchat-realtime-visibility-spec.md` - Subchat design spec

### Testing Tools:
- WebSocket test client for interrupts
- MongoDB transaction test suite
- Load testing for concurrent operations
- Memory profiling for leak detection

### Rollback Plan:
```bash
# If issues arise:
git checkout megha/individual-work
git branch -D megha/subchat-backend-reimpl
# Start over with lessons learned
```

---

## ✅ FINAL CHECKLIST BEFORE MERGE

- [ ] All 10 critical features implemented
- [ ] All 10 tests passing
- [ ] Confirmation step (Step 3) working
- [ ] No memory leaks detected
- [ ] No goroutine leaks detected
- [ ] Performance benchmarks met
- [ ] Multi-tenant tested
- [ ] Backward compatibility verified
- [ ] Code reviewed
- [ ] Documentation updated
- [ ] Ready for production

---

**Report Prepared By:** Claude Code Analysis
**Target Completion:** 2-3 days
**Priority:** CRITICAL
**Approver:** Megha

**Next Steps:**
1. Review this brief
2. Approve approach
3. Create implementation branch
4. Begin Phase 1
5. Test after each phase
6. Complete all phases
7. Final testing
8. Merge to megha/individual-work
