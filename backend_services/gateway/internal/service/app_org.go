package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/eternal-orbit-labs/gateway/internal/domain"
	"github.com/eternal-orbit-labs/gateway/internal/pagination"
	"github.com/eternal-orbit-labs/gateway/internal/repository"
)

var (
	ErrAppNotFound         = errors.New("app not found")
	ErrAppNotAvailable     = errors.New("app is not available")
	ErrOrgNotFound         = errors.New("organization not found")
	ErrInviteNotFound      = errors.New("invite not found")
	ErrInviteNotForUser    = errors.New("invite is not for this user")
	ErrInviteNotPending    = errors.New("invite is not pending")
	ErrNotOrgOwner         = errors.New("not organization owner")
	ErrOrgNameExists       = repository.ErrOrgNameExists
	ErrEmailAlreadyMember  = errors.New("user with this email is already a member")
	ErrEmailAlreadyInvited = errors.New("invite already pending for this email")
	ErrUserNotFound         = errors.New("user not found")
	ErrCannotRemoveOwner    = errors.New("cannot remove organization owner")
	ErrCannotUpdateOwner    = errors.New("cannot update organization owner")
	ErrInvalidMemberStatus  = errors.New("invalid member status")
	ErrMemberNotFound       = errors.New("member not found")
	ErrTeamNotFound         = errors.New("team not found")
	ErrTeamNameExists       = repository.ErrTeamNameExists
	ErrTeamMemberNotFound   = errors.New("team member not found")
	ErrUserNotOrgMember     = errors.New("user is not an organization member")
	ErrInviteNotOrgMember   = errors.New("invite is not in this organization")
)

type AppOrgMembersResult struct {
	Users      []*domain.OrgUserListItem `json:"users"`
	Pagination pagination.Meta           `json:"pagination"`
}

type AppOrgTeamsResult struct {
	Teams      []*domain.TeamListItem `json:"teams"`
	Pagination pagination.Meta        `json:"pagination"`
}

type AppOrgListResult struct {
	Organizations []*domain.AppOrgListItem      `json:"organizations"`
	Invites       []*domain.AppOrgInviteListItem `json:"invites"`
}

type AppOrgService struct {
	apps domain.AppRepository
	orgs domain.AppOrgRepository
	users domain.UserRepository
}

func NewAppOrgService(apps domain.AppRepository, orgs domain.AppOrgRepository, users domain.UserRepository) *AppOrgService {
	return &AppOrgService{apps: apps, orgs: orgs, users: users}
}

func (s *AppOrgService) requireAvailableApp(ctx context.Context, appSlug string) (*domain.App, error) {
	app, err := s.apps.GetBySlug(ctx, appSlug)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, ErrAppNotFound
	}
	if app.Status != "available" {
		return nil, ErrAppNotAvailable
	}
	return app, nil
}

func (s *AppOrgService) ListForUser(ctx context.Context, appSlug, userID, userEmail string) (*AppOrgListResult, error) {
	if _, err := s.requireAvailableApp(ctx, appSlug); err != nil {
		return nil, err
	}

	orgs, err := s.orgs.ListMembershipsForUser(ctx, appSlug, userID)
	if err != nil {
		return nil, err
	}
	invites, err := s.orgs.ListPendingInvitesForEmail(ctx, appSlug, userEmail)
	if err != nil {
		return nil, err
	}

	if orgs == nil {
		orgs = []*domain.AppOrgListItem{}
	}
	if invites == nil {
		invites = []*domain.AppOrgInviteListItem{}
	}

	return &AppOrgListResult{Organizations: orgs, Invites: invites}, nil
}

func (s *AppOrgService) Create(ctx context.Context, appSlug, userID, name string) (*domain.AppOrgListItem, error) {
	if _, err := s.requireAvailableApp(ctx, appSlug); err != nil {
		return nil, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if len(name) > 100 {
		return nil, fmt.Errorf("name must be at most 100 characters")
	}

	return s.orgs.CreateOrgWithOwner(ctx, appSlug, userID, name)
}

func (s *AppOrgService) GetOrg(ctx context.Context, appSlug, userID, orgID string) (*domain.AppOrgListItem, error) {
	if _, err := s.requireAvailableApp(ctx, appSlug); err != nil {
		return nil, err
	}

	org, err := s.orgs.GetOrgForMember(ctx, appSlug, userID, orgID)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, ErrOrgNotFound
	}
	return org, nil
}

