package handler_test

import (
	"net/http"
	"testing"

	"github.com/eternal-orbit-labs/gateway/internal/testutil"
)

func TestHTTP_Signup_Success(t *testing.T) {
	env := setupTestEnv()
	req := jsonRequest("POST", "/api/auth/signup", map[string]string{
		"email": "signup@test.com", "password": "password123", "first_name": "John",
	})
	w := doRequest(env, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_Signup_MissingEmail(t *testing.T) {
	env := setupTestEnv()
	req := jsonRequest("POST", "/api/auth/signup", map[string]string{
		"password": "password123", "first_name": "John",
	})
	w := doRequest(env, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	body := parseJSON(t, w)
	if body["code"] != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR code, got %v", body["code"])
	}
}

func TestHTTP_Signup_ShortPassword(t *testing.T) {
	env := setupTestEnv()
	req := jsonRequest("POST", "/api/auth/signup", map[string]string{
		"email": "short@test.com", "password": "short", "first_name": "John",
	})
	w := doRequest(env, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_Signup_InvalidEmail(t *testing.T) {
	env := setupTestEnv()
	req := jsonRequest("POST", "/api/auth/signup", map[string]string{
		"email": "not-an-email", "password": "password123", "first_name": "John",
	})
	w := doRequest(env, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_Signup_DuplicateEmail(t *testing.T) {
	env := setupTestEnv()
	testutil.SeedTestUser(t, env.users, "dup@test.com", "password123")
	req := jsonRequest("POST", "/api/auth/signup", map[string]string{
		"email": "dup@test.com", "password": "password123", "first_name": "John",
	})
	w := doRequest(env, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_Login_Success(t *testing.T) {
	env := setupTestEnv()
	testutil.SeedTestUser(t, env.users, "login@test.com", "password123")
	req := jsonRequest("POST", "/api/auth/login", map[string]string{
		"email": "login@test.com", "password": "password123",
	})
	w := doRequest(env, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	// Check cookies were set
	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "eol_access" && c.Value != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected eol_access cookie to be set")
	}
}

func TestHTTP_Login_WrongPassword(t *testing.T) {
	env := setupTestEnv()
	testutil.SeedTestUser(t, env.users, "wrong@test.com", "password123")
	req := jsonRequest("POST", "/api/auth/login", map[string]string{
		"email": "wrong@test.com", "password": "wrongpass",
	})
	w := doRequest(env, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_Login_MissingFields(t *testing.T) {
	env := setupTestEnv()
	req := jsonRequest("POST", "/api/auth/login", map[string]string{})
	w := doRequest(env, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_Me_Authenticated(t *testing.T) {
	env := setupTestEnv()
	user := testutil.SeedTestUser(t, env.users, "me@test.com", "password123")
	req := authedRequest(env, "GET", "/api/auth/me", nil, user.ID, user.Email)
	w := doRequest(env, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseJSON(t, w)
	userMap := body["user"].(map[string]interface{})
	if userMap["email"] != "me@test.com" {
		t.Fatalf("expected email me@test.com, got %v", userMap["email"])
	}
}

func TestHTTP_Me_Unauthenticated(t *testing.T) {
	env := setupTestEnv()
	req := jsonRequest("GET", "/api/auth/me", nil)
	w := doRequest(env, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_UpdateMe_Success(t *testing.T) {
	env := setupTestEnv()
	user := testutil.SeedTestUser(t, env.users, "upd@test.com", "password123")
	req := authedRequest(env, "PATCH", "/api/auth/me", map[string]string{
		"first_name": "Updated",
	}, user.ID, user.Email)
	w := doRequest(env, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_ChangePassword_Success(t *testing.T) {
	env := setupTestEnv()
	user := testutil.SeedTestUser(t, env.users, "chpw@test.com", "oldpass123")
	req := authedRequest(env, "POST", "/api/auth/change-password", map[string]string{
		"current_password": "oldpass123", "new_password": "newpass456",
	}, user.ID, user.Email)
	w := doRequest(env, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_ChangePassword_WrongCurrent(t *testing.T) {
	env := setupTestEnv()
	user := testutil.SeedTestUser(t, env.users, "chpwwrong@test.com", "correct123")
	req := authedRequest(env, "POST", "/api/auth/change-password", map[string]string{
		"current_password": "wrong", "new_password": "newpass456",
	}, user.ID, user.Email)
	w := doRequest(env, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_Health(t *testing.T) {
	env := setupTestEnv()
	req := jsonRequest("GET", "/api/health", nil)
	w := doRequest(env, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := parseJSON(t, w)
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", body["status"])
	}
}
