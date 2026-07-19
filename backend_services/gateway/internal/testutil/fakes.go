package testutil

import (
	"context"
	"fmt"
	"sync"

	"github.com/eternal-orbit-labs/gateway/internal/domain"
)

// ── FakeUserRepo ─────────────────────────────────────────────────────

type FakeUserRepo struct {
	mu      sync.Mutex
	users   map[string]*domain.User // by ID
	counter int
}

func NewFakeUserRepo() *FakeUserRepo {
	return &FakeUserRepo{users: make(map[string]*domain.User)}
}

func (r *FakeUserRepo) Create(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counter++
	if user.ID == "" {
		user.ID = fmt.Sprintf("user-%d", r.counter)
	}
	r.users[user.ID] = user
	return nil
}

func (r *FakeUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.users[id], nil
}

func (r *FakeUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, nil
}

func (r *FakeUserRepo) GetByGoogleID(ctx context.Context, googleID string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.GoogleID == googleID {
			return u, nil
		}
	}
	return nil, nil
}

func (r *FakeUserRepo) Update(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[user.ID] = user
	return nil
}

// ── FakeRefreshSessionRepo ───────────────────────────────────────────

type FakeRefreshSessionRepo struct {
	mu       sync.Mutex
	sessions map[string]*domain.RefreshSession // by token hash
	counter  int
}

func NewFakeRefreshSessionRepo() *FakeRefreshSessionRepo {
	return &FakeRefreshSessionRepo{sessions: make(map[string]*domain.RefreshSession)}
}

func (r *FakeRefreshSessionRepo) Create(ctx context.Context, session *domain.RefreshSession) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counter++
	if session.ID == "" {
		session.ID = fmt.Sprintf("sess-%d", r.counter)
	}
	r.sessions[session.TokenHash] = session
	return nil
}

func (r *FakeRefreshSessionRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sessions[tokenHash], nil
}

func (r *FakeRefreshSessionRepo) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, tokenHash)
	return nil
}

func (r *FakeRefreshSessionRepo) DeleteAllForUser(ctx context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for hash, s := range r.sessions {
		if s.UserID == userID {
			delete(r.sessions, hash)
		}
	}
	return nil
}

// ── FakeSessionCache ─────────────────────────────────────────────────

type FakeSessionCache struct {
	mu    sync.Mutex
	cache map[string]*domain.RefreshSession
}

func NewFakeSessionCache() *FakeSessionCache {
	return &FakeSessionCache{cache: make(map[string]*domain.RefreshSession)}
}

func (c *FakeSessionCache) Get(ctx context.Context, tokenHash string) (*domain.RefreshSession, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cache[tokenHash], nil
}

func (c *FakeSessionCache) Set(ctx context.Context, tokenHash string, session *domain.RefreshSession) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[tokenHash] = session
	return nil
}

func (c *FakeSessionCache) Delete(ctx context.Context, tokenHash string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, tokenHash)
	return nil
}

// ── FakeRateLimiter ──────────────────────────────────────────────────

type FakeRateLimiter struct {
	mu       sync.Mutex
	counters map[string]int64
}

func NewFakeRateLimiter() *FakeRateLimiter {
	return &FakeRateLimiter{counters: make(map[string]int64)}
}

func (r *FakeRateLimiter) IncrementLoginAttempts(ctx context.Context, key string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counters[key]++
	return r.counters[key], nil
}

func (r *FakeRateLimiter) GetLoginAttempts(ctx context.Context, key string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.counters[key], nil
}

// ── FakePersonalAPIKeyRepo ───────────────────────────────────────────

type FakePersonalAPIKeyRepo struct {
	mu          sync.Mutex
	keys        map[string]*domain.PersonalAPIKey // by ID
	bySecure    map[string]*domain.PersonalAPIKey // by secureValue
	counter     int
}

func NewFakePersonalAPIKeyRepo() *FakePersonalAPIKeyRepo {
	return &FakePersonalAPIKeyRepo{
		keys:     make(map[string]*domain.PersonalAPIKey),
		bySecure: make(map[string]*domain.PersonalAPIKey),
	}
}

