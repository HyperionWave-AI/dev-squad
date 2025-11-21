# How to Implement JWT Authentication Middleware

**Collection:** howto
**Tags:** jwt, authentication, middleware, security, go
**Version:** 1.0
**Last Updated:** 2025-11-21

---

## Overview

This guide explains how to implement JWT (JSON Web Token) authentication middleware for Go HTTP services using the Gin framework. You'll learn how to validate JWT tokens, extract user claims, and inject user identity into request context for downstream handlers.

## Prerequisites

- Understanding of HTTP middleware pattern
- Familiarity with JWT structure (header, payload, signature)
- Knowledge of Go context and Gin framework
- Review [JWT Authentication Patterns](../security-patterns/jwt-authentication.md)

## When to Use This Guide

- Protecting API endpoints with token-based authentication
- Implementing user identity extraction from JWTs
- Supporting both development and production authentication modes
- Multi-tenant applications requiring company/organization scoping

---

## Steps

### Step 1: Install JWT Library

Add the JWT library to your project:

```bash
go get github.com/golang-jwt/jwt/v5
```

### Step 2: Define JWT Claims Structure

Create `internal/middleware/auth.go` with claims definition:

```go
package middleware

import (
    "errors"
    "fmt"
    "os"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
    "go.uber.org/zap"
)

// CustomClaims extends jwt.RegisteredClaims with application-specific fields
type CustomClaims struct {
    UserID    string `json:"userId"`
    CompanyID string `json:"companyId"`
    Email     string `json:"email,omitempty"`
    Role      string `json:"role,omitempty"`
    jwt.RegisteredClaims
}

var (
    ErrMissingToken    = errors.New("authorization token missing")
    ErrInvalidToken    = errors.New("invalid authorization token")
    ErrExpiredToken    = errors.New("token has expired")
    ErrInvalidClaims   = errors.New("invalid token claims")
)
```

### Step 3: Implement Token Extraction

Extract Bearer token from Authorization header:

```go
// extractBearerToken extracts JWT from "Authorization: Bearer <token>" header
func extractBearerToken(c *gin.Context) (string, error) {
    authHeader := c.GetHeader("Authorization")
    if authHeader == "" {
        return "", ErrMissingToken
    }

    // Check for Bearer prefix
    parts := strings.SplitN(authHeader, " ", 2)
    if len(parts) != 2 || parts[0] != "Bearer" {
        return "", fmt.Errorf("invalid authorization header format")
    }

    token := strings.TrimSpace(parts[1])
    if token == "" {
        return "", ErrMissingToken
    }

    return token, nil
}
```

### Step 4: Implement Token Validation

Parse and validate the JWT token:

```go
// validateToken parses and validates JWT token
func validateToken(tokenString string, jwtSecret string) (*CustomClaims, error) {
    // Parse token with claims
    token, err := jwt.ParseWithClaims(
        tokenString,
        &CustomClaims{},
        func(token *jwt.Token) (interface{}, error) {
            // Verify signing method is HMAC
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
            }
            return []byte(jwtSecret), nil
        },
    )

    if err != nil {
        if errors.Is(err, jwt.ErrTokenExpired) {
            return nil, ErrExpiredToken
        }
        return nil, fmt.Errorf("failed to parse token: %w", err)
    }

    // Extract and validate claims
    if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
        // Basic claims validation
        if claims.UserID == "" {
            return nil, ErrInvalidClaims
        }
        return claims, nil
    }

    return nil, ErrInvalidToken
}
```

### Step 5: Create JWT Middleware

Implement the middleware function:

