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

// List godoc
// @Summary      List user's organizations
// @Description  Returns the authenticated user's organizations and pending invites for the given app.
// @Tags         Organizations
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        slug  path      string  true  "App slug"
// @Success      200   {object}  OrgListResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /api/apps/{slug}/orgs [get]
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

// Create godoc
// @Summary      Create organization
// @Description  Create a new organization under the given app. The authenticated user becomes the owner.
// @Tags         Organizations
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        slug  path      string          true  "App slug"
// @Param        body  body      CreateOrgRequest  true  "Organization details"
// @Success      201   {object}  OrgResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Failure      409   {object}  ErrorResponse
// @Failure      500   {object}  ErrorResponse
// @Router       /api/apps/{slug}/orgs [post]
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

// Get godoc
// @Summary      Get organization
// @Description  Returns organization details. Requires the user to be a member.
// @Tags         Organizations
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        slug   path      string  true  "App slug"
// @Param        orgId  path      string  true  "Organization ID"
// @Success      200    {object}  OrgResponse
// @Failure      401    {object}  ErrorResponse
// @Failure      403    {object}  ErrorResponse
// @Failure      404    {object}  ErrorResponse
// @Failure      500    {object}  ErrorResponse
// @Router       /api/apps/{slug}/orgs/{orgId} [get]
func (h *AppOrgHandler) Get(c *gin.Context) {
	userID := middleware.GetUserID(c)

	org, err := h.svc.GetOrg(c.Request.Context(), c.Param("slug"), userID, c.Param("orgId"))
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	ok(c, gin.H{"organization": org})
}

// AcceptInvite godoc
// @Summary      Accept invite
// @Description  Accept a pending organization invite. The user must be the invite recipient.
// @Tags         Invites
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        slug      path      string  true  "App slug"
// @Param        inviteId  path      string  true  "Invite ID"
// @Success      200       {object}  OrgResponse
// @Failure      401       {object}  ErrorResponse
// @Failure      403       {object}  ErrorResponse
// @Failure      404       {object}  ErrorResponse
// @Failure      500       {object}  ErrorResponse
// @Router       /api/apps/{slug}/invites/{inviteId}/accept [post]
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

// CreateMember godoc
// @Summary      Add member
// @Description  Add an existing user to the organization by email.
// @Tags         Members
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        slug   path  string              true  "App slug"
// @Param        orgId  path  string              true  "Organization ID"
// @Param        body   body  CreateMemberRequest  true  "Member details"
// @Success      204    "No Content"
// @Failure      400    {object}  ErrorResponse
// @Failure      401    {object}  ErrorResponse
// @Failure      403    {object}  ErrorResponse
// @Failure      404    {object}  ErrorResponse
// @Failure      409    {object}  ErrorResponse
// @Failure      500    {object}  ErrorResponse
// @Router       /api/apps/{slug}/orgs/{orgId}/members [post]
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

// CreateInvite godoc
// @Summary      Create invite
// @Description  Send an invitation to join the organization.
// @Tags         Invites
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        slug   path      string              true  "App slug"
// @Param        orgId  path      string              true  "Organization ID"
// @Param        body   body      CreateInviteRequest  true  "Invite details"
// @Success      201    {object}  InviteResponse
// @Failure      400    {object}  ErrorResponse
// @Failure      401    {object}  ErrorResponse
// @Failure      403    {object}  ErrorResponse
// @Failure      404    {object}  ErrorResponse
// @Failure      409    {object}  ErrorResponse
// @Failure      500    {object}  ErrorResponse
// @Router       /api/apps/{slug}/orgs/{orgId}/invites [post]
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

// UpdateInvite godoc
// @Summary      Update invite
// @Description  Update a pending invite's details (email, name, phone, status).
// @Tags         Invites
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        slug      path      string              true  "App slug"
// @Param        orgId     path      string              true  "Organization ID"
// @Param        inviteId  path      string              true  "Invite ID"
// @Param        body      body      UpdateInviteRequest  true  "Updated invite details"
// @Success      200       {object}  InviteResponse
// @Failure      400       {object}  ErrorResponse
// @Failure      401       {object}  ErrorResponse
// @Failure      403       {object}  ErrorResponse
// @Failure      404       {object}  ErrorResponse
// @Failure      500       {object}  ErrorResponse
// @Router       /api/apps/{slug}/orgs/{orgId}/invites/{inviteId} [patch]
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

