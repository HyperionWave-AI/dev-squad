package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestOptionalJWTMiddleware(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Generate test JWT token
	secret := "test-secret"
	generateToken := func(claims map[string]interface{}) string {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(claims))
		tokenString, _ := token.SignedString([]byte(secret))
		return tokenString
	}

	tests := []struct {
		name              string
		enableJWT         string
		jwtSecret         string
		authHeader        string
		expectedStatus    int
		expectedUserID    string
		expectedCompanyID string
		expectError       bool
	}{
		{
			name:              "JWT disabled - dev mode with mock values",
			enableJWT:         "false",
			authHeader:        "",
			expectedStatus:    http.StatusOK,
			expectedUserID:    "dev-user",
			expectedCompanyID: "dev-company",
			expectError:       false,
		},
		{
			name:      "JWT enabled - valid token",
			enableJWT: "true",
			jwtSecret: secret,
			authHeader: "Bearer " + generateToken(map[string]interface{}{
				"userId":    "user-123",
				"companyId": "company-456",
				"exp":       time.Now().Add(time.Hour).Unix(),
			}),
			expectedStatus:    http.StatusOK,
			expectedUserID:    "user-123",
			expectedCompanyID: "company-456",
			expectError:       false,
		},
		{
			name:      "JWT enabled - identity nested format",
			enableJWT: "true",
			jwtSecret: secret,
			authHeader: "Bearer " + generateToken(map[string]interface{}{
				"identity": map[string]interface{}{
					"id":        "user-nested",
					"companyId": "company-nested",
				},
				"exp": time.Now().Add(time.Hour).Unix(),
			}),
			expectedStatus:    http.StatusOK,
			expectedUserID:    "user-nested",
			expectedCompanyID: "company-nested",
			expectError:       false,
		},
		{
			name:           "JWT enabled - missing Authorization header",
			enableJWT:      "true",
			jwtSecret:      secret,
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
		},
		{
			name:           "JWT enabled - invalid Bearer format",
			enableJWT:      "true",
			jwtSecret:      secret,
			authHeader:     "InvalidFormat token",
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
		},
		{
			name:      "JWT enabled - expired token",
			enableJWT: "true",
			jwtSecret: secret,
			authHeader: "Bearer " + generateToken(map[string]interface{}{
				"userId":    "user-123",
				"companyId": "company-456",
				"exp":       time.Now().Add(-time.Hour).Unix(), // Expired
			}),
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
		},
		{
			name:      "JWT enabled - invalid signature",
			enableJWT: "true",
			jwtSecret: secret,
			authHeader: "Bearer " + generateToken(map[string]interface{}{
				"userId":    "user-123",
				"companyId": "company-456",
				"exp":       time.Now().Add(time.Hour).Unix(),
			}) + "tampered",
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
		},
		{
			name:      "JWT enabled - missing userId claim",
			enableJWT: "true",
			jwtSecret: secret,
			authHeader: "Bearer " + generateToken(map[string]interface{}{
				"companyId": "company-456",
				"exp":       time.Now().Add(time.Hour).Unix(),
			}),
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
		},
		{
			name:      "JWT enabled - fallback to userId for companyId",
			enableJWT: "true",
			jwtSecret: secret,
			authHeader: "Bearer " + generateToken(map[string]interface{}{
				"userId": "user-fallback",
				"exp":    time.Now().Add(time.Hour).Unix(),
			}),
			expectedStatus:    http.StatusOK,
			expectedUserID:    "user-fallback",
			expectedCompanyID: "user-fallback", // Falls back to userId
			expectError:       false,
		},
		{
			name:      "JWT enabled - sub claim as userId",
			enableJWT: "true",
			jwtSecret: secret,
			authHeader: "Bearer " + generateToken(map[string]interface{}{
				"sub":       "user-sub",
				"companyId": "company-456",
				"exp":       time.Now().Add(time.Hour).Unix(),
			}),
			expectedStatus:    http.StatusOK,
			expectedUserID:    "user-sub",
			expectedCompanyID: "company-456",
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for this test
			if tt.enableJWT != "" {
				os.Setenv("ENABLE_JWT", tt.enableJWT)
				defer os.Unsetenv("ENABLE_JWT")
			}
			if tt.jwtSecret != "" {
				os.Setenv("JWT_SECRET", tt.jwtSecret)
				defer os.Unsetenv("JWT_SECRET")
			}

			// Create test router with middleware
			router := gin.New()
			router.Use(OptionalJWTMiddleware())
			router.GET("/test", func(c *gin.Context) {
				userID, _ := c.Get("userId")
				companyID, _ := c.Get("companyId")
				c.JSON(http.StatusOK, gin.H{
					"userId":    userID,
					"companyId": companyID,
				})
			})

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			// Serve request
			router.ServeHTTP(w, req)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code, "Status code mismatch")

			if !tt.expectError {
				// Check context values were set correctly
				if w.Code == http.StatusOK {
					// Parse response to check values
					assert.Contains(t, w.Body.String(), tt.expectedUserID, "User ID not found in response")
					assert.Contains(t, w.Body.String(), tt.expectedCompanyID, "Company ID not found in response")
				}
			} else {
				// Check error response
				assert.Contains(t, w.Body.String(), "error", "Error message not found")
			}
		})
	}
}

