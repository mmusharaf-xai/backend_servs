package testutil

import (
	"context"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/eternal-orbit-labs/gateway/internal/domain"
	"github.com/eternal-orbit-labs/gateway/internal/platform"
	"github.com/eternal-orbit-labs/gateway/internal/service"
)

const TestJWTSecret = "test-secret-key-for-testing-only"

// NewTestAuthService creates an AuthService wired to all fakes.
func NewTestAuthService() (*service.AuthService, *FakeUserRepo, *FakeRefreshSessionRepo, *FakeSessionCache, *FakeRateLimiter) {
	users := NewFakeUserRepo()
	sessions := NewFakeRefreshSessionRepo()
	cache := NewFakeSessionCache()
	rateLimiter := NewFakeRateLimiter()
	orgs := NewFakeAppOrgRepo()
	jwt := platform.NewJWTManager(TestJWTSecret, 15*time.Minute)

	svc := service.NewAuthService(users, orgs, sessions, cache, rateLimiter, jwt, 24*time.Hour)
	return svc, users, sessions, cache, rateLimiter
}

// NewTestAPIKeyService creates an APIKeyService wired to a fake repo.
func NewTestAPIKeyService() (*service.APIKeyService, *FakePersonalAPIKeyRepo) {
	repo := NewFakePersonalAPIKeyRepo()
	return service.NewAPIKeyService(repo), repo
}

// SeedTestUser creates a user with a hashed password in the fake repo.
func SeedTestUser(t *testing.T, repo *FakeUserRepo, email, password string) *domain.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 4) // low cost for tests
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := &domain.User{
		Email:        email,
		PasswordHash: string(hash),
		FirstName:    "Test",
		LastName:     "User",
	}
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return user
}

// SeedGoogleUser creates a user with a Google ID (no password).
func SeedGoogleUser(t *testing.T, repo *FakeUserRepo, email, googleID string) *domain.User {
	t.Helper()
	user := &domain.User{
		Email:         email,
		GoogleID:      googleID,
		FirstName:     "Google",
		LastName:      "User",
		EmailVerified: true,
	}
	if err := repo.Create(context.Background(), user); err != nil {
		t.Fatalf("seed google user: %v", err)
	}
	return user
}