// DeleteInvite godoc
// @Summary      Delete invite
// @Description  Permanently delete a pending invitation.
// @Tags         Invites
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        slug      path  string  true  "App slug"
// @Param        orgId     path  string  true  "Organization ID"
// @Param        inviteId  path  string  true  "Invite ID"
// @Success      204       "No Content"
// @Failure      401       {object}  ErrorResponse
// @Failure      403       {object}  ErrorResponse
// @Failure      404       {object}  ErrorResponse
// @Failure      500       {object}  ErrorResponse
// @Router       /api/apps/{slug}/orgs/{orgId}/invites/{inviteId} [delete]
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

// UpdateMember godoc
// @Summary      Update member status
// @Description  Activate or deactivate an organization member.
// @Tags         Members
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        slug      path  string               true  "App slug"
// @Param        orgId     path  string               true  "Organization ID"
// @Param        memberId  path  string               true  "Member user ID"
// @Param        body      body  UpdateMemberRequest   true  "New status"
// @Success      204       "No Content"
// @Failure      400       {object}  ErrorResponse
// @Failure      401       {object}  ErrorResponse
// @Failure      403       {object}  ErrorResponse
// @Failure      404       {object}  ErrorResponse
// @Failure      500       {object}  ErrorResponse
// @Router       /api/apps/{slug}/orgs/{orgId}/members/{memberId} [patch]
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

// RemoveMember godoc
// @Summary      Remove member
// @Description  Remove a member from the organization. Cannot remove the owner.
// @Tags         Members
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        slug      path  string  true  "App slug"
// @Param        orgId     path  string  true  "Organization ID"
// @Param        memberId  path  string  true  "Member user ID"
// @Success      204       "No Content"
// @Failure      401       {object}  ErrorResponse
// @Failure      403       {object}  ErrorResponse
// @Failure      404       {object}  ErrorResponse
// @Failure      500       {object}  ErrorResponse
// @Router       /api/apps/{slug}/orgs/{orgId}/members/{memberId} [delete]
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

// ListMembers godoc
// @Summary      List org members
// @Description  Returns a paginated list of organization members with optional filters.
// @Tags         Members
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        slug        path      string  true   "App slug"
// @Param        orgId       path      string  true   "Organization ID"
// @Param        page        query     int     false  "Page number"               default(1)
// @Param        limit       query     int     false  "Results per page (max 100)" default(20)
// @Param        q           query     string  false  "Search across name, email, phone"
// @Param        first_name  query     string  false  "Filter by first name"
// @Param        last_name   query     string  false  "Filter by last name"
// @Param        email       query     string  false  "Filter by email"
// @Param        phone       query     string  false  "Filter by phone"
// @Param        status      query     string  false  "Filter by status (active, deactive)"
// @Success      200         {object}  MemberListResponse
// @Failure      401         {object}  ErrorResponse
// @Failure      403         {object}  ErrorResponse
// @Failure      404         {object}  ErrorResponse
// @Failure      500         {object}  ErrorResponse
// @Router       /api/apps/{slug}/orgs/{orgId}/members [get]
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

// ListTeams godoc
// @Summary      List teams
// @Description  Returns a paginated list of teams in the organization.
// @Tags         Teams
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        slug   path      string  true   "App slug"
// @Param        orgId  path      string  true   "Organization ID"
// @Param        page   query     int     false  "Page number"               default(1)
// @Param        limit  query     int     false  "Results per page (max 100)" default(20)
// @Param        q      query     string  false  "Search by team name"
// @Param        name   query     string  false  "Filter by team name"
// @Success      200    {object}  TeamListResponse
// @Failure      401    {object}  ErrorResponse
// @Failure      403    {object}  ErrorResponse
// @Failure      404    {object}  ErrorResponse
// @Failure      500    {object}  ErrorResponse
// @Router       /api/apps/{slug}/orgs/{orgId}/teams [get]
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

// CreateTeam godoc
// @Summary      Create team
// @Description  Create a new team in the organization.
// @Tags         Teams
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        slug   path      string           true  "App slug"
// @Param        orgId  path      string           true  "Organization ID"
// @Param        body   body      CreateTeamRequest  true  "Team details"
// @Success      201    {object}  TeamResponse
// @Failure      400    {object}  ErrorResponse
// @Failure      401    {object}  ErrorResponse
// @Failure      403    {object}  ErrorResponse
// @Failure      404    {object}  ErrorResponse
// @Failure      409    {object}  ErrorResponse
// @Failure      500    {object}  ErrorResponse
// @Router       /api/apps/{slug}/orgs/{orgId}/teams [post]
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

