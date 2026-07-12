package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eternal-orbit-labs/gateway/internal/domain"
)

var ErrOrgNameExists = errors.New("organization name already exists")
var ErrTeamNameExists = errors.New("team name already exists")

type AppOrgPG struct {
	pool *pgxpool.Pool
}

func NewAppOrgPG(pool *pgxpool.Pool) *AppOrgPG {
	return &AppOrgPG{pool: pool}
}

func (r *AppOrgPG) ListMembershipsForUser(ctx context.Context, appSlug, userID string) ([]*domain.AppOrgListItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT o.id, o.name, m.role
		FROM app_org_memberships m
		JOIN app_organizations o ON o.id = m.org_id
		WHERE o.app_slug = $1 AND m.user_id = $2 AND m.status = 'active'
		ORDER BY o.name, o.id
	`, appSlug, userID)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}
	defer rows.Close()

	var items []*domain.AppOrgListItem
	for rows.Next() {
		var item domain.AppOrgListItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Role); err != nil {
			return nil, fmt.Errorf("scan membership: %w", err)
		}
		item.Status = "active"
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate memberships: %w", err)
	}
	return items, nil
}

func (r *AppOrgPG) ListPendingInvitesForEmail(ctx context.Context, appSlug, email string) ([]*domain.AppOrgInviteListItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT i.id, i.org_id, o.name, i.status, i.created_at
		FROM app_org_invites i
		JOIN app_organizations o ON o.id = i.org_id
		WHERE o.app_slug = $1
		  AND lower(i.email) = lower($2)
		  AND i.status = 'pending'
		ORDER BY i.created_at, i.id
	`, appSlug, email)
	if err != nil {
		return nil, fmt.Errorf("list invites: %w", err)
	}
	defer rows.Close()

	var items []*domain.AppOrgInviteListItem
	for rows.Next() {
		var item domain.AppOrgInviteListItem
		if err := rows.Scan(&item.ID, &item.OrgID, &item.OrgName, &item.Status, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan invite: %w", err)
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate invites: %w", err)
	}
	return items, nil
}

