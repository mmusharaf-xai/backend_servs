package domain

import "time"

type AppOrganization struct {
	ID              string    `json:"id"`
	AppSlug         string    `json:"app_slug"`
	Name            string    `json:"name"`
	CreatedByUserID string    `json:"created_by_user_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type AppOrgMembership struct {
	OrgID    string    `json:"org_id"`
	UserID   string    `json:"user_id"`
	Role     string    `json:"role"`
	Status   string    `json:"status"`
	JoinedAt time.Time `json:"joined_at"`
}

type AppOrgInvite struct {
	ID              string     `json:"id"`
	OrgID           string     `json:"org_id"`
	Email           string     `json:"email"`
	FirstName       string     `json:"first_name"`
	LastName        string     `json:"last_name"`
	Phone           *string    `json:"phone,omitempty"`
	InvitedByUserID string     `json:"invited_by_user_id"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	AcceptedAt      *time.Time `json:"accepted_at,omitempty"`
}

// AppOrgListItem is returned in list responses for active memberships.
type AppOrgListItem struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Role   string `json:"role"`
	Status string `json:"status"`
}

// AppOrgInviteListItem is returned in list responses for pending invites.
type AppOrgInviteListItem struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	OrgName   string    `json:"org_name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// AppOrgMemberListItem is a user in an organization membership list.
type AppOrgMemberListItem struct {
	ID        string    `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Phone     *string   `json:"phone,omitempty"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	JoinedAt  time.Time `json:"joined_at"`
}

// AppOrgPendingInviteListItem is a pending invite shown on the org users page.
type AppOrgPendingInviteListItem struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Phone     *string   `json:"phone,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// OrgUserListItem is a unified row for the org users table (member or pending invite).
type OrgUserListItem struct {
	ID        string  `json:"id"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Email     string  `json:"email"`
	Phone     *string `json:"phone,omitempty"`
	Role      string  `json:"role"`
	Status    string  `json:"status"`
}

// OrgUserListFilters holds optional server-side filters for listing org users.
// Empty string means no filter on that field.
type OrgUserListFilters struct {
	Q              string // OR search across first_name, last_name, email, phone
	FirstName      string
	LastName       string
	Email          string
	Phone          string
	Status         string // "active", "deactive", or "" (all)
	TeamMembership string // "in", "not_in" — team member listing scope; defaults to "in"
	TeamID         string // required with team_membership=not_in for org-scoped exclusion
}

type AppOrgTeam struct {
	ID              string    `json:"id"`
	OrgID           string    `json:"org_id"`
	Name            string    `json:"name"`
	CreatedByUserID string    `json:"created_by_user_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type TeamCreatedBy struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type TeamListItem struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	MemberCount int           `json:"member_count"`
	CreatedBy   TeamCreatedBy `json:"created_by"`
	CreatedAt   time.Time     `json:"created_at"`
}

type TeamListFilters struct {
	Q    string // search team name
	Name string
}