func (s *AppOrgService) AcceptInvite(ctx context.Context, appSlug, userID, userEmail, inviteID string) (*domain.AppOrgListItem, error) {
	if _, err := s.requireAvailableApp(ctx, appSlug); err != nil {
		return nil, err
	}

	invite, inviteAppSlug, err := s.orgs.GetInviteByID(ctx, inviteID)
	if err != nil {
		return nil, err
	}
	if invite == nil || inviteAppSlug != appSlug {
		return nil, ErrInviteNotFound
	}
	if invite.Status != "pending" {
		return nil, fmt.Errorf("invite is not pending")
	}
	if !strings.EqualFold(strings.TrimSpace(invite.Email), strings.TrimSpace(userEmail)) {
		return nil, ErrInviteNotForUser
	}

	org, err := s.orgs.AcceptInvite(ctx, inviteID, userID)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, ErrInviteNotFound
	}
	return org, nil
}

func (s *AppOrgService) CreateInvite(ctx context.Context, appSlug, userID, orgID, email, firstName, lastName string, phone *string) (*domain.AppOrgInvite, error) {
	if _, err := s.requireAvailableApp(ctx, appSlug); err != nil {
		return nil, err
	}

	if err := s.requireOrgOwner(ctx, appSlug, userID, orgID); err != nil {
		return nil, err
	}

	email = strings.TrimSpace(email)
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	if err := s.ensureEmailAvailableForOrg(ctx, orgID, email, ""); err != nil {
		return nil, err
	}

	return s.orgs.CreateInvite(ctx, orgID, email, userID, firstName, lastName, phone)
}

func (s *AppOrgService) AddMember(
	ctx context.Context,
	appSlug, userID, orgID, email, firstName, lastName string,
	phone *string,
) error {
	if _, err := s.requireAvailableApp(ctx, appSlug); err != nil {
		return err
	}

	if err := s.requireOrgOwner(ctx, appSlug, userID, orgID); err != nil {
		return err
	}

	email = strings.TrimSpace(email)
	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)
	if email == "" {
		return fmt.Errorf("email is required")
	}

	hasMember, err := s.orgs.HasMemberWithEmail(ctx, orgID, email)
	if err != nil {
		return err
	}
	if hasMember {
		return ErrEmailAlreadyMember
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return err
	}
	if user == nil {
		if err := s.ensureEmailAvailableForOrg(ctx, orgID, email, ""); err != nil {
			return err
		}
		deactivated, err := s.orgs.GetDeactivatedInviteForOrg(ctx, orgID, email)
		if err != nil {
			return err
		}
		if deactivated != nil {
			_, err := s.orgs.UpdatePendingInvite(ctx, orgID, deactivated.ID, email, firstName, lastName, phone, true)
			return err
		}
		_, err = s.orgs.CreateInvite(ctx, orgID, email, userID, firstName, lastName, phone)
		return err
	}

	user.FirstName = firstName
	user.LastName = lastName
	user.Phone = phone
	if err := s.users.Update(ctx, user); err != nil {
		return err
	}

	return s.orgs.AddActiveMember(ctx, orgID, user.ID)
}

func (s *AppOrgService) UpdateInvite(
	ctx context.Context,
	appSlug, userID, orgID, inviteID, email, firstName, lastName string,
	phone *string,
	status string,
) (*domain.AppOrgInvite, error) {
	if _, err := s.requireAvailableApp(ctx, appSlug); err != nil {
		return nil, err
	}

	if err := s.requireOrgOwner(ctx, appSlug, userID, orgID); err != nil {
		return nil, err
	}

	invite, appSlugFromInvite, err := s.orgs.GetInviteByID(ctx, inviteID)
	if err != nil {
		return nil, err
	}
	if invite == nil || invite.OrgID != orgID || appSlugFromInvite != appSlug {
		return nil, ErrInviteNotFound
	}
	if invite.Status != "pending" && invite.Status != "deactive" {
		return nil, ErrInviteNotPending
	}

	email = strings.TrimSpace(email)
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	active := status != "deactive"
	if active {
		if err := s.ensureEmailAvailableForOrg(ctx, orgID, email, inviteID); err != nil {
			return nil, err
		}
	}

	updated, err := s.orgs.UpdatePendingInvite(ctx, orgID, inviteID, email, firstName, lastName, phone, active)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrInviteNotFound
	}
	return updated, nil
}