```go
// JWTAuthMiddleware validates JWT tokens and injects user identity into context
func JWTAuthMiddleware(logger *zap.Logger) gin.HandlerFunc {
    jwtSecret := os.Getenv("JWT_SECRET")
    if jwtSecret == "" {
        // Use default for development only
        jwtSecret = "default-secret-change-in-production"
        logger.Warn("Using default JWT_SECRET - INSECURE for production")
    }

    return func(c *gin.Context) {
        // Extract token from header
        tokenString, err := extractBearerToken(c)
        if err != nil {
            logger.Debug("Token extraction failed",
                zap.Error(err),
                zap.String("path", c.Request.URL.Path),
            )
            c.AbortWithStatusJSON(401, gin.H{
                "error": "Unauthorized",
                "message": "Missing or invalid authorization token",
            })
            return
        }

        // Validate token and extract claims
        claims, err := validateToken(tokenString, jwtSecret)
        if err != nil {
            logger.Debug("Token validation failed",
                zap.Error(err),
                zap.String("path", c.Request.URL.Path),
            )
            
            status := 401
            message := "Invalid token"
            
            if errors.Is(err, ErrExpiredToken) {
                message = "Token has expired"
            }
            
            c.AbortWithStatusJSON(status, gin.H{
                "error": "Unauthorized",
                "message": message,
            })
            return
        }

        // Inject user identity into context
        c.Set("userId", claims.UserID)
        c.Set("companyId", claims.CompanyID)
        c.Set("email", claims.Email)
        c.Set("role", claims.Role)
        c.Set("jwtClaims", claims)

        logger.Debug("User authenticated",
            zap.String("userId", claims.UserID),
            zap.String("companyId", claims.CompanyID),
        )

        // Continue to next handler
        c.Next()
    }
}
```

### Step 6: Create Optional JWT Middleware (Dev Mode)

Support disabling JWT for local development:

```go
// OptionalJWTMiddleware allows bypassing JWT validation in development
func OptionalJWTMiddleware(logger *zap.Logger) gin.HandlerFunc {
    enableJWT := os.Getenv("ENABLE_JWT")
    
    // If JWT is disabled, inject mock dev values
    if enableJWT != "true" && enableJWT != "1" {
        logger.Info("JWT authentication DISABLED - using dev mode")
        
        return func(c *gin.Context) {
            // Inject default dev user
            c.Set("userId", "dev-user-id")
            c.Set("companyId", "dev-company-id")
            c.Set("email", "dev@example.com")
            c.Set("role", "admin")
            
            c.Next()
        }
    }

    // JWT enabled - use full middleware
    return JWTAuthMiddleware(logger)
}
```

### Step 7: Create Helper Functions for Context Access

Add utility functions to retrieve user identity from context:

```go
// GetUserID extracts userId from request context
func GetUserID(c *gin.Context) (string, error) {
    userID, exists := c.Get("userId")
    if !exists {
        return "", errors.New("userId not found in context")
    }
    
    if id, ok := userID.(string); ok {
        return id, nil
    }
    
    return "", errors.New("userId is not a string")
}

// GetCompanyID extracts companyId from request context
func GetCompanyID(c *gin.Context) (string, error) {
    companyID, exists := c.Get("companyId")
    if !exists {
        return "", errors.New("companyId not found in context")
    }
    
    if id, ok := companyID.(string); ok {
        return id, nil
    }
    
    return "", errors.New("companyId is not a string")
}

// MustGetUserID extracts userId or panics (use in handlers only)
func MustGetUserID(c *gin.Context) string {
    userID, err := GetUserID(c)
    if err != nil {
        panic(err) // Will be caught by gin.Recovery()
    }
    return userID
}
```

### Step 8: Register Middleware in Router

Apply middleware to protected routes:

```go
package main

import (
    "github.com/gin-gonic/gin"
    "your-project/internal/middleware"
)

func setupRouter(logger *zap.Logger) *gin.Engine {
    router := gin.New()
    
    // Global middleware (all routes)
    router.Use(gin.Recovery())
    
    // Public routes (no auth)
    router.GET("/health", healthHandler)
    router.POST("/auth/login", loginHandler)
    
    // Protected routes (JWT required)
    api := router.Group("/api/v1")
    api.Use(middleware.OptionalJWTMiddleware(logger)) // or JWTAuthMiddleware
    {
        api.GET("/profile", profileHandler)
        api.GET("/tasks", listTasksHandler)
        api.POST("/tasks", createTaskHandler)
    }
    
    return router
}
```

### Step 9: Use User Context in Handlers

Access user identity in your handlers:

```go
package handlers

import (
    "github.com/gin-gonic/gin"
    "your-project/internal/middleware"
)

func GetProfileHandler(c *gin.Context) {
    // Extract user identity
    userID, err := middleware.GetUserID(c)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to get user identity"})
        return
    }
    
    companyID, _ := middleware.GetCompanyID(c)
    
    // Use identity in business logic
    profile, err := fetchUserProfile(userID, companyID)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to fetch profile"})
        return
    }
    
    c.JSON(200, profile)
}
```

### Step 10: Create JWT Tokens (Login Endpoint)

