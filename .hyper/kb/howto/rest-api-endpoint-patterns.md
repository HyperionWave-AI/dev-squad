# How to Implement REST API Endpoints

**Collection:** howto
**Tags:** rest, api, http, gin, endpoints, go
**Version:** 1.0
**Last Updated:** 2025-11-21

---

## Overview

This guide demonstrates how to design and implement REST API endpoints following Hyperion's conventions. You'll learn proper HTTP methods, status codes, request validation, error handling, and response formatting patterns.

## Prerequisites

- Understanding of REST principles
- Go 1.25 with Gin framework
- Knowledge of HTTP status codes
- Familiarity with [API Service Layer](../api-service-layer.md)

## When to Use This Guide

- Creating new REST API endpoints
- Standardizing API design across services
- Implementing proper error handling
- Following RESTful conventions

---

## REST API Conventions

### HTTP Methods

| Method | Purpose | Idempotent | Safe |
|--------|---------|------------|------|
| GET | Retrieve resource(s) | Yes | Yes |
| POST | Create new resource | No | No |
| PUT | Update/replace entire resource | Yes | No |
| PATCH | Partially update resource | No | No |
| DELETE | Remove resource | Yes | No |

### Status Codes

| Code | Meaning | When to Use |
|------|---------|-------------|
| 200 | OK | Successful GET, PUT, PATCH, DELETE |
| 201 | Created | Successful POST (resource created) |
| 204 | No Content | Successful DELETE (no response body) |
| 400 | Bad Request | Invalid input/validation error |
| 401 | Unauthorized | Missing/invalid authentication |
| 403 | Forbidden | Valid auth but insufficient permissions |
| 404 | Not Found | Resource doesn't exist |
| 409 | Conflict | Resource already exists (duplicate) |
| 422 | Unprocessable Entity | Valid format but semantic errors |
| 500 | Internal Server Error | Unexpected server error |

---

## Steps

### Step 1: Define Request/Response Types

Create type-safe structures for requests and responses:

```go
package handlers

import (
    "time"
)

// Request types
type CreateTaskRequest struct {
    Prompt string `json:"prompt" binding:"required,min=1,max=10000"`
}

type UpdateTaskRequest struct {
    Status string  `json:"status" binding:"required,oneof=pending in_progress completed blocked"`
    Notes  *string `json:"notes" binding:"omitempty"`
}

type ListTasksQuery struct {
    Status string `form:"status" binding:"omitempty,oneof=pending in_progress completed blocked"`
    Limit  int    `form:"limit" binding:"omitempty,min=1,max=100"`
    Offset int    `form:"offset" binding:"omitempty,min=0"`
}

// Response types
type TaskResponse struct {
    TaskID    string    `json:"taskId"`
    Prompt    string    `json:"prompt"`
    Status    string    `json:"status"`
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
}

type ListTasksResponse struct {
    Data   []TaskResponse `json:"data"`
    Total  int            `json:"total"`
    Limit  int            `json:"limit"`
    Offset int            `json:"offset"`
}

type ErrorResponse struct {
    Error   string                 `json:"error"`
    Code    string                 `json:"code,omitempty"`
    Details map[string]interface{} `json:"details,omitempty"`
}
```

### Step 2: Implement CRUD Endpoints

#### CREATE - POST /api/v1/tasks

```go
// POST /api/v1/tasks - Create new task
func (h *TaskHandler) CreateTask(c *gin.Context) {
    var req CreateTaskRequest
    
    // Validate request body
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, ErrorResponse{
            Error: "Invalid request body",
            Code:  "VALIDATION_ERROR",
            Details: map[string]interface{}{
                "error": err.Error(),
            },
        })
        return
    }
    
    // Extract user identity from JWT context
    userID, _ := c.Get("userId")
    companyID, _ := c.Get("companyId")
    
    // Create task via service layer
    task, err := h.taskService.Create(c.Request.Context(), &TaskCreateInput{
        Prompt:    req.Prompt,
        UserID:    userID.(string),
        CompanyID: companyID.(string),
    })
    
    if err != nil {
        h.logger.Error("Failed to create task",
            zap.Error(err),
            zap.String("userId", userID.(string)),
        )
        c.JSON(500, ErrorResponse{
            Error: "Failed to create task",
            Code:  "INTERNAL_ERROR",
        })
        return
    }
    
    // Return 201 Created with resource
    c.JSON(201, TaskResponse{
        TaskID:    task.ID,
        Prompt:    task.Prompt,
        Status:    task.Status,
        CreatedAt: task.CreatedAt,
        UpdatedAt: task.UpdatedAt,
    })
}
```

#### READ - GET /api/v1/tasks/:id

