package service

import (
	"context"
	"strings"
	"testing"

	"github.com/eternal-orbit-labs/gateway/internal/domain"
	"github.com/eternal-orbit-labs/gateway/internal/pagination"
)

type fakeAppRepo struct {
	app *domain.App
}

func (f *fakeAppRepo) List(ctx context.Context, q, cursor string, limit int) ([]*domain.App, string, error) {
	return nil, "", nil
}

func (f *fakeAppRepo) GetBySlug(ctx context.Context, slug string) (*domain.App, error) {
	if f.app != nil && f.app.Slug == slug {
		return f.app, nil
	}
	return nil, nil
}

type fakeAppOrgRepo struct {
	org                *domain.AppOrgListItem
	members            []*domain.AppOrgMemberListItem
	pendingInvites     []*domain.AppOrgPendingInviteListItem
	teams              []*domain.TeamListItem
	teamMembers        []*domain.OrgUserListItem
	teamInviteMembers  []*domain.OrgUserListItem
	isOwner            bool
}

func (f *fakeAppOrgRepo) ListMembershipsForUser(ctx context.Context, appSlug, userID string) ([]*domain.AppOrgListItem, error) {
	return nil, nil
}

func (f *fakeAppOrgRepo) ListPendingInvitesForEmail(ctx context.Context, appSlug, email string) ([]*domain.AppOrgInviteListItem, error) {
	return nil, nil
}

func (f *fakeAppOrgRepo) CreateOrgWithOwner(ctx context.Context, appSlug, userID, name string) (*domain.AppOrgListItem, error) {
	return nil, nil
}

func (f *fakeAppOrgRepo) GetOrgForMember(ctx context.Context, appSlug, userID, orgID string) (*domain.AppOrgListItem, error) {
	return f.org, nil
}

func (f *fakeAppOrgRepo) ListMembers(ctx context.Context, orgID string) ([]*domain.AppOrgMemberListItem, error) {
	return f.members, nil
}

