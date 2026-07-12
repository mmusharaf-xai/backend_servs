package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/eternal-orbit-labs/gateway/internal/domain"
	"github.com/eternal-orbit-labs/gateway/internal/platform"
)

type SessionCache interface {
	Get(ctx context.Context, tokenHash string) (*domain.RefreshSession, error)
	Set(ctx context.Context, tokenHash string, session *domain.RefreshSession) error
	Delete(ctx context.Context, tokenHash string) error
}

type AuthService struct {
	users        domain.UserRepository
	orgs         domain.AppOrgRepository
	sessions     domain.RefreshSessionRepository
	sessionCache SessionCache
	rateLimiter  domain.RateLimiterRepository
	jwt          *platform.JWTManager
	refreshTTL   time.Duration
}

func NewAuthService(
	users domain.UserRepository,
	orgs domain.AppOrgRepository,
	sessions domain.RefreshSessionRepository,
	sessionCache SessionCache,
	rateLimiter domain.RateLimiterRepository,
	jwt *platform.JWTManager,
	refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		users:        users,
		orgs:         orgs,
		sessions:     sessions,
		sessionCache: sessionCache,
		rateLimiter:  rateLimiter,
		jwt:          jwt,
		refreshTTL:   refreshTTL,
	}
}

type SignupInput struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
}

type LoginInput struct {
	Email    string
	Password string
}

type AuthResult struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"-"`
	User         *domain.User `json:"user"`
}

func (s *AuthService) Signup(ctx context.Context, in SignupInput) (*AuthResult, error) {
	existing, err := s.users.GetByEmail(ctx, in.Email)
	if err != nil {
		return nil, fmt.Errorf("check existing: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("email already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		Email:        in.Email,
		PasswordHash: string(hash),
		FirstName:    in.FirstName,
		LastName:     in.LastName,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}

	if err := s.orgs.AcceptPendingInvitesForEmail(ctx, user.Email, user.ID); err != nil {
		return nil, fmt.Errorf("accept pending invites: %w", err)
	}

	return s.issueTokens(ctx, user, "", "")
}

func (s *AuthService) Login(ctx context.Context, in LoginInput, ip, userAgent string) (*AuthResult, error) {
	rateLimitKey := ip + ":" + in.Email
	attempts, _ := s.rateLimiter.GetLoginAttempts(ctx, rateLimitKey)
	if attempts >= 10 {
		return nil, fmt.Errorf("too many login attempts, please try again later")
	}

	user, err := s.users.GetByEmail(ctx, in.Email)
	if err != nil {
		return nil, fmt.Errorf("lookup user: %w", err)
	}
	if user == nil {
		s.rateLimiter.IncrementLoginAttempts(ctx, rateLimitKey)
		return nil, fmt.Errorf("invalid credentials")
	}

	if user.PasswordHash == "" {
		return nil, fmt.Errorf("this account uses Google login")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)); err != nil {
		s.rateLimiter.IncrementLoginAttempts(ctx, rateLimitKey)
		return nil, fmt.Errorf("invalid credentials")
	}

	return s.issueTokens(ctx, user, ip, userAgent)
}

func (s *AuthService) Refresh(ctx context.Context, rawToken, ip, userAgent string) (*AuthResult, error) {
	tokenHash := hashToken(rawToken)
	session, err := s.sessions.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("lookup session: %w", err)
	}
	if session == nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// Delete old session from both Postgres and Redis (rotation)
	if err := s.sessions.DeleteByTokenHash(ctx, tokenHash); err != nil {
		return nil, fmt.Errorf("delete old session: %w", err)
	}
	s.sessionCache.Delete(ctx, tokenHash) // best-effort, ignore error

	user, err := s.users.GetByID(ctx, session.UserID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}

	return s.issueTokens(ctx, user, ip, userAgent)
}

func (s *AuthService) Logout(ctx context.Context, rawToken string) error {
	tokenHash := hashToken(rawToken)
	s.sessionCache.Delete(ctx, tokenHash) // best-effort
	return s.sessions.DeleteByTokenHash(ctx, tokenHash)
}

func (s *AuthService) LogoutAll(ctx context.Context, userID string) error {
	return s.sessions.DeleteAllForUser(ctx, userID)
}

func (s *AuthService) GetMe(ctx context.Context, userID string) (*domain.User, error) {
	return s.users.GetByID(ctx, userID)
}

func (s *AuthService) UpdateProfile(ctx context.Context, userID string, firstName, lastName *string) (*domain.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}
	if firstName != nil {
		user.FirstName = *firstName
	}
	if lastName != nil {
		user.LastName = *lastName
	}
	if err := s.users.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}
	return user, nil
}

func (s *AuthService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil || user == nil {
		return fmt.Errorf("user not found")
	}
	if user.PasswordHash != "" {
		if currentPassword == "" {
			return fmt.Errorf("current password incorrect")
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
			return fmt.Errorf("current password incorrect")
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	user.PasswordHash = string(hash)
	return s.users.Update(ctx, user)
}

// GoogleCallback handles the user after Google OAuth verifies them.
func (s *AuthService) GoogleCallback(ctx context.Context, googleID, email, firstName, lastName, avatarURL, ip, userAgent string) (*AuthResult, error) {
	user, err := s.users.GetByGoogleID(ctx, googleID)
	if err != nil {
		return nil, fmt.Errorf("lookup google user: %w", err)
	}

	if user == nil {
		// Check if email already exists (link accounts)
		user, err = s.users.GetByEmail(ctx, email)
		if err != nil {
			return nil, fmt.Errorf("lookup email: %w", err)
		}
		if user != nil {
			user.GoogleID = googleID
			user.EmailVerified = true
			if avatarURL != "" && user.AvatarURL == "" {
				user.AvatarURL = avatarURL
			}
			if err := s.users.Update(ctx, user); err != nil {
				return nil, fmt.Errorf("link google: %w", err)
			}
			if err := s.orgs.AcceptPendingInvitesForEmail(ctx, user.Email, user.ID); err != nil {
				return nil, fmt.Errorf("accept pending invites: %w", err)
			}
		} else {
			user = &domain.User{
				Email:         email,
				FirstName:     firstName,
				LastName:      lastName,
				AvatarURL:     avatarURL,
				GoogleID:      googleID,
				EmailVerified: true,
			}
			if err := s.users.Create(ctx, user); err != nil {
				return nil, err
			}
			if err := s.orgs.AcceptPendingInvitesForEmail(ctx, user.Email, user.ID); err != nil {
				return nil, fmt.Errorf("accept pending invites: %w", err)
			}
		}
	}

	return s.issueTokens(ctx, user, ip, userAgent)
}

func (s *AuthService) issueTokens(ctx context.Context, user *domain.User, ip, userAgent string) (*AuthResult, error) {
	accessToken, err := s.jwt.GenerateAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	rawRefresh, err := generateRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	tokenHash := hashToken(rawRefresh)
	session := &domain.RefreshSession{
		UserID:    user.ID,
		TokenHash: tokenHash,
		IPAddress: ip,
		UserAgent: userAgent,
		ExpiresAt: time.Now().Add(s.refreshTTL),
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Cache session in Redis (best-effort, Postgres is source of truth)
	s.sessionCache.Set(ctx, tokenHash, session)

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		User:         user,
	}, nil
}

func generateRandomToken(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