func (r *AppOrgPG) CreateOrgWithOwner(ctx context.Context, appSlug, userID, name string) (*domain.AppOrgListItem, error) {
	name = strings.TrimSpace(name)
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var orgID string
	err = tx.QueryRow(ctx, `
		INSERT INTO app_organizations (app_slug, name, created_by_user_id)
		VALUES ($1, $2, $3)
		RETURNING id
	`, appSlug, name, userID).Scan(&orgID)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrOrgNameExists
		}
		return nil, fmt.Errorf("insert org: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO app_org_memberships (org_id, user_id, role)
		VALUES ($1, $2, 'owner')
	`, orgID, userID)
	if err != nil {
		return nil, fmt.Errorf("insert membership: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &domain.AppOrgListItem{
		ID:     orgID,
		Name:   name,
		Role:   "owner",
		Status: "active",
	}, nil
}

func (r *AppOrgPG) GetOrgForMember(ctx context.Context, appSlug, userID, orgID string) (*domain.AppOrgListItem, error) {
	var item domain.AppOrgListItem
	err := r.pool.QueryRow(ctx, `
		SELECT o.id, o.name, m.role
		FROM app_organizations o
		JOIN app_org_memberships m ON m.org_id = o.id
		WHERE o.app_slug = $1 AND o.id = $2 AND m.user_id = $3 AND m.status = 'active'
	`, appSlug, orgID, userID).Scan(&item.ID, &item.Name, &item.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get org for member: %w", err)
	}
	item.Status = "active"
	return &item, nil
}

func (r *AppOrgPG) ListMembers(ctx context.Context, orgID string) ([]*domain.AppOrgMemberListItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.first_name, u.last_name, u.email, u.phone, m.role, m.status, m.joined_at
		FROM app_org_memberships m
		JOIN users u ON u.id = m.user_id
		WHERE m.org_id = $1
		ORDER BY u.last_name, u.first_name, u.id
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var items []*domain.AppOrgMemberListItem
	for rows.Next() {
		var item domain.AppOrgMemberListItem
		if err := rows.Scan(
			&item.ID, &item.FirstName, &item.LastName, &item.Email,
			&item.Phone, &item.Role, &item.Status, &item.JoinedAt,
		); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate members: %w", err)
	}
	return items, nil
}

const orgUsersCTE = `
	WITH org_users AS (
		SELECT
			u.id::text AS id,
			u.first_name,
			u.last_name,
			u.email,
			u.phone,
			m.role,
			m.status,
			u.last_name AS sort_last,
			u.first_name AS sort_first
		FROM app_org_memberships m
		JOIN users u ON u.id = m.user_id
		WHERE m.org_id = $1

		UNION ALL

		SELECT
			'invite:' || i.id::text,
			i.first_name,
			i.last_name,
			i.email,
			i.phone,
			'member',
			CASE WHEN i.status = 'pending' THEN 'active' ELSE 'deactive' END,
			i.last_name,
			i.first_name
		FROM app_org_invites i
		WHERE i.org_id = $1 AND i.status IN ('pending', 'deactive')
	)`

func orgUserFilterWhere(filters domain.OrgUserListFilters, startArg int) (string, []any, int) {
	clauses := make([]string, 0, 6)
	args := make([]any, 0, 6)
	arg := startArg

	if filters.Q != "" {
		pattern := "%" + filters.Q + "%"
		clauses = append(clauses, fmt.Sprintf(`(
			first_name ILIKE $%d OR last_name ILIKE $%d OR email ILIKE $%d OR COALESCE(phone, '') ILIKE $%d
		)`, arg, arg, arg, arg))
		args = append(args, pattern)
		arg++
	}
	if filters.FirstName != "" {
		clauses = append(clauses, fmt.Sprintf("first_name ILIKE $%d", arg))
		args = append(args, "%"+filters.FirstName+"%")
		arg++
	}
	if filters.LastName != "" {
		clauses = append(clauses, fmt.Sprintf("last_name ILIKE $%d", arg))
		args = append(args, "%"+filters.LastName+"%")
		arg++
	}
	if filters.Email != "" {
		clauses = append(clauses, fmt.Sprintf("email ILIKE $%d", arg))
		args = append(args, "%"+filters.Email+"%")
		arg++
	}
	if filters.Phone != "" {
		clauses = append(clauses, fmt.Sprintf("COALESCE(phone, '') ILIKE $%d", arg))
		args = append(args, "%"+filters.Phone+"%")
		arg++
	}
	if filters.Status != "" {
		clauses = append(clauses, fmt.Sprintf("status = $%d", arg))
		args = append(args, filters.Status)
		arg++
	}

	if len(clauses) == 0 {
		return "", args, arg
	}
	return " WHERE " + strings.Join(clauses, " AND "), args, arg
}

func orgUserListWhere(filters domain.OrgUserListFilters, startArg int) (string, []any, int) {
	clauses := make([]string, 0, 8)
	args := make([]any, 0, 8)
	arg := startArg

	if filters.TeamMembership == "not_in" && filters.TeamID != "" {
		clauses = append(clauses, fmt.Sprintf(`(
			(id LIKE 'invite:%%' AND NOT EXISTS (
				SELECT 1 FROM app_org_team_invite_memberships tim
				WHERE tim.team_id = $%d AND tim.invite_id = substring(id from 8)::uuid
			))
			OR
			(id NOT LIKE 'invite:%%' AND NOT EXISTS (
				SELECT 1 FROM app_org_team_memberships tm
				WHERE tm.team_id = $%d AND tm.user_id = id::uuid
			))
		)`, arg, arg))
		args = append(args, filters.TeamID)
		arg++
	}

	filterWhere, filterArgs, nextArg := orgUserFilterWhere(filters, arg)
	if filterWhere != "" {
		clauses = append(clauses, strings.TrimPrefix(filterWhere, " WHERE "))
		args = append(args, filterArgs...)
		arg = nextArg
	}

	if len(clauses) == 0 {
		return "", args, arg
	}
	return " WHERE " + strings.Join(clauses, " AND "), args, arg
}

func (r *AppOrgPG) ListOrgUsers(ctx context.Context, orgID string, filters domain.OrgUserListFilters, limit, offset int) ([]*domain.OrgUserListItem, int, error) {
	where, filterArgs, nextArg := orgUserListWhere(filters, 2)

	countQuery := orgUsersCTE + `
		SELECT COUNT(*) FROM org_users` + where
	countArgs := append([]any{orgID}, filterArgs...)

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count org users: %w", err)
	}

	listQuery := orgUsersCTE + `
		SELECT id, first_name, last_name, email, phone, role, status
		FROM org_users` + where + fmt.Sprintf(`
		ORDER BY sort_last, sort_first, id
		LIMIT $%d OFFSET $%d`, nextArg, nextArg+1)
	listArgs := append([]any{orgID}, filterArgs...)
	listArgs = append(listArgs, limit, offset)

	rows, err := r.pool.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list org users: %w", err)
	}
	defer rows.Close()

	var items []*domain.OrgUserListItem
	for rows.Next() {
		var item domain.OrgUserListItem
		if err := rows.Scan(
			&item.ID, &item.FirstName, &item.LastName, &item.Email,
			&item.Phone, &item.Role, &item.Status,
		); err != nil {
			return nil, 0, fmt.Errorf("scan org user: %w", err)
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate org users: %w", err)
	}
	return items, total, nil
}

func (r *AppOrgPG) ListPendingInvitesForOrg(ctx context.Context, orgID string) ([]*domain.AppOrgPendingInviteListItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, email, first_name, last_name, phone, status, created_at
		FROM app_org_invites
		WHERE org_id = $1 AND status = 'pending'
		ORDER BY created_at, id
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list pending invites: %w", err)
	}
	defer rows.Close()

	var items []*domain.AppOrgPendingInviteListItem
	for rows.Next() {
		var item domain.AppOrgPendingInviteListItem
		if err := rows.Scan(
			&item.ID, &item.Email, &item.FirstName, &item.LastName,
			&item.Phone, &item.Status, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan pending invite: %w", err)
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending invites: %w", err)
	}
	return items, nil
}

func (r *AppOrgPG) IsMember(ctx context.Context, orgID, userID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM app_org_memberships
			WHERE org_id = $1 AND user_id = $2 AND status = 'active'
		)
	`, orgID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("is member: %w", err)
	}
	return exists, nil
}

