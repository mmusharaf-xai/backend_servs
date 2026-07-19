package handler

import (
	"github.com/eternal-orbit-labs/gateway/internal/domain"
	"github.com/eternal-orbit-labs/gateway/internal/pagination"
)

// ── Error ────────────────────────────────────────────────────────────

// ErrorResponse is the standard error envelope.
type ErrorResponse struct {
	Error string `json:"error" example:"error message"`
}

// ── Auth ─────────────────────────────────────────────────────────────

type SignupRequest struct {
	Email     string `json:"email" binding:"required,email" example:"user@example.com"`
	Password  string `json:"password" binding:"required,min=8" example:"secureP@ss1"`
	FirstName string `json:"first_name" binding:"required" example:"John"`
	LastName  string `json:"last_name" example:"Doe"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required" example:"secureP@ss1"`
}

type SuccessResponse struct {
	Success bool `json:"success" example:"true"`
}

type MessageResponse struct {
	Message string `json:"message" example:"operation completed"`
}

type UserResponse struct {
	User *domain.User `json:"user"`
}

type UpdateMeRequest struct {
	FirstName *string `json:"first_name" example:"Jane"`
	LastName  *string `json:"last_name" example:"Smith"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" example:"oldP@ss1"`
	NewPassword     string `json:"new_password" binding:"required,min=8" example:"newSecureP@ss1"`
}

// ── API Keys ─────────────────────────────────────────────────────────

type CreateAPIKeyRequest struct {
	Label string `json:"label" binding:"required" example:"my-key"`
}

type CreateAPIKeyResponse struct {
	Key      *domain.PersonalAPIKey `json:"key"`
	RawValue string                 `json:"raw_value" example:"eol_k1_abc123..."`
}

type ListAPIKeysResponse struct {
	Keys []domain.PersonalAPIKey `json:"keys"`
}

// ── Apps ─────────────────────────────────────────────────────────────

type AppListResponse struct {
	Apps       []*domain.App `json:"apps"`
	NextCursor string        `json:"next_cursor" example:""`
}

type AppGetResponse struct {
	App *domain.App `json:"app"`
}

// ── Organizations ────────────────────────────────────────────────────

type CreateOrgRequest struct {
	Name string `json:"name" binding:"required" example:"Acme Corp"`
}

type OrgResponse struct {
	Organization *domain.AppOrgListItem `json:"organization"`
}

type OrgListResponse struct {
	Organizations []*domain.AppOrgListItem       `json:"organizations"`
	Invites       []*domain.AppOrgInviteListItem `json:"invites"`
}

// ── Members ──────────────────────────────────────────────────────────

type CreateMemberRequest struct {
	Email     string  `json:"email" binding:"required,email" example:"member@example.com"`
	FirstName string  `json:"first_name" example:"Alice"`
	LastName  string  `json:"last_name" example:"Johnson"`
	Phone     *string `json:"phone" example:"+1234567890"`
}

type UpdateMemberRequest struct {
	Status string `json:"status" binding:"required" example:"active"`
}

type MemberListResponse struct {
	Data []*domain.OrgUserListItem `json:"data"`
	Meta pagination.Meta           `json:"meta"`
}

// ── Invites ──────────────────────────────────────────────────────────

type CreateInviteRequest struct {
	Email     string  `json:"email" binding:"required,email" example:"invite@example.com"`
	FirstName string  `json:"first_name" example:"Bob"`
	LastName  string  `json:"last_name" example:"Williams"`
	Phone     *string `json:"phone" example:"+1987654321"`
}

type UpdateInviteRequest struct {
	Email     string  `json:"email" binding:"required,email" example:"invite@example.com"`
	FirstName string  `json:"first_name" example:"Bob"`
	LastName  string  `json:"last_name" example:"Williams"`
	Phone     *string `json:"phone" example:"+1987654321"`
	Status    string  `json:"status" example:"active"`
}

type InviteResponse struct {
	Invite *domain.AppOrgInvite `json:"invite"`
}

// ── Teams ────────────────────────────────────────────────────────────

type CreateTeamRequest struct {
	Name string `json:"name" binding:"required" example:"Engineering"`
}

type TeamResponse struct {
	Team *domain.TeamListItem `json:"team"`
}

type TeamListResponse struct {
	Data []*domain.TeamListItem `json:"data"`
	Meta pagination.Meta        `json:"meta"`
}

type AddTeamMembersRequest struct {
	UserIDs []string `json:"user_ids" binding:"required" example:"[\"user-id-1\",\"user-id-2\"]"`
}

type BulkRemoveTeamMembersRequest struct {
	UserIDs []string `json:"user_ids" binding:"required"`
}

type TeamMemberListResponse struct {
	Data []*domain.OrgUserListItem `json:"data"`
	Meta pagination.Meta           `json:"meta"`
}

// ── Sidebar ──────────────────────────────────────────────────────────

type SidebarItem struct {
	ID       string `json:"id" example:"dashboard"`
	Label    string `json:"label" example:"Dashboard"`
	Icon     string `json:"icon" example:"layout-dashboard"`
	Path     string `json:"path" example:"dashboard"`
	Position string `json:"position,omitempty" example:"top"`
}

type SidebarSection struct {
	ID              string        `json:"id" example:"main"`
	Label           *string       `json:"label,omitempty" example:"Main"`
	Collapsible     bool          `json:"collapsible,omitempty" example:"true"`
	DefaultExpanded bool          `json:"default_expanded,omitempty" example:"true"`
	Items           []SidebarItem `json:"items"`
}

type SidebarResponse struct {
	AppSlug  string           `json:"app_slug" example:"my-app"`
	OrgID    string           `json:"org_id" example:"org-123"`
	Sections []SidebarSection `json:"sections"`
}

// ── Health ───────────────────────────────────────────────────────────

type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}
