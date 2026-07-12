package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eternal-orbit-labs/gateway/internal/domain"
)

type RefreshSessionPG struct {
	pool *pgxpool.Pool
}

func NewRefreshSessionPG(pool *pgxpool.Pool) *RefreshSessionPG {
	return &RefreshSessionPG{pool: pool}
}

func (r *RefreshSessionPG) Create(ctx context.Context, s *domain.RefreshSession) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO refresh_sessions (user_id, token_hash, ip_address, user_agent, expires_at)
		VALUES ($1, $2, $3::inet, $4, $5)
		RETURNING id, created_at
	`, s.UserID, s.TokenHash, nullableIP(s.IPAddress), s.UserAgent, s.ExpiresAt,
	).Scan(&s.ID, &s.CreatedAt)
	if err != nil {
		return fmt.Errorf("create refresh session: %w", err)
	}
	return nil
}

func (r *RefreshSessionPG) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshSession, error) {
	var s domain.RefreshSession
	var ip *string
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, token_hash, ip_address::text, user_agent, created_at, expires_at
		FROM refresh_sessions WHERE token_hash = $1 AND expires_at > now()
	`, tokenHash).Scan(&s.ID, &s.UserID, &s.TokenHash, &ip, &s.UserAgent, &s.CreatedAt, &s.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get refresh session: %w", err)
	}
	if ip != nil {
		s.IPAddress = *ip
	}
	return &s, nil
}

func (r *RefreshSessionPG) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM refresh_sessions WHERE token_hash = $1", tokenHash)
	if err != nil {
		return fmt.Errorf("delete refresh session: %w", err)
	}
	return nil
}

func (r *RefreshSessionPG) DeleteAllForUser(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM refresh_sessions WHERE user_id = $1", userID)
	if err != nil {
		return fmt.Errorf("delete all refresh sessions: %w", err)
	}
	return nil
}

func nullableIP(ip string) any {
	if ip == "" {
		return nil
	}
	return ip
}
