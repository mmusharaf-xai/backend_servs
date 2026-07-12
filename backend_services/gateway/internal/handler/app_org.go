package handler

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/eternal-orbit-labs/gateway/internal/domain"
	"github.com/eternal-orbit-labs/gateway/internal/middleware"
	"github.com/eternal-orbit-labs/gateway/internal/pagination"
	"github.com/eternal-orbit-labs/gateway/internal/service"
)

type AppOrgHandler struct {
	svc   *service.AppOrgService
	users domain.UserRepository
}

func NewAppOrgHandler(svc *service.AppOrgService, users domain.UserRepository) *AppOrgHandler {
	return &AppOrgHandler{svc: svc, users: users}
}

func (h *AppOrgHandler) userEmail(c *gin.Context, userID string) (string, error) {
	user, err := h.users.GetByID(c.Request.Context(), userID)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", errors.New("user not found")
	}
	return user.Email, nil
}

func (h *AppOrgHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	email, err := h.userEmail(c, userID)
	if err != nil {
		unauthorized(c, "user not found")
		return
	}

	result, err := h.svc.ListForUser(c.Request.Context(), c.Param("slug"), userID, email)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	ok(c, result)
}

func (h *AppOrgHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	org, err := h.svc.Create(c.Request.Context(), c.Param("slug"), userID, req.Name)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	created(c, gin.H{"organization": org})
}

func (h *AppOrgHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)

	org, err := h.svc.GetOrg(c.Request.Context(), c.Param("slug"), userID, c.Param("orgId"))
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	ok(c, gin.H{"organization": org})
}

func (h *AppOrgHandler) AcceptInvite(c *gin.Context) {
	userID := middleware.GetUserID(c)
	email, err := h.userEmail(c, userID)
	if err != nil {
		unauthorized(c, "user not found")
		return
	}

	org, err := h.svc.AcceptInvite(c.Request.Context(), c.Param("slug"), userID, email, c.Param("inviteId"))
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	ok(c, gin.H{"organization": org})
}

func (h *AppOrgHandler) CreateMember(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		Email     string  `json:"email" binding:"required,email"`
		FirstName string  `json:"first_name"`
		LastName  string  `json:"last_name"`
		Phone     *string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	err := h.svc.AddMember(
		c.Request.Context(),
		c.Param("slug"),
		userID,
		c.Param("orgId"),
		req.Email,
		req.FirstName,
		req.LastName,
		req.Phone,
	)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	c.Status(204)
}

func (h *AppOrgHandler) CreateInvite(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		Email     string  `json:"email" binding:"required,email"`
		FirstName string  `json:"first_name"`
		LastName  string  `json:"last_name"`
		Phone     *string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	invite, err := h.svc.CreateInvite(
		c.Request.Context(),
		c.Param("slug"),
		userID,
		c.Param("orgId"),
		req.Email,
		req.FirstName,
		req.LastName,
		req.Phone,
	)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	created(c, gin.H{"invite": invite})
}

