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

func (h *APIKeyHandler) Create(c *gin.Context) {
	var req struct {
		Label string `json:"label" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
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

func (h *APIKeyHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	keys, err := h.svc.List(c.Request.Context(), userID)
	if err != nil {
		internalError(c)
		return
	}
	ok(c, gin.H{"keys": keys})
}

func (h *APIKeyHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id, userID); err != nil {
		badRequest(c, err.Error())
		return
	}
	ok(c, gin.H{"message": "api key deleted"})
}
