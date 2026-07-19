package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/eternal-orbit-labs/gateway/internal/service"
)

type OAuthHandler struct {
	auth        *service.AuthService
	oauthCfg    *oauth2.Config
	accessTTL   time.Duration
	refreshTTL  time.Duration
	isDev       bool
	frontendURL string
}

func NewOAuthHandler(auth *service.AuthService, clientID, clientSecret, redirectURL, frontendURL string, accessTTL, refreshTTL time.Duration, isDev bool) *OAuthHandler {
	return &OAuthHandler{
		auth:        auth,
		accessTTL:   accessTTL,
		refreshTTL:  refreshTTL,
		isDev:       isDev,
		frontendURL: frontendURL,
		oauthCfg: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		},
	}
}

// GoogleRedirect godoc
// @Summary      Start Google OAuth
// @Description  Redirects the user to Google's consent screen. Sets an oauth_state cookie for CSRF protection.
// @Tags         OAuth
// @Success      307  "Redirect to Google"
// @Failure      400  {object}  ErrorResponse
// @Router       /api/auth/google [get]
func (h *OAuthHandler) GoogleRedirect(c *gin.Context) {
	if h.oauthCfg.ClientID == "" {
		badRequest(c, "google oauth not configured")
		return
	}
	state, _ := generateState()
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("oauth_state", state, 600, "/", "", !h.isDev, true)
	c.Redirect(http.StatusTemporaryRedirect, h.oauthCfg.AuthCodeURL(state))
}

// GoogleCallback godoc
// @Summary      Google OAuth callback
// @Description  Handles the OAuth callback from Google. Validates state, exchanges code for tokens, creates or links user account, sets auth cookies, and redirects to frontend.
// @Tags         OAuth
// @Param        state  query  string  true   "OAuth state parameter"
// @Param        code   query  string  true   "Authorization code"
// @Success      307    "Redirect to frontend"
// @Failure      307    "Redirect to frontend with error"
// @Router       /api/auth/google/callback [get]
func (h *OAuthHandler) GoogleCallback(c *gin.Context) {
	stateCookie, err := c.Cookie("oauth_state")
	if err != nil || stateCookie == "" || stateCookie != c.Query("state") {
		c.Redirect(http.StatusTemporaryRedirect, h.frontendURL+"/signin?error=oauth_cancelled")
		return
	}

	code := c.Query("code")
	if code == "" {
		c.Redirect(http.StatusTemporaryRedirect, h.frontendURL+"/signin?error=oauth_cancelled")
		return
	}

	token, err := h.oauthCfg.Exchange(c.Request.Context(), code)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, h.frontendURL+"/signin?error=social_login_failure")
		return
	}

	userInfo, err := fetchGoogleUserInfo(token.AccessToken)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, h.frontendURL+"/signin?error=social_login_failure")
		return
	}

	result, err := h.auth.GoogleCallback(
		c.Request.Context(),
		userInfo.Sub,
		userInfo.Email,
		userInfo.GivenName,
		userInfo.FamilyName,
		userInfo.Picture,
		clientIP(c),
		c.GetHeader("User-Agent"),
	)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, h.frontendURL+"/signin?error=social_login_failure")
		return
	}

	// Set both auth cookies (and clear any stale old-path cookie)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("eol_access", result.AccessToken, int(h.accessTTL.Seconds()), "/", "", !h.isDev, true)
	c.SetCookie("eol_refresh", result.RefreshToken, int(h.refreshTTL.Seconds()), "/", "", !h.isDev, true)
	c.SetCookie("eol_refresh", "", -1, "/api/auth", "", !h.isDev, true)

	// Redirect to frontend — cookie is all that's needed
	c.Redirect(http.StatusTemporaryRedirect, h.frontendURL+"/auth/callback")
}

type googleUserInfo struct {
	Sub        string `json:"sub"`
	Email      string `json:"email"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
	Picture    string `json:"picture"`
}

func fetchGoogleUserInfo(accessToken string) (*googleUserInfo, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v3/userinfo?access_token=" + accessToken)
	if err != nil {
		return nil, fmt.Errorf("fetch google user info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read google response: %w", err)
	}

	var info googleUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("parse google user info: %w", err)
	}
	if info.Email == "" {
		return nil, fmt.Errorf("no email from google")
	}
	return &info, nil
}

func generateState() (string, error) {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b), nil
}
