package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func signedToken(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return tokenString
}

func TestOptionalJWTMiddleware_InvalidHeaderFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("ENABLE_JWT", "true")
	t.Setenv("JWT_SECRET", "test-secret")

	r := gin.New()
	r.Use(OptionalJWTMiddleware())
	r.GET("/secure", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestOptionalJWTMiddleware_MissingUserClaim(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("ENABLE_JWT", "true")
	secret := "test-secret"
	t.Setenv("JWT_SECRET", secret)

	token := signedToken(t, secret, jwt.MapClaims{
		"companyId": "company-1",
		"exp":       time.Now().Add(time.Hour).Unix(),
	})

	r := gin.New()
	r.Use(OptionalJWTMiddleware())
	r.GET("/secure", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing user claim, got %d", w.Code)
	}
}

func TestOptionalJWTMiddleware_DefaultCompanyIDFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("ENABLE_JWT", "true")
	secret := "test-secret"
	t.Setenv("JWT_SECRET", secret)

	token := signedToken(t, secret, jwt.MapClaims{
		"userId": "user-1",
		"exp":    time.Now().Add(time.Hour).Unix(),
	})

	r := gin.New()
	r.Use(OptionalJWTMiddleware())
	r.GET("/secure", func(c *gin.Context) {
		userID, _ := c.Get("userId")
		companyID, _ := c.Get("companyId")
		c.JSON(http.StatusOK, gin.H{"userId": userID, "companyId": companyID})
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestOptionalJWTMiddleware_IdentityClaimsFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("ENABLE_JWT", "true")
	secret := "test-secret"
	t.Setenv("JWT_SECRET", secret)

	token := signedToken(t, secret, jwt.MapClaims{
		"identity": map[string]interface{}{
			"id":        "user-identity",
			"companyId": "company-identity",
		},
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	r := gin.New()
	r.Use(OptionalJWTMiddleware())
	r.GET("/secure", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for identity claims, got %d", w.Code)
	}
}

func TestOptionalJWTMiddleware_InvalidTokenSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("ENABLE_JWT", "true")
	t.Setenv("JWT_SECRET", "server-secret")

	token := signedToken(t, "different-secret", jwt.MapClaims{
		"userId":    "user-1",
		"companyId": "company-1",
		"exp":       time.Now().Add(time.Hour).Unix(),
	})

	r := gin.New()
	r.Use(OptionalJWTMiddleware())
	r.GET("/secure", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid signature, got %d", w.Code)
	}
}