```go
// GET /api/v1/tasks/:id - Get single task
func (h *TaskHandler) GetTask(c *gin.Context) {
    taskID := c.Param("id")
    
    // Validate UUID format
    if !isValidUUID(taskID) {
        c.JSON(400, ErrorResponse{
            Error: "Invalid task ID format",
            Code:  "INVALID_ID",
        })
        return
    }
    
    // Get user identity
    userID, _ := c.Get("userId")
    companyID, _ := c.Get("companyId")
    
    // Fetch task (automatically scoped by companyId)
    task, err := h.taskService.GetByID(c.Request.Context(), taskID, companyID.(string))
    
    if err != nil {
        if errors.Is(err, ErrTaskNotFound) {
            c.JSON(404, ErrorResponse{
                Error: "Task not found",
                Code:  "NOT_FOUND",
            })
            return
        }
        
        h.logger.Error("Failed to get task", zap.Error(err))
        c.JSON(500, ErrorResponse{
            Error: "Failed to retrieve task",
            Code:  "INTERNAL_ERROR",
        })
        return
    }
    
    c.JSON(200, TaskResponse{
        TaskID:    task.ID,
        Prompt:    task.Prompt,
        Status:    task.Status,
        CreatedAt: task.CreatedAt,
        UpdatedAt: task.UpdatedAt,
    })
}
```

#### LIST - GET /api/v1/tasks

```go
// GET /api/v1/tasks - List tasks with pagination
func (h *TaskHandler) ListTasks(c *gin.Context) {
    var query ListTasksQuery
    
    // Parse query parameters
    if err := c.ShouldBindQuery(&query); err != nil {
        c.JSON(400, ErrorResponse{
            Error: "Invalid query parameters",
            Code:  "VALIDATION_ERROR",
            Details: map[string]interface{}{
                "error": err.Error(),
            },
        })
        return
    }
    
    // Set defaults
    if query.Limit == 0 {
        query.Limit = 20
    }
    
    // Get user identity
    companyID, _ := c.Get("companyId")
    
    // Fetch tasks
    tasks, total, err := h.taskService.List(c.Request.Context(), &ListTasksInput{
        CompanyID: companyID.(string),
        Status:    query.Status,
        Limit:     query.Limit,
        Offset:    query.Offset,
    })
    
    if err != nil {
        h.logger.Error("Failed to list tasks", zap.Error(err))
        c.JSON(500, ErrorResponse{
            Error: "Failed to retrieve tasks",
            Code:  "INTERNAL_ERROR",
        })
        return
    }
    
    // Convert to response format
    taskResponses := make([]TaskResponse, len(tasks))
    for i, task := range tasks {
        taskResponses[i] = TaskResponse{
            TaskID:    task.ID,
            Prompt:    task.Prompt,
            Status:    task.Status,
            CreatedAt: task.CreatedAt,
            UpdatedAt: task.UpdatedAt,
        }
    }
    
    c.JSON(200, ListTasksResponse{
        Data:   taskResponses,
        Total:  total,
        Limit:  query.Limit,
        Offset: query.Offset,
    })
}
```

#### UPDATE - PUT /api/v1/tasks/:id

```go
// PUT /api/v1/tasks/:id - Update task
func (h *TaskHandler) UpdateTask(c *gin.Context) {
    taskID := c.Param("id")
    
    var req UpdateTaskRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, ErrorResponse{
            Error: "Invalid request body",
            Code:  "VALIDATION_ERROR",
        })
        return
    }
    
    companyID, _ := c.Get("companyId")
    
    // Update task
    task, err := h.taskService.Update(c.Request.Context(), taskID, &TaskUpdateInput{
        CompanyID: companyID.(string),
        Status:    req.Status,
        Notes:     req.Notes,
    })
    
    if err != nil {
        if errors.Is(err, ErrTaskNotFound) {
            c.JSON(404, ErrorResponse{
                Error: "Task not found",
                Code:  "NOT_FOUND",
            })
            return
        }
        
        c.JSON(500, ErrorResponse{
            Error: "Failed to update task",
            Code:  "INTERNAL_ERROR",
        })
        return
    }
    
    c.JSON(200, TaskResponse{
        TaskID:    task.ID,
        Prompt:    task.Prompt,
        Status:    task.Status,
        CreatedAt: task.CreatedAt,
        UpdatedAt: task.UpdatedAt,
    })
}
```

#### DELETE - DELETE /api/v1/tasks/:id

```go
// DELETE /api/v1/tasks/:id - Delete task
func (h *TaskHandler) DeleteTask(c *gin.Context) {
    taskID := c.Param("id")
    companyID, _ := c.Get("companyId")
    
    err := h.taskService.Delete(c.Request.Context(), taskID, companyID.(string))
    
    if err != nil {
        if errors.Is(err, ErrTaskNotFound) {
            c.JSON(404, ErrorResponse{
                Error: "Task not found",
                Code:  "NOT_FOUND",
            })
            return
        }
        
        c.JSON(500, ErrorResponse{
            Error: "Failed to delete task",
            Code:  "INTERNAL_ERROR",
        })
        return
    }
    
    // 204 No Content - successful deletion
    c.Status(204)
}
```

### Step 3: Register Routes

Group related endpoints and apply middleware:

