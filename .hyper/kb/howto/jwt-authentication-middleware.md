# JWT Authentication Middleware Pattern

## Overview

Gin middleware pattern for JWT token validation with support for both development and production modes, flexible claim extraction, and automatic context injection.

## Technology

- Gin Framework
- golang-jwt/jwt library
- HMAC token signing

## Use Case

Use this pattern when implementing JWT authentication in Hyperion HTTP services. The middleware supports development mode with mock credentials and production mode with full JWT validation.

## Implementation

### Optional JWT Middleware

**File Reference**: `hyper/internal/middleware/auth.go:13-159`

```go
func OptionalJWTMiddleware() gin.HandlerFunc {
    enableJWT := os.Getenv("ENABLE_JWT")
    jwtEnabled := enableJWT == "true" || enableJWT == "1"

    if !jwtEnabled {
        // Dev mode: inject mock values
        return func(c *gin.Context) {
            c.Set("userId", "dev-user")
            c.Set("companyId", "dev-company")
            c.Next()
        }
    }

    // Production mode: validate JWT
    jwtSecret := os.Getenv("JWT_SECRET")
    return func(c *gin.Context) {
        // 1. Extract Bearer token
        authHeader := c.GetHeader("Authorization")
        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid format"})
            c.Abort()
            return
        }

        // 2. Parse and validate
        token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, jwt.ErrSignatureInvalid
            }
            return []byte(jwtSecret), nil
        })

        if err != nil || !token.Valid {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }

        // 3. Extract claims (flexible format support)
        claims := token.Claims.(jwt.MapClaims)

        // Try multiple claim formats
        userId := claims["userId"] // or "user_id", "sub", "identity.id"
        companyId := claims["companyId"] // or "company_id"

        // 4. Inject into context
        c.Set("userId", userId)
        c.Set("companyId", companyId)
        c.Set("jwtClaims", claims)
        c.Next()
    }
}
```

## Key Points

### Environment Configuration

- **ENABLE_JWT**: Toggle between dev and production modes (`"true"` or `"1"` enables JWT)
- **JWT_SECRET**: HMAC signing secret (required in production mode)

### Mode Behavior

**Development Mode** (`ENABLE_JWT` not set or false):
- Injects mock `userId` and `companyId` values
- No token validation required
- Useful for local development and testing

**Production Mode** (`ENABLE_JWT=true`):
- Validates Bearer token from Authorization header
- Verifies HMAC signature
- Extracts and injects claims into context

### Claim Extraction

The middleware supports multiple JWT claim formats:

```go
// Try multiple formats for userId
userId := claims["userId"]
if userId == nil {
    userId = claims["user_id"]
}
if userId == nil {
    userId = claims["sub"]
}
if userId == nil {
    if identity, ok := claims["identity"].(map[string]interface{}); ok {
        userId = identity["id"]
    }
}
```

### Context Injection

After validation, claims are injected into Gin context:

```go
c.Set("userId", userId)
c.Set("companyId", companyId)
c.Set("jwtClaims", claims) // Full claims map for advanced use
```

### Using Claims in Handlers

```go
func (h *Handler) GetUserData(c *gin.Context) {
    // Extract from context
    userId, exists := c.Get("userId")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "No user context"})
        return
    }

    companyId, _ := c.Get("companyId")

    // Use in business logic
    data, err := h.service.GetUserData(userId.(string), companyId.(string))
    // ...
}
```

### Security Features

1. **Bearer Token Format**: Validates `Authorization: Bearer <token>` format
2. **HMAC Validation**: Only accepts HMAC-signed tokens
3. **Signature Verification**: Validates token signature against JWT_SECRET
4. **Abort on Failure**: Stops request chain on validation failure
5. **Multi-Tenant Support**: Extracts companyId for data isolation

### Best Practices

1. **Dev/Prod Toggle**: Use environment variable for mode switching
2. **Flexible Claims**: Support multiple claim name formats for compatibility
3. **Context Injection**: Store claims in Gin context for handler access
4. **Early Abort**: Return 401 and abort on validation failure
5. **Full Claims Access**: Store complete claims map for advanced scenarios
6. **Backward Compatible**: Falls back if optional claims missing

### Error Handling

```go
// Invalid format
if len(parts) != 2 || parts[0] != "Bearer" {
    c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
    c.Abort()
    return
}

// Invalid token or signature
if err != nil || !token.Valid {
    c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
    c.Abort()
    return
}
```

## Metadata

- **Domain**: authentication
- **Language**: go
- **Pattern**: middleware
- **Technology**: jwt, gin