func (r *AppOrgPG) IsOwner(ctx context.Context, orgID, userID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM app_org_memberships
			WHERE org_id = $1 AND user_id = $2 AND role = 'owner' AND status = 'active'
		)
	`, orgID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("is owner: %w", err)
	}
	return exists, nil
}

func (r *AppOrgPG) GetInviteByID(ctx context.Context, inviteID string) (*domain.AppOrgInvite, string, error) {
	var inv domain.AppOrgInvite
	var appSlug string
	err := r.pool.QueryRow(ctx, `
		SELECT i.id, i.org_id, i.email, i.first_name, i.last_name, i.phone, i.invited_by_user_id, i.status, i.created_at, i.accepted_at, o.app_slug
		FROM app_org_invites i
		JOIN app_organizations o ON o.id = i.org_id
		WHERE i.id = $1
	`, inviteID).Scan(&inv.ID, &inv.OrgID, &inv.Email, &inv.FirstName, &inv.LastName, &inv.Phone,
		&inv.InvitedByUserID, &inv.Status, &inv.CreatedAt, &inv.AcceptedAt, &appSlug)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("get invite: %w", err)
	}
	return &inv, appSlug, nil
}

func (r *AppOrgPG) CreateInvite(ctx context.Context, orgID, email, invitedByUserID, firstName, lastName string, phone *string) (*domain.AppOrgInvite, error) {
	email = strings.TrimSpace(email)
	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)
	var inv domain.AppOrgInvite
	err := r.pool.QueryRow(ctx, `
		INSERT INTO app_org_invites (org_id, email, first_name, last_name, phone, invited_by_user_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, org_id, email, first_name, last_name, phone, invited_by_user_id, status, created_at, accepted_at
	`, orgID, email, firstName, lastName, phone, invitedByUserID).Scan(
		&inv.ID, &inv.OrgID, &inv.Email, &inv.FirstName, &inv.LastName, &inv.Phone,
		&inv.InvitedByUserID, &inv.Status, &inv.CreatedAt, &inv.AcceptedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("invite already pending for this email")
		}
		return nil, fmt.Errorf("create invite: %w", err)
	}
	return &inv, nil
}

func (r *AppOrgPG) AddActiveMember(ctx context.Context, orgID, memberUserID string) error {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO app_org_memberships (org_id, user_id, role, status)
		VALUES ($1, $2, 'member', 'active')
		ON CONFLICT (org_id, user_id) DO UPDATE
		SET status = 'active'
		WHERE app_org_memberships.role <> 'owner'
	`, orgID, memberUserID)
	if err != nil {
		return fmt.Errorf("add active member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("member not found")
	}
	return nil
}

