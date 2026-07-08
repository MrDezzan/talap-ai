package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"talkd-backend/auth"
)

func TestAuthRequired_NoHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthRequired())
	r.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 Unauthorized, got %d", w.Code)
	}
}

func TestAuthRequired_InvalidPrefix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthRequired())
	r.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Token invalid_token_format")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 Unauthorized for invalid prefix, got %d", w.Code)
	}
}

func TestAuthRequired_InvalidToken(t *testing.T) {
	os.Setenv("JWT_SECRET", "test_jwt_secret")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AuthRequired())
	r.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.jwt.token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 Unauthorized for invalid token, got %d", w.Code)
	}
}

func TestAuthRequired_ValidToken(t *testing.T) {
	os.Setenv("JWT_SECRET", "test_jwt_secret")
	gin.SetMode(gin.TestMode)

	// Generate a valid token
	token, err := auth.GenerateToken(42, "test@example.com", "Test User")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	r := gin.New()
	r.Use(AuthRequired())
	
	var contextUserId uint
	var contextEmail string
	var contextName string

	r.GET("/protected", func(c *gin.Context) {
		uid, _ := c.Get("user_id")
		email, _ := c.Get("email")
		name, _ := c.Get("name")

		contextUserId = uid.(uint)
		contextEmail = email.(string)
		contextName = name.(string)
		
		c.Status(http.StatusOK)
	})

	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	if contextUserId != 42 {
		t.Errorf("Expected user_id 42 in context, got %d", contextUserId)
	}
	if contextEmail != "test@example.com" {
		t.Errorf("Expected email test@example.com in context, got %s", contextEmail)
	}
	if contextName != "Test User" {
		t.Errorf("Expected name 'Test User' in context, got %s", contextName)
	}
}