func (s *AppOrgService) DeleteInvite(ctx context.Context, appSlug, userID, orgID, inviteID string) error {
	if _, err := s.requireAvailableApp(ctx, appSlug); err != nil {
		return err
	}

	if err := s.requireOrgOwner(ctx, appSlug, userID, orgID); err != nil {
		return err
	}

	invite, appSlugFromInvite, err := s.orgs.GetInviteByID(ctx, inviteID)
	if err != nil {
		return err
	}
	if invite == nil || invite.OrgID != orgID || appSlugFromInvite != appSlug {
		return ErrInviteNotFound
	}
	if invite.Status == "deactive" {
		return s.orgs.DeleteInvitePermanently(ctx, orgID, inviteID)
	}
	if invite.Status != "pending" {
		return ErrInviteNotPending
	}

	return s.orgs.DeactivateInvite(ctx, orgID, inviteID)
}

func (s *AppOrgService) UpdateMemberStatus(
	ctx context.Context,
	appSlug, userID, orgID, memberUserID, status string,
) error {
	if _, err := s.requireAvailableApp(ctx, appSlug); err != nil {
		return err
	}

	if err := s.requireOrgOwner(ctx, appSlug, userID, orgID); err != nil {
		return err
	}

	if status != "active" && status != "deactive" {
		return ErrInvalidMemberStatus
	}

	members, err := s.orgs.ListMembers(ctx, orgID)
	if err != nil {
		return err
	}

	var target *domain.AppOrgMemberListItem
	for _, member := range members {
		if member.ID == memberUserID {
			target = member
			break
		}
	}
	if target == nil {
		return ErrMemberNotFound
	}
	if target.Role == "owner" {
		return ErrCannotUpdateOwner
	}
	if target.Status == status {
		return nil
	}

	return s.orgs.UpdateMemberStatus(ctx, orgID, memberUserID, status)
}

func (s *AppOrgService) RemoveMember(ctx context.Context, appSlug, userID, orgID, memberUserID string) error {
	if _, err := s.requireAvailableApp(ctx, appSlug); err != nil {
		return err
	}

	if err := s.requireOrgOwner(ctx, appSlug, userID, orgID); err != nil {
		return err
	}

	members, err := s.orgs.ListMembers(ctx, orgID)
	if err != nil {
		return err
	}

	var target *domain.AppOrgMemberListItem
	for _, member := range members {
		if member.ID == memberUserID {
			target = member
			break
		}
	}
	if target == nil {
		return ErrMemberNotFound
	}
	if target.Role == "owner" {
		return ErrCannotRemoveOwner
	}
	if target.Status == "deactive" {
		return s.orgs.DeleteMember(ctx, orgID, memberUserID)
	}
	if target.Status != "active" {
		return ErrMemberNotFound
	}

	return s.orgs.RemoveMember(ctx, orgID, memberUserID)
}

func (s *AppOrgService) requireOrgOwner(ctx context.Context, appSlug, userID, orgID string) error {
	org, err := s.orgs.GetOrgForMember(ctx, appSlug, userID, orgID)
	if err != nil {
		return err
	}
	if org == nil {
		return ErrOrgNotFound
	}

	isOwner, err := s.orgs.IsOwner(ctx, orgID, userID)
	if err != nil {
		return err
	}
	if !isOwner {
		return ErrNotOrgOwner
	}
	return nil
}

func (s *AppOrgService) ensureEmailAvailableForOrg(ctx context.Context, orgID, email, excludeInviteID string) error {
	hasMember, err := s.orgs.HasMemberWithEmail(ctx, orgID, email)
	if err != nil {
		return err
	}
	if hasMember {
		return ErrEmailAlreadyMember
	}

	hasInvite, err := s.orgs.HasPendingInviteWithEmail(ctx, orgID, email, excludeInviteID)
	if err != nil {
		return err
	}
	if hasInvite {
		return ErrEmailAlreadyInvited
	}
	return nil
}

func normalizeOrgUserListFilters(filters domain.OrgUserListFilters) (domain.OrgUserListFilters, error) {
	filters.Q = strings.TrimSpace(filters.Q)
	filters.FirstName = strings.TrimSpace(filters.FirstName)
	filters.LastName = strings.TrimSpace(filters.LastName)
	filters.Email = strings.TrimSpace(filters.Email)
	filters.Phone = strings.ReplaceAll(strings.TrimSpace(filters.Phone), " ", "")
	filters.Status = strings.TrimSpace(filters.Status)
	if filters.Status != "" && filters.Status != "active" && filters.Status != "deactive" {
		return filters, ErrInvalidMemberStatus
	}
	filters.TeamMembership = strings.TrimSpace(filters.TeamMembership)
	filters.TeamID = strings.TrimSpace(filters.TeamID)
	if filters.TeamMembership == "" {
		filters.TeamMembership = "in"
	}
	if filters.TeamMembership != "in" && filters.TeamMembership != "not_in" {
		return filters, fmt.Errorf("invalid team_membership")
	}
	if filters.TeamMembership == "not_in" && filters.TeamID == "" {
		filters.TeamMembership = "in"
	}
	return filters, nil
}