func (r *FakePersonalAPIKeyRepo) Create(ctx context.Context, key *domain.PersonalAPIKey, secureValue string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counter++
	if key.ID == "" {
		key.ID = fmt.Sprintf("apikey-%d", r.counter)
	}
	r.keys[key.ID] = key
	r.bySecure[secureValue] = key
	return nil
}

func (r *FakePersonalAPIKeyRepo) GetBySecureValue(ctx context.Context, secureValue string) (*domain.PersonalAPIKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.bySecure[secureValue], nil
}

func (r *FakePersonalAPIKeyRepo) ListByUserID(ctx context.Context, userID string) ([]domain.PersonalAPIKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []domain.PersonalAPIKey
	for _, k := range r.keys {
		if k.UserID == userID {
			result = append(result, *k)
		}
	}
	return result, nil
}

func (r *FakePersonalAPIKeyRepo) Delete(ctx context.Context, id, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k, ok := r.keys[id]
	if !ok || k.UserID != userID {
		return nil
	}
	// Remove from bySecure map as well
	for sv, key := range r.bySecure {
		if key.ID == id {
			delete(r.bySecure, sv)
			break
		}
	}
	delete(r.keys, id)
	return nil
}

func (r *FakePersonalAPIKeyRepo) TouchLastUsed(ctx context.Context, id string) error {
	return nil // no-op
}

// ── FakeAppRepo ──────────────────────────────────────────────────────

type FakeAppRepo struct {
	Apps []*domain.App
}

func NewFakeAppRepo() *FakeAppRepo {
	return &FakeAppRepo{}
}

func (r *FakeAppRepo) List(ctx context.Context, q, cursor string, limit int) ([]*domain.App, string, error) {
	return r.Apps, "", nil
}

func (r *FakeAppRepo) GetBySlug(ctx context.Context, slug string) (*domain.App, error) {
	for _, a := range r.Apps {
		if a.Slug == slug {
			return a, nil
		}
	}
	return nil, nil
}

// ── FakeAppOrgRepo ───────────────────────────────────────────────────

type FakeAppOrgRepo struct{}

func NewFakeAppOrgRepo() *FakeAppOrgRepo {
	return &FakeAppOrgRepo{}
}

func (r *FakeAppOrgRepo) ListMembershipsForUser(ctx context.Context, appSlug, userID string) ([]*domain.AppOrgListItem, error) {
	return nil, nil
}

func (r *FakeAppOrgRepo) ListPendingInvitesForEmail(ctx context.Context, appSlug, email string) ([]*domain.AppOrgInviteListItem, error) {
	return nil, nil
}

func (r *FakeAppOrgRepo) CreateOrgWithOwner(ctx context.Context, appSlug, userID, name string) (*domain.AppOrgListItem, error) {
	return nil, nil
}

func (r *FakeAppOrgRepo) GetOrgForMember(ctx context.Context, appSlug, userID, orgID string) (*domain.AppOrgListItem, error) {
	return nil, nil
}

func (r *FakeAppOrgRepo) ListMembers(ctx context.Context, orgID string) ([]*domain.AppOrgMemberListItem, error) {
	return nil, nil
}

func (r *FakeAppOrgRepo) ListOrgUsers(ctx context.Context, orgID string, filters domain.OrgUserListFilters, limit, offset int) ([]*domain.OrgUserListItem, int, error) {
	return nil, 0, nil
}

func (r *FakeAppOrgRepo) ListPendingInvitesForOrg(ctx context.Context, orgID string) ([]*domain.AppOrgPendingInviteListItem, error) {
	return nil, nil
}

func (r *FakeAppOrgRepo) IsMember(ctx context.Context, orgID, userID string) (bool, error) {
	return false, nil
}

func (r *FakeAppOrgRepo) IsOwner(ctx context.Context, orgID, userID string) (bool, error) {
	return false, nil
}

func (r *FakeAppOrgRepo) GetInviteByID(ctx context.Context, inviteID string) (*domain.AppOrgInvite, string, error) {
	return nil, "", nil
}

