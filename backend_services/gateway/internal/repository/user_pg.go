package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eternal-orbit-labs/gateway/internal/domain"
)

type UserPG struct {
	pool *pgxpool.Pool
}

func NewUserPG(pool *pgxpool.Pool) *UserPG {
	return &UserPG{pool: pool}
}

func (r *UserPG) Create(ctx context.Context, u *domain.User) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, first_name, last_name, avatar_url, email_verified, google_id)
		VALUES (lower($1), $2, $3, $4, $5, $6, NULLIF($7, ''))
		RETURNING id, created_at, updated_at
	`, u.Email, u.PasswordHash, u.FirstName, u.LastName, u.AvatarURL, u.EmailVerified, u.GoogleID,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "users_email_unique") {
			return fmt.Errorf("email already exists")
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *UserPG) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return r.scanUser(r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, first_name, last_name, phone, avatar_url, email_verified, COALESCE(google_id,''), created_at, updated_at
		FROM users WHERE id = $1
	`, id))
}

func (r *UserPG) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.scanUser(r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, first_name, last_name, phone, avatar_url, email_verified, COALESCE(google_id,''), created_at, updated_at
		FROM users WHERE lower(email) = lower($1)
	`, email))
}

func (r *UserPG) GetByGoogleID(ctx context.Context, googleID string) (*domain.User, error) {
	return r.scanUser(r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, first_name, last_name, phone, avatar_url, email_verified, COALESCE(google_id,''), created_at, updated_at
		FROM users WHERE google_id = $1
	`, googleID))
}

func (r *UserPG) Update(ctx context.Context, u *domain.User) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users SET email=lower($2), password_hash=$3, first_name=$4, last_name=$5,
		                 avatar_url=$6, email_verified=$7, google_id=NULLIF($8,''), updated_at=now()
		WHERE id=$1
	`, u.ID, u.Email, u.PasswordHash, u.FirstName, u.LastName, u.AvatarURL, u.EmailVerified, u.GoogleID)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

func (r *UserPG) scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName, &u.Phone,
		&u.AvatarURL, &u.EmailVerified, &u.GoogleID, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return &u, nil
}