// GetTeam godoc
// @Summary      Get team
// @Description  Returns team details by ID.
// @Tags         Teams
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        slug    path      string  true  "App slug"
// @Param        orgId   path      string  true  "Organization ID"
// @Param        teamId  path      string  true  "Team ID"
// @Success      200     {object}  TeamResponse
// @Failure      401     {object}  ErrorResponse
// @Failure      403     {object}  ErrorResponse
// @Failure      404     {object}  ErrorResponse
// @Failure      500     {object}  ErrorResponse
// @Router       /api/apps/{slug}/orgs/{orgId}/teams/{teamId} [get]
func (h *AppOrgHandler) GetTeam(c *gin.Context) {
	userID := middleware.GetUserID(c)

	team, err := h.svc.GetTeam(c.Request.Context(), c.Param("slug"), userID, c.Param("orgId"), c.Param("teamId"))
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	ok(c, gin.H{"team": team})
}

// ListTeamMembers godoc
// @Summary      List team members
// @Description  Returns a paginated list of members in the team.
// @Tags         Teams
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        slug    path      string  true   "App slug"
// @Param        orgId   path      string  true   "Organization ID"
// @Param        teamId  path      string  true   "Team ID"
// @Param        page    query     int     false  "Page number"               default(1)
// @Param        limit   query     int     false  "Results per page (max 100)" default(20)
// @Param        q       query     string  false  "Search across name, email, phone"
// @Param        status  query     string  false  "Filter by status"
// @Success      200     {object}  TeamMemberListResponse
// @Failure      401     {object}  ErrorResponse
// @Failure      403     {object}  ErrorResponse
// @Failure      404     {object}  ErrorResponse
// @Failure      500     {object}  ErrorResponse
// @Router       /api/apps/{slug}/orgs/{orgId}/teams/{teamId}/members [get]
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

// AddTeamMembers godoc
// @Summary      Add team members
// @Description  Add one or more organization members to the team.
// @Tags         Teams
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        slug    path  string                 true  "App slug"
// @Param        orgId   path  string                 true  "Organization ID"
// @Param        teamId  path  string                 true  "Team ID"
// @Param        body    body  AddTeamMembersRequest   true  "User IDs to add"
// @Success      204     "No Content"
// @Failure      400     {object}  ErrorResponse
// @Failure      401     {object}  ErrorResponse
// @Failure      403     {object}  ErrorResponse
// @Failure      404     {object}  ErrorResponse
// @Failure      500     {object}  ErrorResponse
// @Router       /api/apps/{slug}/orgs/{orgId}/teams/{teamId}/members [post]
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

// RemoveTeamMember godoc
// @Summary      Remove team member
// @Description  Remove a single member from the team.
// @Tags         Teams
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        slug    path  string  true  "App slug"
// @Param        orgId   path  string  true  "Organization ID"
// @Param        teamId  path  string  true  "Team ID"
// @Param        userId  path  string  true  "User ID to remove"
// @Success      204     "No Content"
// @Failure      401     {object}  ErrorResponse
// @Failure      403     {object}  ErrorResponse
// @Failure      404     {object}  ErrorResponse
// @Failure      500     {object}  ErrorResponse
// @Router       /api/apps/{slug}/orgs/{orgId}/teams/{teamId}/members/{userId} [delete]
func (h *AppOrgHandler) RemoveTeamMember(c *gin.Context) {
	userID := middleware.GetUserID(c)

	err := h.svc.RemoveTeamMember(c.Request.Context(), c.Param("slug"), userID, c.Param("orgId"), c.Param("teamId"), c.Param("userId"))
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	c.Status(204)
}

// BulkRemoveTeamMembers godoc
// @Summary      Bulk remove team members
// @Description  Remove multiple members from the team in a single request.
// @Tags         Teams
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Security     BearerAPIKey
// @Param        slug    path  string                        true  "App slug"
// @Param        orgId   path  string                        true  "Organization ID"
// @Param        teamId  path  string                        true  "Team ID"
// @Param        body    body  BulkRemoveTeamMembersRequest   true  "User IDs to remove"
// @Success      204     "No Content"
// @Failure      400     {object}  ErrorResponse
// @Failure      401     {object}  ErrorResponse
// @Failure      403     {object}  ErrorResponse
// @Failure      404     {object}  ErrorResponse
// @Failure      500     {object}  ErrorResponse
// @Router       /api/apps/{slug}/orgs/{orgId}/teams/{teamId}/members [delete]
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