func (s *AppOrgService) ListMembers(
	ctx context.Context,
	appSlug, userID, orgID string,
	params pagination.Params,
	filters domain.OrgUserListFilters,
) (*AppOrgMembersResult, error) {
	if _, err := s.requireAvailableApp(ctx, appSlug); err != nil {
		return nil, err
	}

	org, err := s.orgs.GetOrgForMember(ctx, appSlug, userID, orgID)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, ErrOrgNotFound
	}

	filters, err = normalizeOrgUserListFilters(filters)
	if err != nil {
		return nil, err
	}

	users, total, err := s.orgs.ListOrgUsers(ctx, orgID, filters, params.Limit, params.Offset())
	if err != nil {
		return nil, err
	}
	if users == nil {
		users = []*domain.OrgUserListItem{}
	}

	return &AppOrgMembersResult{
		Users:      users,
		Pagination: pagination.NewMeta(total, params.Page, params.Limit),
	}, nil
}

func normalizeTeamListFilters(filters domain.TeamListFilters) domain.TeamListFilters {
	filters.Q = strings.TrimSpace(filters.Q)
	filters.Name = strings.TrimSpace(filters.Name)
	return filters
}

func (s *AppOrgService) ListTeams(
	ctx context.Context,
	appSlug, userID, orgID string,
	params pagination.Params,
	filters domain.TeamListFilters,
) (*AppOrgTeamsResult, error) {
	if _, err := s.requireAvailableApp(ctx, appSlug); err != nil {
		return nil, err
	}

	org, err := s.orgs.GetOrgForMember(ctx, appSlug, userID, orgID)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, ErrOrgNotFound
	}

	filters = normalizeTeamListFilters(filters)

	teams, total, err := s.orgs.ListTeams(ctx, orgID, filters, params.Limit, params.Offset())
	if err != nil {
		return nil, err
	}
	if teams == nil {
		teams = []*domain.TeamListItem{}
	}

	return &AppOrgTeamsResult{
		Teams:      teams,
		Pagination: pagination.NewMeta(total, params.Page, params.Limit),
	}, nil
}

func (s *AppOrgService) CreateTeam(ctx context.Context, appSlug, userID, orgID, name string) (*domain.TeamListItem, error) {
	if _, err := s.requireAvailableApp(ctx, appSlug); err != nil {
		return nil, err
	}

	if err := s.requireOrgOwner(ctx, appSlug, userID, orgID); err != nil {
		return nil, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if len(name) > 100 {
		return nil, fmt.Errorf("name must be at most 100 characters")
	}

	return s.orgs.CreateTeam(ctx, orgID, name, userID)
}

func (s *AppOrgService) GetTeam(ctx context.Context, appSlug, userID, orgID, teamID string) (*domain.TeamListItem, error) {
	if _, err := s.requireAvailableApp(ctx, appSlug); err != nil {
		return nil, err
	}

	org, err := s.orgs.GetOrgForMember(ctx, appSlug, userID, orgID)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, ErrOrgNotFound
	}

	team, err := s.orgs.GetTeam(ctx, orgID, teamID)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return nil, ErrTeamNotFound
	}
	return team, nil
}

func (s *AppOrgService) ListTeamMembers(
	ctx context.Context,
	appSlug, userID, orgID, teamID string,
	params pagination.Params,
	filters domain.OrgUserListFilters,
) (*AppOrgMembersResult, error) {
	if _, err := s.requireAvailableApp(ctx, appSlug); err != nil {
		return nil, err
	}

	org, err := s.orgs.GetOrgForMember(ctx, appSlug, userID, orgID)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, ErrOrgNotFound
	}

	team, err := s.orgs.GetTeam(ctx, orgID, teamID)
	if err != nil {
		return nil, err
	}
	if team == nil {
		return nil, ErrTeamNotFound
	}

	if strings.TrimSpace(filters.TeamMembership) == "not_in" {
		filters.TeamMembership = "not_in"
		filters.TeamID = teamID
	}

	filters, err = normalizeOrgUserListFilters(filters)
	if err != nil {
		return nil, err
	}

	users, total, err := s.orgs.ListTeamMembers(ctx, teamID, orgID, filters, params.Limit, params.Offset())
	if err != nil {
		return nil, err
	}
	if users == nil {
		users = []*domain.OrgUserListItem{}
	}

	return &AppOrgMembersResult{
		Users:      users,
		Pagination: pagination.NewMeta(total, params.Page, params.Limit),
	}, nil
}

