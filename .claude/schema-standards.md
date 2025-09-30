# 🔗 MCP Schema Standardization Rules

## 🚨 CRITICAL: CAMEL CASE MANDATORY FOR ALL MCP APIs

### **ZERO TOLERANCE POLICY: CAMEL CASE ONLY**

All MCP tool parameters, response fields, and JSON API interfaces **MUST** use camelCase convention. No exceptions.

## ❌ CRITICAL VIOLATIONS FOUND

### 1. **person_assign_task Tool - COMPLETELY NON-COMPLIANT**
**Current (WRONG):**
```json
{
  "person_id": "123",
  "task_name": "Test Task", 
  "task_description": "Description",
  "due_date": "2024-12-31"
}
```

**Required (CORRECT):**
```json
{
  "personId": "123",
  "taskName": "Test Task",
  "taskDescription": "Description", 
  "dueDate": "2024-12-31"
}
```

### 2. **Chart Tools - Inconsistent snake_case**
**Current (WRONG):**
```json
{
  "chart_type": "bar",
  "conversation_id": "123",
  "x_axis_label": "X Axis",
  "y_axis_label": "Y Axis"
}
```

**Required (CORRECT):**
```json
{
  "chartType": "bar",
  "conversationId": "123", 
  "xAxisLabel": "X Axis",
  "yAxisLabel": "Y Axis"
}
```

## 📋 MANDATORY SCHEMA STANDARDS

### **Parameter Naming Convention**

1. **ID Fields**: Always use `entityId` pattern
   - ✅ `taskId`, `documentId`, `memoryId`, `personId`
   - ❌ `task_id`, `person_id`, `id` (generic)

2. **Entity Actions**: Use `entity_action` pattern for tool names
   - ✅ `task_create`, `document_search`, `person_update`
   - ❌ `person_assign_task` (should be `task_assign_to_person`)

3. **Date/Time Fields**: Standardize temporal naming
   - ✅ `createdAt`, `updatedAt`, `expiresAt`, `dueAt`, `startsAt`
   - ❌ `due_date`, `create_time`, `dueDateTime`

4. **Content Fields**: Consistent main content naming
   - ✅ `content` for main text (documents, messages, memories)
   - ✅ `description` for summaries/explanations (tasks, processes)
   - ❌ `task_description` (redundant prefix)

### **Required vs Optional Parameters**

1. **Name Field**: Always required for creation, consistent naming
   - ✅ `name` (required)
   - ❌ `task_name`, `document_name`

2. **Assignment Structure**: Standardize assignment objects
   - ✅ Complex object: `{ "assignedTo": { "type": "human", "id": "123", "name": "User" } }`
   - ❌ Simple ID: `person_id: "123"`

### **Response Structure Standards**

1. **List Responses**: Always paginated
```json
{
  "items": [...],
  "total": 100,
  "page": 1,
  "pageSize": 20
}
```

2. **Error Responses**: Consistent structure
```json
{
  "error": "Error message",
  "code": "ERROR_CODE", 
  "details": { ... }
}
```

## 🛠️ IMMEDIATE FIXES REQUIRED

### Priority 1: Critical Violations
1. **Rename person_assign_task → task_assign_to_person**
2. **Fix all parameter names in person_assign_task**
3. **Standardize chart tool parameters**

### Priority 2: Schema Alignment
1. **Unify task creation interfaces** (agent_task_create vs person_task_create)
2. **Standardize ID parameter naming** across all tools
3. **Fix date field inconsistencies**

### Priority 3: Documentation
1. **Update all MCP tool schemas** in handlers
2. **Validate parameter naming** in tests
3. **Update API documentation**

## 🔍 VALIDATION CHECKLIST

### Before Any MCP Tool Development:
- [ ] All parameters use camelCase
- [ ] ID fields follow `entityId` pattern  
- [ ] Date fields use `*At` suffix
- [ ] Content fields consistently named
- [ ] No redundant entity prefixes (task_name → name)
- [ ] Error messages reference correct parameter names

### Go Code Standards:
```go
// ✅ CORRECT - Struct tags use camelCase for JSON
type TaskRequest struct {
    PersonID    string `json:"personId" validate:"required"`
    TaskName    string `json:"taskName" validate:"required"`
    Description string `json:"description" validate:"required"`
    DueDate     string `json:"dueDate,omitempty"`
}

// ❌ WRONG - snake_case in JSON tags
type TaskRequest struct {
    PersonID    string `json:"person_id" validate:"required"`
    TaskName    string `json:"task_name" validate:"required"`
    Description string `json:"task_description" validate:"required"`
    DueDate     string `json:"due_date,omitempty"`
}
```

### TypeScript/Frontend Standards:
```typescript
// ✅ CORRECT - camelCase interfaces
interface TaskRequest {
  personId: string;
  taskName: string;
  description: string;
  dueDate?: string;
}

// ❌ WRONG - snake_case properties
interface TaskRequest {
  person_id: string;
  task_name: string;
  task_description: string;
  due_date?: string;
}
```

## 🚨 ENFORCEMENT

1. **All new MCP tools must pass schema validation**
2. **Existing non-compliant tools must be fixed**
3. **API clients must be updated to match**
4. **Tests must validate correct parameter names**
5. **Documentation must reflect standardized schemas**

**NO EXCEPTIONS - CAMEL CASE IS MANDATORY!**