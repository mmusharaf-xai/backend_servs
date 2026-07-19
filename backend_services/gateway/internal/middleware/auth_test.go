package middleware_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/eternal-orbit-labs/gateway/internal/middleware"
	"github.com/eternal-orbit-labs/gateway/internal/platform"
	"github.com/eternal-orbit-labs/gateway/internal/service"
	"github.com/eternal-orbit-labs/gateway/internal/testutil"
)

func setupAuthTestRouter() (*gin.Engine, *platform.JWTManager, *testutil.FakeRefreshSessionRepo, *testutil.FakeSessionCache, *service.APIKeyService, *testutil.FakePersonalAPIKeyRepo) {
	gin.SetMode(gin.TestMode)

	sessions := testutil.NewFakeRefreshSessionRepo()
	cache := testutil.NewFakeSessionCache()
	apiKeyRepo := testutil.NewFakePersonalAPIKeyRepo()
	apiKeySvc := service.NewAPIKeyService(apiKeyRepo)
	jwtMgr := platform.NewJWTManager("test-secret", 15*time.Minute)

	r := gin.New()
	r.Use(middleware.Auth(jwtMgr, apiKeySvc, sessions, cache, 15*time.Minute, true))
	r.GET("/protected", func(c *gin.Context) {
		userID := middleware.GetUserID(c)
		method := c.GetString("auth_method")
		c.JSON(200, gin.H{"user_id": userID, "auth_method": method})
	})

	return r, jwtMgr, sessions, cache, apiKeySvc, apiKeyRepo
}

func TestAuth_ValidJWTCookie(t *testing.T) {
	r, jwtMgr, _, _, _, _ := setupAuthTestRouter()
	token, _ := jwtMgr.GenerateAccessToken("user-123", "test@test.com")

	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "eol_access", Value: token})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["user_id"] != "user-123" {
		t.Fatalf("expected user_id user-123, got %v", body["user_id"])
	}
	if body["auth_method"] != "jwt" {
		t.Fatalf("expected auth_method jwt, got %v", body["auth_method"])
	}
}

func TestAuth_ValidAPIKey(t *testing.T) {
	r, _, _, _, apiKeySvc, _ := setupAuthTestRouter()
	result, _ := apiKeySvc.Create(context.Background(), "user-456", "Test Key")

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+result.RawValue)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["user_id"] != "user-456" {
		t.Fatalf("expected user_id user-456, got %v", body["user_id"])
	}
	if body["auth_method"] != "api_key" {
		t.Fatalf("expected auth_method api_key, got %v", body["auth_method"])
	}
}

func TestAuth_NoAuth(t *testing.T) {
	r, _, _, _, _, _ := setupAuthTestRouter()
	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_InvalidJWT(t *testing.T) {
	r, _, _, _, _, _ := setupAuthTestRouter()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "eol_access", Value: "invalid-jwt-token"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should fall through to refresh cookie, then fail with 401
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_InvalidBearerToken(t *testing.T) {
	r, _, _, _, _, _ := setupAuthTestRouter()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer not-eol-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAuth_InvalidAPIKey(t *testing.T) {
	r, _, _, _, _, _ := setupAuthTestRouter()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer eol_k1_0000000000000000000000000000000000000000000000000000000000000000")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}
