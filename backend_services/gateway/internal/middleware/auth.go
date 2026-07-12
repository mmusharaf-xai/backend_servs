package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/eternal-orbit-labs/gateway/internal/domain"
	"github.com/eternal-orbit-labs/gateway/internal/platform"
	"github.com/eternal-orbit-labs/gateway/internal/service"
)

type SessionCache interface {
	Get(ctx context.Context, tokenHash string) (*domain.RefreshSession, error)
	Set(ctx context.Context, tokenHash string, session *domain.RefreshSession) error
}

func Auth(jwt *platform.JWTManager, apiKeySvc *service.APIKeyService, sessions domain.RefreshSessionRepository, sessionCache SessionCache, accessTTL time.Duration, isDev bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		header := c.GetHeader("Authorization")

		// 1. Try Bearer token (API key only)
		if strings.HasPrefix(header, "Bearer ") {
			tokenStr := strings.TrimPrefix(header, "Bearer ")

			if strings.HasPrefix(tokenStr, "eol_k1_") {
				key, err := apiKeySvc.Authenticate(ctx, tokenStr)
				if err != nil {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
					return
				}
				c.Set("user_id", key.UserID)
				c.Set("auth_method", "api_key")
				c.Next()
				return
			}

			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		// 2. Try eol_access cookie (JWT — zero DB hit)
		if accessToken, err := c.Cookie("eol_access"); err == nil && accessToken != "" {
			claims, err := jwt.ValidateAccessToken(accessToken)
			if err == nil {
				c.Set("user_id", claims.UserID)
				c.Set("auth_method", "jwt")
				c.Next()
				return
			}
			// JWT expired or invalid — fall through to refresh
		}

		// 3. Try eol_refresh cookie — Redis first, then Postgres
		for _, ck := range c.Request.Cookies() {
			if ck.Name != "eol_refresh" || ck.Value == "" {
				continue
			}

			h := sha256.Sum256([]byte(ck.Value))
			tokenHash := hex.EncodeToString(h[:])

			// 3a. Check Redis cache
			session, err := sessionCache.Get(ctx, tokenHash)

			// 3b. Cache miss — check Postgres
			if err != nil || session == nil {
				session, err = sessions.GetByTokenHash(ctx, tokenHash)
				if err != nil || session == nil {
					continue // try next cookie
				}
				// Backfill Redis cache
				sessionCache.Set(ctx, tokenHash, session)
			}

			// 3c. Validate expiry
			if session.ExpiresAt.Before(time.Now()) {
				continue
			}

			// 3d. Issue a new access token JWT and set as cookie
			//     TTL = min(accessTTL, time until refresh expires)
			ttl := accessTTL
			if untilRefreshExpires := time.Until(session.ExpiresAt); untilRefreshExpires < ttl {
				ttl = untilRefreshExpires
			}
			newAccessToken, err := jwt.GenerateAccessTokenWithTTL(session.UserID, "", ttl)
			if err != nil {
				// JWT generation failed — still authenticate the request
				c.Set("user_id", session.UserID)
				c.Set("auth_method", "refresh_fallback")
				c.Next()
				return
			}

			// Set the new access cookie on the response
			c.SetSameSite(http.SameSiteLaxMode)
			c.SetCookie("eol_access", newAccessToken, int(ttl.Seconds()), "/", "", !isDev, true)

			c.Set("user_id", session.UserID)
			c.Set("auth_method", "refresh")
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization"})
	}
}

func GetUserID(c *gin.Context) string {
	v, _ := c.Get("user_id")
	s, _ := v.(string)
	return s
}
