package domain

import "context"

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByGoogleID(ctx context.Context, googleID string) (*User, error)
	Update(ctx context.Context, user *User) error
}

type RefreshSessionRepository interface {
	Create(ctx context.Context, session *RefreshSession) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*RefreshSession, error)
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
	DeleteAllForUser(ctx context.Context, userID string) error
}

type PersonalAPIKeyRepository interface {
	Create(ctx context.Context, key *PersonalAPIKey, secureValue string) error
	GetBySecureValue(ctx context.Context, secureValue string) (*PersonalAPIKey, error)
	ListByUserID(ctx context.Context, userID string) ([]PersonalAPIKey, error)
	Delete(ctx context.Context, id, userID string) error
	TouchLastUsed(ctx context.Context, id string) error
}

type RateLimiterRepository interface {
	IncrementLoginAttempts(ctx context.Context, key string) (int64, error)
	GetLoginAttempts(ctx context.Context, key string) (int64, error)
}

type AppRepository interface {
	List(ctx context.Context, q, cursor string, limit int) (apps []*App, nextCursor string, err error)
	GetBySlug(ctx context.Context, slug string) (*App, error)
}

type AppOrgRepository interface {
	ListMembershipsForUser(ctx context.Context, appSlug, userID string) ([]*AppOrgListItem, error)
	ListPendingInvitesForEmail(ctx context.Context, appSlug, email string) ([]*AppOrgInviteListItem, error)
	CreateOrgWithOwner(ctx context.Context, appSlug, userID, name string) (*AppOrgListItem, error)
	GetOrgForMember(ctx context.Context, appSlug, userID, orgID string) (*AppOrgListItem, error)
	ListMembers(ctx context.Context, orgID string) ([]*AppOrgMemberListItem, error)
	ListOrgUsers(ctx context.Context, orgID string, filters OrgUserListFilters, limit, offset int) ([]*OrgUserListItem, int, error)
	ListPendingInvitesForOrg(ctx context.Context, orgID string) ([]*AppOrgPendingInviteListItem, error)
	IsMember(ctx context.Context, orgID, userID string) (bool, error)
	IsOwner(ctx context.Context, orgID, userID string) (bool, error)
	GetInviteByID(ctx context.Context, inviteID string) (*AppOrgInvite, string, error)
	CreateInvite(ctx context.Context, orgID, email, invitedByUserID string, firstName, lastName string, phone *string) (*AppOrgInvite, error)
	AddActiveMember(ctx context.Context, orgID, memberUserID string) error
	UpdatePendingInvite(ctx context.Context, orgID, inviteID, email, firstName, lastName string, phone *string, active bool) (*AppOrgInvite, error)
	GetDeactivatedInviteForOrg(ctx context.Context, orgID, email string) (*AppOrgInvite, error)
	DeactivateInvite(ctx context.Context, orgID, inviteID string) error
	DeletePendingInvite(ctx context.Context, orgID, inviteID string) error
	DeleteMember(ctx context.Context, orgID, memberUserID string) error
	DeleteInvitePermanently(ctx context.Context, orgID, inviteID string) error
	UpdateMemberStatus(ctx context.Context, orgID, memberUserID, status string) error
	RemoveMember(ctx context.Context, orgID, memberUserID string) error
	HasMemberWithEmail(ctx context.Context, orgID, email string) (bool, error)
	HasPendingInviteWithEmail(ctx context.Context, orgID, email, excludeInviteID string) (bool, error)
	AcceptInvite(ctx context.Context, inviteID, userID string) (*AppOrgListItem, error)
	AcceptPendingInvitesForEmail(ctx context.Context, email, userID string) error
	ListTeams(ctx context.Context, orgID string, filters TeamListFilters, limit, offset int) ([]*TeamListItem, int, error)
	CreateTeam(ctx context.Context, orgID, name, createdByUserID string) (*TeamListItem, error)
	GetTeam(ctx context.Context, orgID, teamID string) (*TeamListItem, error)
	ListTeamMembers(ctx context.Context, teamID, orgID string, filters OrgUserListFilters, limit, offset int) ([]*OrgUserListItem, int, error)
	AddTeamMembers(ctx context.Context, teamID string, userIDs []string) error
	RemoveTeamMember(ctx context.Context, teamID, userID string) error
	BulkRemoveTeamMembers(ctx context.Context, teamID string, userIDs []string) error
	IsTeamMember(ctx context.Context, teamID, userID string) (bool, error)
	IsTeamInviteMember(ctx context.Context, teamID, inviteID string) (bool, error)
	AreOrgMembers(ctx context.Context, orgID string, userIDs []string) (bool, error)
	AreOrgInvites(ctx context.Context, orgID string, inviteIDs []string) (bool, error)
}
