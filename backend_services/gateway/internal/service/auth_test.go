package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/eternal-orbit-labs/gateway/internal/service"
	"github.com/eternal-orbit-labs/gateway/internal/testutil"
)

// ── Signup ────────────────────────────────────────────────────────────

func TestSignup_Success(t *testing.T) {
	svc, _, _, _, _ := testutil.NewTestAuthService()
	result, err := svc.Signup(context.Background(), service.SignupInput{
		Email: "new@test.com", Password: "password123", FirstName: "John", LastName: "Doe",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AccessToken == "" {
		t.Fatal("expected access token")
	}
	if result.RefreshToken == "" {
		t.Fatal("expected refresh token")
	}
	if result.User == nil {
		t.Fatal("expected user")
	}
	if result.User.Email != "new@test.com" {
		t.Fatalf("expected email new@test.com, got %s", result.User.Email)
	}
}

func TestSignup_DuplicateEmail(t *testing.T) {
	svc, users, _, _, _ := testutil.NewTestAuthService()
	testutil.SeedTestUser(t, users, "dup@test.com", "password123")
	_, err := svc.Signup(context.Background(), service.SignupInput{
		Email: "dup@test.com", Password: "password123", FirstName: "John",
	})
	if !errors.Is(err, service.ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
	}
}

// ── Login ─────────────────────────────────────────────────────────────

func TestLogin_Success(t *testing.T) {
	svc, users, _, _, _ := testutil.NewTestAuthService()
	testutil.SeedTestUser(t, users, "login@test.com", "password123")
	result, err := svc.Login(context.Background(), service.LoginInput{
		Email: "login@test.com", Password: "password123",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AccessToken == "" {
		t.Fatal("expected access token")
	}
	if result.User.Email != "login@test.com" {
		t.Fatalf("wrong email: %s", result.User.Email)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, users, _, _, _ := testutil.NewTestAuthService()
	testutil.SeedTestUser(t, users, "wrong@test.com", "password123")
	_, err := svc.Login(context.Background(), service.LoginInput{
		Email: "wrong@test.com", Password: "wrongpassword",
	}, "127.0.0.1", "test-agent")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_NonexistentUser(t *testing.T) {
	svc, _, _, _, _ := testutil.NewTestAuthService()
	_, err := svc.Login(context.Background(), service.LoginInput{
		Email: "nobody@test.com", Password: "password123",
	}, "127.0.0.1", "test-agent")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_GoogleOnlyAccount(t *testing.T) {
	svc, users, _, _, _ := testutil.NewTestAuthService()
	testutil.SeedGoogleUser(t, users, "google@test.com", "gid-123")
	_, err := svc.Login(context.Background(), service.LoginInput{
		Email: "google@test.com", Password: "anything",
	}, "127.0.0.1", "test-agent")
	if !errors.Is(err, service.ErrGoogleLoginOnly) {
		t.Fatalf("expected ErrGoogleLoginOnly, got %v", err)
	}
}

func TestLogin_RateLimited(t *testing.T) {
	svc, users, _, _, rateLimiter := testutil.NewTestAuthService()
	testutil.SeedTestUser(t, users, "rate@test.com", "password123")
	// Simulate 10 failed attempts
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		rateLimiter.IncrementLoginAttempts(ctx, "127.0.0.1:rate@test.com")
	}
	_, err := svc.Login(ctx, service.LoginInput{
		Email: "rate@test.com", Password: "password123",
	}, "127.0.0.1", "test-agent")
	if !errors.Is(err, service.ErrTooManyLoginAttempts) {
		t.Fatalf("expected ErrTooManyLoginAttempts, got %v", err)
	}
}

// ── Refresh ───────────────────────────────────────────────────────────

func TestRefresh_Success(t *testing.T) {
	svc, users, _, _, _ := testutil.NewTestAuthService()
	testutil.SeedTestUser(t, users, "refresh@test.com", "password123")
	loginResult, err := svc.Login(context.Background(), service.LoginInput{
		Email: "refresh@test.com", Password: "password123",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	result, err := svc.Refresh(context.Background(), loginResult.RefreshToken, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if result.AccessToken == "" {
		t.Fatal("expected new access token")
	}
	if result.RefreshToken == "" {
		t.Fatal("expected new refresh token")
	}
	if result.RefreshToken == loginResult.RefreshToken {
		t.Fatal("refresh token should have rotated")
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	svc, _, _, _, _ := testutil.NewTestAuthService()
	_, err := svc.Refresh(context.Background(), "bogus-token", "127.0.0.1", "test-agent")
	if !errors.Is(err, service.ErrInvalidRefreshToken) {
		t.Fatalf("expected ErrInvalidRefreshToken, got %v", err)
	}
}

// ── Logout ────────────────────────────────────────────────────────────

func TestLogout_Success(t *testing.T) {
	svc, users, _, _, _ := testutil.NewTestAuthService()
	testutil.SeedTestUser(t, users, "logout@test.com", "password123")
	loginResult, _ := svc.Login(context.Background(), service.LoginInput{
		Email: "logout@test.com", Password: "password123",
	}, "127.0.0.1", "test-agent")
	err := svc.Logout(context.Background(), loginResult.RefreshToken)
	if err != nil {
		t.Fatalf("logout failed: %v", err)
	}
	// Refresh should fail after logout
	_, err = svc.Refresh(context.Background(), loginResult.RefreshToken, "127.0.0.1", "test-agent")
	if !errors.Is(err, service.ErrInvalidRefreshToken) {
		t.Fatalf("expected refresh to fail after logout, got %v", err)
	}
}

// ── GetMe ─────────────────────────────────────────────────────────────

func TestGetMe_Success(t *testing.T) {
	svc, users, _, _, _ := testutil.NewTestAuthService()
	user := testutil.SeedTestUser(t, users, "me@test.com", "password123")
	got, err := svc.GetMe(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Email != "me@test.com" {
		t.Fatalf("wrong email: %s", got.Email)
	}
}

func TestGetMe_NotFound(t *testing.T) {
	svc, _, _, _, _ := testutil.NewTestAuthService()
	got, err := svc.GetMe(context.Background(), "nonexistent-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatal("expected nil user for nonexistent ID")
	}
}

// ── UpdateProfile ─────────────────────────────────────────────────────

func TestUpdateProfile_Success(t *testing.T) {
	svc, users, _, _, _ := testutil.NewTestAuthService()
	user := testutil.SeedTestUser(t, users, "update@test.com", "password123")
	first := "Updated"
	last := "Name"
	got, err := svc.UpdateProfile(context.Background(), user.ID, &first, &last)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.FirstName != "Updated" {
		t.Fatalf("first name not updated: %s", got.FirstName)
	}
	if got.LastName != "Name" {
		t.Fatalf("last name not updated: %s", got.LastName)
	}
}

func TestUpdateProfile_NotFound(t *testing.T) {
	svc, _, _, _, _ := testutil.NewTestAuthService()
	first := "X"
	_, err := svc.UpdateProfile(context.Background(), "nonexistent", &first, nil)
	if !errors.Is(err, service.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

// ── ChangePassword ────────────────────────────────────────────────────

func TestChangePassword_Success(t *testing.T) {
	svc, users, _, _, _ := testutil.NewTestAuthService()
	user := testutil.SeedTestUser(t, users, "chpw@test.com", "oldpass123")
	err := svc.ChangePassword(context.Background(), user.ID, "oldpass123", "newpass456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Old password should no longer work
	_, err = svc.Login(context.Background(), service.LoginInput{Email: "chpw@test.com", Password: "oldpass123"}, "", "")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("old password should not work: %v", err)
	}
	// New password should work
	_, err = svc.Login(context.Background(), service.LoginInput{Email: "chpw@test.com", Password: "newpass456"}, "", "")
	if err != nil {
		t.Fatalf("new password should work: %v", err)
	}
}

func TestChangePassword_WrongCurrent(t *testing.T) {
	svc, users, _, _, _ := testutil.NewTestAuthService()
	user := testutil.SeedTestUser(t, users, "wrongcur@test.com", "correct123")
	err := svc.ChangePassword(context.Background(), user.ID, "wrong123", "newpass456")
	if !errors.Is(err, service.ErrCurrentPasswordIncorrect) {
		t.Fatalf("expected ErrCurrentPasswordIncorrect, got %v", err)
	}
}

func TestChangePassword_GoogleUser(t *testing.T) {
	svc, users, _, _, _ := testutil.NewTestAuthService()
	user := testutil.SeedGoogleUser(t, users, "googlecp@test.com", "gid-456")
	// Google user with empty password hash, providing empty current password
	err := svc.ChangePassword(context.Background(), user.ID, "", "newpass456")
	if err != nil {
		t.Fatalf("google user setting first password should work: %v", err)
	}
}

func TestChangePassword_NotFound(t *testing.T) {
	svc, _, _, _, _ := testutil.NewTestAuthService()
	err := svc.ChangePassword(context.Background(), "nonexistent", "old", "new12345")
	if !errors.Is(err, service.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

// ── GoogleCallback ────────────────────────────────────────────────────

func TestGoogleCallback_NewUser(t *testing.T) {
	svc, _, _, _, _ := testutil.NewTestAuthService()
	result, err := svc.GoogleCallback(context.Background(), "gid-new", "newgoogle@test.com", "First", "Last", "https://avatar.url", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.User.Email != "newgoogle@test.com" {
		t.Fatalf("wrong email: %s", result.User.Email)
	}
	if result.User.GoogleID != "gid-new" {
		t.Fatal("google ID not set")
	}
	if result.AccessToken == "" {
		t.Fatal("expected access token")
	}
}

func TestGoogleCallback_ExistingGoogleUser(t *testing.T) {
	svc, users, _, _, _ := testutil.NewTestAuthService()
	testutil.SeedGoogleUser(t, users, "existing@test.com", "gid-existing")
	result, err := svc.GoogleCallback(context.Background(), "gid-existing", "existing@test.com", "First", "Last", "", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.User.Email != "existing@test.com" {
		t.Fatalf("wrong email: %s", result.User.Email)
	}
}

func TestGoogleCallback_LinkExistingEmailUser(t *testing.T) {
	svc, users, _, _, _ := testutil.NewTestAuthService()
	testutil.SeedTestUser(t, users, "link@test.com", "password123")
	result, err := svc.GoogleCallback(context.Background(), "gid-link", "link@test.com", "First", "Last", "https://pic.url", "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.User.GoogleID != "gid-link" {
		t.Fatal("google ID not linked")
	}
	if result.User.EmailVerified != true {
		t.Fatal("email should be verified after Google link")
	}
}