func (r *FakeAppOrgRepo) CreateInvite(ctx context.Context, orgID, email, invitedByUserID string, firstName, lastName string, phone *string) (*domain.AppOrgInvite, error) {
	return nil, nil
}

func (r *FakeAppOrgRepo) AddActiveMember(ctx context.Context, orgID, memberUserID string) error {
	return nil
}

func (r *FakeAppOrgRepo) UpdatePendingInvite(ctx context.Context, orgID, inviteID, email, firstName, lastName string, phone *string, active bool) (*domain.AppOrgInvite, error) {
	return nil, nil
}

func (r *FakeAppOrgRepo) GetDeactivatedInviteForOrg(ctx context.Context, orgID, email string) (*domain.AppOrgInvite, error) {
	return nil, nil
}

func (r *FakeAppOrgRepo) DeactivateInvite(ctx context.Context, orgID, inviteID string) error {
	return nil
}

func (r *FakeAppOrgRepo) DeletePendingInvite(ctx context.Context, orgID, inviteID string) error {
	return nil
}

func (r *FakeAppOrgRepo) DeleteMember(ctx context.Context, orgID, memberUserID string) error {
	return nil
}

func (r *FakeAppOrgRepo) DeleteInvitePermanently(ctx context.Context, orgID, inviteID string) error {
	return nil
}

func (r *FakeAppOrgRepo) UpdateMemberStatus(ctx context.Context, orgID, memberUserID, status string) error {
	return nil
}

func (r *FakeAppOrgRepo) RemoveMember(ctx context.Context, orgID, memberUserID string) error {
	return nil
}

func (r *FakeAppOrgRepo) HasMemberWithEmail(ctx context.Context, orgID, email string) (bool, error) {
	return false, nil
}

func (r *FakeAppOrgRepo) HasPendingInviteWithEmail(ctx context.Context, orgID, email, excludeInviteID string) (bool, error) {
	return false, nil
}

func (r *FakeAppOrgRepo) AcceptInvite(ctx context.Context, inviteID, userID string) (*domain.AppOrgListItem, error) {
	return nil, nil
}

func (r *FakeAppOrgRepo) AcceptPendingInvitesForEmail(ctx context.Context, email, userID string) error {
	return nil
}

func (r *FakeAppOrgRepo) ListTeams(ctx context.Context, orgID string, filters domain.TeamListFilters, limit, offset int) ([]*domain.TeamListItem, int, error) {
	return nil, 0, nil
}

func (r *FakeAppOrgRepo) CreateTeam(ctx context.Context, orgID, name, createdByUserID string) (*domain.TeamListItem, error) {
	return nil, nil
}

func (r *FakeAppOrgRepo) GetTeam(ctx context.Context, orgID, teamID string) (*domain.TeamListItem, error) {
	return nil, nil
}

func (r *FakeAppOrgRepo) ListTeamMembers(ctx context.Context, teamID, orgID string, filters domain.OrgUserListFilters, limit, offset int) ([]*domain.OrgUserListItem, int, error) {
	return nil, 0, nil
}

func (r *FakeAppOrgRepo) AddTeamMembers(ctx context.Context, teamID string, userIDs []string) error {
	return nil
}

func (r *FakeAppOrgRepo) RemoveTeamMember(ctx context.Context, teamID, userID string) error {
	return nil
}

func (r *FakeAppOrgRepo) BulkRemoveTeamMembers(ctx context.Context, teamID string, userIDs []string) error {
	return nil
}

func (r *FakeAppOrgRepo) IsTeamMember(ctx context.Context, teamID, userID string) (bool, error) {
	return false, nil
}

func (r *FakeAppOrgRepo) IsTeamInviteMember(ctx context.Context, teamID, inviteID string) (bool, error) {
	return false, nil
}

func (r *FakeAppOrgRepo) AreOrgMembers(ctx context.Context, orgID string, userIDs []string) (bool, error) {
	return false, nil
}

func (r *FakeAppOrgRepo) AreOrgInvites(ctx context.Context, orgID string, inviteIDs []string) (bool, error) {
	return false, nil
}