func (s *AppOrgService) AddTeamMembers(ctx context.Context, appSlug, userID, orgID, teamID string, userIDs []string) error {
	if _, err := s.requireAvailableApp(ctx, appSlug); err != nil {
		return err
	}

	if err := s.requireOrgOwner(ctx, appSlug, userID, orgID); err != nil {
		return err
	}

	team, err := s.orgs.GetTeam(ctx, orgID, teamID)
	if err != nil {
		return err
	}
	if team == nil {
		return ErrTeamNotFound
	}

	if len(userIDs) == 0 {
		return fmt.Errorf("user_ids is required")
	}

	memberUserIDs, inviteIDs := splitTeamMemberIDs(userIDs)

	if len(memberUserIDs) > 0 {
		ok, err := s.orgs.AreOrgMembers(ctx, orgID, memberUserIDs)
		if err != nil {
			return err
		}
		if !ok {
			return ErrUserNotOrgMember
		}
	}

	if len(inviteIDs) > 0 {
		ok, err := s.orgs.AreOrgInvites(ctx, orgID, inviteIDs)
		if err != nil {
			return err
		}
		if !ok {
			return ErrInviteNotOrgMember
		}
	}

	return s.orgs.AddTeamMembers(ctx, teamID, userIDs)
}

func (s *AppOrgService) RemoveTeamMember(ctx context.Context, appSlug, userID, orgID, teamID, memberUserID string) error {
	if _, err := s.requireAvailableApp(ctx, appSlug); err != nil {
		return err
	}

	if err := s.requireOrgOwner(ctx, appSlug, userID, orgID); err != nil {
		return err
	}

	team, err := s.orgs.GetTeam(ctx, orgID, teamID)
	if err != nil {
		return err
	}
	if team == nil {
		return ErrTeamNotFound
	}

	isMember, err := isTeamMemberRef(ctx, s.orgs, teamID, memberUserID)
	if err != nil {
		return err
	}
	if !isMember {
		return ErrTeamMemberNotFound
	}

	if err := s.orgs.RemoveTeamMember(ctx, teamID, memberUserID); err != nil {
		if strings.Contains(err.Error(), "team member not found") {
			return ErrTeamMemberNotFound
		}
		return err
	}
	return nil
}

func (s *AppOrgService) BulkRemoveTeamMembers(ctx context.Context, appSlug, userID, orgID, teamID string, userIDs []string) error {
	if _, err := s.requireAvailableApp(ctx, appSlug); err != nil {
		return err
	}

	if err := s.requireOrgOwner(ctx, appSlug, userID, orgID); err != nil {
		return err
	}

	team, err := s.orgs.GetTeam(ctx, orgID, teamID)
	if err != nil {
		return err
	}
	if team == nil {
		return ErrTeamNotFound
	}

	if len(userIDs) == 0 {
		return fmt.Errorf("user_ids is required")
	}

	if err := s.orgs.BulkRemoveTeamMembers(ctx, teamID, userIDs); err != nil {
		if strings.Contains(err.Error(), "team member not found") {
			return ErrTeamMemberNotFound
		}
		return err
	}
	return nil
}

func splitTeamMemberIDs(ids []string) (userIDs []string, inviteIDs []string) {
	for _, id := range ids {
		if strings.HasPrefix(id, "invite:") {
			inviteIDs = append(inviteIDs, strings.TrimPrefix(id, "invite:"))
			continue
		}
		userIDs = append(userIDs, id)
	}
	return userIDs, inviteIDs
}

func isTeamMemberRef(ctx context.Context, orgs domain.AppOrgRepository, teamID, memberRef string) (bool, error) {
	if strings.HasPrefix(memberRef, "invite:") {
		return orgs.IsTeamInviteMember(ctx, teamID, strings.TrimPrefix(memberRef, "invite:"))
	}
	return orgs.IsTeamMember(ctx, teamID, memberRef)
}
