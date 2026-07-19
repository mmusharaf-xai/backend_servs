package handler_test

import (
	"net/http"
	"testing"

	"github.com/eternal-orbit-labs/gateway/internal/domain"
	"github.com/eternal-orbit-labs/gateway/internal/testutil"
)

func TestHTTP_ListApps(t *testing.T) {
	env := setupTestEnv()
	env.appRepo.Apps = []*domain.App{
		{ID: "1", Slug: "app-one", Name: "App One"},
		{ID: "2", Slug: "app-two", Name: "App Two"},
	}
	user := testutil.SeedTestUser(t, env.users, "apps@test.com", "password123")
	req := authedRequest(env, "GET", "/api/apps", nil, user.ID, user.Email)
	w := doRequest(env, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_ListApps_Unauthenticated(t *testing.T) {
	env := setupTestEnv()
	req := jsonRequest("GET", "/api/apps", nil)
	w := doRequest(env, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestHTTP_GetApp_Found(t *testing.T) {
	env := setupTestEnv()
	env.appRepo.Apps = []*domain.App{
		{ID: "1", Slug: "my-app", Name: "My App"},
	}
	user := testutil.SeedTestUser(t, env.users, "getapp@test.com", "password123")
	req := authedRequest(env, "GET", "/api/apps/my-app", nil, user.ID, user.Email)
	w := doRequest(env, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_GetApp_NotFound(t *testing.T) {
	env := setupTestEnv()
	env.appRepo.Apps = []*domain.App{}
	user := testutil.SeedTestUser(t, env.users, "getapp404@test.com", "password123")
	req := authedRequest(env, "GET", "/api/apps/nonexistent", nil, user.ID, user.Email)
	w := doRequest(env, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