func (r *AppOrgPG) UpdatePendingInvite(
	ctx context.Context,
	orgID, inviteID, email, firstName, lastName string,
	phone *string,
	active bool,
) (*domain.AppOrgInvite, error) {
	email = strings.TrimSpace(email)
	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)
	inviteStatus := "deactive"
	if active {
		inviteStatus = "pending"
	}
	var inv domain.AppOrgInvite
	err := r.pool.QueryRow(ctx, `
		UPDATE app_org_invites
		SET email = $3, first_name = $4, last_name = $5, phone = $6, status = $7
		WHERE id = $1 AND org_id = $2 AND status IN ('pending', 'deactive')
		RETURNING id, org_id, email, first_name, last_name, phone, invited_by_user_id, status, created_at, accepted_at
	`, inviteID, orgID, email, firstName, lastName, phone, inviteStatus).Scan(
		&inv.ID, &inv.OrgID, &inv.Email, &inv.FirstName, &inv.LastName, &inv.Phone,
		&inv.InvitedByUserID, &inv.Status, &inv.CreatedAt, &inv.AcceptedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("invite already pending for this email")
		}
		return nil, fmt.Errorf("update invite: %w", err)
	}
	return &inv, nil
}

func (r *AppOrgPG) GetDeactivatedInviteForOrg(ctx context.Context, orgID, email string) (*domain.AppOrgInvite, error) {
	var inv domain.AppOrgInvite
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, email, first_name, last_name, phone, invited_by_user_id, status, created_at, accepted_at
		FROM app_org_invites
		WHERE org_id = $1 AND lower(email) = lower($2) AND status = 'deactive'
	`, orgID, email).Scan(
		&inv.ID, &inv.OrgID, &inv.Email, &inv.FirstName, &inv.LastName, &inv.Phone,
		&inv.InvitedByUserID, &inv.Status, &inv.CreatedAt, &inv.AcceptedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get deactivated invite: %w", err)
	}
	return &inv, nil
}

func (r *AppOrgPG) DeactivateInvite(ctx context.Context, orgID, inviteID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE app_org_invites
		SET status = 'deactive'
		WHERE id = $1 AND org_id = $2 AND status = 'pending'
	`, inviteID, orgID)
	if err != nil {
		return fmt.Errorf("deactivate invite: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("invite not found")
	}
	return nil
}

func (r *AppOrgPG) DeleteInvitePermanently(ctx context.Context, orgID, inviteID string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM app_org_invites
		WHERE id = $1 AND org_id = $2 AND status = 'deactive'
	`, inviteID, orgID)
	if err != nil {
		return fmt.Errorf("delete invite: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("invite not found")
	}
	return nil
}

func (r *AppOrgPG) DeletePendingInvite(ctx context.Context, orgID, inviteID string) error {
	return r.DeactivateInvite(ctx, orgID, inviteID)
}

func (r *AppOrgPG) DeleteMember(ctx context.Context, orgID, memberUserID string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM app_org_memberships
		WHERE org_id = $1 AND user_id = $2 AND role <> 'owner' AND status = 'deactive'
	`, orgID, memberUserID)
	if err != nil {
		return fmt.Errorf("delete member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("member not found")
	}
	return nil
}

func (r *AppOrgPG) UpdateMemberStatus(ctx context.Context, orgID, memberUserID, status string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE app_org_memberships
		SET status = $3
		WHERE org_id = $1 AND user_id = $2 AND role <> 'owner'
	`, orgID, memberUserID, status)
	if err != nil {
		return fmt.Errorf("update member status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("member not found")
	}
	return nil
}

func (r *AppOrgPG) RemoveMember(ctx context.Context, orgID, memberUserID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE app_org_memberships
		SET status = 'deactive'
		WHERE org_id = $1 AND user_id = $2 AND role <> 'owner' AND status = 'active'
	`, orgID, memberUserID)
	if err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("member not found")
	}
	return nil
}

