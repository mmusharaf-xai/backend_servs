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

// Signup godoc
// @Summary      Create a new account
// @Description  Register a new user with email and password. Sets httpOnly auth cookies on success.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      SignupRequest  true  "Signup credentials"
// @Success      201   {object}  SuccessResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /api/auth/signup [post]
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

// Login godoc
// @Summary      Log in
// @Description  Authenticate with email and password. Sets httpOnly auth cookies on success.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      LoginRequest  true  "Login credentials"
// @Success      200   {object}  SuccessResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      429   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /api/auth/login [post]
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

// Refresh godoc
// @Summary      Refresh access token
// @Description  Exchange the refresh token cookie for a new access token. Rotates the refresh token.
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  UserResponse
// @Failure      401  {object}  ErrorResponse
// @Router       /api/auth/refresh [post]
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

// Logout godoc
// @Summary      Log out
// @Description  Invalidate the current session and clear auth cookies.
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  MessageResponse
// @Router       /api/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	rawToken, _ := c.Cookie("eol_refresh")
	if rawToken != "" {
		h.auth.Logout(c.Request.Context(), rawToken)
	}
	h.clearAuthCookies(c)
	ok(c, gin.H{"message": "logged out"})
}

// Me godoc
// @Summary      Get current user
// @Description  Returns the authenticated user's profile.
// @Tags         Auth
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Success      200  {object}  UserResponse
// @Failure      401  {object}  ErrorResponse
// @Router       /api/auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	userID := middleware.GetUserID(c)
	user, err := h.auth.GetMe(c.Request.Context(), userID)
	if err != nil || user == nil {
		unauthorized(c, "user not found")
		return
	}
	ok(c, gin.H{"user": user})
}

// UpdateMe godoc
// @Summary      Update profile
// @Description  Update the authenticated user's first name and/or last name.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        body  body      UpdateMeRequest  true  "Fields to update"
// @Success      200   {object}  UserResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /api/auth/me [patch]
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

// ChangePassword godoc
// @Summary      Change password
// @Description  Change the authenticated user's password. Requires current password (unless the account was created via OAuth).
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        body  body      ChangePasswordRequest  true  "Password change payload"
// @Success      200   {object}  MessageResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /api/auth/change-password [post]
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