```go
package api

import (
    "github.com/gin-gonic/gin"
    "your-project/internal/handlers"
    "your-project/internal/middleware"
)

func RegisterRoutes(
    router *gin.Engine,
    taskHandler *handlers.TaskHandler,
    logger *zap.Logger,
) {
    // Health check (no auth)
    router.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "healthy"})
    })
    
    // API v1 routes
    v1 := router.Group("/api/v1")
    
    // Apply JWT middleware to all v1 routes
    v1.Use(middleware.JWTAuthMiddleware(logger))
    
    // Task routes
    tasks := v1.Group("/tasks")
    {
        tasks.POST("", taskHandler.CreateTask)
        tasks.GET("", taskHandler.ListTasks)
        tasks.GET("/:id", taskHandler.GetTask)
        tasks.PUT("/:id", taskHandler.UpdateTask)
        tasks.DELETE("/:id", taskHandler.DeleteTask)
    }
}
```

### Step 4: Add Request Validation

Use Gin binding tags for automatic validation:

```go
type CreateUserRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Name     string `json:"name" binding:"required,min=2,max=100"`
    Age      int    `json:"age" binding:"omitempty,min=18,max=120"`
    Role     string `json:"role" binding:"required,oneof=admin user guest"`
    Password string `json:"password" binding:"required,min=8"`
}

// Custom validation
func (r *CreateUserRequest) Validate() error {
    if strings.Contains(r.Email, "+") {
        return fmt.Errorf("email cannot contain + character")
    }
    return nil
}

func handler(c *gin.Context) {
    var req CreateUserRequest
    
    // Automatic validation via binding tags
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, ErrorResponse{
            Error:   "Validation failed",
            Details: map[string]interface{}{"errors": err.Error()},
        })
        return
    }
    
    // Custom validation
    if err := req.Validate(); err != nil {
        c.JSON(400, ErrorResponse{
            Error: err.Error(),
            Code:  "CUSTOM_VALIDATION_ERROR",
        })
        return
    }
    
    // Process request...
}
```

---

## Best Practices

### 1. Consistent Error Format
```go
// Always use same error structure
type ErrorResponse struct {
    Error   string                 `json:"error"`   // Human-readable message
    Code    string                 `json:"code"`    // Machine-readable code
    Details map[string]interface{} `json:"details"` // Additional context
}
```

### 2. Proper Status Codes
```go
// ✅ GOOD - Appropriate status codes
c.JSON(201, resource)      // Created
c.JSON(404, ErrorResponse) // Not found
c.Status(204)              // No content

// ❌ BAD - Always returning 200
c.JSON(200, gin.H{"error": "not found"})
```

### 3. Pagination for Lists
```go
// Always paginate list endpoints
type PaginatedResponse struct {
    Data   []interface{} `json:"data"`
    Total  int           `json:"total"`
    Limit  int           `json:"limit"`
    Offset int           `json:"offset"`
}
```

### 4. Idempotency for PUT/DELETE
```go
// PUT should be idempotent - same result if called multiple times
// DELETE should return 404 if resource already deleted
```

### 5. Use Service Layer
```go
// ✅ GOOD - Business logic in service
task, err := h.taskService.Create(ctx, input)

// ❌ BAD - Business logic in handler
task := &Task{...}
db.Insert(task)
```

---

## Common Pitfalls

### 1. Not Validating Input
```go
// ❌ BAD
taskID := c.Param("id")
task, _ := service.Get(taskID)

// ✅ GOOD
taskID := c.Param("id")
if !isValidUUID(taskID) {
    c.JSON(400, ErrorResponse{Error: "Invalid ID format"})
    return
}
```

### 2. Exposing Internal Errors
```go
// ❌ BAD - Leaks internal details
c.JSON(500, gin.H{"error": err.Error()})

// ✅ GOOD - Generic message + logging
logger.Error("Internal error", zap.Error(err))
c.JSON(500, ErrorResponse{Error: "Internal server error"})
```

### 3. Not Using HTTP Methods Correctly
```go
// ❌ BAD - Using POST for everything
router.POST("/api/v1/get-task", handler)
router.POST("/api/v1/delete-task", handler)

// ✅ GOOD - Proper HTTP methods
router.GET("/api/v1/tasks/:id", getHandler)
router.DELETE("/api/v1/tasks/:id", deleteHandler)
```

---

## Related Documentation

- [API Service Layer](../api-service-layer.md) - API architecture
- [Data Contracts](../data-contracts.md) - Type definitions
- [JWT Authentication](./jwt-authentication-middleware.md) - Auth middleware

---

## Troubleshooting

### Issue: "Validation always fails"

**Solution:**
```go
// Use correct binding tag
type Request struct {
    Email string `json:"email" binding:"required,email"` // Correct
    // Not: validate:"required,email"
}
```

### Issue: "Can't parse JSON"

**Solution:**
```go
// Ensure Content-Type header is set
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"prompt":"task"}' \
  http://localhost:8080/api/v1/tasks
```