func (r *AppOrgPG) HasMemberWithEmail(ctx context.Context, orgID, email string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM app_org_memberships m
			JOIN users u ON u.id = m.user_id
			WHERE m.org_id = $1 AND lower(u.email) = lower($2) AND m.status = 'active'
		)
	`, orgID, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("has member with email: %w", err)
	}
	return exists, nil
}

func (r *AppOrgPG) HasPendingInviteWithEmail(ctx context.Context, orgID, email, excludeInviteID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM app_org_invites
			WHERE org_id = $1
			  AND lower(email) = lower($2)
			  AND status = 'pending'
			  AND ($3 = '' OR id <> $3::uuid)
		)
	`, orgID, email, excludeInviteID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("has pending invite with email: %w", err)
	}
	return exists, nil
}

func (r *AppOrgPG) AcceptInvite(ctx context.Context, inviteID, userID string) (*domain.AppOrgListItem, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var (
		orgID   string
		orgName string
		status  string
	)
	err = tx.QueryRow(ctx, `
		SELECT i.status, i.org_id, o.name
		FROM app_org_invites i
		JOIN app_organizations o ON o.id = i.org_id
		WHERE i.id = $1
		FOR UPDATE OF i
	`, inviteID).Scan(&status, &orgID, &orgName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lock invite: %w", err)
	}
	if status != "pending" {
		return nil, fmt.Errorf("invite is not pending")
	}

	var membershipStatus string
	err = tx.QueryRow(ctx, `
		SELECT status FROM app_org_memberships WHERE org_id = $1 AND user_id = $2
	`, orgID, userID).Scan(&membershipStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = tx.Exec(ctx, `
			INSERT INTO app_org_memberships (org_id, user_id, role, status)
			VALUES ($1, $2, 'member', 'active')
		`, orgID, userID)
		if err != nil {
			return nil, fmt.Errorf("insert membership: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("check membership: %w", err)
	} else if membershipStatus == "deactive" {
		_, err = tx.Exec(ctx, `
			UPDATE app_org_memberships
			SET status = 'active', role = 'member'
			WHERE org_id = $1 AND user_id = $2
		`, orgID, userID)
		if err != nil {
			return nil, fmt.Errorf("reactivate membership: %w", err)
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE app_org_invites
		SET status = 'accepted', accepted_at = now()
		WHERE id = $1
	`, inviteID)
	if err != nil {
		return nil, fmt.Errorf("update invite: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO app_org_team_memberships (team_id, user_id)
		SELECT tim.team_id, $2
		FROM app_org_team_invite_memberships tim
		WHERE tim.invite_id = $1
		ON CONFLICT (team_id, user_id) DO NOTHING
	`, inviteID, userID)
	if err != nil {
		return nil, fmt.Errorf("promote team invite memberships: %w", err)
	}

	_, err = tx.Exec(ctx, `
		DELETE FROM app_org_team_invite_memberships
		WHERE invite_id = $1
	`, inviteID)
	if err != nil {
		return nil, fmt.Errorf("clear team invite memberships: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &domain.AppOrgListItem{
		ID:     orgID,
		Name:   orgName,
		Role:   "member",
		Status: "active",
	}, nil
}

func (r *AppOrgPG) AcceptPendingInvitesForEmail(ctx context.Context, email, userID string) error {
	rows, err := r.pool.Query(ctx, `
		SELECT id FROM app_org_invites
		WHERE lower(email) = lower($1) AND status = 'pending'
		ORDER BY created_at, id
	`, email)
	if err != nil {
		return fmt.Errorf("list pending invites for email: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var inviteID string
		if err := rows.Scan(&inviteID); err != nil {
			return fmt.Errorf("scan invite id: %w", err)
		}
		if _, err := r.AcceptInvite(ctx, inviteID, userID); err != nil {
			return fmt.Errorf("accept invite %s: %w", inviteID, err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate pending invites: %w", err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func teamFilterWhere(filters domain.TeamListFilters, startArg int) (string, []any, int) {
	clauses := make([]string, 0, 2)
	args := make([]any, 0, 2)
	arg := startArg

	if filters.Q != "" {
		clauses = append(clauses, fmt.Sprintf("t.name ILIKE $%d", arg))
		args = append(args, "%"+filters.Q+"%")
		arg++
	}
	if filters.Name != "" {
		clauses = append(clauses, fmt.Sprintf("t.name ILIKE $%d", arg))
		args = append(args, "%"+filters.Name+"%")
		arg++
	}

	if len(clauses) == 0 {
		return "", args, arg
	}
	return " AND " + strings.Join(clauses, " AND "), args, arg
}

func (r *AppOrgPG) ListTeams(ctx context.Context, orgID string, filters domain.TeamListFilters, limit, offset int) ([]*domain.TeamListItem, int, error) {
	where, filterArgs, nextArg := teamFilterWhere(filters, 2)

	countQuery := `
		SELECT COUNT(*)
		FROM app_org_teams t
		WHERE t.org_id = $1` + where
	countArgs := append([]any{orgID}, filterArgs...)

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count teams: %w", err)
	}

	listQuery := fmt.Sprintf(`
		SELECT
			t.id,
			t.name,
			(
				(SELECT COUNT(*) FROM app_org_team_memberships tm WHERE tm.team_id = t.id)
				+ (SELECT COUNT(*) FROM app_org_team_invite_memberships tim WHERE tim.team_id = t.id)
			),
			u.id,
			u.first_name,
			u.last_name,
			t.created_at
		FROM app_org_teams t
		JOIN users u ON u.id = t.created_by_user_id
		WHERE t.org_id = $1%s
		ORDER BY t.name, t.id
		LIMIT $%d OFFSET $%d`, where, nextArg, nextArg+1)
	listArgs := append([]any{orgID}, filterArgs...)
	listArgs = append(listArgs, limit, offset)

	rows, err := r.pool.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list teams: %w", err)
	}
	defer rows.Close()

	var items []*domain.TeamListItem
	for rows.Next() {
		var item domain.TeamListItem
		if err := rows.Scan(
			&item.ID, &item.Name, &item.MemberCount,
			&item.CreatedBy.ID, &item.CreatedBy.FirstName, &item.CreatedBy.LastName,
			&item.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan team: %w", err)
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate teams: %w", err)
	}
	return items, total, nil
}

func (r *AppOrgPG) CreateTeam(ctx context.Context, orgID, name, createdByUserID string) (*domain.TeamListItem, error) {
	name = strings.TrimSpace(name)
	var item domain.TeamListItem
	err := r.pool.QueryRow(ctx, `
		INSERT INTO app_org_teams (org_id, name, created_by_user_id)
		VALUES ($1, $2, $3)
		RETURNING id, name, created_at
	`, orgID, name, createdByUserID).Scan(&item.ID, &item.Name, &item.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrTeamNameExists
		}
		return nil, fmt.Errorf("create team: %w", err)
	}

	item.MemberCount = 0
	item.CreatedBy.ID = createdByUserID
	err = r.pool.QueryRow(ctx, `
		SELECT first_name, last_name FROM users WHERE id = $1
	`, createdByUserID).Scan(&item.CreatedBy.FirstName, &item.CreatedBy.LastName)
	if err != nil {
		return nil, fmt.Errorf("load creator: %w", err)
	}
	return &item, nil
}

func (r *AppOrgPG) GetTeam(ctx context.Context, orgID, teamID string) (*domain.TeamListItem, error) {
	var item domain.TeamListItem
	err := r.pool.QueryRow(ctx, `
		SELECT
			t.id,
			t.name,
			(
				(SELECT COUNT(*) FROM app_org_team_memberships tm WHERE tm.team_id = t.id)
				+ (SELECT COUNT(*) FROM app_org_team_invite_memberships tim WHERE tim.team_id = t.id)
			),
			u.id,
			u.first_name,
			u.last_name,
			t.created_at
		FROM app_org_teams t
		JOIN users u ON u.id = t.created_by_user_id
		WHERE t.org_id = $1 AND t.id = $2
	`, orgID, teamID).Scan(
		&item.ID, &item.Name, &item.MemberCount,
		&item.CreatedBy.ID, &item.CreatedBy.FirstName, &item.CreatedBy.LastName,
		&item.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get team: %w", err)
	}
	return &item, nil
}

const teamMembersBase = `
	SELECT
		u.id::text,
		u.first_name,
		u.last_name,
		u.email,
		u.phone,
		m.role,
		m.status,
		u.last_name AS sort_last,
		u.first_name AS sort_first
	FROM app_org_team_memberships tm
	JOIN users u ON u.id = tm.user_id
	JOIN app_org_memberships m ON m.org_id = $1 AND m.user_id = u.id
	WHERE tm.team_id = $2

	UNION ALL

	SELECT
		'invite:' || i.id::text,
		i.first_name,
		i.last_name,
		i.email,
		i.phone,
		'member',
		CASE WHEN i.status = 'pending' THEN 'active' ELSE 'deactive' END,
		i.last_name,
		i.first_name
	FROM app_org_team_invite_memberships tim
	JOIN app_org_invites i ON i.id = tim.invite_id
	WHERE tim.team_id = $2 AND i.org_id = $1`

func teamMemberFilterWhere(filters domain.OrgUserListFilters, startArg int) (string, []any, int) {
	clauses := make([]string, 0, 6)
	args := make([]any, 0, 6)
	arg := startArg

	if filters.Q != "" {
		pattern := "%" + filters.Q + "%"
		clauses = append(clauses, fmt.Sprintf(`(
			first_name ILIKE $%d OR last_name ILIKE $%d OR email ILIKE $%d OR COALESCE(phone, '') ILIKE $%d
		)`, arg, arg, arg, arg))
		args = append(args, pattern)
		arg++
	}
	if filters.FirstName != "" {
		clauses = append(clauses, fmt.Sprintf("first_name ILIKE $%d", arg))
		args = append(args, "%"+filters.FirstName+"%")
		arg++
	}
	if filters.LastName != "" {
		clauses = append(clauses, fmt.Sprintf("last_name ILIKE $%d", arg))
		args = append(args, "%"+filters.LastName+"%")
		arg++
	}
	if filters.Email != "" {
		clauses = append(clauses, fmt.Sprintf("email ILIKE $%d", arg))
		args = append(args, "%"+filters.Email+"%")
		arg++
	}
	if filters.Phone != "" {
		clauses = append(clauses, fmt.Sprintf("COALESCE(phone, '') ILIKE $%d", arg))
		args = append(args, "%"+filters.Phone+"%")
		arg++
	}
	if filters.Status != "" {
		clauses = append(clauses, fmt.Sprintf("status = $%d", arg))
		args = append(args, filters.Status)
		arg++
	}

	if len(clauses) == 0 {
		return "", args, arg
	}
	return " WHERE " + strings.Join(clauses, " AND "), args, arg
}

func (r *AppOrgPG) ListTeamMembers(ctx context.Context, teamID, orgID string, filters domain.OrgUserListFilters, limit, offset int) ([]*domain.OrgUserListItem, int, error) {
	membership := filters.TeamMembership
	if membership == "" {
		membership = "in"
	}

	if membership == "not_in" {
		filters.TeamMembership = "not_in"
		filters.TeamID = teamID
		return r.ListOrgUsers(ctx, orgID, filters, limit, offset)
	}

	baseQuery := teamMembersBase

	where, filterArgs, nextArg := teamMemberFilterWhere(filters, 3)

	countQuery := `SELECT COUNT(*) FROM (` + baseQuery + `) team_members` + where
	countArgs := append([]any{orgID, teamID}, filterArgs...)

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count team members: %w", err)
	}

	listQuery := `SELECT id, first_name, last_name, email, phone, role, status FROM (` +
		baseQuery + `) team_members` + where + fmt.Sprintf(`
	ORDER BY sort_last, sort_first, id
	LIMIT $%d OFFSET $%d`, nextArg, nextArg+1)
	listArgs := append([]any{orgID, teamID}, filterArgs...)
	listArgs = append(listArgs, limit, offset)

	rows, err := r.pool.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list team members: %w", err)
	}
	defer rows.Close()

	var items []*domain.OrgUserListItem
	for rows.Next() {
		var item domain.OrgUserListItem
		if err := rows.Scan(
			&item.ID, &item.FirstName, &item.LastName, &item.Email,
			&item.Phone, &item.Role, &item.Status,
		); err != nil {
			return nil, 0, fmt.Errorf("scan team member: %w", err)
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate team members: %w", err)
	}
	return items, total, nil
}

func (r *AppOrgPG) AddTeamMembers(ctx context.Context, teamID string, memberIDs []string) error {
	if len(memberIDs) == 0 {
		return nil
	}
	for _, memberID := range memberIDs {
		if strings.HasPrefix(memberID, "invite:") {
			inviteID := strings.TrimPrefix(memberID, "invite:")
			_, err := r.pool.Exec(ctx, `
				INSERT INTO app_org_team_invite_memberships (team_id, invite_id)
				VALUES ($1, $2)
				ON CONFLICT (team_id, invite_id) DO NOTHING
			`, teamID, inviteID)
			if err != nil {
				return fmt.Errorf("add team invite member: %w", err)
			}
			continue
		}
		_, err := r.pool.Exec(ctx, `
			INSERT INTO app_org_team_memberships (team_id, user_id)
			VALUES ($1, $2)
			ON CONFLICT (team_id, user_id) DO NOTHING
		`, teamID, memberID)
		if err != nil {
			return fmt.Errorf("add team member: %w", err)
		}
	}
	return nil
}

func (r *AppOrgPG) RemoveTeamMember(ctx context.Context, teamID, memberID string) error {
	if strings.HasPrefix(memberID, "invite:") {
		inviteID := strings.TrimPrefix(memberID, "invite:")
		tag, err := r.pool.Exec(ctx, `
			DELETE FROM app_org_team_invite_memberships
			WHERE team_id = $1 AND invite_id = $2
		`, teamID, inviteID)
		if err != nil {
			return fmt.Errorf("remove team invite member: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("team member not found")
		}
		return nil
	}

	tag, err := r.pool.Exec(ctx, `
		DELETE FROM app_org_team_memberships
		WHERE team_id = $1 AND user_id = $2
	`, teamID, memberID)
	if err != nil {
		return fmt.Errorf("remove team member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("team member not found")
	}
	return nil
}

func (r *AppOrgPG) BulkRemoveTeamMembers(ctx context.Context, teamID string, memberIDs []string) error {
	if len(memberIDs) == 0 {
		return nil
	}

	var userIDs []string
	var inviteIDs []string
	for _, memberID := range memberIDs {
		if strings.HasPrefix(memberID, "invite:") {
			inviteIDs = append(inviteIDs, strings.TrimPrefix(memberID, "invite:"))
			continue
		}
		userIDs = append(userIDs, memberID)
	}

	var affected int64
	if len(userIDs) > 0 {
		tag, err := r.pool.Exec(ctx, `
			DELETE FROM app_org_team_memberships
			WHERE team_id = $1 AND user_id = ANY($2)
		`, teamID, userIDs)
		if err != nil {
			return fmt.Errorf("bulk remove team members: %w", err)
		}
		affected += tag.RowsAffected()
	}
	if len(inviteIDs) > 0 {
		tag, err := r.pool.Exec(ctx, `
			DELETE FROM app_org_team_invite_memberships
			WHERE team_id = $1 AND invite_id = ANY($2)
		`, teamID, inviteIDs)
		if err != nil {
			return fmt.Errorf("bulk remove team invite members: %w", err)
		}
		affected += tag.RowsAffected()
	}
	if affected == 0 {
		return fmt.Errorf("team member not found")
	}
	return nil
}

func (r *AppOrgPG) IsTeamMember(ctx context.Context, teamID, userID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM app_org_team_memberships
			WHERE team_id = $1 AND user_id = $2
		)
	`, teamID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("is team member: %w", err)
	}
	return exists, nil
}

func (r *AppOrgPG) IsTeamInviteMember(ctx context.Context, teamID, inviteID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM app_org_team_invite_memberships
			WHERE team_id = $1 AND invite_id = $2
		)
	`, teamID, inviteID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("is team invite member: %w", err)
	}
	return exists, nil
}

func (r *AppOrgPG) AreOrgMembers(ctx context.Context, orgID string, userIDs []string) (bool, error) {
	if len(userIDs) == 0 {
		return true, nil
	}
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM app_org_memberships
		WHERE org_id = $1 AND user_id = ANY($2)
	`, orgID, userIDs).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check org members: %w", err)
	}
	return count == len(userIDs), nil
}

func (r *AppOrgPG) AreOrgInvites(ctx context.Context, orgID string, inviteIDs []string) (bool, error) {
	if len(inviteIDs) == 0 {
		return true, nil
	}
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM app_org_invites
		WHERE org_id = $1 AND id = ANY($2) AND status IN ('pending', 'deactive')
	`, orgID, inviteIDs).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check org invites: %w", err)
	}
	return count == len(inviteIDs), nil
}
