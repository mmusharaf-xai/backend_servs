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

// List godoc
// @Summary      List apps
// @Description  Returns a paginated list of apps. Supports cursor-based pagination and search.
// @Tags         Apps
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        q       query     string  false  "Search query"
// @Param        cursor  query     string  false  "Pagination cursor"
// @Param        limit   query     int     false  "Results per page (max 100)"  default(20)
// @Success      200     {object}  AppListResponse
// @Failure      401     {object}  ErrorResponse
// @Failure      500     {object}  ErrorResponse
// @Router       /api/apps [get]
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

// Get godoc
// @Summary      Get app by slug
// @Description  Returns a single app by its URL slug.
// @Tags         Apps
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        slug  path      string  true  "App slug"
// @Success      200   {object}  AppGetResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /api/apps/{slug} [get]
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