Implement token generation for authentication:

```go
package handlers

import (
    "time"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
)

func LoginHandler(c *gin.Context) {
    var loginReq struct {
        Email    string `json:"email" binding:"required"`
        Password string `json:"password" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&loginReq); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }
    
    // Validate credentials (implement your logic)
    user, err := authenticateUser(loginReq.Email, loginReq.Password)
    if err != nil {
        c.JSON(401, gin.H{"error": "Invalid credentials"})
        return
    }
    
    // Generate JWT token
    token, err := generateToken(user)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to generate token"})
        return
    }
    
    c.JSON(200, gin.H{
        "token": token,
        "userId": user.ID,
        "email": user.Email,
    })
}

func generateToken(user *User) (string, error) {
    jwtSecret := os.Getenv("JWT_SECRET")
    
    // Create claims
    claims := &middleware.CustomClaims{
        UserID:    user.ID,
        CompanyID: user.CompanyID,
        Email:     user.Email,
        Role:      user.Role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Issuer:    "your-service",
        },
    }
    
    // Create token
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    
    // Sign token
    return token.SignedString([]byte(jwtSecret))
}
```

---

## Best Practices

### 1. Use Strong Secrets in Production
```bash
# Generate a secure random secret
openssl rand -base64 32

# Set in environment
export JWT_SECRET="your-generated-secret"
```

### 2. Set Appropriate Token Expiration
```go
// Short-lived tokens (1-24 hours)
ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour))

// Consider refresh tokens for longer sessions
```

### 3. Validate Signing Method
Always verify the token uses expected signing algorithm to prevent algorithm confusion attacks:
```go
if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
    return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
}
```

### 4. Use Structured Logging
Log authentication events for security auditing:
```go
logger.Warn("Authentication failed",
    zap.String("reason", "invalid_token"),
    zap.String("ip", c.ClientIP()),
    zap.String("path", c.Request.URL.Path),
)
```

### 5. Handle Token Refresh
Implement token refresh endpoint to avoid forcing users to re-login frequently.

---

## Common Pitfalls

### 1. Storing Secrets in Code
```go
// ❌ BAD - Hardcoded secret
jwtSecret := "my-secret-key"

// ✅ GOOD - Load from environment
jwtSecret := os.Getenv("JWT_SECRET")
```

### 2. Not Validating Claims
```go
// ❌ BAD - Accepting empty userId
claims.UserID // could be ""

// ✅ GOOD - Validate required claims
if claims.UserID == "" {
    return nil, ErrInvalidClaims
}
```

### 3. Using Wrong Status Codes
- `401 Unauthorized` - Missing/invalid token
- `403 Forbidden` - Valid token but insufficient permissions
- `500 Internal Server Error` - Server-side validation errors

### 4. Not Supporting Development Mode
Always provide a way to disable JWT locally for faster development iteration.

---

## Related Documentation

- [JWT Authentication Patterns](../security-patterns/jwt-authentication.md) - Security architecture
- [MongoDB Secure Connection](./mongodb-secure-connection.md) - User-scoped database access
- [REST API Patterns](./rest-api-endpoint-patterns.md) - Protected endpoint design
- [Rate Limiting](../security-patterns/rate-limiting.md) - Additional security layer

---

## Troubleshooting

### Issue: "Token signature invalid"

**Cause:** JWT_SECRET mismatch between token generation and validation

**Solution:**
```bash
# Ensure JWT_SECRET is consistent
echo $JWT_SECRET

# Regenerate tokens with correct secret
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"pass"}'
```

### Issue: "userId not found in context"

**Cause:** Middleware not registered or route not protected

**Solution:**
```go
// Ensure middleware is applied to route group
api := router.Group("/api/v1")
api.Use(middleware.JWTAuthMiddleware(logger)) // ADD THIS
{
    api.GET("/profile", profileHandler)
}
```

### Issue: "Token has expired"

**Cause:** Token lifetime exceeded

**Solution:**
- Implement token refresh endpoint
- Increase token expiration time
- Clear local storage and re-login

### Issue: "CORS error when sending Authorization header"

**Cause:** CORS middleware not configured to allow Authorization header

**Solution:**
```go
// Configure CORS to allow Authorization header
router.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"http://localhost:3000"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders:     []string{"Content-Type", "Authorization"}, // ADD THIS
    AllowCredentials: true,
}))
```
