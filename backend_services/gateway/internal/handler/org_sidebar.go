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
