package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/eternal-orbit-labs/gateway/internal/middleware"
	"github.com/eternal-orbit-labs/gateway/internal/service"
)

type OrgSidebarHandler struct {
	svc *service.OrgSidebarService
}

func NewOrgSidebarHandler(svc *service.OrgSidebarService) *OrgSidebarHandler {
	return &OrgSidebarHandler{svc: svc}
}

// Get godoc
// @Summary      Get org sidebar
// @Description  Returns the sidebar navigation configuration for an organization workspace.
// @Tags         Sidebar
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        slug   path      string  true  "App slug"
// @Param        orgId  path      string  true  "Organization ID"
// @Success      200    {object}  SidebarResponse
// @Failure      401    {object}  ErrorResponse
// @Failure      403    {object}  ErrorResponse
// @Failure      404    {object}  ErrorResponse
// @Failure      500    {object}  ErrorResponse
// @Router       /api/apps/{slug}/orgs/{orgId}/sidebar [get]
func (h *OrgSidebarHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)

	resp, err := h.svc.GetSidebar(c.Request.Context(), c.Param("slug"), userID, c.Param("orgId"))
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	ok(c, resp)
}

func (h *OrgSidebarHandler) handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrAppNotFound),
		errors.Is(err, service.ErrOrgNotFound),
		errors.Is(err, service.ErrSidebarNotConfigured):
		notFound(c, err.Error())
	case errors.Is(err, service.ErrAppNotAvailable):
		forbidden(c, err.Error())
	default:
		internalError(c)
	}
}
