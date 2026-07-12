package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/eternal-orbit-labs/gateway/internal/middleware"
	"github.com/eternal-orbit-labs/gateway/internal/service"
)

type AuthHandler struct {
	auth        *service.AuthService
	accessTTL   time.Duration
	refreshTTL  time.Duration
	isDev       bool
}

func NewAuthHandler(auth *service.AuthService, accessTTL, refreshTTL time.Duration, isDev bool) *AuthHandler {
	return &AuthHandler{auth: auth, accessTTL: accessTTL, refreshTTL: refreshTTL, isDev: isDev}
}

func (h *AuthHandler) Signup(c *gin.Context) {
	var req struct {
		Email     string `json:"email" binding:"required,email"`
		Password  string `json:"password" binding:"required,min=8"`
		FirstName string `json:"first_name" binding:"required"`
		LastName  string `json:"last_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	result, err := h.auth.Signup(c.Request.Context(), service.SignupInput{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		if strings.Contains(err.Error(), "email already exists") {
			badRequest(c, "email already exists")
			return
		}
		internalError(c)
		return
	}

	h.setAuthCookies(c, result.AccessToken, result.RefreshToken)
	created(c, gin.H{"success": true})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	result, err := h.auth.Login(c.Request.Context(), service.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	}, clientIP(c), c.GetHeader("User-Agent"))
	if err != nil {
		if strings.Contains(err.Error(), "invalid credentials") || strings.Contains(err.Error(), "Google login") {
			unauthorized(c, err.Error())
			return
		}
		if strings.Contains(err.Error(), "too many") {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
			return
		}
		internalError(c)
		return
	}

	h.setAuthCookies(c, result.AccessToken, result.RefreshToken)
	ok(c, gin.H{"success": true})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	rawToken, err := c.Cookie("eol_refresh")
	if err != nil || rawToken == "" {
		unauthorized(c, "missing refresh token")
		return
	}

	result, err := h.auth.Refresh(c.Request.Context(), rawToken, clientIP(c), c.GetHeader("User-Agent"))
	if err != nil {
		h.clearAuthCookies(c)
		unauthorized(c, "invalid refresh token")
		return
	}

	h.setAuthCookies(c, result.AccessToken, result.RefreshToken)
	ok(c, gin.H{
		"user": result.User,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	rawToken, _ := c.Cookie("eol_refresh")
	if rawToken != "" {
		h.auth.Logout(c.Request.Context(), rawToken)
	}
	h.clearAuthCookies(c)
	ok(c, gin.H{"message": "logged out"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := middleware.GetUserID(c)
	user, err := h.auth.GetMe(c.Request.Context(), userID)
	if err != nil || user == nil {
		unauthorized(c, "user not found")
		return
	}
	ok(c, gin.H{"user": user})
}

func (h *AuthHandler) UpdateMe(c *gin.Context) {
	var req struct {
		FirstName *string `json:"first_name"`
		LastName  *string `json:"last_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	user, err := h.auth.UpdateProfile(c.Request.Context(), userID, req.FirstName, req.LastName)
	if err != nil {
		internalError(c)
		return
	}
	ok(c, gin.H{"user": user})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	if err := h.auth.ChangePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		if strings.Contains(err.Error(), "incorrect") || strings.Contains(err.Error(), "no password") {
			badRequest(c, err.Error())
			return
		}
		internalError(c)
		return
	}
	ok(c, gin.H{"message": "password changed"})
}

func (h *AuthHandler) setAuthCookies(c *gin.Context, accessToken, refreshToken string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("eol_access", accessToken, int(h.accessTTL.Seconds()), "/", "", !h.isDev, true)
	c.SetCookie("eol_refresh", refreshToken, int(h.refreshTTL.Seconds()), "/", "", !h.isDev, true)
	// Clear any stale cookie that was set with the old "/api/auth" path
	c.SetCookie("eol_refresh", "", -1, "/api/auth", "", !h.isDev, true)
}

func (h *AuthHandler) clearAuthCookies(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("eol_access", "", -1, "/", "", !h.isDev, true)
	c.SetCookie("eol_refresh", "", -1, "/", "", !h.isDev, true)
	c.SetCookie("eol_refresh", "", -1, "/api/auth", "", !h.isDev, true)
}
