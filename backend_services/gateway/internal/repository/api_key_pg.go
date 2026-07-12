package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/eternal-orbit-labs/gateway/internal/domain"
)

type PersonalAPIKeyPG struct {
	pool *pgxpool.Pool
}

func NewPersonalAPIKeyPG(pool *pgxpool.Pool) *PersonalAPIKeyPG {
	return &PersonalAPIKeyPG{pool: pool}
}

func (r *PersonalAPIKeyPG) Create(ctx context.Context, k *domain.PersonalAPIKey, secureValue string) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO personal_api_keys (user_id, label, prefix, secure_value)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`, k.UserID, k.Label, k.Prefix, secureValue).Scan(&k.ID, &k.CreatedAt)
	if err != nil {
		return fmt.Errorf("create api key: %w", err)
	}
	return nil
}

func (r *PersonalAPIKeyPG) GetBySecureValue(ctx context.Context, secureValue string) (*domain.PersonalAPIKey, error) {
	var k domain.PersonalAPIKey
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, label, prefix, last_used_at, created_at
		FROM personal_api_keys WHERE secure_value = $1
	`, secureValue).Scan(&k.ID, &k.UserID, &k.Label, &k.Prefix, &k.LastUsedAt, &k.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get api key by hash: %w", err)
	}
	return &k, nil
}

func (r *PersonalAPIKeyPG) ListByUserID(ctx context.Context, userID string) ([]domain.PersonalAPIKey, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, label, prefix, last_used_at, created_at
		FROM personal_api_keys WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	var keys []domain.PersonalAPIKey
	for rows.Next() {
		var k domain.PersonalAPIKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.Label, &k.Prefix, &k.LastUsedAt, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (r *PersonalAPIKeyPG) Delete(ctx context.Context, id, userID string) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM personal_api_keys WHERE id = $1 AND user_id = $2", id, userID)
	if err != nil {
		return fmt.Errorf("delete api key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("api key not found")
	}
	return nil
}

func (r *PersonalAPIKeyPG) TouchLastUsed(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "UPDATE personal_api_keys SET last_used_at = now() WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("touch api key: %w", err)
	}
	return nil
}
