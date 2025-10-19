package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"instant/internal/session"

	"github.com/gin-gonic/gin"
)

// Mock session manager for testing
type mockSessionManager struct {
	getFunc      func(ctx context.Context, sessionID string) (*session.Session, error)
	validateFunc func(ctx context.Context, sessionID string) (bool, error)
}

func (m *mockSessionManager) Get(ctx context.Context, sessionID string) (*session.Session, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, sessionID)
	}
	return nil, errors.New("session not found")
}

func (m *mockSessionManager) Create(ctx context.Context, userID, email string, maxAge int) (string, error) {
	return "", nil
}

func (m *mockSessionManager) Delete(ctx context.Context, sessionID string) error {
	return nil
}

func (m *mockSessionManager) Validate(ctx context.Context, sessionID string) (bool, error) {
	if m.validateFunc != nil {
		return m.validateFunc(ctx, sessionID)
	}
	return true, nil
}

func TestSessionAuthMiddleware_ValidSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockMgr := &mockSessionManager{
		getFunc: func(ctx context.Context, sessionID string) (*session.Session, error) {
			return &session.Session{
				ID:        sessionID,
				UserID:    "test-user-id",
				Email:     "test@example.com",
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(1 * time.Hour),
			}, nil
		},
	}

	r := gin.New()
	r.Use(SessionAuthMiddleware(mockMgr))
	r.GET("/test", func(c *gin.Context) {
		// Check that headers were injected into the request
		userID := c.Request.Header.Get("X-User-ID")
		email := c.Request.Header.Get("X-User-Email")

		// Also check Gin context
		userIDCtx, _ := c.Get("user_id")
		emailCtx, _ := c.Get("email")

		c.JSON(http.StatusOK, gin.H{
			"user_id":      userIDCtx,
			"email":        emailCtx,
			"header_user":  userID,
			"header_email": email,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: "valid-session-id",
	})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response to check injected values
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["user_id"] != "test-user-id" {
		t.Errorf("Expected user_id to be test-user-id, got %v", response["user_id"])
	}
	if response["email"] != "test@example.com" {
		t.Errorf("Expected email to be test@example.com, got %v", response["email"])
	}
	if response["header_user"] != "test-user-id" {
		t.Errorf("Expected header_user to be test-user-id, got %v", response["header_user"])
	}
}

func TestSessionAuthMiddleware_NoSessionCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockMgr := &mockSessionManager{}
	r := gin.New()
	r.Use(SessionAuthMiddleware(mockMgr))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No session cookie
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestSessionAuthMiddleware_InvalidSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockMgr := &mockSessionManager{
		getFunc: func(ctx context.Context, sessionID string) (*session.Session, error) {
			return nil, errors.New("session not found")
		},
	}

	r := gin.New()
	r.Use(SessionAuthMiddleware(mockMgr))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: "invalid-session-id",
	})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestSessionAuthMiddleware_ExpiredSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockMgr := &mockSessionManager{
		getFunc: func(ctx context.Context, sessionID string) (*session.Session, error) {
			return &session.Session{
				ID:        sessionID,
				UserID:    "test-user-id",
				Email:     "test@example.com",
				CreatedAt: time.Now().Add(-2 * time.Hour),
				ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
			}, nil
		},
	}

	r := gin.New()
	r.Use(SessionAuthMiddleware(mockMgr))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: "expired-session-id",
	})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestSessionAuthMiddleware_HeaderInjection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockMgr := &mockSessionManager{
		getFunc: func(ctx context.Context, sessionID string) (*session.Session, error) {
			return &session.Session{
				ID:        sessionID,
				UserID:    "test-user-123",
				Email:     "user@example.com",
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(1 * time.Hour),
			}, nil
		},
	}

	r := gin.New()
	r.Use(SessionAuthMiddleware(mockMgr))
	r.GET("/test", func(c *gin.Context) {
		// Check headers that should be injected
		userID := c.Request.Header.Get("X-User-ID")
		email := c.Request.Header.Get("X-User-Email")

		if userID != "test-user-123" {
			t.Errorf("Expected X-User-ID to be test-user-123, got %s", userID)
		}
		if email != "user@example.com" {
			t.Errorf("Expected X-User-Email to be user@example.com, got %s", email)
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: "valid-session",
	})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(CORSMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// Check CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Error("Expected CORS Allow-Origin header")
	}
	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("Expected CORS Allow-Credentials header")
	}
}

func TestCORSMiddleware_OPTIONS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(CORSMiddleware())
	r.OPTIONS("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
	})

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// OPTIONS should return 204 No Content
	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status 204 for OPTIONS, got %d", w.Code)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(LoggingMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	// Note: Logging output would go to stdout, which we're not capturing here
	// This test just ensures the middleware doesn't break the request flow
}
