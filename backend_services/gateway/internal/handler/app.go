package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/eternal-orbit-labs/gateway/internal/domain"
	"github.com/eternal-orbit-labs/gateway/internal/pagination"
	"github.com/eternal-orbit-labs/gateway/internal/service"
)

type AppHandler struct {
	svc *service.AppService
}

func NewAppHandler(svc *service.AppService) *AppHandler {
	return &AppHandler{svc: svc}
}

func (h *AppHandler) List(c *gin.Context) {
	q := c.Query("q")
	cursor := c.Query("cursor")

	limit := 0
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = pagination.NormalizeLimit(parsed)
		}
	}

	apps, nextCursor, err := h.svc.List(c.Request.Context(), q, cursor, limit)
	if err != nil {
		internalError(c)
		return
	}

	// Always return a JSON array (never null) for an empty catalog/page.
	if apps == nil {
		apps = []*domain.App{}
	}

	ok(c, gin.H{"apps": apps, "next_cursor": nextCursor})
}

func (h *AppHandler) Get(c *gin.Context) {
	slug := c.Param("slug")
	app, err := h.svc.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		internalError(c)
		return
	}
	if app == nil {
		notFound(c, "app not found")
		return
	}
	ok(c, gin.H{"app": app})
}
