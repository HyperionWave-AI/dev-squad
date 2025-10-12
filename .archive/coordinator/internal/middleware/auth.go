package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// OptionalJWTMiddleware provides optional JWT authentication
// If ENABLE_JWT is not set or set to "false" (default), it injects dev mock values
// If ENABLE_JWT is "true", it validates JWT tokens and extracts claims
func OptionalJWTMiddleware() gin.HandlerFunc {
	enableJWT := os.Getenv("ENABLE_JWT")
	jwtEnabled := enableJWT == "true" || enableJWT == "1"

	// Get logger (optional, for debugging)
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	if !jwtEnabled {
		logger.Info("JWT authentication DISABLED - using dev mock values")
		// Return middleware that injects mock values for development
		return func(c *gin.Context) {
			c.Set("userId", "dev-user")
			c.Set("companyId", "dev-company")
			c.Next()
		}
	}

	logger.Info("JWT authentication ENABLED - validating tokens")

	// JWT secret from environment
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		logger.Warn("JWT_SECRET not set, using default (INSECURE for production)")
		jwtSecret = "hyperion-default-secret-change-in-production"
	}

	// Return middleware that validates JWT tokens
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Missing Authorization header",
			})
			c.Abort()
			return
		}

		// Extract Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid Authorization header format. Expected: Bearer <token>",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Parse and validate JWT token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})

		if err != nil {
			logger.Error("JWT validation failed", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token: " + err.Error(),
			})
			c.Abort()
			return
		}

		if !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
			})
			c.Abort()
			return
		}

		// Extract claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token claims",
			})
			c.Abort()
			return
		}

		// Extract userId and companyId from claims
		// Try different claim formats to be flexible
		var userId, companyId string

		// Try to get userId
		if id, ok := claims["userId"].(string); ok {
			userId = id
		} else if id, ok := claims["user_id"].(string); ok {
			userId = id
		} else if id, ok := claims["sub"].(string); ok {
			userId = id
		} else if identity, ok := claims["identity"].(map[string]interface{}); ok {
			if id, ok := identity["id"].(string); ok {
				userId = id
			}
		}

		// Try to get companyId
		if id, ok := claims["companyId"].(string); ok {
			companyId = id
		} else if id, ok := claims["company_id"].(string); ok {
			companyId = id
		} else if identity, ok := claims["identity"].(map[string]interface{}); ok {
			if id, ok := identity["companyId"].(string); ok {
				companyId = id
			}
		}

		// Validate required claims
		if userId == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Token missing userId claim",
			})
			c.Abort()
			return
		}

		if companyId == "" {
			// If no companyId in token, use userId as default (for backward compatibility)
			companyId = userId
			logger.Warn("Token missing companyId claim, using userId as default",
				zap.String("userId", userId))
		}

		// Set claims in context
		c.Set("userId", userId)
		c.Set("companyId", companyId)

		// Store full claims for additional context if needed
		c.Set("jwtClaims", claims)

		logger.Debug("JWT validated successfully",
			zap.String("userId", userId),
			zap.String("companyId", companyId))

		c.Next()
	}
}
