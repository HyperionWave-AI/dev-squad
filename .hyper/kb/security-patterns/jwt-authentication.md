# Hyperion JWT Authentication & Authorization

**Collection:** security-patterns
**Tags:** authentication, JWT, authorization, security
**File Reference:** middleware/auth.go
**Version:** 1.0

---

HYPERION JWT AUTHENTICATION & AUTHORIZATION

JWT Authentication Middleware (middleware/auth.go:13-159):

AUTHENTICATION FLOW:
1. OptionalJWTMiddleware() checks ENABLE_JWT environment variable
2. If disabled (default): Injects mock dev values (userId="dev-user", companyId="dev-company")
3. If enabled (ENABLE_JWT=true|1): Validates actual JWT tokens

JWT VALIDATION PROCESS:
Request Header: Authorization: Bearer <token>
Parsing:
- Extracts Bearer token from Authorization header (lines 44-64)
- Validates header format: must be "Bearer <token>"
- Returns 401 Unauthorized if missing or malformed

Token Validation:
- Parses JWT using HS256 HMAC algorithm (line 67-72)
- Validates signing method (must be HMAC, not RS256/ES256)
- Uses JWT_SECRET from environment or default (insecure) fallback
- Returns 401 if signature invalid or token expired

CLAIMS EXTRACTION (lines 102-128):
Flexible claim parsing supports multiple formats:
- userId sources: "userId", "user_id", "sub", "identity.id"
- companyId sources: "companyId", "company_id", "identity.companyId"
- Backward compatibility: Uses userId as companyId if missing
- Stores full claims in context for downstream use

CONTEXT INJECTION (lines 146-151):
Sets Gin context values:
- c.Set("userId", userId)
- c.Set("companyId", companyId)
- c.Set("jwtClaims", claims) - full token claims

DEVELOPMENT MODE:
ENABLE_JWT not set (default):
- Skips validation for rapid development
- Injects mock values without token
- Logs: "JWT authentication DISABLED"

PRODUCTION SECURITY:
ENABLE_JWT=true:
- Requires valid JWT token on every request
- Validates signature using JWT_SECRET
- Returns 401 for missing/invalid tokens
- Logs all validation failures

CONFIGURATION:
- JWT_SECRET: HMAC secret key (env variable)
- ENABLE_JWT: Enable/disable validation (env variable)
- Default secret: "hyperion-default-secret-change-in-production" (INSECURE)

SECURITY NOTES:
- JWT tokens tied to userId and companyId
- No built-in expiration validation (relies on token issuer)
- HMAC-only: Doesn't support public-key algorithms
- Claims format flexible to support different token sources
