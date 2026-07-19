package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/eternal-orbit-labs/gateway/internal/testutil"
)

func TestHTTP_CreateAPIKey_Success(t *testing.T) {
	env := setupTestEnv()
	user := testutil.SeedTestUser(t, env.users, "key@test.com", "password123")
	req := authedRequest(env, "POST", "/api/auth/apikeys", map[string]string{
		"label": "My Test Key",
	}, user.ID, user.Email)
	w := doRequest(env, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	body := parseJSON(t, w)
	if body["raw_value"] == nil || body["raw_value"] == "" {
		t.Fatal("expected raw_value in response")
	}
}

func TestHTTP_CreateAPIKey_MissingLabel(t *testing.T) {
	env := setupTestEnv()
	user := testutil.SeedTestUser(t, env.users, "key2@test.com", "password123")
	req := authedRequest(env, "POST", "/api/auth/apikeys", map[string]string{}, user.ID, user.Email)
	w := doRequest(env, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_CreateAPIKey_Unauthenticated(t *testing.T) {
	env := setupTestEnv()
	req := jsonRequest("POST", "/api/auth/apikeys", map[string]string{"label": "test"})
	w := doRequest(env, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_ListAPIKeys(t *testing.T) {
	env := setupTestEnv()
	user := testutil.SeedTestUser(t, env.users, "listkey@test.com", "password123")
	// Create a key first
	env.apiKeySvc.Create(context.Background(), user.ID, "Key A")
	req := authedRequest(env, "GET", "/api/auth/apikeys", nil, user.ID, user.Email)
	w := doRequest(env, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_DeleteAPIKey(t *testing.T) {
	env := setupTestEnv()
	user := testutil.SeedTestUser(t, env.users, "delkey@test.com", "password123")
	result, _ := env.apiKeySvc.Create(context.Background(), user.ID, "To Delete")
	req := authedRequest(env, "DELETE", "/api/auth/apikeys/"+result.Key.ID, nil, user.ID, user.Email)
	w := doRequest(env, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
