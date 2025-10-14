package middleware

import (
    "net/http"
    "net/http/httptest"
    "os"
    "testing"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
)

func TestOptionalJWTMiddleware_Disabled(t *testing.T) {
    // Ensure JWT is disabled
    os.Unsetenv("ENABLE_JWT")
    os.Unsetenv("JWT_SECRET")

    r := gin.New()
    r.Use(OptionalJWTMiddleware())
    r.GET("/test", func(c *gin.Context) {
        userID, _ := c.Get("userId")
        companyID, _ := c.Get("companyId")
        c.JSON(http.StatusOK, gin.H{"userId": userID, "companyId": companyID})
    })

    req := httptest.NewRequest(http.MethodGet, "/test", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("expected status 200, got %d", w.Code)
    }
    // Simple check that response contains dev values
    body := w.Body.String()
    if body != "{\"companyId\":\"dev-company\",\"userId\":\"dev-user\"}" && body != "{\"userId\":\"dev-user\",\"companyId\":\"dev-company\"}" {
        t.Fatalf("unexpected response body: %s", body)
    }
}

func TestOptionalJWTMiddleware_EnabledValidToken(t *testing.T) {
    os.Setenv("ENABLE_JWT", "true")
    secret := "test-secret"
    os.Setenv("JWT_SECRET", secret)

    // Create a JWT token with required claims
    claims := jwt.MapClaims{
        "userId":    "test-user",
        "companyId": "test-company",
        "exp":       time.Now().Add(time.Hour).Unix(),
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, err := token.SignedString([]byte(secret))
    if err != nil {
        t.Fatalf("failed to sign token: %v", err)
    }

    r := gin.New()
    r.Use(OptionalJWTMiddleware())
    r.GET("/secure", func(c *gin.Context) {
        userID, _ := c.Get("userId")
        companyID, _ := c.Get("companyId")
        c.JSON(http.StatusOK, gin.H{"userId": userID, "companyId": companyID})
    })

    req := httptest.NewRequest(http.MethodGet, "/secure", nil)
    req.Header.Set("Authorization", "Bearer "+tokenString)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("expected status 200, got %d", w.Code)
    }
    body := w.Body.String()
    if body != "{\"companyId\":\"test-company\",\"userId\":\"test-user\"}" && body != "{\"userId\":\"test-user\",\"companyId\":\"test-company\"}" {
        t.Fatalf("unexpected response body: %s", body)
    }
}

func TestOptionalJWTMiddleware_EnabledMissingHeader(t *testing.T) {
    os.Setenv("ENABLE_JWT", "true")
    os.Setenv("JWT_SECRET", "any")

    r := gin.New()
    r.Use(OptionalJWTMiddleware())
    r.GET("/secure", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"ok": true})
    })

    req := httptest.NewRequest(http.MethodGet, "/secure", nil)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    if w.Code != http.StatusUnauthorized {
        t.Fatalf("expected status 401, got %d", w.Code)
    }
}
