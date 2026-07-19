package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/eternal-orbit-labs/gateway/internal/middleware"
	"github.com/eternal-orbit-labs/gateway/internal/service"
)

type APIKeyHandler struct {
	svc *service.APIKeyService
}

func NewAPIKeyHandler(svc *service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{svc: svc}
}

// Create godoc
// @Summary      Create API key
// @Description  Generate a new personal API key. The raw value is only returned once.
// @Tags         API Keys
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        body  body      CreateAPIKeyRequest   true  "API key label"
// @Success      201   {object}  CreateAPIKeyResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /api/auth/apikeys [post]
func (h *APIKeyHandler) Create(c *gin.Context) {
	var req struct {
		Label string `json:"label" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		validationError(c, err)
		return
	}

	userID := middleware.GetUserID(c)
	result, err := h.svc.Create(c.Request.Context(), userID, req.Label)
	if err != nil {
		internalError(c)
		return
	}

	created(c, gin.H{
		"key":       result.Key,
		"raw_value": result.RawValue,
	})
}

// List godoc
// @Summary      List API keys
// @Description  Returns all personal API keys for the authenticated user.
// @Tags         API Keys
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Success      200  {object}  ListAPIKeysResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/auth/apikeys [get]
func (h *APIKeyHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	keys, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		internalError(c)
		return
	}
	ok(c, gin.H{"keys": keys})
}

// Delete godoc
// @Summary      Delete API key
// @Description  Permanently delete a personal API key by ID.
// @Tags         API Keys
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        id   path      string  true  "API Key ID"
// @Success      200  {object}  MessageResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Router       /api/auth/apikeys/{id} [delete]
func (h *APIKeyHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id, userID); err != nil {
		badRequest(c, err.Error())
		return
	}
	ok(c, gin.H{"message": "api key deleted"})
}