// Test default JWT secret warning
func TestDefaultJWTSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Enable JWT but don't set JWT_SECRET
	os.Setenv("ENABLE_JWT", "true")
	defer os.Unsetenv("ENABLE_JWT")
	os.Unsetenv("JWT_SECRET") // Ensure it's not set

	// Create middleware (should use default secret)
	middleware := OptionalJWTMiddleware()

	// Create valid token with default secret
	defaultSecret := "hyperion-default-secret-change-in-production"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId":    "user-123",
		"companyId": "company-456",
		"exp":       time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte(defaultSecret))

	router := gin.New()
	router.Use(middleware)
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should still work with default secret
	assert.Equal(t, http.StatusOK, w.Code)
}

// Test middleware chain execution
func TestMiddlewareChain(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Disable JWT for simple test
	os.Setenv("ENABLE_JWT", "false")
	defer os.Unsetenv("ENABLE_JWT")

	router := gin.New()
	router.Use(OptionalJWTMiddleware())

	executed := false
	router.GET("/test", func(c *gin.Context) {
		executed = true
		userID, exists := c.Get("userId")
		assert.True(t, exists, "userId should be set in context")
		assert.Equal(t, "dev-user", userID)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.True(t, executed, "Handler should have been executed")
	assert.Equal(t, http.StatusOK, w.Code)
}

// Test multiple claim formats
func TestMultipleClaimFormats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret"

	os.Setenv("ENABLE_JWT", "true")
	os.Setenv("JWT_SECRET", secret)
	defer os.Unsetenv("ENABLE_JWT")
	defer os.Unsetenv("JWT_SECRET")

	claimFormats := []map[string]interface{}{
		// Standard format
		{"userId": "user-1", "companyId": "company-1"},
		// Underscore format
		{"user_id": "user-2", "company_id": "company-2"},
		// Sub format
		{"sub": "user-3", "companyId": "company-3"},
		// Nested identity
		{"identity": map[string]interface{}{"id": "user-4", "companyId": "company-4"}},
	}

	for i, claims := range claimFormats {
		claims["exp"] = time.Now().Add(time.Hour).Unix()

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(claims))
		tokenString, _ := token.SignedString([]byte(secret))

		router := gin.New()
		router.Use(OptionalJWTMiddleware())
		router.GET("/test", func(c *gin.Context) {
			userID, _ := c.Get("userId")
			c.JSON(http.StatusOK, gin.H{"userId": userID})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "Test case %d failed", i)
	}
}