func (h *AppOrgHandler) UpdateInvite(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		Email     string  `json:"email" binding:"required,email"`
		FirstName string  `json:"first_name"`
		LastName  string  `json:"last_name"`
		Phone     *string `json:"phone"`
		Status    string  `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}
	if req.Status == "" {
		req.Status = "active"
	}

	invite, err := h.svc.UpdateInvite(
		c.Request.Context(),
		c.Param("slug"),
		userID,
		c.Param("orgId"),
		c.Param("inviteId"),
		req.Email,
		req.FirstName,
		req.LastName,
		req.Phone,
		req.Status,
	)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	ok(c, gin.H{"invite": invite})
}

func (h *AppOrgHandler) DeleteInvite(c *gin.Context) {
	userID := middleware.GetUserID(c)

	err := h.svc.DeleteInvite(
		c.Request.Context(),
		c.Param("slug"),
		userID,
		c.Param("orgId"),
		c.Param("inviteId"),
	)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	c.Status(204)
}

func (h *AppOrgHandler) UpdateMember(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	err := h.svc.UpdateMemberStatus(
		c.Request.Context(),
		c.Param("slug"),
		userID,
		c.Param("orgId"),
		c.Param("memberId"),
		req.Status,
	)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	c.Status(204)
}

func (h *AppOrgHandler) RemoveMember(c *gin.Context) {
	userID := middleware.GetUserID(c)

	err := h.svc.RemoveMember(
		c.Request.Context(),
		c.Param("slug"),
		userID,
		c.Param("orgId"),
		c.Param("memberId"),
	)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	c.Status(204)
}

func parseOrgUserListFilters(c *gin.Context) domain.OrgUserListFilters {
	return domain.OrgUserListFilters{
		Q:              strings.TrimSpace(c.Query("q")),
		FirstName:      strings.TrimSpace(c.Query("first_name")),
		LastName:       strings.TrimSpace(c.Query("last_name")),
		Email:          strings.TrimSpace(c.Query("email")),
		Phone:          strings.TrimSpace(c.Query("phone")),
		Status:         strings.TrimSpace(c.Query("status")),
		TeamMembership: strings.TrimSpace(c.Query("team_membership")),
		TeamID:         strings.TrimSpace(c.Query("team_id")),
	}
}

func (h *AppOrgHandler) ListMembers(c *gin.Context) {
	userID := middleware.GetUserID(c)
	params := pagination.ParsePageLimit(c)
	filters := parseOrgUserListFilters(c)

	result, err := h.svc.ListMembers(c.Request.Context(), c.Param("slug"), userID, c.Param("orgId"), params, filters)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	ok(c, result)
}

func parseTeamListFilters(c *gin.Context) domain.TeamListFilters {
	q := strings.TrimSpace(c.Query("q"))
	name := strings.TrimSpace(c.Query("name"))
	if name == "" && q != "" {
		name = q
	}
	return domain.TeamListFilters{Q: q, Name: name}
}

func (h *AppOrgHandler) ListTeams(c *gin.Context) {
	userID := middleware.GetUserID(c)
	params := pagination.ParsePageLimit(c)
	filters := parseTeamListFilters(c)

	result, err := h.svc.ListTeams(c.Request.Context(), c.Param("slug"), userID, c.Param("orgId"), params, filters)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	ok(c, result)
}

func (h *AppOrgHandler) CreateTeam(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	team, err := h.svc.CreateTeam(c.Request.Context(), c.Param("slug"), userID, c.Param("orgId"), req.Name)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	created(c, gin.H{"team": team})
}

func (h *AppOrgHandler) GetTeam(c *gin.Context) {
	userID := middleware.GetUserID(c)

	team, err := h.svc.GetTeam(c.Request.Context(), c.Param("slug"), userID, c.Param("orgId"), c.Param("teamId"))
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	ok(c, gin.H{"team": team})
}

func (h *AppOrgHandler) ListTeamMembers(c *gin.Context) {
	userID := middleware.GetUserID(c)
	params := pagination.ParsePageLimit(c)
	filters := parseOrgUserListFilters(c)

	result, err := h.svc.ListTeamMembers(c.Request.Context(), c.Param("slug"), userID, c.Param("orgId"), c.Param("teamId"), params, filters)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	ok(c, result)
}

func (h *AppOrgHandler) AddTeamMembers(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		UserIDs []string `json:"user_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	err := h.svc.AddTeamMembers(c.Request.Context(), c.Param("slug"), userID, c.Param("orgId"), c.Param("teamId"), req.UserIDs)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	c.Status(204)
}

func (h *AppOrgHandler) RemoveTeamMember(c *gin.Context) {
	userID := middleware.GetUserID(c)

	err := h.svc.RemoveTeamMember(c.Request.Context(), c.Param("slug"), userID, c.Param("orgId"), c.Param("teamId"), c.Param("userId"))
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	c.Status(204)
}

func (h *AppOrgHandler) BulkRemoveTeamMembers(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		UserIDs []string `json:"user_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, err.Error())
		return
	}

	err := h.svc.BulkRemoveTeamMembers(c.Request.Context(), c.Param("slug"), userID, c.Param("orgId"), c.Param("teamId"), req.UserIDs)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	c.Status(204)
}

func (h *AppOrgHandler) handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrAppNotFound), errors.Is(err, service.ErrOrgNotFound), errors.Is(err, service.ErrInviteNotFound), errors.Is(err, service.ErrMemberNotFound), errors.Is(err, service.ErrUserNotFound), errors.Is(err, service.ErrTeamNotFound), errors.Is(err, service.ErrTeamMemberNotFound):
		notFound(c, err.Error())
	case errors.Is(err, service.ErrAppNotAvailable), errors.Is(err, service.ErrNotOrgOwner), errors.Is(err, service.ErrInviteNotForUser), errors.Is(err, service.ErrCannotRemoveOwner), errors.Is(err, service.ErrCannotUpdateOwner):
		forbidden(c, err.Error())
	case errors.Is(err, service.ErrOrgNameExists):
		conflict(c, "organization name already exists")
	case errors.Is(err, service.ErrTeamNameExists):
		conflict(c, "team name already exists")
	case errors.Is(err, service.ErrEmailAlreadyMember):
		conflict(c, "user with this email is already a member")
	case errors.Is(err, service.ErrEmailAlreadyInvited):
		conflict(c, "invite already pending for this email")
	default:
		if errors.Is(err, service.ErrInvalidMemberStatus) ||
			errors.Is(err, service.ErrUserNotOrgMember) ||
			errors.Is(err, service.ErrInviteNotOrgMember) ||
			strings.Contains(err.Error(), "name is required") ||
			strings.Contains(err.Error(), "name must be") ||
			strings.Contains(err.Error(), "email is required") ||
			strings.Contains(err.Error(), "user_ids is required") ||
			strings.Contains(err.Error(), "invite is not pending") ||
			strings.Contains(err.Error(), "invite already pending") {
			badRequest(c, err.Error())
			return
		}
		internalError(c)
	}
}