func (f *fakeAppOrgRepo) ListOrgUsers(ctx context.Context, orgID string, filters domain.OrgUserListFilters, limit, offset int) ([]*domain.OrgUserListItem, int, error) {
	all := make([]*domain.OrgUserListItem, 0, len(f.members)+len(f.pendingInvites))
	for _, member := range f.members {
		all = append(all, &domain.OrgUserListItem{
			ID:        member.ID,
			FirstName: member.FirstName,
			LastName:  member.LastName,
			Email:     member.Email,
			Phone:     member.Phone,
			Role:      member.Role,
			Status:    member.Status,
		})
	}
	for _, invite := range f.pendingInvites {
		all = append(all, &domain.OrgUserListItem{
			ID:        "invite:" + invite.ID,
			FirstName: invite.FirstName,
			LastName:  invite.LastName,
			Email:     invite.Email,
			Phone:     invite.Phone,
			Role:      "member",
			Status:    "active",
		})
	}

	if filters.TeamMembership == "not_in" && filters.TeamID != "" {
		onTeam := make(map[string]struct{}, len(f.teamMembers))
		for _, member := range f.teamMembers {
			onTeam[member.ID] = struct{}{}
		}
		onTeamInvites := make(map[string]struct{}, len(f.teamInviteMembers))
		for _, member := range f.teamInviteMembers {
			onTeamInvites[member.ID] = struct{}{}
		}
		filtered := make([]*domain.OrgUserListItem, 0, len(all))
		for _, user := range all {
			if strings.HasPrefix(user.ID, "invite:") {
				if _, ok := onTeamInvites[user.ID]; !ok {
					filtered = append(filtered, user)
				}
				continue
			}
			if _, ok := onTeam[user.ID]; !ok {
				filtered = append(filtered, user)
			}
		}
		all = filtered
	}

	if filters.Status != "" {
		filtered := make([]*domain.OrgUserListItem, 0, len(all))
		for _, user := range all {
			if user.Status == filters.Status {
				filtered = append(filtered, user)
			}
		}
		all = filtered
	}

	total := len(all)
	if offset >= total {
		return []*domain.OrgUserListItem{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func (f *fakeAppOrgRepo) ListPendingInvitesForOrg(ctx context.Context, orgID string) ([]*domain.AppOrgPendingInviteListItem, error) {
	return f.pendingInvites, nil
}

func (f *fakeAppOrgRepo) IsMember(ctx context.Context, orgID, userID string) (bool, error) {
	return false, nil
}

func (f *fakeAppOrgRepo) IsOwner(ctx context.Context, orgID, userID string) (bool, error) {
	return f.isOwner, nil
}

func (f *fakeAppOrgRepo) GetInviteByID(ctx context.Context, inviteID string) (*domain.AppOrgInvite, string, error) {
	return nil, "", nil
}

func (f *fakeAppOrgRepo) CreateInvite(ctx context.Context, orgID, email, invitedByUserID, firstName, lastName string, phone *string) (*domain.AppOrgInvite, error) {
	return nil, nil
}

func (f *fakeAppOrgRepo) AddActiveMember(ctx context.Context, orgID, memberUserID string) error {
	return nil
}

func (f *fakeAppOrgRepo) UpdatePendingInvite(ctx context.Context, orgID, inviteID, email, firstName, lastName string, phone *string, active bool) (*domain.AppOrgInvite, error) {
	return nil, nil
}

func (f *fakeAppOrgRepo) GetDeactivatedInviteForOrg(ctx context.Context, orgID, email string) (*domain.AppOrgInvite, error) {
	return nil, nil
}

func (f *fakeAppOrgRepo) DeactivateInvite(ctx context.Context, orgID, inviteID string) error {
	return nil
}

func (f *fakeAppOrgRepo) DeletePendingInvite(ctx context.Context, orgID, inviteID string) error {
	return nil
}

func (f *fakeAppOrgRepo) DeleteMember(ctx context.Context, orgID, memberUserID string) error {
	return nil
}

func (f *fakeAppOrgRepo) DeleteInvitePermanently(ctx context.Context, orgID, inviteID string) error {
	return nil
}

func (f *fakeAppOrgRepo) UpdateMemberStatus(ctx context.Context, orgID, memberUserID, status string) error {
	return nil
}

func (f *fakeAppOrgRepo) RemoveMember(ctx context.Context, orgID, memberUserID string) error {
	return nil
}

func (f *fakeAppOrgRepo) HasMemberWithEmail(ctx context.Context, orgID, email string) (bool, error) {
	return false, nil
}

func (f *fakeAppOrgRepo) HasPendingInviteWithEmail(ctx context.Context, orgID, email, excludeInviteID string) (bool, error) {
	return false, nil
}

func (f *fakeAppOrgRepo) AcceptPendingInvitesForEmail(ctx context.Context, email, userID string) error {
	return nil
}

func (f *fakeAppOrgRepo) AcceptInvite(ctx context.Context, inviteID, userID string) (*domain.AppOrgListItem, error) {
	return nil, nil
}

func (f *fakeAppOrgRepo) ListTeams(ctx context.Context, orgID string, filters domain.TeamListFilters, limit, offset int) ([]*domain.TeamListItem, int, error) {
	total := len(f.teams)
	if offset >= total {
		return []*domain.TeamListItem{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return f.teams[offset:end], total, nil
}

func (f *fakeAppOrgRepo) CreateTeam(ctx context.Context, orgID, name, createdByUserID string) (*domain.TeamListItem, error) {
	team := &domain.TeamListItem{
		ID:   "team-1",
		Name: name,
		CreatedBy: domain.TeamCreatedBy{ID: createdByUserID, FirstName: "Jane", LastName: "Doe"},
	}
	f.teams = append(f.teams, team)
	return team, nil
}

func (f *fakeAppOrgRepo) GetTeam(ctx context.Context, orgID, teamID string) (*domain.TeamListItem, error) {
	for _, team := range f.teams {
		if team.ID == teamID {
			return team, nil
		}
	}
	return nil, nil
}

func (f *fakeAppOrgRepo) ListTeamMembers(ctx context.Context, teamID, orgID string, filters domain.OrgUserListFilters, limit, offset int) ([]*domain.OrgUserListItem, int, error) {
	if filters.TeamMembership == "not_in" {
		filters.TeamID = teamID
		return f.ListOrgUsers(ctx, orgID, filters, limit, offset)
	}

	total := len(f.teamMembers) + len(f.teamInviteMembers)
	combined := append([]*domain.OrgUserListItem{}, f.teamMembers...)
	combined = append(combined, f.teamInviteMembers...)
	if offset >= total {
		return []*domain.OrgUserListItem{}, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return combined[offset:end], total, nil
}

func (f *fakeAppOrgRepo) AddTeamMembers(ctx context.Context, teamID string, userIDs []string) error {
	return nil
}

func (f *fakeAppOrgRepo) RemoveTeamMember(ctx context.Context, teamID, userID string) error {
	return nil
}

func (f *fakeAppOrgRepo) BulkRemoveTeamMembers(ctx context.Context, teamID string, userIDs []string) error {
	return nil
}

func (f *fakeAppOrgRepo) IsTeamMember(ctx context.Context, teamID, userID string) (bool, error) {
	return true, nil
}

func (f *fakeAppOrgRepo) IsTeamInviteMember(ctx context.Context, teamID, inviteID string) (bool, error) {
	for _, member := range f.teamInviteMembers {
		if member.ID == "invite:"+inviteID {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeAppOrgRepo) AreOrgMembers(ctx context.Context, orgID string, userIDs []string) (bool, error) {
	return true, nil
}

func (f *fakeAppOrgRepo) AreOrgInvites(ctx context.Context, orgID string, inviteIDs []string) (bool, error) {
	return true, nil
}

type fakeUserRepo struct{}

func (f *fakeUserRepo) Create(ctx context.Context, user *domain.User) error { return nil }
func (f *fakeUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return nil, nil
}
func (f *fakeUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return nil, nil
}
func (f *fakeUserRepo) GetByGoogleID(ctx context.Context, googleID string) (*domain.User, error) {
	return nil, nil
}
func (f *fakeUserRepo) Update(ctx context.Context, user *domain.User) error { return nil }

func TestListMembersReturnsMembersForOrgMember(t *testing.T) {
	svc := NewAppOrgService(
		&fakeAppRepo{app: &domain.App{Slug: "surveillance-pro", Status: "available"}},
		&fakeAppOrgRepo{
			org: &domain.AppOrgListItem{ID: "org-1", Name: "Acme", Role: "owner", Status: "active"},
			members: []*domain.AppOrgMemberListItem{
				{ID: "user-1", FirstName: "Jane", LastName: "Doe", Email: "jane@example.com", Role: "owner", Status: "active"},
			},
		},
		&fakeUserRepo{},
	)

	members, err := svc.ListMembers(context.Background(), "surveillance-pro", "user-1", "org-1", pagination.Params{Page: 1, Limit: 20}, domain.OrgUserListFilters{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(members.Users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(members.Users))
	}
	if members.Users[0].Email != "jane@example.com" {
		t.Fatalf("unexpected user email %q", members.Users[0].Email)
	}
	if members.Pagination.Total != 1 {
		t.Fatalf("expected total 1, got %d", members.Pagination.Total)
	}
}

func TestListMembersNotFoundWhenNotMember(t *testing.T) {
	svc := NewAppOrgService(
		&fakeAppRepo{app: &domain.App{Slug: "surveillance-pro", Status: "available"}},
		&fakeAppOrgRepo{org: nil},
		&fakeUserRepo{},
	)

	_, err := svc.ListMembers(context.Background(), "surveillance-pro", "user-1", "org-1", pagination.Params{Page: 1, Limit: 20}, domain.OrgUserListFilters{})
	if err != ErrOrgNotFound {
		t.Fatalf("expected ErrOrgNotFound, got %v", err)
	}
}

func TestListMembersRejectsInvalidStatusFilter(t *testing.T) {
	svc := NewAppOrgService(
		&fakeAppRepo{app: &domain.App{Slug: "surveillance-pro", Status: "available"}},
		&fakeAppOrgRepo{
			org: &domain.AppOrgListItem{ID: "org-1", Name: "Acme", Role: "owner", Status: "active"},
		},
		&fakeUserRepo{},
	)

	_, err := svc.ListMembers(
		context.Background(),
		"surveillance-pro",
		"user-1",
		"org-1",
		pagination.Params{Page: 1, Limit: 20},
		domain.OrgUserListFilters{Status: "invalid"},
	)
	if err != ErrInvalidMemberStatus {
		t.Fatalf("expected ErrInvalidMemberStatus, got %v", err)
	}
}

func TestListTeamsReturnsTeamsForOrgMember(t *testing.T) {
	svc := NewAppOrgService(
		&fakeAppRepo{app: &domain.App{Slug: "surveillance-pro", Status: "available"}},
		&fakeAppOrgRepo{
			org: &domain.AppOrgListItem{ID: "org-1", Name: "Acme", Role: "owner", Status: "active"},
			teams: []*domain.TeamListItem{
				{
					ID:          "team-1",
					Name:        "Engineering",
					MemberCount: 2,
					CreatedBy:   domain.TeamCreatedBy{ID: "user-1", FirstName: "Jane", LastName: "Doe"},
				},
			},
		},
		&fakeUserRepo{},
	)

	result, err := svc.ListTeams(context.Background(), "surveillance-pro", "user-1", "org-1", pagination.Params{Page: 1, Limit: 20}, domain.TeamListFilters{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Teams) != 1 {
		t.Fatalf("expected 1 team, got %d", len(result.Teams))
	}
	if result.Teams[0].Name != "Engineering" {
		t.Fatalf("unexpected team name %q", result.Teams[0].Name)
	}
}

func TestCreateTeamRequiresOwner(t *testing.T) {
	svc := NewAppOrgService(
		&fakeAppRepo{app: &domain.App{Slug: "surveillance-pro", Status: "available"}},
		&fakeAppOrgRepo{
			org:     &domain.AppOrgListItem{ID: "org-1", Name: "Acme", Role: "member", Status: "active"},
			isOwner: false,
		},
		&fakeUserRepo{},
	)

	_, err := svc.CreateTeam(context.Background(), "surveillance-pro", "user-1", "org-1", "Engineering")
	if err != ErrNotOrgOwner {
		t.Fatalf("expected ErrNotOrgOwner, got %v", err)
	}
}

func TestListTeamMembersNotInReturnsOrgUsersNotOnTeam(t *testing.T) {
	repo := &fakeAppOrgRepo{
		org: &domain.AppOrgListItem{ID: "org-1", Name: "Acme", Role: "owner", Status: "active"},
		members: []*domain.AppOrgMemberListItem{
			{ID: "owner-1", FirstName: "Owner", LastName: "User", Email: "owner@eol.dev", Role: "owner", Status: "active"},
			{ID: "member-1", FirstName: "Member", LastName: "One", Email: "one@eol.dev", Role: "member", Status: "active"},
			{ID: "member-2", FirstName: "Member", LastName: "Two", Email: "two@eol.dev", Role: "member", Status: "active"},
		},
		pendingInvites: []*domain.AppOrgPendingInviteListItem{
			{ID: "invite-1", FirstName: "Pending", LastName: "User", Email: "pending@eol.dev", Status: "pending"},
		},
		teams: []*domain.TeamListItem{{ID: "team-1", Name: "Engineering"}},
		teamMembers: []*domain.OrgUserListItem{
			{ID: "member-1", FirstName: "Member", LastName: "One", Email: "one@eol.dev", Role: "member", Status: "active"},
		},
	}
	svc := NewAppOrgService(
		&fakeAppRepo{app: &domain.App{Slug: "surveillance-pro", Status: "available"}},
		repo,
		&fakeUserRepo{},
	)

	result, err := svc.ListTeamMembers(
		context.Background(),
		"surveillance-pro",
		"user-1",
		"org-1",
		"team-1",
		pagination.Params{Page: 1, Limit: 20},
		domain.OrgUserListFilters{TeamMembership: "not_in"},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Pagination.Total != 3 {
		t.Fatalf("expected 3 users not on team, got %d", result.Pagination.Total)
	}
}

func TestListTeamMembersNotFoundWhenTeamMissing(t *testing.T) {
	svc := NewAppOrgService(
		&fakeAppRepo{app: &domain.App{Slug: "surveillance-pro", Status: "available"}},
		&fakeAppOrgRepo{
			org:   &domain.AppOrgListItem{ID: "org-1", Name: "Acme", Role: "owner", Status: "active"},
			teams: []*domain.TeamListItem{},
		},
		&fakeUserRepo{},
	)

	_, err := svc.ListTeamMembers(context.Background(), "surveillance-pro", "user-1", "org-1", "team-1", pagination.Params{Page: 1, Limit: 20}, domain.OrgUserListFilters{})
	if err != ErrTeamNotFound {
		t.Fatalf("expected ErrTeamNotFound, got %v", err)
	}
}
